package repository

import (
	"golos/internal/database"
	"golos/internal/models"

	"gorm.io/gorm"
)

type EmailVerificationRepository struct {
	db *gorm.DB
}

func NewEmailVerificationRepository() *EmailVerificationRepository {
	return &EmailVerificationRepository{
		db: database.GetDB(),
	}
}

func (r *EmailVerificationRepository) Create(ev *models.EmailVerification) error {
	return r.db.Create(ev).Error
}

func (r *EmailVerificationRepository) FindByEmailAndCode(email, code string) (*models.EmailVerification, error) {
	var ev models.EmailVerification
	err := r.db.Where("email = ? AND code = ?", email, code).First(&ev).Error
	if err != nil {
		return nil, err
	}
	return &ev, nil
}

func (r *EmailVerificationRepository) DeleteByEmail(email string) error {
	return r.db.Where("email = ?", email).Delete(&models.EmailVerification{}).Error
}
