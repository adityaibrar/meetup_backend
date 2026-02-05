package handlers

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/gofiber/fiber/v2"
)

// UploadHandler handles file uploads
type UploadHandler struct{}

func NewUploadHandler() *UploadHandler {
	return &UploadHandler{}
}

// UploadImage handles image uploads and returns the file URL
func (h *UploadHandler) UploadImage(c *fiber.Ctx) error {
	// Parse the multipart form:
	file, err := c.FormFile("image")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Image file is required",
		})
	}

	// Validate file type (simple check extension)
	ext := filepath.Ext(file.Filename)
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Only .jpg, .jpeg, and .png files are allowed",
		})
	}

	// Generate unique filename
	filename := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)

	// Create destination path
	// Ensure this directory exists! We might need to create it in main.go or here
	destination := fmt.Sprintf("./uploads/products/%s", filename)

	if err := c.SaveFile(file, destination); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not save file",
		})
	}

	// Return the public URL
	// Assuming static files are served from /uploads
	imageURL := fmt.Sprintf("/uploads/products/%s", filename)

	return c.JSON(fiber.Map{
		"url": imageURL,
	})
}
