package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AdvertiserRequestStatus string

const (
	AdvertiserRequestPending  AdvertiserRequestStatus = "pending"
	AdvertiserRequestApproved AdvertiserRequestStatus = "approved"
	AdvertiserRequestRejected AdvertiserRequestStatus = "rejected"
)

type AdvertiserRequest struct {
	ID           uuid.UUID               `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID       uuid.UUID               `gorm:"type:uuid;not null;index" json:"userId"`
	User         User                    `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Status       AdvertiserRequestStatus `gorm:"type:varchar(20);default:'pending'" json:"status"`
	Comment      string                  `gorm:"type:text" json:"comment"`
	AdminComment string                  `gorm:"type:text" json:"adminComment"`
	CreatedAt    time.Time               `json:"createdAt"`
	UpdatedAt    time.Time               `json:"updatedAt"`
	DeletedAt    gorm.DeletedAt          `gorm:"index" json:"-"`
}

func (a *AdvertiserRequest) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}
