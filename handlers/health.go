package handlers

import (
	"net/http"

	"bekend/database"

	"github.com/gin-gonic/gin"
)

type HealthHandler struct{}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

// HealthCheck godoc
// @Summary Проверка здоровья сервиса
// @Description Проверка доступности сервиса и подключения к базе данных
// @Tags Система
// @Accept json
// @Produce json
// @Success 200 {object} map[string]string "Сервис работает"
// @Failure 503 {object} map[string]string "Сервис недоступен"
// @Router /health [get]
func (h *HealthHandler) HealthCheck(c *gin.Context) {
	sqlDB, err := database.DB.DB()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":  "unhealthy",
			"message": "Database connection error",
		})
		return
	}

	if err := sqlDB.Ping(); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":  "unhealthy",
			"message": "Database ping failed",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"message": "Service is running",
	})
}

