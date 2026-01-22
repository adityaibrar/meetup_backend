package config

import (
	"log"
	"meetup_backend/models"
	"meetup_backend/utils"

	"gorm.io/gorm"
)

func SeedUsers(db *gorm.DB) {
	log.Println("🌱 Seeding users...")

	password, _ := utils.HashPassword("password123")

	users := []models.User{
		{
			Username: "user1",
			Email:    "user1@example.com",
			Password: password,
			FullName: "User One",
			Role:     "user",
		},
		{
			Username: "user2",
			Email:    "user2@example.com",
			Password: password,
			FullName: "User Two",
			Role:     "user",
		},
	}

	for _, user := range users {
		var existingUser models.User
		if err := db.Where("email = ?", user.Email).First(&existingUser).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				if err := db.Create(&user).Error; err != nil {
					log.Printf("Failed to seed user %s: %v", user.Username, err)
				} else {
					log.Printf("User seeded: %s (ID: %d)", user.Username, user.ID)
				}
			}
		} else {
			log.Printf("User already exists: %s", user.Username)
		}
	}

	log.Println("✅ Seeding complete.")
}
