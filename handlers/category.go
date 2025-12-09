package handlers

import (
	"net/http"

	"bekend/database"
	"bekend/dto"
	"bekend/models"
	"bekend/utils"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type CategoryHandler struct {
	logger *zap.Logger
}

func NewCategoryHandler() *CategoryHandler {
	return &CategoryHandler{
		logger: utils.GetLogger(),
	}
}

func (h *CategoryHandler) GetCategories(c *gin.Context) {
	var categories []models.Category
	if err := database.DB.Order("name ASC").Find(&categories).Error; err != nil {
		h.logger.Error("Ошибка получения категорий", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении категорий"})
		return
	}

	result := make([]dto.CategoryInfo, len(categories))
	for i, cat := range categories {
		result[i] = dto.CategoryInfo{
			ID:   cat.ID.String(),
			Name: cat.Name,
		}
	}

	c.JSON(http.StatusOK, result)
}

func (h *CategoryHandler) CreateCategory(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Неверные данные при создании категории", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверные данные"})
		return
	}

	if !utils.ValidateStringLength(req.Name, 1, 100) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Название категории должно быть от 1 до 100 символов"})
		return
	}

	category := models.Category{
		Name:        req.Name,
		Description: req.Description,
	}

	if err := database.DB.Create(&category).Error; err != nil {
		h.logger.Error("Ошибка создания категории в БД", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при создании категории"})
		return
	}

	c.JSON(http.StatusCreated, dto.CategoryInfo{
		ID:   category.ID.String(),
		Name: category.Name,
	})
}

func (h *CategoryHandler) UpdateCategory(c *gin.Context) {
	categoryID := c.Param("id")

	if !utils.ValidateUUID(categoryID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат ID категории"})
		return
	}

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Неверные данные при обновлении категории", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверные данные"})
		return
	}

	var category models.Category
	if err := database.DB.Where("id = ?", categoryID).First(&category).Error; err != nil {
		h.logger.Error("Категория не найдена", zap.String("categoryID", categoryID), zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "Категория не найдена"})
		return
	}

	if req.Name != "" {
		if !utils.ValidateStringLength(req.Name, 1, 100) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Название категории должно быть от 1 до 100 символов"})
			return
		}
		category.Name = req.Name
	}

	if req.Description != "" {
		category.Description = req.Description
	}

	if err := database.DB.Save(&category).Error; err != nil {
		h.logger.Error("Ошибка сохранения категории", zap.String("categoryID", categoryID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при обновлении категории"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Категория обновлена"})
}

func (h *CategoryHandler) DeleteCategory(c *gin.Context) {
	categoryID := c.Param("id")

	if !utils.ValidateUUID(categoryID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат ID категории"})
		return
	}

	if err := database.DB.Where("id = ?", categoryID).Delete(&models.Category{}).Error; err != nil {
		h.logger.Error("Ошибка удаления категории", zap.String("categoryID", categoryID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при удалении категории"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Категория удалена"})
}

