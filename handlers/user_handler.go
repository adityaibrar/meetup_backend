package handlers

import (
	"meetup_backend/models"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type UserHandler struct {
	DB *gorm.DB
}

func NewUserHandler(db *gorm.DB) *UserHandler {
	return &UserHandler{DB: db}
}

// SearchUsers allows searching for users by username or email
func (h *UserHandler) SearchUsers(c *fiber.Ctx) error {
	query := c.Query("q")
	if query == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Query parameter 'q' is required",
		})
	}

	// Get current user ID from locals (set by auth middleware)
	currentUserID := c.Locals("user_id")

	var users []models.User
	// Search for users where username OR email matches the query
	// AND the user is NOT the current user
	err := h.DB.Select("id, username, email, full_name, image_url").
		Where("(username LIKE ? OR email LIKE ?) AND id != ?", "%"+query+"%", "%"+query+"%", currentUserID).
		Limit(10). // Limit results
		Find(&users).Error

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not search users",
		})
	}

	return c.JSON(fiber.Map{
		"data": users,
	})
}
