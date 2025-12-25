package models

import (
	"database/sql/driver"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type EventStatus string

const (
	EventStatusActive   EventStatus = "Активное"
	EventStatusPast     EventStatus = "Прошедшее"
	EventStatusRejected EventStatus = "Отклоненное"
)

type Event struct {
	ID              uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Title           string    `gorm:"not null" json:"title"`
	ShortDescription string   `gorm:"type:text" json:"shortDescription"`
	FullDescription string    `gorm:"type:text;not null" json:"fullDescription"`
	StartDate       time.Time `gorm:"not null" json:"startDate"`
	EndDate         time.Time `gorm:"not null" json:"endDate"`
	ImageURL        string    `gorm:"type:text" json:"imageURL"`
	PaymentInfo     string    `gorm:"type:text" json:"paymentInfo"`
	MaxParticipants *int      `json:"maxParticipants"`
	Status          EventStatus `gorm:"type:varchar(50);default:'Активное'" json:"status"`
	OrganizerID     uuid.UUID `gorm:"type:uuid;not null" json:"organizerID"`
	Organizer       User      `gorm:"foreignKey:OrganizerID" json:"organizer"`
	Participants    []EventParticipant `gorm:"foreignKey:EventID" json:"participants"`
	Categories      []Category `gorm:"many2many:event_categories;" json:"categories"` // Категории события
	Tags            StringArray `gorm:"type:text[]" json:"tags"` // Теги события (массив строк)
	// Место проведения
	Address         string    `gorm:"type:text" json:"address"` // Адрес места проведения
	Latitude        *float64  `gorm:"type:decimal(10,8)" json:"latitude"` // Широта
	Longitude       *float64  `gorm:"type:decimal(11,8)" json:"longitude"` // Долгота
	YandexMapLink   string    `gorm:"type:text" json:"yandexMapLink"` // Ссылка на Яндекс.Карты
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
}

type EventCategory struct {
	EventID    uuid.UUID `gorm:"type:uuid;primary_key" json:"eventID"`
	CategoryID uuid.UUID `gorm:"type:uuid;primary_key" json:"categoryID"`
	CreatedAt  time.Time `json:"createdAt"`
}

func (e *Event) BeforeCreate(tx *gorm.DB) error {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	return nil
}

func (e *Event) GetParticipantsCount() int {
	return len(e.Participants)
}

type EventParticipant struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	EventID   uuid.UUID `gorm:"type:uuid;not null;index" json:"eventID"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index" json:"userID"`
	Event     Event     `gorm:"foreignKey:EventID" json:"event"`
	User      User      `gorm:"foreignKey:UserID" json:"user"`
	CreatedAt time.Time `json:"createdAt"`
}

func (ep *EventParticipant) BeforeCreate(tx *gorm.DB) error {
	if ep.ID == uuid.Nil {
		ep.ID = uuid.New()
	}
	return nil
}

type StringArray []string

func (a *StringArray) Scan(value interface{}) error {
	if value == nil {
		*a = []string{}
		return nil
	}
	
	var str string
	switch v := value.(type) {
	case []byte:
		str = string(v)
	case string:
		str = v
	default:
		return errors.New("cannot scan into StringArray")
	}
	
	if str == "{}" || str == "" {
		*a = []string{}
		return nil
	}
	
	str = strings.TrimPrefix(str, "{")
	str = strings.TrimSuffix(str, "}")
	if str == "" {
		*a = []string{}
		return nil
	}
	
	parts := strings.Split(str, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		part = strings.TrimPrefix(part, `"`)
		part = strings.TrimSuffix(part, `"`)
		part = strings.ReplaceAll(part, `\"`, `"`)
		part = strings.ReplaceAll(part, `\\`, `\`)
		result = append(result, part)
	}
	
	*a = result
	return nil
}

func (a StringArray) Value() (driver.Value, error) {
	if len(a) == 0 {
		return "{}", nil
	}
	result := "{"
	for i, s := range a {
		if i > 0 {
			result += ","
		}
		escaped := escapeString(s)
		result += escaped
	}
	result += "}"
	return result, nil
}

func escapeString(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	return `"` + s + `"`
}

