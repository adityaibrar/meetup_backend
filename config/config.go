package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	// Server Settings
	AppPort     string
	HOST        string
	DatabaseURL string

	// JWT Settings
	JWTSecret     string
	JWTExpiration string

	// CORS Settings
	CORSAllowOrigins []string
	CORSAllowMethods []string
	CORSAllowHeaders []string
}

func LoadConfig() *Config {
	// Implementation to load configuration from environment variables or config files
	err := godotenv.Load()
	if err != nil {
		panic("Error loading .env file")
	}

	config := &Config{
		AppPort:       os.Getenv("PORT"),
		DatabaseURL:   os.Getenv("DATABASE_URL"),
		HOST:          os.Getenv("HOST"),
		
		JWTSecret:     os.Getenv("JWT_SECRET"),
		JWTExpiration: os.Getenv("JWT_EXPIRES_IN"),

		CORSAllowOrigins: []string{"*"},
		CORSAllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		CORSAllowHeaders: []string{"Origin", "Content-Type", "Accept", "Authorization"},
	}

	return config
}
