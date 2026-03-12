package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SubscriptionPlan — тариф подписки организатора
type SubscriptionPlan string

const (
	SubscriptionPlanBasic   SubscriptionPlan = "Базовый"   // Бесплатно
	SubscriptionPlanPro     SubscriptionPlan = "Про"      // 990 ₽/мес
	SubscriptionPlanBusiness SubscriptionPlan = "Бизнес"  // 2990 ₽/мес
)

// SubscriptionStatus — статус подписки
type SubscriptionStatus string

const (
	SubscriptionStatusActive   SubscriptionStatus = "Активна"
	SubscriptionStatusExpired  SubscriptionStatus = "Истекла"
	SubscriptionStatusCancelled SubscriptionStatus = "Отменена"
)

// Subscription — подписка организатора на тариф
type Subscription struct {
	ID        uuid.UUID          `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID    uuid.UUID          `gorm:"type:uuid;not null;index" json:"userID"`
	User      User               `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Plan      SubscriptionPlan   `gorm:"type:varchar(50);not null" json:"plan"`
	Status    SubscriptionStatus  `gorm:"type:varchar(50);default:'Активна'" json:"status"`
	Price     float64            `gorm:"type:decimal(12,2);default:0" json:"price"` // Цена в рублях
	StartDate time.Time          `gorm:"not null" json:"startDate"`
	EndDate   time.Time          `gorm:"not null" json:"endDate"`
	CreatedAt time.Time          `json:"createdAt"`
	UpdatedAt time.Time          `json:"updatedAt"`
	DeletedAt gorm.DeletedAt     `gorm:"index" json:"-"`
}

func (s *Subscription) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}

// IsActive возвращает true, если подписка активна и не истекла
func (s *Subscription) IsActive() bool {
	return s.Status == SubscriptionStatusActive && time.Now().Before(s.EndDate)
}
