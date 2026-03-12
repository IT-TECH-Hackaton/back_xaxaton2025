package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PaymentProvider — платёжная система
type PaymentProvider string

const (
	PaymentProviderYooKassa   PaymentProvider = "yookassa"
	PaymentProviderCloudPayments PaymentProvider = "cloudpayments"
	PaymentProviderStripe     PaymentProvider = "stripe"
)

// PaymentType — тип платежа
type PaymentType string

const (
	PaymentTypeSubscription PaymentType = "subscription"
	PaymentTypeTicket       PaymentType = "ticket"
	PaymentTypePromotion    PaymentType = "promotion"
	PaymentTypeAdBanner     PaymentType = "ad_banner"
)

// PaymentStatus — статус платежа
type PaymentStatus string

const (
	PaymentStatusPending   PaymentStatus = "pending"
	PaymentStatusSucceeded PaymentStatus = "succeeded"
	PaymentStatusFailed    PaymentStatus = "failed"
	PaymentStatusRefunded  PaymentStatus = "refunded"
	PaymentStatusCancelled PaymentStatus = "cancelled"
)

// JSONMap — для хранения произвольных данных в Metadata
type JSONMap map[string]interface{}

func (j JSONMap) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

func (j *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("неверный тип для JSONMap")
	}
	return json.Unmarshal(bytes, j)
}

// Payment — запись о платеже
type Payment struct {
	ID          uuid.UUID       `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID      uuid.UUID       `gorm:"type:uuid;not null;index" json:"userID"`
	User        User            `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Amount      float64         `gorm:"type:decimal(12,2);not null" json:"amount"`      // Сумма
	Currency    string          `gorm:"type:varchar(3);default:'RUB'" json:"currency"`  // RUB, USD, EUR
	Status      PaymentStatus   `gorm:"type:varchar(50);default:'pending'" json:"status"`
	Provider    PaymentProvider `gorm:"type:varchar(50);not null" json:"provider"`
	PaymentType PaymentType     `gorm:"type:varchar(50);not null" json:"paymentType"`
	RelatedID   *uuid.UUID      `gorm:"type:uuid;index" json:"relatedID"`   // ID связанной сущности (Subscription, EventPromotion)
	ExternalID  string          `gorm:"type:varchar(255);index" json:"-"`   // ID в платёжной системе
	Description string          `gorm:"type:text" json:"description"`      // Описание платежа
	Metadata    JSONMap         `gorm:"type:jsonb" json:"metadata,omitempty"` // Доп. данные от провайдера
	CreatedAt   time.Time       `json:"createdAt"`
	UpdatedAt   time.Time       `json:"updatedAt"`
	DeletedAt   gorm.DeletedAt  `gorm:"index" json:"-"`
}

func (p *Payment) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}
