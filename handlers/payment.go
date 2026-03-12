package handlers

import (
	"encoding/json"
	"io"
	"net/http"

	"bekend/config"
	"bekend/dto"
	"bekend/models"
	"bekend/services"
	"bekend/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type PaymentHandler struct {
	yooKassa *services.YooKassaService
	logger   *zap.Logger
}

func NewPaymentHandler() *PaymentHandler {
	return &PaymentHandler{
		yooKassa: services.NewYooKassaService(),
		logger:   utils.GetLogger(),
	}
}

// CreatePayment создаёт платёж и возвращает ссылку на оплату
// @Summary Создать платёж
// @Description Создаёт платёж в ЮKassa и возвращает ссылку для перенаправления пользователя
// @Tags Платежи
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body dto.CreatePaymentRequest true "Данные платежа"
// @Success 200 {object} dto.CreatePaymentResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /payments [post]
func (h *PaymentHandler) CreatePayment(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Необходима авторизация"})
		return
	}
	uid := userID.(uuid.UUID)

	var req dto.CreatePaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверные данные: " + err.Error()})
		return
	}

	paymentType := models.PaymentType(req.PaymentType)
	var relatedID *uuid.UUID
	if req.RelatedID != "" {
		parsed, err := uuid.Parse(req.RelatedID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат relatedID"})
			return
		}
		relatedID = &parsed
	}
	if paymentType == models.PaymentTypeAdBanner && (req.RelatedID == "" || relatedID == nil) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "relatedID обязателен для оплаты баннера"})
		return
	}

	payment, confirmationURL, err := h.yooKassa.CreatePayment(
		uid,
		req.Amount,
		paymentType,
		req.Description,
		relatedID,
		req.ReturnURL,
		req.CancelURL,
		nil,
	)
	if err != nil {
		h.logger.Error("Ошибка создания платежа", zap.Error(err), zap.String("userID", uid.String()))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.CreatePaymentResponse{
		PaymentID:       payment.ID.String(),
		ConfirmationURL: confirmationURL,
		Status:          string(payment.Status),
	})
}

// YooKassaWebhook обрабатывает webhook от ЮKassa (без авторизации)
// @Summary Webhook ЮKassa
// @Description Принимает уведомления о статусе платежей от ЮKassa
// @Tags Платежи
// @Accept json
// @Produce json
// @Param body body object true "Тело webhook"
// @Success 200 "OK"
// @Router /payments/webhook/yookassa [post]
func (h *PaymentHandler) YooKassaWebhook(c *gin.Context) {
	// Проверка IP whitelist (в prod — WEBHOOK_IP_CHECK=true)
	if config.AppConfig.WebhookIPCheck {
		clientIP := c.ClientIP()
		if !utils.IsYooKassaIP(clientIP) {
			h.logger.Warn("Webhook: запрос с недоверенного IP", zap.String("ip", clientIP))
			c.JSON(http.StatusForbidden, gin.H{"error": "Доступ запрещён"})
			return
		}
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		h.logger.Error("Ошибка чтения webhook", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ошибка чтения тела запроса"})
		return
	}

	// Логирование payload для аудита (без чувствительных данных)
	var audit struct {
		Type   string `json:"type"`
		Event  string `json:"event"`
		Object struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"object"`
	}
	if err := json.Unmarshal(body, &audit); err == nil {
		h.logger.Info("Webhook получен",
			zap.String("type", audit.Type),
			zap.String("event", audit.Event),
			zap.String("object_id", audit.Object.ID),
			zap.String("object_status", audit.Object.Status),
			zap.String("client_ip", c.ClientIP()))
	}

	if err := h.yooKassa.HandleWebhook(body); err != nil {
		h.logger.Error("Ошибка обработки webhook", zap.Error(err), zap.String("object_id", audit.Object.ID))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка обработки"})
		return
	}

	// ЮKassa ожидает 200 OK
	c.Status(http.StatusOK)
}
