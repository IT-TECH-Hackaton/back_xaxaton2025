package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserRole string

const (
	RoleUser  UserRole = "Пользователь"
	RoleAdmin UserRole = "Администратор"
)

type UserStatus string

const (
	UserStatusActive UserStatus = "Активен"
	UserStatusDeleted UserStatus = "Удален"
)

type User struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	FullName  string    `gorm:"not null" json:"fullName"`
	Email     string    `gorm:"uniqueIndex;not null" json:"email"`
	Password  string    `gorm:"not null" json:"-"`
	Role      UserRole  `gorm:"type:varchar(50);default:'Пользователь'" json:"role"`
	Status    UserStatus `gorm:"type:varchar(50);default:'Активен'" json:"status"`
	EmailVerified bool   `gorm:"default:false" json:"emailVerified"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}

