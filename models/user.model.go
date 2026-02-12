package models

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID uint `gorm:"primaryKey" json:"id"`

	// Informasi Login
	Username string `gorm:"unique;not null;size:50" json:"username"` // Size membatasi panjang karakter
	Email    string `gorm:"unique;not null;size:100" json:"email"`
	Password string `gorm:"not null" json:"-"`

	// Profil
	FullName string  `gorm:"size:100" json:"full_name"`
	Phone    *string `gorm:"unique;size:20" json:"phone"`
	ImageURL string  `json:"image_url"`

	// Role & Status
	Role       string `gorm:"default:'user';size:20" json:"role"` // user, admin, moderator
	IsVerified bool   `gorm:"default:false" json:"is_verified"`
	IsOnline   bool   `gorm:"default:false" json:"is_online"`
	Points     int    `gorm:"default:10" json:"points"`

	// Lokasi (Indexed untuk performa pencarian geospasial)
	Latitude  float64 `gorm:"index:idx_location" json:"latitude"`
	Longitude float64 `gorm:"index:idx_location" json:"longitude"`
	Address   string  `gorm:"type:text" json:"address"` // Alamat lengkap opsional

	// System Timestamps
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Soft Delete yang Benar
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at"`
}
