package models

import (
	"time"

	"gorm.io/gorm"
)

type Message struct {
	ID         uint `gorm:"primaryKey" json:"id"`
	ChatRoomID uint `gorm:"index;not null" json:"chat_room_id"` // Pesan ini milik room mana?
	SenderID   uint `gorm:"index;not null" json:"sender_id"`    // Siapa pengirimnya?

	Content   string `gorm:"type:text;not null" json:"content"`
	MediaType string `gorm:"default:'text'" json:"media_type"` // 'text', 'image', 'video', 'file'
	MediaURL  string `json:"media_url,omitempty"`              // Jika kirim gambar/file

	ProductInfo string `gorm:"type:text" json:"product_info"` // Snapshot product data

	IsRead bool `gorm:"default:false" json:"is_read"` // Sederhana (bagus untuk Private Chat)

	CreatedAt time.Time      `json:"created_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at"`

	// Relasi
	Sender User `gorm:"foreignKey:SenderID" json:"sender"`
}
