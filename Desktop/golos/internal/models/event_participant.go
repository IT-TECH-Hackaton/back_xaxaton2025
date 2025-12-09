package models

import (
	"time"
)

type EventParticipant struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	EventID     uint      `gorm:"not null;index" json:"event_id"`
	Event       Event     `gorm:"foreignKey:EventID" json:"event,omitempty"`
	UserID      uint      `gorm:"not null;index" json:"user_id"`
	User        User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
	ConfirmedAt time.Time `gorm:"not null" json:"confirmed_at"`
	CreatedAt   time.Time `json:"created_at"`
}

func (EventParticipant) TableName() string {
	return "event_participants"
}
