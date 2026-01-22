package middleware

import (
	"meetup_backend/models"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
)

// SetupMiddleware configures all application middleware
func SetupMiddleware(app *fiber.App) {
	// Request ID middleware - adds unique ID to each request
	app.Use(requestid.New())

	// Logger middleware - logs all requests
	app.Use(logger.New(logger.Config{
		Format:     "[${time}] ${status} - ${method} ${path} - ${ip} - ${latency}\n",
		TimeFormat: "2006-01-02 15:04:05",
		TimeZone:   "Local",
	}))

	// Recover middleware - recovers from panics
	app.Use(recover.New(recover.Config{
		EnableStackTrace: true,
	}))

	// Security middleware
	app.Use(helmet.New(helmet.Config{
		XSSProtection:      "1; mode=block",
		ContentTypeNosniff: "nosniff",
		XFrameOptions:      "DENY",
		HSTSMaxAge:         31536000,
	}))

	// CORS middleware
	app.Use(cors.New(cors.Config{
		AllowOrigins:     "*",
		AllowMethods:     "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders:     "Origin,Content-Type,Accept,Authorization",
		AllowCredentials: false,
		ExposeHeaders:    "X-Request-ID",
		MaxAge:           86400, // 24 hours
	}))

	// Content-Type middleware for JSON APIs
	app.Use(func(c *fiber.Ctx) error {
		// Set default content type for API responses
		if len(c.Path()) > 4 && c.Path()[:4] == "/api" {
			c.Set("Content-Type", "application/json")
		}
		return c.Next()
	})
}

// SetupErrorHandler configures global error handling
func SetupErrorHandler(app *fiber.App) {
	app.Use(func(c *fiber.Ctx) error {
		// Handle 404 errors
		response := models.ErrorResponse("Not Found", "The requested resource was not found")
		return c.Status(fiber.StatusNotFound).JSON(response)
	})
}
