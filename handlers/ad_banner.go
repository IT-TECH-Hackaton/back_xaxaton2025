package handlers

import (
	"net/http"
	"strings"

	"bekend/database"
	"bekend/models"
	"bekend/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type AdBannerHandler struct {
	logger *zap.Logger
}

func NewAdBannerHandler() *AdBannerHandler {
	return &AdBannerHandler{logger: utils.GetLogger()}
}

// GetActiveBanners godoc
// @Summary Получить активные рекламные баннеры
// @Description Возвращает баннеры для вставки в ленту (1 на 12 афиш)
// @Tags Реклама
// @Produce json
// @Success 200 {array} object
// @Router /ads [get]
func (h *AdBannerHandler) GetActiveBanners(c *gin.Context) {
	var banners []models.AdBanner
	if err := database.DB.Where("status = ? AND impressions < target_impressions", models.AdBannerStatusActive).
		Order("created_at DESC").
		Limit(50).
		Find(&banners).Error; err != nil {
		h.logger.Error("GetActiveBanners: ошибка БД", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка получения баннеров"})
		return
	}

	result := make([]gin.H, 0, len(banners))
	for _, b := range banners {
		result = append(result, gin.H{
			"id":                b.ID.String(),
			"imageURL":          b.ImageURL,
			"linkURL":           b.LinkURL,
			"title":             b.Title,
			"targetImpressions": b.TargetImpressions,
			"impressions":       b.Impressions,
			"clicks":            b.Clicks,
			"price":             b.Price,
		})
	}

	c.JSON(http.StatusOK, result)
}

// GetMyBanners godoc
// @Summary Получить баннеры текущего рекламодателя
// @Tags Реклама
// @Security BearerAuth
// @Produce json
// @Success 200 {array} object
// @Router /ads/my [get]
func (h *AdBannerHandler) GetMyBanners(c *gin.Context) {
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
	if user.Role != models.RoleAdvertiser {
		c.JSON(http.StatusForbidden, gin.H{"error": "Доступ только для рекламодателей"})
		return
	}

	var banners []models.AdBanner
	if err := database.DB.Where("advertiser_id = ?", userID).Order("created_at DESC").Find(&banners).Error; err != nil {
		h.logger.Error("GetMyBanners: ошибка БД", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка получения баннеров"})
		return
	}

	result := make([]gin.H, 0, len(banners))
	for _, b := range banners {
		item := gin.H{
			"id":                b.ID.String(),
			"imageURL":          b.ImageURL,
			"linkURL":           b.LinkURL,
			"title":             b.Title,
			"targetImpressions": b.TargetImpressions,
			"impressions":       b.Impressions,
			"clicks":            b.Clicks,
			"price":             b.Price,
			"status":            b.Status,
		}
		if b.RejectionReason != "" {
			item["rejectionReason"] = b.RejectionReason
		}
		if b.ApprovedAt != nil {
			item["approvedAt"] = b.ApprovedAt
		}
		if b.PaidAt != nil {
			item["paidAt"] = b.PaidAt
		}
		result = append(result, item)
	}

	c.JSON(http.StatusOK, result)
}

// RecordImpression godoc
// @Summary Записать показ баннера
// @Tags Реклама
// @Param id path string true "ID баннера"
// @Success 200
// @Router /ads/{id}/impression [post]
func (h *AdBannerHandler) RecordImpression(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID"})
		return
	}

	var banner models.AdBanner
	if err := database.DB.First(&banner, "id = ? AND status = ?", id, models.AdBannerStatusActive).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Баннер не найден"})
		return
	}

	banner.Impressions++
	if banner.Impressions >= banner.TargetImpressions {
		banner.Status = models.AdBannerStatusFinished
	}
	if err := database.DB.Save(&banner).Error; err != nil {
		h.logger.Error("RecordImpression: ошибка сохранения", zap.Error(err))
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// RecordClick godoc
// @Summary Записать клик по баннеру
// @Tags Реклама
// @Param id path string true "ID баннера"
// @Success 200
// @Router /ads/{id}/click [post]
func (h *AdBannerHandler) RecordClick(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID"})
		return
	}

	var banner models.AdBanner
	if err := database.DB.First(&banner, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Баннер не найден"})
		return
	}
	banner.Clicks++
	if err := database.DB.Save(&banner).Error; err != nil {
		h.logger.Error("RecordClick: ошибка сохранения", zap.Error(err))
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// CreateBannerRequest — запрос на создание баннера
type CreateBannerRequest struct {
	ImageURL          string  `json:"imageURL" binding:"required"`
	LinkURL           string  `json:"linkURL" binding:"required"`
	Title             string  `json:"title"`
	TargetImpressions int     `json:"targetImpressions" binding:"required,min=1"`
	Price             float64 `json:"price" binding:"required,min=0"`
	SubmitForReview   bool    `json:"submitForReview"` // true = отправить на модерацию
}

// CreateBanner godoc
// @Summary Создать рекламный баннер (только для рекламодателей)
// @Tags Реклама
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body CreateBannerRequest true "Данные баннера"
// @Success 201 {object} object
// @Router /ads [post]
func (h *AdBannerHandler) CreateBanner(c *gin.Context) {
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
	if user.Role != models.RoleAdvertiser {
		c.JSON(http.StatusForbidden, gin.H{"error": "Только рекламодатели могут создавать баннеры. Обратитесь к администратору для получения роли."})
		return
	}

	var req CreateBannerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверные данные: " + err.Error()})
		return
	}

	req.LinkURL = strings.TrimSpace(req.LinkURL)
	if !strings.HasPrefix(req.LinkURL, "http://") && !strings.HasPrefix(req.LinkURL, "https://") {
		req.LinkURL = "https://" + req.LinkURL
	}

	initialStatus := models.AdBannerStatusDraft
	if req.SubmitForReview {
		initialStatus = models.AdBannerStatusPendingReview
	}
	banner := models.AdBanner{
		AdvertiserID:      userID,
		ImageURL:          strings.TrimSpace(req.ImageURL),
		LinkURL:           req.LinkURL,
		Title:             strings.TrimSpace(req.Title),
		TargetImpressions: req.TargetImpressions,
		Price:             req.Price,
		Status:            initialStatus,
	}
	if err := database.DB.Create(&banner).Error; err != nil {
		h.logger.Error("CreateBanner: ошибка БД", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка создания баннера"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":                banner.ID.String(),
		"imageURL":          banner.ImageURL,
		"linkURL":           banner.LinkURL,
		"title":             banner.Title,
		"targetImpressions": banner.TargetImpressions,
		"price":             banner.Price,
		"impressions":       banner.Impressions,
		"clicks":            banner.Clicks,
		"status":            banner.Status,
	})
}

// SubmitBannerForReview — отправить черновик на модерацию
func (h *AdBannerHandler) SubmitBannerForReview(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID"})
		return
	}

	rawID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Требуется авторизация"})
		return
	}
	userID := rawID.(uuid.UUID)

	var banner models.AdBanner
	if err := database.DB.First(&banner, "id = ? AND advertiser_id = ?", id, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Баннер не найден"})
		return
	}
	if banner.Status != models.AdBannerStatusDraft {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Только черновики можно отправить на модерацию"})
		return
	}

	banner.Status = models.AdBannerStatusPendingReview
	if err := database.DB.Save(&banner).Error; err != nil {
		h.logger.Error("SubmitBannerForReview: ошибка сохранения", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка сохранения"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true, "status": banner.Status})
}

// GetPricePlans — получить тарифы на рекламу (публичный)
func (h *AdBannerHandler) GetPricePlans(c *gin.Context) {
	var plans []models.AdPricePlan
	if err := database.DB.Where("is_active = ?", true).Order("sort_order ASC, target_impressions ASC").Find(&plans).Error; err != nil {
		h.logger.Error("GetPricePlans: ошибка БД", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка получения тарифов"})
		return
	}
	if len(plans) == 0 {
		// Дефолтные тарифы если в БД пусто
		plans = []models.AdPricePlan{
			{Name: "1 000 показов", TargetImpressions: 1000, Price: 500, SortOrder: 1},
			{Name: "5 000 показов", TargetImpressions: 5000, Price: 2000, SortOrder: 2},
			{Name: "10 000 показов", TargetImpressions: 10000, Price: 3500, SortOrder: 3},
			{Name: "25 000 показов", TargetImpressions: 25000, Price: 7500, SortOrder: 4},
		}
	}
	result := make([]gin.H, len(plans))
	for i, p := range plans {
		result[i] = gin.H{
			"id":                p.ID.String(),
			"name":              p.Name,
			"targetImpressions": p.TargetImpressions,
			"price":             p.Price,
		}
	}
	c.JSON(http.StatusOK, result)
}
