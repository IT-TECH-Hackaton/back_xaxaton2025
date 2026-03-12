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

type PromotionHandler struct {
	yooKassa *services.YooKassaService
	logger   *zap.Logger
}

func NewPromotionHandler() *PromotionHandler {
	return &PromotionHandler{
		yooKassa: services.NewYooKassaService(),
		logger:   utils.GetLogger(),
	}
}

// GetPackages возвращает список пакетов продвижения
// @Summary Список пакетов продвижения
// @Description Возвращает доступные пакеты продвижения событий
// @Tags Продвижение
// @Produce json
// @Success 200 {array} models.PromotionPackage
// @Router /promotion/packages [get]
func (h *PromotionHandler) GetPackages(c *gin.Context) {
	var packages []models.PromotionPackage
	if err := database.DB.Find(&packages).Error; err != nil {
		h.logger.Error("Ошибка получения пакетов", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка получения данных"})
		return
	}

	result := make([]dto.PromotionPackageResponse, len(packages))
	for i, p := range packages {
		result[i] = dto.PromotionPackageResponse{
			ID:           p.ID.String(),
			Name:         string(p.Name),
			Price:        p.Price,
			DurationDays: p.DurationDays,
			Description:  p.Description,
		}
	}

	c.JSON(http.StatusOK, result)
}

// PurchasePromotion создаёт платёж за продвижение события
// @Summary Купить продвижение события
// @Description Создаёт платёж в ЮKassa за продвижение события
// @Tags Продвижение
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body dto.PurchasePromotionRequest true "Данные покупки"
// @Success 200 {object} dto.CreatePaymentResponse
// @Failure 400 {object} map[string]string
// @Router /promotion/purchase [post]
func (h *PromotionHandler) PurchasePromotion(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Необходима авторизация"})
		return
	}
	uid := userID.(uuid.UUID)

	var req dto.PurchasePromotionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверные данные: " + err.Error()})
		return
	}

	eventID, err := uuid.Parse(req.EventID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат eventID"})
		return
	}
	packageID, err := uuid.Parse(req.PackageID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат packageID"})
		return
	}

	// Проверяем, что пользователь — организатор события
	var event models.Event
	if err := database.DB.First(&event, "id = ?", eventID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Событие не найдено"})
		return
	}
	if event.OrganizerID != uid {
		c.JSON(http.StatusForbidden, gin.H{"error": "Только организатор может купить продвижение для своего события"})
		return
	}

	// Проверяем пакет
	var pkg models.PromotionPackage
	if err := database.DB.First(&pkg, "id = ?", packageID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Пакет продвижения не найден"})
		return
	}

	// Проверяем, нет ли уже активного продвижения этого типа
	var existing models.EventPromotion
	if database.DB.Where("event_id = ? AND package_id = ? AND status = ?", eventID, packageID, models.EventPromotionStatusActive).
		Where("end_date > ?", time.Now()).
		First(&existing).Error == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "У события уже активно продвижение этого типа"})
		return
	}

	description := "Продвижение события: " + string(pkg.Name)
	metadata := map[string]string{"package_id": packageID.String()}
	payment, confirmationURL, err := h.yooKassa.CreatePayment(
		uid,
		pkg.Price,
		models.PaymentTypePromotion,
		description,
		&eventID,
		req.ReturnURL,
		req.CancelURL,
		metadata,
	)
	if err != nil {
		h.logger.Error("Ошибка создания платежа за продвижение", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.CreatePaymentResponse{
		PaymentID:       payment.ID.String(),
		ConfirmationURL: confirmationURL,
		Status:          string(payment.Status),
	})
}
