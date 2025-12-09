package models

import (
	"time"
)

type EmailVerification struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Email        string    `gorm:"not null;index" json:"email"`
	Code         string    `gorm:"not null" json:"-"`
	PasswordHash string    `gorm:"not null" json:"-"`
	FullName     string    `gorm:"not null" json:"-"`
	ExpiresAt    time.Time `gorm:"not null" json:"expires_at"`
	CreatedAt    time.Time `json:"created_at"`
}

func (ev *EmailVerification) IsExpired() bool {
	return time.Now().After(ev.ExpiresAt)
}
