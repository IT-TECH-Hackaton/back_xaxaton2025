package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MatchStatus string

const (
	MatchStatusLooking    MatchStatus = "Ищу компанию"
	MatchStatusFound      MatchStatus = "Нашел компанию"
	MatchStatusGoingAlone MatchStatus = "Иду один"
)

type EventMatching struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID      uuid.UUID `gorm:"type:uuid;not null;index" json:"userID"`
	EventID     uuid.UUID `gorm:"type:uuid;not null;index" json:"eventID"`
	Status      MatchStatus `gorm:"type:varchar(50);default:'Ищу компанию'" json:"status"`
	Preferences string    `gorm:"type:text" json:"preferences"`
	User        User      `gorm:"foreignKey:UserID" json:"user"`
	Event       Event     `gorm:"foreignKey:EventID" json:"event"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type MatchRequestStatus string

const (
	MatchRequestStatusPending  MatchRequestStatus = "pending"
	MatchRequestStatusAccepted MatchRequestStatus = "accepted"
	MatchRequestStatusRejected MatchRequestStatus = "rejected"
)

type MatchRequest struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	FromUserID uuid.UUID `gorm:"type:uuid;not null;index" json:"fromUserID"`
	ToUserID   uuid.UUID `gorm:"type:uuid;not null;index" json:"toUserID"`
	EventID    uuid.UUID `gorm:"type:uuid;not null;index" json:"eventID"`
	Status    MatchRequestStatus `gorm:"type:varchar(50);default:'pending'" json:"status"`
	Message   string    `gorm:"type:text" json:"message"`
	FromUser  User      `gorm:"foreignKey:FromUserID" json:"fromUser"`
	ToUser    User      `gorm:"foreignKey:ToUserID" json:"toUser"`
	Event     Event     `gorm:"foreignKey:EventID" json:"event"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (em *EventMatching) BeforeCreate(tx *gorm.DB) error {
	if em.ID == uuid.Nil {
		em.ID = uuid.New()
	}
	return nil
}

func (mr *MatchRequest) BeforeCreate(tx *gorm.DB) error {
	if mr.ID == uuid.Nil {
		mr.ID = uuid.New()
	}
	return nil
}

