package handlers

import (
	"meetup_backend/models"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type CategoryHandler struct {
	DB *gorm.DB
}

func NewCategoryHandler(db *gorm.DB) *CategoryHandler {
	return &CategoryHandler{DB: db}
}

// GetCategories - GET /api/categories
func (h *CategoryHandler) GetCategories(c *fiber.Ctx) error {
	var categories []models.Category
	if err := h.DB.Find(&categories).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not fetch categories"})
	}
	return c.JSON(fiber.Map{"data": categories})
}
