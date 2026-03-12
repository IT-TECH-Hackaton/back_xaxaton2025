package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AdBannerStatus string

const (
	AdBannerStatusDraft        AdBannerStatus = "draft"
	AdBannerStatusPendingReview AdBannerStatus = "pending_review"
	AdBannerStatusApproved     AdBannerStatus = "approved"
	AdBannerStatusRejected     AdBannerStatus = "rejected"
	AdBannerStatusActive       AdBannerStatus = "active"
	AdBannerStatusPaused       AdBannerStatus = "paused"
	AdBannerStatusFinished     AdBannerStatus = "finished"
)

type AdBanner struct {
	ID                uuid.UUID       `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	AdvertiserID      uuid.UUID      `gorm:"type:uuid;not null" json:"advertiserId"`
	ImageURL          string          `gorm:"type:text;not null" json:"imageURL"`
	LinkURL           string          `gorm:"type:text;not null" json:"linkURL"`
	Title             string          `gorm:"type:varchar(200)" json:"title"`
	TargetImpressions int             `gorm:"not null;default:0" json:"targetImpressions"`
	Price             float64        `gorm:"type:decimal(12,2);not null;default:0" json:"price"`
	Impressions       int             `gorm:"default:0" json:"impressions"`
	Clicks            int             `gorm:"default:0" json:"clicks"`
	Status            AdBannerStatus  `gorm:"type:varchar(30);default:'draft'" json:"status"`
	RejectionReason   string          `gorm:"type:text" json:"rejectionReason"`
	ApprovedAt        *time.Time      `json:"approvedAt"`
	PaidAt            *time.Time      `json:"paidAt"`
	CreatedAt         time.Time       `json:"createdAt"`
	UpdatedAt         time.Time       `json:"updatedAt"`
	DeletedAt         gorm.DeletedAt  `gorm:"index" json:"-"`
}

func (a *AdBanner) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}
