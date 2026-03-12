package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type EventReview struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	EventID   uuid.UUID `gorm:"type:uuid;not null;index" json:"eventID"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index" json:"userID"`
	Rating    int       `gorm:"not null;check:rating >= 1 AND rating <= 5" json:"rating"`
	Comment   string    `gorm:"type:text" json:"comment"`
	Event     Event     `gorm:"foreignKey:EventID" json:"event"`
	User      User      `gorm:"foreignKey:UserID" json:"user"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (er *EventReview) BeforeCreate(tx *gorm.DB) error {
	if er.ID == uuid.Nil {
		er.ID = uuid.New()
	}
	return nil
}

