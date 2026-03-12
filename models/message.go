package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Message struct {
	ID         uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	SenderID   uuid.UUID      `gorm:"type:uuid;not null;index" json:"senderId"`
	ReceiverID uuid.UUID      `gorm:"type:uuid;not null;index" json:"receiverId"`
	Content    string         `gorm:"type:text;not null" json:"content"`
	ReadAt     *time.Time     `json:"readAt,omitempty"`
	CreatedAt  time.Time      `json:"createdAt"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}

func (m *Message) BeforeCreate(tx *gorm.DB) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return nil
}
