package main

import (
	"log"
	"meetup_backend/config"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

func main() {
	cfg := config.LoadConfig()

	app := fiber.New(fiber.Config{
		AppName:      "Meetup Backend",
		ServerHeader: "Meetup Backend Server/1.0",
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			// Default 500 statuscode
			code := fiber.StatusInternalServerError
			msg := "Internal Server Error"

			// Retrieve the custom statuscode if it's a *fiber.Error
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
				msg = e.Message
			}

			// Send custom error page
			return c.Status(code).JSON(fiber.Map{
				"error":   true,
				"message": msg,
			})
		},
	})

	app.Use(logger.New(logger.Config{
		Format:     "[${time}] ${status} - ${method} ${path} - ${ip} - ${latency}\n",
		TimeFormat: "2006-01-02 15:04:05",
		TimeZone:   "Local",
	}))

	// Health Check Endpoint
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"status":  "success",
			"message": "API is healthy",
		})
	})

	log.Printf("🚀 Server starting on host %s in port %s mode", cfg.HOST, cfg.AppPort)

	if err := app.Listen(cfg.HOST + ":" + cfg.AppPort); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
