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

		// Register to Hub
		client.Hub.Register <- client

		// Send offline/unread messages
		go client.SendUnreadMessages()

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
