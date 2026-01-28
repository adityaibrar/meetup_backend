package main

import (
	"flag"
	"log"
	"meetup_backend/config"
	"meetup_backend/handlers"
	"meetup_backend/internal/ws"
	"meetup_backend/middleware"
	"meetup_backend/utils"
	"os"

	"github.com/gofiber/contrib/websocket"
	"github.com/golang-jwt/jwt/v5"

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

	// Parse command line flags
	resetFlag := flag.Bool("reset", false, "Reset database and migrate")
	flag.Parse()

	// Run Migrations
	if *resetFlag {
		log.Println("⚠️ Resetting database...")
		if err := config.ResetAndMigrate(db); err != nil {
			log.Fatal("Failed to reset and migrate database:", err)
		}
	} else {
		if err := config.Migrate(db); err != nil {
			log.Fatal("Failed to migrate database:", err)
		}
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

	// WebSocket Configuration
	hub := ws.NewHub()
	go hub.Run()

	authHandler := handlers.NewAuthHandler(db)
	chatHandler := handlers.NewChatHandler(hub, db)

	// API Routes
	api := app.Group("/api")

	// Auth Routes
	auth := api.Group("/auth")
	auth.Post("/register", authHandler.Register)
	auth.Post("/login", authHandler.Login)

	// Chat Routes (Protected)
	chat := api.Group("/chat", utils.AuthMiddleware)
	chat.Post("/private", chatHandler.InitPrivateChat)
	chat.Get("/room/:roomID/messages", chatHandler.GetChatMessages)
	chat.Get("/room/:roomID/status", chatHandler.GetRoomStatus)

	// Middleware for WebSocket Upgrade & Auth
	app.Use("/ws", func(c *fiber.Ctx) error {
		// 1. Check if it's a websocket upgrade
		if !websocket.IsWebSocketUpgrade(c) {
			return fiber.ErrUpgradeRequired
		}

		// 2. Validate Token from Query Param
		tokenString := c.Query("token")
		if tokenString == "" {
			return fiber.ErrUnauthorized
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return []byte(os.Getenv("JWT_SECRET")), nil
		})

		if err != nil || !token.Valid {
			return fiber.ErrUnauthorized
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return fiber.ErrUnauthorized
		}

		// Extract user_id and pass to locals using the exact key expected by ChatHandler
		// Note: ChatHandler currently reads from Query("user_id"), we need to change that OR
		// we just update ChatHandler to read from Locals if we were using middleware there.
		// BUT: fiber/contrib/websocket creates a new *conn context. Locals MIGHT NOT persist easily strictly
		// inside the websocket handler without some tricks.
		// EASIER PATH: The ChatHandler parses query param "user_id".
		// We should VALIDATE that the "user_id" in query param matches the Token.

		userIDFromToken := uint(claims["user_id"].(float64))

		// For simplicity/security, we will OVERRIDE the user_id query param with the one from token
		// or just rely on the token validation.

		// Let's pass the valid user ID in Locals, and update ChatHandler to check Locals OR Query.
		// However, standard websocket handler in fiber doesn't easy share Locals to the conn wrapper?
		// Actually it does: c.Locals key is copied to conn.Locals.

		c.Locals("user_id", userIDFromToken)

		return c.Next()
	})

	app.Get("/ws", chatHandler.Handler())
	middleware.SetupErrorHandler(app)

	log.Printf("🚀 Server starting on host %s in port %s mode", cfg.HOST, cfg.AppPort)

	if err := app.Listen(cfg.HOST + ":" + cfg.AppPort); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
