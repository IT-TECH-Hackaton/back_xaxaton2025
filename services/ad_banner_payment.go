package services

import (
	"time"

	"bekend/database"
	"bekend/models"
)

// ActivateAdBannerFromPayment активирует баннер после успешной оплаты
func ActivateAdBannerFromPayment(payment *models.Payment) error {
	if payment.PaymentType != models.PaymentTypeAdBanner || payment.RelatedID == nil {
		return nil
	}
	bannerID := *payment.RelatedID

	var banner models.AdBanner
	if database.DB.First(&banner, "id = ?", bannerID).Error != nil {
		return nil
	}
	if banner.AdvertiserID != payment.UserID {
		return nil // Баннер принадлежит другому пользователю
	}
	if banner.Status != models.AdBannerStatusApproved {
		return nil // Только одобренные баннеры активируются после оплаты
	}

	now := time.Now()
	banner.Status = models.AdBannerStatusActive
	banner.PaidAt = &now
	return database.DB.Save(&banner).Error
}
