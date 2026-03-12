package utils

import "bekend/models"

// PlanLimits — лимиты по тарифам организаторов
// -1 означает без ограничений
var PlanLimits = map[models.SubscriptionPlan]struct {
	MaxEventsPerMonth int     // Макс. событий в месяц
	PriceMonthly      float64 // Цена в рублях/месяц (0 = бесплатно)
}{
	models.SubscriptionPlanBasic:   {MaxEventsPerMonth: 5, PriceMonthly: 0},
	models.SubscriptionPlanPro:    {MaxEventsPerMonth: 50, PriceMonthly: 990},
	models.SubscriptionPlanBusiness: {MaxEventsPerMonth: -1, PriceMonthly: 2990},
}
