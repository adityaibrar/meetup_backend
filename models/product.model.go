package models

import (
	"time"

	"gorm.io/gorm"
)

type Product struct {
	ID          uint    `gorm:"primaryKey" json:"id"`
	SellerID    uint    `gorm:"index" json:"seller_id"`
	Title       string  `gorm:"size:255;not null" json:"title"`
	Description string  `gorm:"type:text" json:"description"`
	Price       float64 `gorm:"not null" json:"price"`
	Category    string  `gorm:"size:50;index" json:"category"` // electronics, automotive, etc.
	Condition   string  `gorm:"size:20" json:"condition"`      // new, used
	ImageURL    string  `json:"image_url"`
	Status      string  `gorm:"default:'available';size:20" json:"status"` // available, sold

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at"`

	// Relations
	Seller User `gorm:"foreignKey:SellerID" json:"seller"`
}
