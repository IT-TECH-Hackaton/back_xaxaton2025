package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RefreshToken struct {
	ID        uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID    uuid.UUID      `gorm:"type:uuid;not null;index" json:"userId"`
	TokenHash string         `gorm:"type:varchar(128);not null;uniqueIndex" json:"-"`
	ExpiresAt time.Time      `gorm:"not null;index" json:"expiresAt"`
	CreatedAt time.Time      `json:"createdAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (r *RefreshToken) BeforeCreate(tx *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}
