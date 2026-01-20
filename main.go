package main

import (
	"log"
	"meetup_backend/config"
	"meetup_backend/middleware"

	"github.com/gofiber/fiber/v2"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
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

	// Database Configuration
	gormConfig := &gorm.Config{}
	if cfg.Debug {
		gormConfig.Logger = logger.Default.LogMode(logger.Info)
	} else {
		gormConfig.Logger = logger.Default.LogMode(logger.Silent)
	}
	db, err := gorm.Open(mysql.Open(cfg.DatabaseURL), gormConfig)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	sqlDb, err := db.DB()
	if err != nil {
		log.Fatal("Failed to get database instance:", err)
	}

	// Set database connection pool settings
	sqlDb.SetMaxOpenConns(100)
	sqlDb.SetMaxIdleConns(10)

	// Run Migrations

	if err := config.Migrate(db); err != nil {
		log.Fatal("Failed to migrate database:", err)
	}

	// Logger Middleware
	middleware.SetupMiddleware(app)

	// Global Error Handler

	// Health Check Endpoint
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"status":  "success",
			"message": "API is healthy",
		})
	})
	middleware.SetupErrorHandler(app)

	log.Printf("🚀 Server starting on host %s in port %s mode", cfg.HOST, cfg.AppPort)

	if err := app.Listen(cfg.HOST + ":" + cfg.AppPort); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
