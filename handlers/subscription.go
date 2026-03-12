package handlers

import (
	"net/http"
	"time"

	"bekend/database"
	"bekend/dto"
	"bekend/models"
	"bekend/services"
	"bekend/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type SubscriptionHandler struct {
	yooKassa *services.YooKassaService
	logger   *zap.Logger
}

func NewSubscriptionHandler() *SubscriptionHandler {
	return &SubscriptionHandler{
		yooKassa: services.NewYooKassaService(),
		logger:   utils.GetLogger(),
	}
}

// GetSubscription возвращает текущий тариф и лимиты пользователя
// @Summary Получить текущую подписку
// @Description Возвращает текущий тариф, лимиты и дату окончания подписки
// @Tags Подписка
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Router /subscription [get]
func (h *SubscriptionHandler) GetSubscription(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Необходима авторизация"})
		return
	}
	uid := userID.(uuid.UUID)

	plan := services.GetUserPlan(uid)
	limits, ok := utils.PlanLimits[plan]
	if !ok {
		limits = utils.PlanLimits[models.SubscriptionPlanBasic]
	}

	// Количество созданных событий в этом месяце
	now := time.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	var eventsCount int64
	database.DB.Model(&models.Event{}).
		Where("organizer_id = ? AND created_at >= ?", uid, startOfMonth).
		Count(&eventsCount)

	maxEvents := limits.MaxEventsPerMonth
	if maxEvents == -1 {
		maxEvents = 999999
	}

	// Активная подписка (если есть)
	var sub models.Subscription
	hasActiveSub := database.DB.Where("user_id = ? AND status = ?", uid, models.SubscriptionStatusActive).
		Where("end_date > ?", time.Now()).
		First(&sub).Error == nil

	resp := gin.H{
		"plan":         string(plan),
		"maxEvents":    maxEvents,
		"eventsUsed":   int(eventsCount),
		"canCreate":    limits.MaxEventsPerMonth == -1 || int(eventsCount) < limits.MaxEventsPerMonth,
		"priceMonthly": limits.PriceMonthly,
	}
	if hasActiveSub {
		resp["subscriptionEndDate"] = sub.EndDate.Format(time.RFC3339)
		resp["subscriptionId"] = sub.ID.String()
	}

	c.JSON(http.StatusOK, resp)
}

// UpgradeSubscription создаёт платёж для перехода на платный тариф
// @Summary Оформить подписку
// @Description Создаёт платёж в ЮKassa для перехода на тариф Про или Бизнес
// @Tags Подписка
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body dto.UpgradeSubscriptionRequest true "Тариф для оформления"
// @Success 200 {object} dto.CreatePaymentResponse
// @Failure 400 {object} map[string]string
// @Router /subscription/upgrade [post]
func (h *SubscriptionHandler) UpgradeSubscription(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Необходима авторизация"})
		return
	}
	uid := userID.(uuid.UUID)

	var req dto.UpgradeSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверные данные: " + err.Error()})
		return
	}

	plan := models.SubscriptionPlan(req.Plan)
	if plan != models.SubscriptionPlanPro && plan != models.SubscriptionPlanBusiness {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Укажите тариф: Про или Бизнес"})
		return
	}

	limits, ok := utils.PlanLimits[plan]
	if !ok || limits.PriceMonthly <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный тариф"})
		return
	}

	// Проверяем, нет ли уже активной подписки на этот или более высокий тариф
	currentPlan := services.GetUserPlan(uid)
	if currentPlan == plan || (plan == models.SubscriptionPlanPro && currentPlan == models.SubscriptionPlanBusiness) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "У вас уже активна подписка на этот тариф или выше"})
		return
	}

	description := "Подписка MyAfisha: " + string(plan) + " на 1 месяц"
	payment, confirmationURL, err := h.yooKassa.CreatePayment(
		uid,
		limits.PriceMonthly,
		models.PaymentTypeSubscription,
		description,
		nil,
		req.ReturnURL,
		req.CancelURL,
		nil,
	)
	if err != nil {
		h.logger.Error("Ошибка создания платежа за подписку", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Сохраняем plan в metadata платежа для webhook
	// (уже передаётся через RelatedID = nil, plan определим по amount или добавим в metadata)
	// Для простоты - webhook будет смотреть payment_type=subscription и amount -> plan
	_ = payment

	c.JSON(http.StatusOK, dto.CreatePaymentResponse{
		PaymentID:       payment.ID.String(),
		ConfirmationURL: confirmationURL,
		Status:          string(payment.Status),
	})
}
