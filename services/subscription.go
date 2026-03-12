package services

import (
	"time"

	"bekend/database"
	"bekend/models"
	"bekend/utils"

	"github.com/google/uuid"
)

// GetUserPlan возвращает текущий тариф пользователя.
// Если нет активной подписки — считается Базовый.
func GetUserPlan(userID uuid.UUID) models.SubscriptionPlan {
	var sub models.Subscription
	err := database.DB.Where("user_id = ? AND status = ?", userID, models.SubscriptionStatusActive).
		Where("end_date > ?", time.Now()).
		Order("end_date DESC").
		First(&sub).Error
	if err != nil {
		return models.SubscriptionPlanBasic
	}
	return sub.Plan
}

// CanCreateEvent проверяет, может ли пользователь создать событие в этом месяце
func CanCreateEvent(userID uuid.UUID) (bool, string) {
	plan := GetUserPlan(userID)
	limits, ok := utils.PlanLimits[plan]
	if !ok {
		limits = utils.PlanLimits[models.SubscriptionPlanBasic]
	}

	if limits.MaxEventsPerMonth == -1 {
		return true, ""
	}

	now := time.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	var count int64
	database.DB.Model(&models.Event{}).
		Where("organizer_id = ? AND created_at >= ?", userID, startOfMonth).
		Count(&count)

	if int(count) >= limits.MaxEventsPerMonth {
		return false, "Достигнут лимит событий для вашего тарифа. Перейдите на тариф Про или Бизнес для создания большего количества событий."
	}
	return true, ""
}

// PlanFromAmount определяет тариф по сумме платежа
func PlanFromAmount(amount float64) models.SubscriptionPlan {
	if amount >= 2990 {
		return models.SubscriptionPlanBusiness
	}
	if amount >= 990 {
		return models.SubscriptionPlanPro
	}
	return ""
}

// CreateSubscriptionFromPayment создаёт запись подписки после успешной оплаты
func CreateSubscriptionFromPayment(userID uuid.UUID, plan models.SubscriptionPlan, paymentID uuid.UUID) (*models.Subscription, error) {
	if plan == "" {
		return nil, nil
	}
	limits, ok := utils.PlanLimits[plan]
	if !ok || limits.PriceMonthly == 0 {
		return nil, nil
	}

	now := time.Now()
	endDate := now.AddDate(0, 1, 0) // +1 месяц

	sub := &models.Subscription{
		UserID:    userID,
		Plan:      plan,
		Status:    models.SubscriptionStatusActive,
		Price:     limits.PriceMonthly,
		StartDate: now,
		EndDate:   endDate,
	}
	if err := database.DB.Create(sub).Error; err != nil {
		return nil, err
	}
	return sub, nil
}
