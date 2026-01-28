package config

import (
	"log"
	"meetup_backend/models"

	"gorm.io/gorm"
)

func Migrate(db *gorm.DB) error {
	// Migrate the schema
	err := db.AutoMigrate(
		&models.User{},
		&models.ChatRoom{},
		&models.ChatParticipant{},
		&models.Message{},
		&models.Product{},
	)

	if err != nil {
		log.Printf("Failed to migrate database schema: %v", err)
		return err
	}

	log.Println("Database Migrations completed succesfully...")
	return err
}

func ResetAndMigrate(db *gorm.DB) error {
	// Drop all tables
	models := []interface{}{
		&models.User{},
		&models.ChatRoom{},
		&models.ChatParticipant{},
		&models.Message{},
		&models.Product{},
	}

	if err := db.Migrator().DropTable(models...); err != nil {
		log.Printf("Failed to drop tables: %v", err)
		return err
	}

	log.Println("All tables dropped successfully.")

	if err := db.AutoMigrate(models...); err != nil {
		log.Printf("Failed to auto migrate: %v", err)
		return err
	}

	// Seed Users
	SeedUsers(db)
	SeedProducts(db)

	log.Println("Database reset and migration completed successfully.")
	return nil
}
