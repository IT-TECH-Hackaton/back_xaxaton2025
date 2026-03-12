package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PromotionPackageName — название пакета продвижения
type PromotionPackageName string

const (
	PromotionPackageTop         PromotionPackageName = "Топ"          // Событие в топе списка
	PromotionPackageRecommended PromotionPackageName = "Рекомендуемое" // В блоке рекомендаций
	PromotionPackageHot         PromotionPackageName = "Горящее"       // В блоке горящих событий
)

// PromotionPackage — шаблон пакета продвижения (справочник)
type PromotionPackage struct {
	ID          uuid.UUID             `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Name        PromotionPackageName  `gorm:"type:varchar(100);uniqueIndex;not null" json:"name"`
	Price       float64               `gorm:"type:decimal(12,2);not null" json:"price"`       // Цена в рублях
	DurationDays int                  `gorm:"not null" json:"durationDays"`                     // Длительность в днях
	Description string                `gorm:"type:text" json:"description"`
	CreatedAt   time.Time             `json:"createdAt"`
	UpdatedAt   time.Time             `json:"updatedAt"`
	DeletedAt   gorm.DeletedAt        `gorm:"index" json:"-"`
}

func (p *PromotionPackage) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

// EventPromotionStatus — статус продвижения события
type EventPromotionStatus string

const (
	EventPromotionStatusActive   EventPromotionStatus = "Активно"
	EventPromotionStatusExpired  EventPromotionStatus = "Истекло"
	EventPromotionStatusCancelled EventPromotionStatus = "Отменено"
)

// EventPromotion — продвижение конкретного события (купленное)
type EventPromotion struct {
	ID             uuid.UUID             `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	EventID        uuid.UUID             `gorm:"type:uuid;not null;index" json:"eventID"`
	Event          Event                 `gorm:"foreignKey:EventID" json:"event,omitempty"`
	PackageID      uuid.UUID             `gorm:"type:uuid;not null" json:"packageID"`
	Package        PromotionPackage      `gorm:"foreignKey:PackageID" json:"package,omitempty"`
	UserID         uuid.UUID             `gorm:"type:uuid;not null" json:"userID"` // Кто оплатил (организатор)
	PaymentID      *uuid.UUID            `gorm:"type:uuid;index" json:"paymentID"`
	Status         EventPromotionStatus  `gorm:"type:varchar(50);default:'Активно'" json:"status"`
	StartDate      time.Time             `gorm:"not null" json:"startDate"`
	EndDate        time.Time             `gorm:"not null" json:"endDate"`
	CreatedAt      time.Time             `json:"createdAt"`
	UpdatedAt      time.Time             `json:"updatedAt"`
	DeletedAt      gorm.DeletedAt        `gorm:"index" json:"-"`
}

func (e *EventPromotion) BeforeCreate(tx *gorm.DB) error {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	return nil
}

// IsActive возвращает true, если продвижение активно
func (e *EventPromotion) IsActive() bool {
	return e.Status == EventPromotionStatusActive &&
		time.Now().After(e.StartDate) &&
		time.Now().Before(e.EndDate)
}
