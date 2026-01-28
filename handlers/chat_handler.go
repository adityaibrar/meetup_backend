package handlers

import (
	"log"
	"meetup_backend/internal/ws"
	"meetup_backend/models"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type ChatHandler struct {
	Hub *ws.Hub
	DB  *gorm.DB
}

func NewChatHandler(hub *ws.Hub, db *gorm.DB) *ChatHandler {
	return &ChatHandler{
		Hub: hub,
		DB:  db,
	}
}

// WebSocketUpgradeMiddleware ensures the client is trying to upgrade to WebSocket
func (h *ChatHandler) WebSocketUpgradeMiddleware(c *fiber.Ctx) error {
	if websocket.IsWebSocketUpgrade(c) {
		c.Locals("allowed", true)
		return c.Next()
	}
	return fiber.ErrUpgradeRequired
}

// Handler returns the websocket handler function
func (h *ChatHandler) Handler() fiber.Handler {
	return websocket.New(func(c *websocket.Conn) {
		// Retrieve user_id from Locals (set by main.go middleware)
		userID, ok := c.Locals("user_id").(uint)
		if !ok || userID == 0 {
			log.Println("Invalid or missing User ID in WebSocket connection")
			c.Close()
			return
		}

		// Create Client
		client := &ws.Client{
			Hub:    h.Hub,
			Conn:   c,
			Send:   make(chan []byte, 256),
			UserID: uint(userID),
			DB:     h.DB, // Pass DB connection
		}

		// Register to Hub (this will trigger limitRegister which sends unread messages)
		client.Hub.Register <- client

		// Start Pumps
		go client.WritePump()
		client.ReadPump()
	})
}

// InitPrivateChatRequest defines payload for starting a chat
type InitPrivateChatRequest struct {
	TargetUserID uint `json:"target_user_id"`
}

// InitPrivateChat gets an existing private room or creates a new one
func (h *ChatHandler) InitPrivateChat(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	var req InitPrivateChatRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input"})
	}

	if userID == req.TargetUserID {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Cannot chat with yourself"})
	}

	// 1. Check if room exists
	// Query is complex: Find a room where both users are participants
	// Simplified approach: Find all private rooms for User A, then filter for User B
	// SQL approach:
	// SELECT cr.id FROM chat_rooms cr
	// JOIN chat_participants cp1 ON cr.id = cp1.chat_room_id AND cp1.user_id = ?
	// JOIN chat_participants cp2 ON cr.id = cp2.chat_room_id AND cp2.user_id = ?
	// WHERE cr.type = 'private'

	var roomID uint
	query := `
		SELECT cr.id 
		FROM chat_rooms cr
		JOIN chat_participants cp1 ON cr.id = cp1.chat_room_id
		JOIN chat_participants cp2 ON cr.id = cp2.chat_room_id
		WHERE cr.type = 'private' 
		AND cp1.user_id = ? 
		AND cp2.user_id = ?
		LIMIT 1
	`
	if err := h.DB.Raw(query, userID, req.TargetUserID).Scan(&roomID).Error; err != nil {
		// Just continue if error or not found (roomID will be 0)
	}

	if roomID != 0 {
		return c.JSON(fiber.Map{
			"room_id": roomID,
			"created": false,
		})
	}

	// 2. Create new room
	newRoom := models.ChatRoom{
		Type: "private",
	}

	tx := h.DB.Begin()
	if err := tx.Create(&newRoom).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not create room"})
	}

	// Add participants
	participants := []models.ChatParticipant{
		{ChatRoomID: newRoom.ID, UserID: userID},
		{ChatRoomID: newRoom.ID, UserID: req.TargetUserID},
	}

	if err := tx.Create(&participants).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not add participants"})
	}

	tx.Commit()

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"room_id": newRoom.ID,
		"created": true,
	})
}

// GetChatMessages retrieves messages for a specific chat room
func (h *ChatHandler) GetChatMessages(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	roomID, err := c.ParamsInt("roomID")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid room ID"})
	}

	// 1. Verify User is Participant
	var count int64
	h.DB.Model(&models.ChatParticipant{}).
		Where("chat_room_id = ? AND user_id = ?", roomID, userID).
		Count(&count)

	if count == 0 {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "You are not a member of this chat room"})
	}

	// 2. Fetch Messages
	var messages []models.Message
	// Pagination
	limit := c.QueryInt("limit", 50)
	offset := c.QueryInt("offset", 0)

	if err := h.DB.Preload("Sender").
		Where("chat_room_id = ?", roomID).
		Order("created_at DESC"). // Newest first
		Limit(limit).
		Offset(offset).
		Find(&messages).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch messages"})
	}

	// Reverse to Oldest First for Chat UI usually, or keep Newest First and Client reverses
	// Let's keep Newest First (Desc) as it's standard for pagination, Client should handle display order.

	return c.JSON(fiber.Map{
		"messages": messages,
	})
}

// GetRoomStatus returns who is currently online/active in a specific chat room
func (h *ChatHandler) GetRoomStatus(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	roomID, err := c.ParamsInt("roomID")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid room ID"})
	}

	// 1. Verify User is Participant
	var count int64
	h.DB.Model(&models.ChatParticipant{}).
		Where("chat_room_id = ? AND user_id = ?", roomID, userID).
		Count(&count)

	if count == 0 {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "You are not a member of this chat room"})
	}

	// 2. Get all participants in this room
	var participants []models.ChatParticipant
	h.DB.Where("chat_room_id = ?", roomID).Find(&participants)

	// 3. Get users currently active in this room from WebSocket hub
	usersInRoom := h.Hub.GetUsersInRoom(uint(roomID))
	usersInRoomMap := make(map[uint]bool)
	for _, uid := range usersInRoom {
		usersInRoomMap[uid] = true
	}

	// 4. Build response with status for each participant
	type UserRoomStatus struct {
		UserID   uint `json:"user_id"`
		InRoom   bool `json:"in_room"`
		IsOnline bool `json:"is_online"`
	}

	var statuses []UserRoomStatus
	for _, p := range participants {
		// Check if user is currently in this specific room
		inRoom := usersInRoomMap[p.UserID]

		// Check if user is online (has any active WebSocket connection) - in-memory check
		isOnline := h.Hub.IsUserOnline(p.UserID)

		statuses = append(statuses, UserRoomStatus{
			UserID:   p.UserID,
			InRoom:   inRoom,
			IsOnline: isOnline,
		})
	}

	return c.JSON(fiber.Map{
		"room_id":  roomID,
		"statuses": statuses,
	})
}
