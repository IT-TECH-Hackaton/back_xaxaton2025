package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Interest struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Name        string    `gorm:"uniqueIndex;not null" json:"name"`
	Category    string    `gorm:"index" json:"category"`
	Description string    `gorm:"type:text" json:"description"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type UserInterest struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index" json:"userID"`
	InterestID uuid.UUID `gorm:"type:uuid;not null;index" json:"interestID"`
	Weight    int       `gorm:"default:5" json:"weight"`
	User      User      `gorm:"foreignKey:UserID" json:"user"`
	Interest  Interest  `gorm:"foreignKey:InterestID" json:"interest"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (ui *UserInterest) BeforeCreate(tx *gorm.DB) error {
	if ui.ID == uuid.Nil {
		ui.ID = uuid.New()
	}
	return nil
}

func (i *Interest) BeforeCreate(tx *gorm.DB) error {
	if i.ID == uuid.Nil {
		i.ID = uuid.New()
	}
	return nil
}

