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

type CreateUserRequest struct {
	FullName string `json:"fullName" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
	Role     string `json:"role"`
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

	page := c.DefaultQuery("page", "1")
	limit := c.DefaultQuery("limit", "20")
	pageInt := 1
	limitInt := 20

	if p, err := strconv.Atoi(page); err == nil && p > 0 {
		pageInt = p
	}
	if l, err := strconv.Atoi(limit); err == nil && l > 0 && l <= 100 {
		limitInt = l
	}

	offset := (pageInt - 1) * limitInt

	fullName := c.Query("fullName")
	if fullName != "" {
		if !utils.ValidateStringLength(fullName, 2, 100) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ФИО должно быть от 2 до 100 символов"})
			return
		}
		query = query.Where("full_name ILIKE ?", "%"+fullName+"%")
	}

	role := c.Query("role")
	if role != "" {
		if !models.IsValidUserRole(models.UserRole(role)) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Неверная роль пользователя"})
			return
		}
		query = query.Where("role = ?", role)
	}

	status := c.Query("status")
	if status != "" {
		if !models.IsValidUserStatus(models.UserStatus(status)) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный статус пользователя"})
			return
		}
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

	var total int64
	query.Model(&models.User{}).Count(&total)

	if err := query.Offset(offset).Limit(limitInt).Order("created_at DESC").Find(&users).Error; err != nil {
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

	totalPages := int((total + int64(limitInt) - 1) / int64(limitInt))
	c.JSON(http.StatusOK, gin.H{
		"data": result,
		"pagination": gin.H{
			"page":       pageInt,
			"limit":     limitInt,
			"total":     total,
			"totalPages": totalPages,
		},
	})
}

func (h *AdminHandler) GetUser(c *gin.Context) {
	userID := c.Param("id")

	if !utils.ValidateUUID(userID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат ID пользователя"})
		return
	}

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

	if !utils.ValidateUUID(userID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат ID пользователя"})
		return
	}

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
		if !utils.ValidateStringLength(req.FullName, 2, 100) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ФИО должно быть от 2 до 100 символов"})
			return
		}
		user.FullName = req.FullName
	}

	if req.Role != "" {
		if !models.IsValidUserRole(models.UserRole(req.Role)) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Неверная роль. Допустимые значения: Пользователь, Администратор"})
			return
		}
		user.Role = models.UserRole(req.Role)
	}

	if req.Status != "" {
		if !models.IsValidUserStatus(models.UserStatus(req.Status)) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный статус. Допустимые значения: Активен, Удален"})
			return
		}
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

	if !utils.ValidateUUID(userID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат ID пользователя"})
		return
	}

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
	user.EmailVerified = false
	if err := database.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при обновлении пароля"})
		return
	}

	resetToken, err := utils.GenerateResetToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка генерации токена"})
		return
	}

	passwordReset := models.PasswordReset{
		Email:     user.Email,
		Token:     resetToken,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		Used:      false,
	}

	database.DB.Where("email = ?", user.Email).Delete(&models.PasswordReset{})
	if err := database.DB.Create(&passwordReset).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при создании токена сброса"})
		return
	}

	resetURL := fmt.Sprintf("%s/auth/reset-password?token=%s", config.AppConfig.FrontendURL, resetToken)
	if err := h.emailService.SendPasswordResetLink(user.Email, resetToken); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"message": "Пароль изменен. Ссылка для установки нового пароля отправлена на почту пользователя",
			"resetURL": resetURL,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Пароль изменен. Ссылка для установки нового пароля отправлена на почту пользователя"})
}

func (h *AdminHandler) DeleteUser(c *gin.Context) {
	userID := c.Param("id")

	if !utils.ValidateUUID(userID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат ID пользователя"})
		return
	}

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

func (h *AdminHandler) CreateUser(c *gin.Context) {
	var req dto.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверные данные"})
		return
	}

	if !utils.ValidateEmail(req.Email) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат email"})
		return
	}

	if !utils.ValidatePassword(req.Password) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Пароль должен содержать минимум 8 символов, латинские буквы, цифры и специальные символы"})
		return
	}

	if !utils.ValidateFullName(req.FullName) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ФИО должно содержать только русские буквы"})
		return
	}

	if !utils.ValidateStringLength(req.FullName, 2, 100) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ФИО должно быть от 2 до 100 символов"})
		return
	}

	var existingUser models.User
	if err := database.DB.Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Пользователь с таким email уже существует"})
		return
	}

	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при хешировании пароля"})
		return
	}

	role := models.RoleUser
	if req.Role != "" {
		if !models.IsValidUserRole(models.UserRole(req.Role)) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Неверная роль пользователя"})
			return
		}
		role = models.UserRole(req.Role)
	}

	user := models.User{
		FullName:      req.FullName,
		Email:         req.Email,
		Password:      hashedPassword,
		Role:          role,
		Status:        models.UserStatusActive,
		EmailVerified: true,
		AuthProvider:  "email",
	}

	if err := database.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при создании пользователя"})
		return
	}

	go h.emailService.SendWelcomeEmail(user.Email, user.FullName)

	c.JSON(http.StatusCreated, gin.H{
		"message": "Пользователь успешно создан",
		"user": gin.H{
			"id":       user.ID,
			"email":    user.Email,
			"fullName": user.FullName,
			"role":     user.Role,
		},
	})
}

func (h *AdminHandler) GetAdminEvents(c *gin.Context) {
	var events []models.Event
	query := database.DB.Preload("Organizer").Preload("Participants")

	page := c.DefaultQuery("page", "1")
	limit := c.DefaultQuery("limit", "20")
	pageInt := 1
	limitInt := 20

	if p, err := strconv.Atoi(page); err == nil && p > 0 {
		pageInt = p
	}
	if l, err := strconv.Atoi(limit); err == nil && l > 0 && l <= 100 {
		limitInt = l
	}

	offset := (pageInt - 1) * limitInt

	status := c.Query("status")
	if status != "" {
		if !models.IsValidEventStatus(models.EventStatus(status)) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный статус события"})
			return
		}
		query = query.Where("status = ?", status)
	}

	var total int64
	query.Model(&models.Event{}).Count(&total)

	if err := query.Offset(offset).Limit(limitInt).Order("created_at DESC").Find(&events).Error; err != nil {
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
			"address":         event.Address,
			"latitude":        event.Latitude,
			"longitude":       event.Longitude,
			"yandexMapLink":   event.YandexMapLink,
			"organizer": gin.H{
				"id":   event.Organizer.ID,
				"name": event.Organizer.FullName,
			},
			"createdAt": event.CreatedAt,
		}
	}

	totalPages := int((total + int64(limitInt) - 1) / int64(limitInt))
	c.JSON(http.StatusOK, gin.H{
		"data": result,
		"pagination": gin.H{
			"page":       pageInt,
			"limit":     limitInt,
			"total":     total,
			"totalPages": totalPages,
		},
	})
}

