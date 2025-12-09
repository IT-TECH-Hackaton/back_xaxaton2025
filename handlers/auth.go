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

type AuthHandler struct {
	emailService *services.EmailService
}

func NewAuthHandler() *AuthHandler {
	return &AuthHandler{
		emailService: services.NewEmailService(),
	}
}

type RegisterRequest struct {
	FullName string `json:"fullName" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type VerifyEmailRequest struct {
	Email string `json:"email" binding:"required,email"`
	Code  string `json:"code" binding:"required"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type ResetPasswordRequest struct {
	Token    string `json:"token" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверные данные"})
		return
	}

	if !utils.ValidateFullName(req.FullName) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ФИО должно содержать только русские буквы"})
		return
	}

	if !utils.ValidateEmail(req.Email) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат электронной почты"})
		return
	}

	if !utils.ValidatePassword(req.Password) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Пароль должен содержать минимум 8 символов, латинские буквы, цифры и специальные символы"})
		return
	}

	var existingUser models.User
	if err := database.DB.Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Пользователь с такой почтой уже существует"})
		return
	}

	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при создании пользователя"})
		return
	}

	code, err := utils.GenerateVerificationCode()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка генерации кода"})
		return
	}

	verification := models.EmailVerification{
		Email:     req.Email,
		Code:      code,
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}

	if err := database.DB.Create(&verification).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при создании кода подтверждения"})
		return
	}

	user := models.User{
		FullName:      req.FullName,
		Email:         req.Email,
		Password:      hashedPassword,
		Role:          models.RoleUser,
		Status:        models.UserStatusActive,
		EmailVerified: false,
	}

	if err := database.DB.Create(&user).Error; err != nil {
		database.DB.Where("email = ?", req.Email).Delete(&models.EmailVerification{})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при создании пользователя"})
		return
	}

	if err := h.emailService.SendVerificationCode(req.Email, code); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"message": "Пользователь создан, но не удалось отправить письмо. Используйте /auth/resend-code для повторной отправки",
			"email": req.Email,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Код подтверждения отправлен на почту"})
}

func (h *AuthHandler) VerifyEmail(c *gin.Context) {
	var req VerifyEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверные данные"})
		return
	}

	var verification models.EmailVerification
	if err := database.DB.Where("email = ? AND code = ? AND expires_at > ?", req.Email, req.Code, time.Now()).First(&verification).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный или истекший код подтверждения"})
		return
	}

	var user models.User
	if err := database.DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Пользователь не найден"})
		return
	}

	user.EmailVerified = true
	if err := database.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при обновлении пользователя"})
		return
	}

	database.DB.Delete(&verification)

	if err := h.emailService.SendWelcomeEmail(user.Email, user.FullName); err != nil {
	}

	token, err := utils.GenerateToken(user.ID, user.Email, string(user.Role))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка генерации токена"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user": gin.H{
			"id":    user.ID,
			"email": user.Email,
			"role":  user.Role,
		},
	})
}

func (h *AuthHandler) ResendCode(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required,email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверные данные"})
		return
	}

	var user models.User
	if err := database.DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Пользователь не найден"})
		return
	}

	if user.EmailVerified {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Почта уже подтверждена"})
		return
	}

	code, err := utils.GenerateVerificationCode()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка генерации кода"})
		return
	}

	verification := models.EmailVerification{
		Email:     req.Email,
		Code:      code,
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}

	database.DB.Where("email = ?", req.Email).Delete(&models.EmailVerification{})
	if err := database.DB.Create(&verification).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при создании кода подтверждения"})
		return
	}

	if err := h.emailService.SendVerificationCode(req.Email, code); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка отправки письма. Попробуйте позже"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Код подтверждения отправлен на почту"})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверные данные"})
		return
	}

	var user models.User
	if err := database.DB.Where("email = ? AND status = ?", req.Email, models.UserStatusActive).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Неверный email или пароль"})
		return
	}

	if !user.EmailVerified {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Почта не подтверждена"})
		return
	}

	if !utils.CheckPasswordHash(req.Password, user.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Неверный email или пароль"})
		return
	}

	token, err := utils.GenerateToken(user.ID, user.Email, string(user.Role))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка генерации токена"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user": gin.H{
			"id":    user.ID,
			"email": user.Email,
			"role":  user.Role,
		},
	})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Выход выполнен успешно"})
}

func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверные данные"})
		return
	}

	var user models.User
	if err := database.DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "Если пользователь с такой почтой существует, письмо отправлено"})
		return
	}

	token, err := utils.GenerateResetToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка генерации токена"})
		return
	}

	passwordReset := models.PasswordReset{
		Email:     req.Email,
		Token:     token,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		Used:      false,
	}

	database.DB.Where("email = ?", req.Email).Delete(&models.PasswordReset{})
	if err := database.DB.Create(&passwordReset).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при создании запроса на сброс пароля"})
		return
	}

	if err := h.emailService.SendPasswordResetLink(req.Email, token); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка отправки письма"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Если пользователь с такой почтой существует, письмо отправлено"})
}

func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверные данные"})
		return
	}

	if !utils.ValidatePassword(req.Password) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Пароль должен содержать минимум 8 символов, латинские буквы, цифры и специальные символы"})
		return
	}

	var passwordReset models.PasswordReset
	if err := database.DB.Where("token = ? AND expires_at > ? AND used = ?", req.Token, time.Now(), false).First(&passwordReset).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверная или истекшая ссылка для сброса пароля"})
		return
	}

	var user models.User
	if err := database.DB.Where("email = ?", passwordReset.Email).First(&user).Error; err != nil {
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

	passwordReset.Used = true
	database.DB.Save(&passwordReset)

	if err := h.emailService.SendPasswordChangedNotification(user.Email); err != nil {
	}

	c.JSON(http.StatusOK, gin.H{"message": "Пароль успешно изменен"})
}

