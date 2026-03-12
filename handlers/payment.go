package handlers

import (
	"io"
	"net/http"

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
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		h.logger.Error("Ошибка чтения webhook", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ошибка чтения тела запроса"})
		return
	}

	if err := h.yooKassa.HandleWebhook(body); err != nil {
		h.logger.Error("Ошибка обработки webhook", zap.Error(err), zap.String("body", string(body)))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка обработки"})
		return
	}

	// ЮKassa ожидает 200 OK
	c.Status(http.StatusOK)
}
