package models

import (
	"time"

	"gorm.io/gorm"
)

type Event struct {
	ID               uint               `gorm:"primaryKey" json:"id"`
	Title            string             `gorm:"not null" json:"title"`
	ShortDescription string             `gorm:"type:text" json:"short_description"`
	FullDescription  string             `gorm:"type:text;not null" json:"full_description"`
	StartDate        time.Time          `gorm:"not null;index" json:"start_date"`
	EndDate          time.Time          `gorm:"not null;index" json:"end_date"`
	ImageURL         string             `gorm:"not null" json:"image_url"`
	PaymentInfo      string             `gorm:"type:text" json:"payment_info"`
	MaxParticipants  *int               `json:"max_participants"`
	Status           EventStatus        `gorm:"type:varchar(20);default:'ACTIVE';index" json:"status"`
	OrganizerID      uint               `gorm:"not null;index" json:"organizer_id"`
	Organizer        User               `gorm:"foreignKey:OrganizerID" json:"organizer"`
	Participants     []EventParticipant `gorm:"foreignKey:EventID" json:"participants,omitempty"`
	CreatedAt        time.Time          `json:"created_at"`
	UpdatedAt        time.Time          `json:"updated_at"`
	DeletedAt        gorm.DeletedAt     `gorm:"index" json:"-"`
}

func (e *Event) IsActive() bool {
	now := time.Now()
	return now.After(e.StartDate) && now.Before(e.EndDate) && e.Status == EventStatusActive
}

func (e *Event) IsPast() bool {
	return time.Now().After(e.EndDate) || e.Status == EventStatusPast
}

func (e *Event) GetParticipantsCount() int64 {
	return int64(len(e.Participants))
}

func (e *Event) IsParticipantLimitReached() bool {
	if e.MaxParticipants == nil {
		return false
	}
	return e.GetParticipantsCount() >= int64(*e.MaxParticipants)
}
