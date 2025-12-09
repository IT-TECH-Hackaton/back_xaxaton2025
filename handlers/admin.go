package handlers

import (
	"net/http"
	"time"

	"bekend/database"
	"bekend/models"
	"bekend/services"
	"bekend/utils"

	"github.com/gin-gonic/gin"
)

type AdminHandler struct {
	emailService *services.EmailService
}

func NewAdminHandler() *AdminHandler {
	return &AdminHandler{
		emailService: services.NewEmailService(),
	}
}

type UpdateUserRequest struct {
	FullName string `json:"fullName"`
	Role     string `json:"role"`
	Status   string `json:"status"`
}

type ResetUserPasswordRequest struct {
	Password string `json:"password" binding:"required"`
}

type UserFilterRequest struct {
	FullName   string    `json:"fullName"`
	Role       string    `json:"role"`
	Status     string    `json:"status"`
	DateFrom   time.Time `json:"dateFrom"`
	DateTo     time.Time `json:"dateTo"`
}

func (h *AdminHandler) GetUsers(c *gin.Context) {
	var users []models.User
	query := database.DB

	fullName := c.Query("fullName")
	if fullName != "" {
		query = query.Where("full_name ILIKE ?", "%"+fullName+"%")
	}

	role := c.Query("role")
	if role != "" {
		query = query.Where("role = ?", role)
	}

	status := c.Query("status")
	if status != "" {
		query = query.Where("status = ?", status)
	}

	dateFrom := c.Query("dateFrom")
	if dateFrom != "" {
		if t, err := time.Parse("2006-01-02", dateFrom); err == nil {
			query = query.Where("created_at >= ?", t)
		}
	}

	dateTo := c.Query("dateTo")
	if dateTo != "" {
		if t, err := time.Parse("2006-01-02", dateTo); err == nil {
			query = query.Where("created_at <= ?", t)
		}
	}

	if err := query.Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении пользователей"})
		return
	}

	result := make([]gin.H, len(users))
	for i, user := range users {
		result[i] = gin.H{
			"id":        user.ID,
			"fullName":  user.FullName,
			"email":     user.Email,
			"role":      string(user.Role),
			"status":    string(user.Status),
			"createdAt": user.CreatedAt,
		}
	}

	c.JSON(http.StatusOK, result)
}

func (h *AdminHandler) GetUser(c *gin.Context) {
	userID := c.Param("id")

	var user models.User
	if err := database.DB.Where("id = ?", userID).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Пользователь не найден"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":        user.ID,
		"fullName":  user.FullName,
		"email":     user.Email,
		"role":      string(user.Role),
		"status":    string(user.Status),
		"createdAt": user.CreatedAt,
	})
}

func (h *AdminHandler) UpdateUser(c *gin.Context) {
	userID := c.Param("id")
	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверные данные"})
		return
	}

	var user models.User
	if err := database.DB.Where("id = ?", userID).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Пользователь не найден"})
		return
	}

	if req.FullName != "" {
		if !utils.ValidateFullName(req.FullName) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ФИО должно содержать только русские буквы"})
			return
		}
		user.FullName = req.FullName
	}

	if req.Role != "" {
		user.Role = models.UserRole(req.Role)
	}

	if req.Status != "" {
		user.Status = models.UserStatus(req.Status)
	}

	if err := database.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при обновлении пользователя"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Пользователь обновлен"})
}

func (h *AdminHandler) ResetUserPassword(c *gin.Context) {
	userID := c.Param("id")
	var req ResetUserPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверные данные"})
		return
	}

	if !utils.ValidatePassword(req.Password) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Пароль должен содержать минимум 8 символов, латинские буквы, цифры и специальные символы"})
		return
	}

	var user models.User
	if err := database.DB.Where("id = ?", userID).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Пользователь не найден"})
		return
	}

	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при обновлении пароля"})
		return
	}

	user.Password = hashedPassword
	if err := database.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при обновлении пароля"})
		return
	}

	if err := h.emailService.SendEmail(user.Email, "Пароль изменен администратором", "Ваш новый пароль: "+req.Password); err != nil {
	}

	c.JSON(http.StatusOK, gin.H{"message": "Пароль успешно изменен и отправлен на почту пользователя"})
}

func (h *AdminHandler) DeleteUser(c *gin.Context) {
	userID := c.Param("id")

	var user models.User
	if err := database.DB.Where("id = ?", userID).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Пользователь не найден"})
		return
	}

	user.Status = models.UserStatusDeleted
	if err := database.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при удалении пользователя"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Пользователь удален"})
}

func (h *AdminHandler) GetAdminEvents(c *gin.Context) {
	var events []models.Event
	query := database.DB.Preload("Organizer").Preload("Participants")

	status := c.Query("status")
	if status != "" {
		query = query.Where("status = ?", status)
	}

	if err := query.Find(&events).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении событий"})
		return
	}

	result := make([]gin.H, len(events))
	for i, event := range events {
		result[i] = gin.H{
			"id":              event.ID,
			"title":           event.Title,
			"shortDescription": event.ShortDescription,
			"fullDescription": event.FullDescription,
			"startDate":       event.StartDate,
			"endDate":         event.EndDate,
			"imageURL":        event.ImageURL,
			"paymentInfo":    event.PaymentInfo,
			"maxParticipants": event.MaxParticipants,
			"status":          event.Status,
			"participantsCount": event.GetParticipantsCount(),
			"organizer": gin.H{
				"id":   event.Organizer.ID,
				"name": event.Organizer.FullName,
			},
			"createdAt": event.CreatedAt,
		}
	}

	c.JSON(http.StatusOK, result)
}

