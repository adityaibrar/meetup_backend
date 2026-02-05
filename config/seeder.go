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

func SeedProducts(db *gorm.DB) {
	log.Println("🌱 Seeding products...")

	var user1, user2 models.User
	if err := db.Where("username = ?", "user1").First(&user1).Error; err != nil {
		log.Printf("Error finding user1: %v", err)
		return
	}
	if err := db.Where("username = ?", "user2").First(&user2).Error; err != nil {
		log.Printf("Error finding user2: %v", err)
		return
	}

	products := []models.Product{
		{
			SellerID:    user1.ID,
			Title:       "Es Joshua",
			Description: "Segar dan nikmat",
			Price:       5000,
			Category:    "beverages",
			Condition:   "new",
			Status:      "available",
			ImageURL:    "https://images.unsplash.com/photo-1543253687-c599f5e08fd8?auto=format&fit=crop&w=600",
		},
		{
			SellerID:    user1.ID,
			Title:       "Es Nutrisari",
			Description: "Jeruk peras asli",
			Price:       3500,
			Category:    "beverages",
			Condition:   "new",
			Status:      "available",
			ImageURL:    "https://images.unsplash.com/photo-1613478223719-2ab802602423?auto=format&fit=crop&w=600",
		},
		{
			SellerID:    user2.ID,
			Title:       "Nasi Kucing",
			Description: "Porsi pas untuk sarapan",
			Price:       5000,
			Category:    "food",
			Condition:   "new",
			Status:      "available",
			ImageURL:    "https://images.unsplash.com/photo-1626082927389-6cd097cdc6ec?auto=format&fit=crop&w=600",
		},
		{
			SellerID:    user2.ID,
			Title:       "Sate Satean",
			Description: "Sate angkringan mantap",
			Price:       2000,
			Category:    "food",
			Condition:   "new",
			Status:      "available",
			ImageURL:    "https://images.unsplash.com/photo-1555939594-58d7cb561ad1?auto=format&fit=crop&w=600",
		},
	}

	for _, p := range products {
		if err := db.Create(&p).Error; err != nil {
			log.Printf("Failed to seed product %s: %v", p.Title, err)
		} else {
			log.Printf("Product seeded: %s (ID: %d)", p.Title, p.ID)
		}
	}
	log.Println("✅ Product seeding complete.")
}
