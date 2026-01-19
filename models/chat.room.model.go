package models

import (
	"time"

	"gorm.io/gorm"
)

type ChatRoom struct {
	ID   uint    `gorm:"primaryKey" json:"id"`
	Name *string `gorm:"size:100" json:"name"`          // Nullable. Diisi jika Group Chat. Kosong jika DM.
	Type string  `gorm:"default:'private'" json:"type"` // 'private' (1-on-1) atau 'group'

	// Field optimasi untuk menampilkan list chat (agar tidak perlu query message terakhir terus menerus)
	LastMessageContent string    `gorm:"type:text" json:"last_message"`
	LastMessageAt      time.Time `json:"last_message_at"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at"`

	// Relasi
	Participants []ChatParticipant `json:"participants"`
	Messages     []Message         `json:"messages"`
}
