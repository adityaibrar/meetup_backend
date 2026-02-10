package handlers

import (
	"meetup_backend/models"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type ProductHandler struct {
	DB *gorm.DB
}

func NewProductHandler(db *gorm.DB) *ProductHandler {
	return &ProductHandler{DB: db}
}

// CreateProductRequest
type CreateProductRequest struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Price       float64  `json:"price"`
	Category    string   `json:"category"`
	Condition   string   `json:"condition"`
	ImageURL    string   `json:"image_url"`
	Images      []string `json:"images"`
}

// CreateProduct - POST /api/products
func (h *ProductHandler) CreateProduct(c *fiber.Ctx) error {
	var req CreateProductRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input"})
	}

	userID := c.Locals("user_id").(uint)

	product := models.Product{
		SellerID:    userID,
		Title:       req.Title,
		Description: req.Description,
		Price:       req.Price,
		Category:    req.Category,
		Condition:   req.Condition,
		ImageURL:    req.ImageURL,
		Images:      req.Images,
		Status:      "available",
	}

	if err := h.DB.Create(&product).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not create product"})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"message": "Product created", "data": product})
}

// GetAllProducts - GET /api/products
func (h *ProductHandler) GetAllProducts(c *fiber.Ctx) error {
	var products []models.Product
	query := h.DB.Preload("Seller", func(db *gorm.DB) *gorm.DB {
		return db.Select("id, username, full_name, image_url")
	}).Where("status = ?", "available")

	// Filter by Category
	if category := c.Query("category"); category != "" {
		query = query.Where("category = ?", category)
	}

	// Search by Title
	if q := c.Query("q"); q != "" {
		query = query.Where("title LIKE ?", "%"+q+"%")
	}

	// Sort by Newest
	query = query.Order("created_at desc")

	if err := query.Find(&products).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not fetch products"})
	}

	return c.JSON(fiber.Map{"data": products})
}

// GetProduct - GET /api/products/:id
func (h *ProductHandler) GetProduct(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	var product models.Product

	if err := h.DB.Preload("Seller", func(db *gorm.DB) *gorm.DB {
		return db.Select("id, username, full_name, image_url, email") // Include email for contact/search
	}).First(&product, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Product not found"})
	}

	return c.JSON(fiber.Map{"data": product})
}

// DeleteProduct - DELETE /api/products/:id
func (h *ProductHandler) DeleteProduct(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	userIDVal := c.Locals("user_id")
	var userID uint
	if idVal, ok := userIDVal.(uint); ok {
		userID = idVal
	} else if idVal, ok := userIDVal.(float64); ok {
		userID = uint(idVal)
	} else {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid user session"})
	}

	var product models.Product
	if err := h.DB.First(&product, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Product not found"})
	}

	// Check ownership
	if product.SellerID != userID {
		// Assuming no admin role check for now, just ownership
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Not authorized"})
	}

	if err := h.DB.Delete(&product).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not delete product"})
	}

	return c.JSON(fiber.Map{"message": "Product deleted"})
}

// GetMyProducts - GET /api/my-products
func (h *ProductHandler) GetMyProducts(c *fiber.Ctx) error {
	userIDVal := c.Locals("user_id")
	var userID uint

	if id, ok := userIDVal.(uint); ok {
		userID = id
	} else if id, ok := userIDVal.(float64); ok {
		userID = uint(id)
	} else {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid user session"})
	}

	var products []models.Product

	if err := h.DB.Preload("Seller", func(db *gorm.DB) *gorm.DB {
		return db.Select("id, username, full_name, image_url")
	}).Where("seller_id = ? AND status = ?", userID, "available").Order("created_at desc").Find(&products).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not fetch products"})
	}

	return c.JSON(fiber.Map{"data": products})
}

// UpdateProduct - PUT /api/products/:id
func (h *ProductHandler) UpdateProduct(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	userIDVal := c.Locals("user_id")
	var userID uint
	if idVal, ok := userIDVal.(uint); ok {
		userID = idVal
	} else if idVal, ok := userIDVal.(float64); ok {
		userID = uint(idVal)
	} else {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid user session"})
	}

	var product models.Product
	if err := h.DB.First(&product, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Product not found"})
	}

	// Check ownership
	if product.SellerID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Not authorized"})
	}

	var req CreateProductRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input"})
	}

	// Update fields
	product.Title = req.Title
	product.Description = req.Description
	product.Price = req.Price
	product.Category = req.Category
	product.Condition = req.Condition
	product.ImageURL = req.ImageURL
	product.Images = req.Images

	if err := h.DB.Save(&product).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not update product"})
	}

	return c.JSON(fiber.Map{"message": "Product updated", "data": product})
}
