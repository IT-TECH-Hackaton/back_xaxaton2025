package services

import (
	"time"

	"bekend/database"
	"bekend/models"

	"github.com/google/uuid"
)

// CreateEventPromotionFromPayment создаёт EventPromotion после успешной оплаты
func CreateEventPromotionFromPayment(payment *models.Payment) error {
	if payment.PaymentType != models.PaymentTypePromotion || payment.RelatedID == nil {
		return nil
	}
	eventID := *payment.RelatedID

	var packageIDStr string
	switch v := payment.Metadata["package_id"].(type) {
	case string:
		packageIDStr = v
	default:
		return nil
	}
	if packageIDStr == "" {
		return nil
	}
	packageID, err := uuid.Parse(packageIDStr)
	if err != nil {
		return err
	}

	var pkg models.PromotionPackage
	if database.DB.First(&pkg, "id = ?", packageID).Error != nil {
		return nil
	}

	now := time.Now()
	endDate := now.AddDate(0, 0, pkg.DurationDays)

	ep := &models.EventPromotion{
		EventID:   eventID,
		PackageID: packageID,
		UserID:    payment.UserID,
		PaymentID: &payment.ID,
		Status:    models.EventPromotionStatusActive,
		StartDate: now,
		EndDate:   endDate,
	}
	return database.DB.Create(ep).Error
}
