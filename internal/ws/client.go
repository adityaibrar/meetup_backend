package ws

import (
	"encoding/json"
	"log"
	"meetup_backend/models"
	"sync"
	"time"

	"github.com/gofiber/contrib/websocket"
	"gorm.io/gorm"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 4096 // 4KB
)

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	Hub *Hub

	// The websocket connection.
	Conn *websocket.Conn

	// Buffered channel of outbound messages.
	Send chan []byte

	// User ID derived from authentication
	UserID uint

	// Database connection for persistence/deletion
	DB *gorm.DB

	// Active Room Tracking
	ActiveRoomID uint
	mu           sync.Mutex
}

// WSMessage defines the structure of messages sent over WebSocket
type WSMessage struct {
	Type        string          `json:"type"` // 'chat', 'read', 'typing'
	ChatRoomID  uint            `json:"chat_room_id,omitempty"`
	RecipientID uint            `json:"recipient_id,omitempty"` // Optional if sending to specific user, but usually ChatRoomID is enough
	Content     string          `json:"content,omitempty"`
	MessageID   uint            `json:"message_id,omitempty"` // Used for 'read' receipts
	Payload     json.RawMessage `json:"payload,omitempty"`    // Flexible payload
	Product     json.RawMessage `json:"product,omitempty"`    // Product snapshot
}

// readPump pumps messages from the websocket connection to the hub.
func (c *Client) ReadPump() {
	defer func() {
		c.Hub.Unregister <- c
		c.Conn.Close()
	}()
	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error { c.Conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}

		// Process the message
		c.handleMessage(message)
	}
}

// writePump pumps messages from the hub to the websocket connection.
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued chat messages to the current websocket message.
			n := len(c.Send)
			for i := 0; i < n; i++ {
				w.Write(<-c.Send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Client) handleMessage(message []byte) {
	var wsMsg WSMessage
	if err := json.Unmarshal(message, &wsMsg); err != nil {
		log.Printf("Error unmarshalling message: %v", err)
		return
	}

	switch wsMsg.Type {
	case "chat":
		c.processChatMessage(&wsMsg)
	case "read":
		c.processReadReceipt(&wsMsg)
	case "join_room":
		c.mu.Lock()
		previousRoom := c.ActiveRoomID
		c.ActiveRoomID = wsMsg.ChatRoomID
		c.mu.Unlock()

		log.Printf("User %d joined room %d", c.UserID, wsMsg.ChatRoomID)

		// If leaving a previous room, notify participants of that room
		if previousRoom != 0 && previousRoom != wsMsg.ChatRoomID {
			c.broadcastRoomStatus(previousRoom, false)
		}

		// Notify other participants in the new room that this user joined
		c.broadcastRoomStatus(wsMsg.ChatRoomID, true)

		// Send unread messages for this specific room
		go c.SendUnreadMessagesForRoom(wsMsg.ChatRoomID)

	case "leave_room":
		c.mu.Lock()
		prevRoom := c.ActiveRoomID
		c.ActiveRoomID = 0
		c.mu.Unlock()

		log.Printf("User %d left room %d", c.UserID, prevRoom)

		// Notify other participants that this user left the room
		if prevRoom != 0 {
			c.broadcastRoomStatus(prevRoom, false)
		}
	}
}

// broadcastRoomStatus notifies other participants in a room about this user's presence
func (c *Client) broadcastRoomStatus(roomID uint, inRoom bool) {
	// Find other participants in this room
	var room models.ChatRoom
	if err := c.DB.Preload("Participants").First(&room, roomID).Error; err != nil {
		log.Printf("Error finding room for status broadcast: %v", err)
		return
	}

	statusJSON, _ := json.Marshal(map[string]interface{}{
		"type":         "room_status",
		"user_id":      c.UserID,
		"chat_room_id": roomID,
		"in_room":      inRoom,
	})

	// Send to all other participants
	for _, p := range room.Participants {
		if p.UserID != c.UserID {
			c.Hub.SendToUser(p.UserID, statusJSON)
		}
	}
}

func (c *Client) processChatMessage(wsMsg *WSMessage) {
	// 1. Find Chat Room and Participants (needed to know who to send to)
	var room models.ChatRoom
	if err := c.DB.Preload("Participants").First(&room, wsMsg.ChatRoomID).Error; err != nil {
		log.Printf("Error finding chat room: %v", err)
		return
	}

	// 2. Determine Recipient
	var recipientID uint
	for _, p := range room.Participants {
		if p.UserID != c.UserID {
			recipientID = p.UserID
			break
		}
	}

	// 3. Check if Recipient is currently IN the room (viewing the chat screen)
	recipientInRoom := false
	if recipientID != 0 {
		recipientInRoom = c.Hub.IsUserInRoom(recipientID, wsMsg.ChatRoomID)
	}

	// 4. Fetch Sender Info for JSON payload
	var sender models.User
	if err := c.DB.First(&sender, c.UserID).Error; err != nil {
		log.Printf("Error fetching sender info: %v", err)
	}

	log.Printf("Processing message. RecipientID: %d, InRoom: %v", recipientID, recipientInRoom)

	if recipientInRoom {
		// CASE 1: Recipient IS in room - Direct delivery, NO database save (true ephemeral)
		// Create a temporary message object for JSON (no DB save)
		tempMsg := map[string]interface{}{
			"id":           0, // No DB ID since not saved
			"chat_room_id": wsMsg.ChatRoomID,
			"sender_id":    c.UserID,
			"content":      wsMsg.Content,
			"media_type":   "text",
			"is_read":      true, // Already read since recipient is in room
			"created_at":   time.Now(),
			"sender":       sender,
			"product":      wsMsg.Product,
		}

		responseJSON, _ := json.Marshal(map[string]interface{}{
			"type":         "chat",
			"message":      tempMsg,
			"sender_id":    c.UserID,
			"chat_room_id": wsMsg.ChatRoomID,
			"product":      wsMsg.Product, // Include top-level for convenience if needed, but message.product is better
		})

		// Send to recipient
		c.Hub.SendToUser(recipientID, responseJSON)

		// Also send to sender so their UI shows the message with is_read=true
		c.Send <- responseJSON

		log.Printf("Message sent directly to recipient (ephemeral, no DB save)")
	} else {
		// CASE 2: Recipient NOT in room - Save to database for later retrieval
		newMsg := models.Message{
			ChatRoomID:  wsMsg.ChatRoomID,
			SenderID:    c.UserID,
			Content:     wsMsg.Content,
			IsRead:      false,
			Sender:      sender,
			ProductInfo: string(wsMsg.Product),
		}

		if err := c.DB.Omit("Sender").Create(&newMsg).Error; err != nil {
			log.Printf("Error saving message: %v", err)
			return
		}

		// Construct response for both Sender (Echo) and Recipient (Notification/Update)
		responseJSON, _ := json.Marshal(map[string]interface{}{
			"type":         "chat",
			"message":      newMsg,
			"sender_id":    c.UserID,
			"chat_room_id": wsMsg.ChatRoomID,
			"product":      wsMsg.Product, // Explicitly send as JSON object so client sees it
		})

		// Echo back to sender so they see it (Unread) and have the ID for future read receipt
		c.Send <- responseJSON

		// NEW: Always send to recipient if they are online, even if not "in room"
		// This ensures real-time updates for list view or if they are actually in room (false negative)
		if recipientID != 0 {
			c.Hub.SendToUser(recipientID, responseJSON)
			log.Printf("Message saved to DB AND sent to recipient %d (Real-time delivery)", recipientID)
		} else {
			log.Printf("Message saved to DB (ID: %d) - Recipient ID 0??", newMsg.ID)
		}
	}

	// 5. Update Chat Room Metadata (Last Message & Time)
	// This is crucial for the Chat List API (GetMyChats) to show the correct preview and order.
	if err := c.DB.Model(&models.ChatRoom{}).Where("id = ?", wsMsg.ChatRoomID).Updates(map[string]interface{}{
		"last_message_content": wsMsg.Content,
		"last_message_at":      time.Now(),
	}).Error; err != nil {
		log.Printf("Error updating chat room metadata: %v", err)
	}
}

func (c *Client) processReadReceipt(wsMsg *WSMessage) {
	// 1. "Delete on Read" Logic with Notification
	if wsMsg.MessageID != 0 {
		var msg models.Message
		// Get Message first to know who sent it
		if err := c.DB.First(&msg, wsMsg.MessageID).Error; err != nil {
			// Message might already be deleted or invalid
			return
		}

		senderID := msg.SenderID
		chatRoomID := msg.ChatRoomID

		// Hard Delete the message
		if err := c.DB.Unscoped().Delete(&models.Message{}, wsMsg.MessageID).Error; err != nil {
			log.Printf("Error deleting read message: %v", err)
			return // Don't notify if delete failed? or notify regardless? Let's return.
		}

		log.Printf("Message %d read by user %d --> DELETED from DB", wsMsg.MessageID, c.UserID)

		// Notify the Sender that their message was read
		if senderID != c.UserID { // Should always be true but good to check
			receiptJSON, _ := json.Marshal(map[string]interface{}{
				"type":         "read_receipt",
				"message_id":   wsMsg.MessageID,
				"chat_room_id": chatRoomID,
				"read_by":      c.UserID,
			})
			c.Hub.SendToUser(senderID, receiptJSON)
		}
	}
}

// SendUnreadMessages fetches and delivers all unread messages for the connected user (used on initial connect)
func (c *Client) SendUnreadMessages() {
	var unreadMessages []models.Message
	// Find messages where the user is a participant of the room, sender is NOT the user, and is_read is false
	// Using a subquery approach or JOIN for clarity

	// GORM Joins with conditions
	// We need: Message.ChatRoomID -> ChatParticipant.ChatRoomID AND ChatParticipant.UserID = c.UserID
	// AND Message.SenderID != c.UserID
	// AND Message.IsRead = false

	// PRELOAD Sender so the client can display who sent it!
	err := c.DB.Preload("Sender").Joins("JOIN chat_participants cp ON cp.chat_room_id = messages.chat_room_id").
		Where("cp.user_id = ? AND messages.sender_id != ? AND messages.is_read = ?", c.UserID, c.UserID, false).
		Find(&unreadMessages).Error

	if err != nil {
		log.Printf("Error fetching unread messages: %v", err)
		return
	}

	if len(unreadMessages) > 0 {
		log.Printf("Found %d unread messages for user %d", len(unreadMessages), c.UserID)
		for _, msg := range unreadMessages {
			responseJSON, _ := json.Marshal(map[string]interface{}{
				"type":         "chat",
				"message":      msg,
				"sender_id":    msg.SenderID,
				"chat_room_id": msg.ChatRoomID,
			})
			c.Send <- responseJSON
		}
	}
}

// SendUnreadMessagesForRoom fetches and delivers unread messages for a specific room
// This is called when user joins a room to get messages they missed while not in the room
// After sending, messages are DELETED from the database (ephemeral behavior)
func (c *Client) SendUnreadMessagesForRoom(roomID uint) {
	var unreadMessages []models.Message

	// Fetch unread messages for this specific room
	err := c.DB.Preload("Sender").
		Where("chat_room_id = ? AND sender_id != ? AND is_read = ?", roomID, c.UserID, false).
		Order("created_at ASC").
		Find(&unreadMessages).Error

	if err != nil {
		log.Printf("Error fetching unread messages for room %d: %v", roomID, err)
		return
	}

	if len(unreadMessages) > 0 {
		log.Printf("Found %d unread messages in room %d for user %d", len(unreadMessages), roomID, c.UserID)

		// Collect message IDs for deletion
		var messageIDs []uint

		for _, msg := range unreadMessages {
			responseJSON, _ := json.Marshal(map[string]interface{}{
				"type":         "chat",
				"message":      msg,
				"sender_id":    msg.SenderID,
				"chat_room_id": msg.ChatRoomID,
			})
			c.Send <- responseJSON
			messageIDs = append(messageIDs, msg.ID)

			// Notify sender that their message was read/delivered
			receiptJSON, _ := json.Marshal(map[string]interface{}{
				"type":         "read_receipt",
				"message_id":   msg.ID,
				"chat_room_id": roomID,
				"read_by":      c.UserID,
			})
			c.Hub.SendToUser(msg.SenderID, receiptJSON)
		}

		// Delete all fetched messages from database (ephemeral)
		if len(messageIDs) > 0 {
			if err := c.DB.Unscoped().Where("id IN ?", messageIDs).Delete(&models.Message{}).Error; err != nil {
				log.Printf("Error deleting messages after fetch: %v", err)
			} else {
				log.Printf("Deleted %d messages from room %d after delivery to user %d", len(messageIDs), roomID, c.UserID)
			}
		}
	}
}
