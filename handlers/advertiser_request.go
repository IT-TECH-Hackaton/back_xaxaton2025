package handlers

import (
	"net/http"

	"bekend/database"
	"bekend/models"
	"bekend/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type AdvertiserRequestHandler struct {
	logger *zap.Logger
}

func NewAdvertiserRequestHandler() *AdvertiserRequestHandler {
	return &AdvertiserRequestHandler{logger: utils.GetLogger()}
}

// CreateRequestRequest — тело запроса на создание заявки
type CreateRequestRequest struct {
	Comment string `json:"comment"`
}

// CreateRequest godoc
// @Summary Подать заявку на статус рекламодателя
// @Tags Реклама
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body CreateRequestRequest true "Комментарий"
// @Success 201 {object} object
// @Failure 400 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /advertiser/request [post]
func (h *AdvertiserRequestHandler) CreateRequest(c *gin.Context) {
	rawID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Требуется авторизация"})
		return
	}
	userID := rawID.(uuid.UUID)

	var user models.User
	if err := database.DB.First(&user, "id = ?", userID).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Пользователь не найден"})
		return
	}
	if user.Role == models.RoleAdvertiser {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Вы уже являетесь рекламодателем"})
		return
	}

	var count int64
	database.DB.Model(&models.AdvertiserRequest{}).Where("user_id = ? AND status = ?", userID, models.AdvertiserRequestPending).Count(&count)
	if count > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "У вас уже есть активная заявка на рассмотрении"})
		return
	}

	var req CreateRequestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверные данные"})
		return
	}
	if len(req.Comment) > 500 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Комментарий не более 500 символов"})
		return
	}

	ar := models.AdvertiserRequest{
		UserID:  userID,
		Status:  models.AdvertiserRequestPending,
		Comment: req.Comment,
	}
	if err := database.DB.Create(&ar).Error; err != nil {
		h.logger.Error("CreateRequest: ошибка БД", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка создания заявки"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":        ar.ID.String(),
		"status":    ar.Status,
		"createdAt": ar.CreatedAt,
	})
}

// GetRequestStatus godoc
// @Summary Получить статус своей заявки
// @Tags Реклама
// @Security BearerAuth
// @Produce json
// @Success 200 {object} object
// @Failure 404 {object} map[string]string
// @Router /advertiser/request/status [get]
func (h *AdvertiserRequestHandler) GetRequestStatus(c *gin.Context) {
	rawID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Требуется авторизация"})
		return
	}
	userID := rawID.(uuid.UUID)

	var ar models.AdvertiserRequest
	if err := database.DB.Where("user_id = ?", userID).Order("created_at DESC").First(&ar).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"hasRequest": false})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"hasRequest":   true,
		"id":           ar.ID.String(),
		"status":       ar.Status,
		"comment":      ar.Comment,
		"adminComment": ar.AdminComment,
		"createdAt":    ar.CreatedAt,
	})
}
