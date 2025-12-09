package repository

import (
	"golos/internal/database"
	"golos/internal/models"

	"gorm.io/gorm"
)

type PasswordResetRepository struct {
	db *gorm.DB
}

func NewPasswordResetRepository() *PasswordResetRepository {
	return &PasswordResetRepository{
		db: database.GetDB(),
	}
}

func (r *PasswordResetRepository) Create(pr *models.PasswordReset) error {
	return r.db.Create(pr).Error
}

func (r *PasswordResetRepository) FindByToken(token string) (*models.PasswordReset, error) {
	var pr models.PasswordReset
	err := r.db.Where("token = ?", token).Preload("User").First(&pr).Error
	if err != nil {
		return nil, err
	}
	return &pr, nil
}

func (r *PasswordResetRepository) Delete(id uint) error {
	return r.db.Delete(&models.PasswordReset{}, id).Error
}
