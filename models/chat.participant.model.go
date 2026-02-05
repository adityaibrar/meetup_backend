package models

import (
	"time"

	"gorm.io/gorm"
)

type ChatParticipant struct {
	ID         uint `gorm:"primaryKey" json:"id"`
	ChatRoomID uint `gorm:"index" json:"chat_room_id"`
	UserID     uint `gorm:"index" json:"user_id"`

	// Metadata member
	Role      string         `gorm:"default:'member'" json:"role"` // 'admin', 'member' (berguna untuk group)
	JoinedAt  time.Time      `gorm:"autoCreateTime" json:"joined_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Relasi
	User     User     `gorm:"foreignKey:UserID" json:"user"`
	ChatRoom ChatRoom `gorm:"foreignKey:ChatRoomID" json:"chat_room"`
}
