package handlers

import (
	"net/http"
	"strings"

	"bekend/database"
	"bekend/dto"
	"bekend/models"
	"bekend/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type InterestHandler struct {
	logger *zap.Logger
}

func NewInterestHandler() *InterestHandler {
	return &InterestHandler{
		logger: utils.GetLogger(),
	}
}

func (h *InterestHandler) GetInterests(c *gin.Context) {
	category := c.Query("category")
	search := c.Query("search")

	var interests []models.Interest
	query := database.DB

	if category != "" {
		query = query.Where("category = ?", category)
	}

	if search != "" {
		query = query.Where("name ILIKE ? OR description ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	if err := query.Order("name ASC").Find(&interests).Error; err != nil {
		h.logger.Error("Ошибка получения интересов", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении интересов"})
		return
	}

	result := make([]dto.InterestResponse, len(interests))
	for i, interest := range interests {
		result[i] = dto.InterestResponse{
			ID:          interest.ID.String(),
			Name:        interest.Name,
			Category:    interest.Category,
			Description: interest.Description,
		}
	}

	c.JSON(http.StatusOK, result)
}

func (h *InterestHandler) GetUserInterests(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Требуется авторизация"})
		return
	}

	var userInterests []models.UserInterest
	if err := database.DB.Preload("Interest").Where("user_id = ?", userID).Find(&userInterests).Error; err != nil {
		h.logger.Error("Ошибка получения интересов пользователя", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении интересов"})
		return
	}

	result := make([]dto.UserInterestResponse, len(userInterests))
	for i, ui := range userInterests {
		result[i] = dto.UserInterestResponse{
			ID: ui.ID.String(),
			Interest: dto.InterestResponse{
				ID:          ui.Interest.ID.String(),
				Name:        ui.Interest.Name,
				Category:    ui.Interest.Category,
				Description: ui.Interest.Description,
			},
			Weight:    ui.Weight,
			CreatedAt: ui.CreatedAt.Format("2006-01-02T15:04:05Z"),
		}
	}

	c.JSON(http.StatusOK, result)
}

func (h *InterestHandler) AddUserInterest(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Требуется авторизация"})
		return
	}

	var req dto.UserInterestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверные данные"})
		return
	}

	if !utils.ValidateUUID(req.InterestID.String()) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат ID интереса"})
		return
	}

	var interest models.Interest
	if err := database.DB.Where("id = ?", req.InterestID).First(&interest).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Интерес не найден"})
		return
	}

	var existing models.UserInterest
	if err := database.DB.Where("user_id = ? AND interest_id = ?", userID, req.InterestID).First(&existing).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Интерес уже добавлен"})
		return
	}

	weight := req.Weight
	if weight < 1 || weight > 10 {
		weight = 5
	}

	userInterest := models.UserInterest{
		UserID:     userID.(uuid.UUID),
		InterestID: req.InterestID,
		Weight:     weight,
	}

	if err := database.DB.Create(&userInterest).Error; err != nil {
		h.logger.Error("Ошибка добавления интереса", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при добавлении интереса"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Интерес добавлен"})
}

func (h *InterestHandler) RemoveUserInterest(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Требуется авторизация"})
		return
	}

	interestID := c.Param("id")
	if !utils.ValidateUUID(interestID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат ID"})
		return
	}

	if err := database.DB.Where("user_id = ? AND interest_id = ?", userID, interestID).Delete(&models.UserInterest{}).Error; err != nil {
		h.logger.Error("Ошибка удаления интереса", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при удалении интереса"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Интерес удален"})
}

func (h *InterestHandler) UpdateUserInterestWeight(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Требуется авторизация"})
		return
	}

	interestID := c.Param("id")
	if !utils.ValidateUUID(interestID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат ID"})
		return
	}

	var req struct {
		Weight int `json:"weight" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверные данные"})
		return
	}

	if req.Weight < 1 || req.Weight > 10 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Вес должен быть от 1 до 10"})
		return
	}

	var userInterest models.UserInterest
	if err := database.DB.Where("user_id = ? AND interest_id = ?", userID, interestID).First(&userInterest).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Интерес не найден"})
		return
	}

	userInterest.Weight = req.Weight
	if err := database.DB.Save(&userInterest).Error; err != nil {
		h.logger.Error("Ошибка обновления веса интереса", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при обновлении интереса"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Вес интереса обновлен"})
}

func (h *InterestHandler) GetCategories(c *gin.Context) {
	var categories []string
	if err := database.DB.Model(&models.Interest{}).Distinct("category").Pluck("category", &categories).Error; err != nil {
		h.logger.Error("Ошибка получения категорий", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении категорий"})
		return
	}

	c.JSON(http.StatusOK, categories)
}

func (h *InterestHandler) CreateInterest(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Category    string `json:"category" binding:"required"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверные данные"})
		return
	}

	if !utils.ValidateStringLength(req.Name, 1, 100) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Название должно быть от 1 до 100 символов"})
		return
	}

	if !utils.ValidateStringLength(req.Category, 1, 50) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Категория должна быть от 1 до 50 символов"})
		return
	}

	var existing models.Interest
	if err := database.DB.Where("name = ?", strings.TrimSpace(req.Name)).First(&existing).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Интерес с таким названием уже существует"})
		return
	}

	interest := models.Interest{
		Name:        strings.TrimSpace(req.Name),
		Category:    strings.TrimSpace(req.Category),
		Description: req.Description,
	}

	if err := database.DB.Create(&interest).Error; err != nil {
		h.logger.Error("Ошибка создания интереса", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при создании интереса"})
		return
	}

	c.JSON(http.StatusCreated, dto.InterestResponse{
		ID:          interest.ID.String(),
		Name:        interest.Name,
		Category:    interest.Category,
		Description: interest.Description,
	})
}

