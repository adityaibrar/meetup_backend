package ws

import (
	"encoding/json"
	"log"
	"meetup_backend/models"
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
}

// WSMessage defines the structure of messages sent over WebSocket
type WSMessage struct {
	Type        string          `json:"type"` // 'chat', 'read', 'typing'
	ChatRoomID  uint            `json:"chat_room_id,omitempty"`
	RecipientID uint            `json:"recipient_id,omitempty"` // Optional if sending to specific user, but usually ChatRoomID is enough
	Content     string          `json:"content,omitempty"`
	MessageID   uint            `json:"message_id,omitempty"` // Used for 'read' receipts
	Payload     json.RawMessage `json:"payload,omitempty"`    // Flexible payload
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
	}
}

func (c *Client) processChatMessage(wsMsg *WSMessage) {
	// 1. Save to Database (Ephemeral persistence)
	newMsg := models.Message{
		ChatRoomID: wsMsg.ChatRoomID,
		SenderID:   c.UserID,
		Content:    wsMsg.Content,
		IsRead:     false,
	}

	if err := c.DB.Create(&newMsg).Error; err != nil {
		log.Printf("Error saving message: %v", err)
		return
	}

	// 2. Prepare payload to send to recipient
	responseJSON, _ := json.Marshal(map[string]interface{}{
		"type":         "chat",
		"message":      newMsg,
		"sender_id":    c.UserID,
		"chat_room_id": wsMsg.ChatRoomID,
	})

	// 3. Find recipients via ChatRoom participants
	// Retrieve participants for this chat room
	var room models.ChatRoom
	if err := c.DB.Preload("Participants").First(&room, wsMsg.ChatRoomID).Error; err != nil {
		log.Printf("Error finding chat room: %v", err)
		return
	}

	for _, p := range room.Participants {
		if p.UserID != c.UserID {
			// Send to this user
			c.Hub.SendToUser(p.UserID, responseJSON)
		}
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

// SendUnreadMessages fetches and delivers all unread messages for the connected user
func (c *Client) SendUnreadMessages() {
	var unreadMessages []models.Message
	// Find messages where the user is a participant of the room, sender is NOT the user, and is_read is false
	// Using a subquery approach or JOIN for clarity

	// GORM Joins with conditions
	// We need: Message.ChatRoomID -> ChatParticipant.ChatRoomID AND ChatParticipant.UserID = c.UserID
	// AND Message.SenderID != c.UserID
	// AND Message.IsRead = false

	err := c.DB.Joins("JOIN chat_participants cp ON cp.chat_room_id = messages.chat_room_id").
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
