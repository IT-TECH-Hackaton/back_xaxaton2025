package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AdPricePlan struct {
	ID                uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Name              string    `gorm:"type:varchar(100);not null" json:"name"`
	TargetImpressions int       `gorm:"not null" json:"targetImpressions"`
	Price             float64   `gorm:"type:decimal(12,2);not null" json:"price"`
	IsActive          bool      `gorm:"default:true" json:"isActive"`
	SortOrder         int       `gorm:"default:0" json:"sortOrder"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`
}

func (a *AdPricePlan) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}
