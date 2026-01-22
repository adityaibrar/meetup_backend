package utils

import (
	"fmt"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

func generateJWTSecret(userId uint) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userId,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(os.Getenv("JWT_SECRET")))
	return tokenString, err
}

func AuthMiddleware(c *fiber.Ctx) error {
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{ // Fixed: StatusBadRequest -> StatusUnauthorized
			"error": "No Token Provided",
		})
	}

	var tokenString string
	fmt.Sscanf(authHeader, "Bearer %s", &tokenString)

	if tokenString == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{ // Fixed: StatusBadGateway -> StatusUnauthorized
			"error": "Token format is invalid",
		})
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(os.Getenv("JWT_SECRET")), nil
	})

	if err != nil || !token.Valid {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{ // Fixed: StatusBadRequest -> StatusUnauthorized
			"error": "Token is invalid",
		})
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{ // Fixed: StatusBadRequest -> StatusUnauthorized
			"error": "Invalid token claims",
		})
	}

	// Check token expiration
	if exp, ok := claims["exp"].(float64); ok {
		if time.Now().Unix() > int64(exp) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Token has expired",
			})
		}
	}

	// Convert user_id to uint to avoid type assertion issues later
	if userIDFloat, ok := claims["user_id"].(float64); ok {
		c.Locals("user_id", uint(userIDFloat))
	} else {
		c.Locals("user_id", claims["user_id"])
	}
	c.Locals("role", claims["role"])

	return c.Next()
}
