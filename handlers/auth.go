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
	"go.uber.org/zap"
)

type AuthHandler struct {
	emailService *services.EmailService
	logger       *zap.Logger
}

func NewAuthHandler() *AuthHandler {
	return &AuthHandler{
		emailService: services.NewEmailService(),
		logger:       utils.GetLogger(),
	}
}

// Register godoc
// @Summary Регистрация нового пользователя
// @Description Регистрация пользователя с отправкой кода подтверждения на email
// @Tags Авторизация
// @Accept json
// @Produce json
// @Param request body dto.RegisterRequest true "Данные для регистрации"
// @Success 200 {object} map[string]interface{} "Код подтверждения отправлен"
// @Failure 400 {object} map[string]string "Ошибка валидации"
// @Failure 409 {object} map[string]string "Пользователь уже существует"
// @Failure 500 {object} map[string]string "Внутренняя ошибка сервера"
// @Router /auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверные данные"})
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

	if !utils.ValidateEmail(req.Email) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат электронной почты"})
		return
	}

	if !utils.ValidatePassword(req.Password) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Пароль должен содержать минимум 8 символов, латинские буквы, цифры и специальные символы"})
		return
	}

	if req.Password != req.PasswordConfirm {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Пароли не совпадают"})
		return
	}

	var existingUser models.User
	if err := database.DB.Where("email = ? AND status != ?", req.Email, models.UserStatusDeleted).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Пользователь с такой почтой уже существует"})
		return
	}

	if err := database.DB.Unscoped().Where("email = ? AND status = ?", req.Email, models.UserStatusDeleted).First(&existingUser).Error; err == nil {
		database.DB.Unscoped().Delete(&existingUser)
	}

	var existingPending models.RegistrationPending
	if err := database.DB.Where("email = ?", req.Email).First(&existingPending).Error; err == nil {
		database.DB.Delete(&existingPending)
	}

	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		h.logger.Error("Ошибка хеширования пароля", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при создании пользователя"})
		return
	}

	code, err := utils.GenerateVerificationCode()
	if err != nil {
		h.logger.Error("Ошибка генерации кода", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка генерации кода"})
		return
	}

	registrationPending := models.RegistrationPending{
		Email:        req.Email,
		FullName:     req.FullName,
		PasswordHash: hashedPassword,
		Code:         code,
		ExpiresAt:    time.Now().Add(10 * time.Minute),
	}

	if err := database.DB.Create(&registrationPending).Error; err != nil {
		h.logger.Error("Ошибка создания записи регистрации", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при создании кода подтверждения"})
		return
	}

	if err := h.emailService.SendVerificationCode(req.Email, code); err != nil {
		h.logger.Error("Ошибка отправки кода подтверждения", zap.String("email", req.Email), zap.Error(err))
		c.JSON(http.StatusOK, gin.H{
			"message": "Код подтверждения не удалось отправить. Используйте /api/auth/resend-code для повторной отправки",
			"email":   req.Email,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Код подтверждения отправлен на вашу электронную почту. Пожалуйста, проверьте почту для завершения регистрации.",
		"email":   req.Email,
	})
}

// VerifyEmail godoc
// @Summary Подтверждение email
// @Description Подтверждение email с помощью кода из письма и создание учетной записи
// @Tags Авторизация
// @Accept json
// @Produce json
// @Param request body dto.VerifyEmailRequest true "Email и код подтверждения"
// @Success 200 {object} dto.AuthResponse "Учетная запись создана, токен выдан"
// @Failure 400 {object} map[string]interface{} "Неверный или истекший код"
// @Failure 409 {object} map[string]string "Пользователь уже существует"
// @Failure 500 {object} map[string]string "Внутренняя ошибка сервера"
// @Router /auth/verify-email [post]
func (h *AuthHandler) VerifyEmail(c *gin.Context) {
	var req dto.VerifyEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверные данные"})
		return
	}

	if !utils.ValidateVerificationCode(req.Code) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Неверный код подтверждения",
			"message": "Код должен состоять из 6 цифр. Вы можете запросить новый код.",
		})
		return
	}

	var registrationPending models.RegistrationPending
	if err := database.DB.Where("email = ? AND code = ? AND expires_at > ?", req.Email, req.Code, time.Now()).First(&registrationPending).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Неверный или истекший код подтверждения",
			"message": "Код подтверждения неверен или истек. Вы можете запросить новый код.",
		})
		return
	}

	var existingUser models.User
	if err := database.DB.Where("email = ? AND status != ?", req.Email, models.UserStatusDeleted).First(&existingUser).Error; err == nil {
		database.DB.Delete(&registrationPending)
		c.JSON(http.StatusConflict, gin.H{"error": "Пользователь с такой почтой уже существует"})
		return
	}

	var deletedUser models.User
	if err := database.DB.Unscoped().Where("email = ? AND status = ?", req.Email, models.UserStatusDeleted).First(&deletedUser).Error; err == nil {
		database.DB.Unscoped().Delete(&deletedUser)
	}

	user := models.User{
		FullName:      registrationPending.FullName,
		Email:         registrationPending.Email,
		Password:      registrationPending.PasswordHash,
		Role:          models.RoleUser,
		Status:        models.UserStatusActive,
		EmailVerified: true,
		AuthProvider:  "email",
	}

	tx := database.DB.Begin()
	if err := tx.Create(&user).Error; err != nil {
		tx.Rollback()
		h.logger.Error("Ошибка создания пользователя после подтверждения", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при создании учетной записи"})
		return
	}

	if err := tx.Delete(&registrationPending).Error; err != nil {
		tx.Rollback()
		h.logger.Error("Ошибка удаления записи регистрации", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при завершении регистрации"})
		return
	}

	if err := tx.Commit().Error; err != nil {
		h.logger.Error("Ошибка коммита транзакции", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при завершении регистрации"})
		return
	}

	go func() {
		if err := h.emailService.SendWelcomeEmail(user.Email, user.FullName); err != nil {
			h.logger.Error("Ошибка отправки приветственного письма", zap.String("email", user.Email), zap.Error(err))
		}
	}()

	token, err := utils.GenerateToken(user.ID, user.Email, string(user.Role))
	if err != nil {
		h.logger.Error("Ошибка генерации токена", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка генерации токена"})
		return
	}

	c.JSON(http.StatusOK, dto.AuthResponse{
		Token: token,
		User: dto.UserInfo{
			ID:    user.ID.String(),
			Email: user.Email,
			Role:  string(user.Role),
		},
	})
}

// ResendCode godoc
// @Summary Повторная отправка кода подтверждения
// @Description Отправка нового кода подтверждения на email
// @Tags Авторизация
// @Accept json
// @Produce json
// @Param request body object{email=string} true "Email для повторной отправки кода"
// @Success 200 {object} map[string]string "Новый код отправлен"
// @Failure 400 {object} map[string]string "Ошибка валидации или почта уже подтверждена"
// @Failure 404 {object} map[string]string "Запрос на регистрацию не найден"
// @Failure 500 {object} map[string]string "Внутренняя ошибка сервера"
// @Router /auth/resend-code [post]
func (h *AuthHandler) ResendCode(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required,email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверные данные"})
		return
	}

	var registrationPending models.RegistrationPending
	if err := database.DB.Where("email = ?", req.Email).First(&registrationPending).Error; err != nil {
		var user models.User
		if err := database.DB.Where("email = ?", req.Email).First(&user).Error; err == nil {
			if user.EmailVerified {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Почта уже подтверждена"})
				return
			}
		} else {
			c.JSON(http.StatusNotFound, gin.H{"error": "Запрос на регистрацию не найден"})
			return
		}
	}

	code, err := utils.GenerateVerificationCode()
	if err != nil {
		h.logger.Error("Ошибка генерации кода", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка генерации кода"})
		return
	}

	registrationPending.Code = code
	registrationPending.ExpiresAt = time.Now().Add(10 * time.Minute)

	if err := database.DB.Save(&registrationPending).Error; err != nil {
		h.logger.Error("Ошибка обновления кода", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при создании кода подтверждения"})
		return
	}

	if err := h.emailService.SendVerificationCode(req.Email, code); err != nil {
		h.logger.Error("Ошибка отправки кода", zap.String("email", req.Email), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка отправки письма. Попробуйте позже"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Новый код подтверждения отправлен на вашу почту"})
}

// Login godoc
// @Summary Вход в систему
// @Description Авторизация пользователя по email и паролю
// @Tags Авторизация
// @Accept json
// @Produce json
// @Param request body dto.LoginRequest true "Email и пароль"
// @Success 200 {object} dto.AuthResponse "Успешный вход, токен выдан"
// @Failure 400 {object} map[string]string "Ошибка валидации"
// @Failure 401 {object} map[string]interface{} "Неверные данные или почта не подтверждена"
// @Failure 500 {object} map[string]string "Внутренняя ошибка сервера"
// @Router /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
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
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "Почта не подтверждена",
			"message": "Пожалуйста, подтвердите свою электронную почту перед входом",
		})
		return
	}

	if user.AuthProvider == "yandex" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Войдите через Яндекс"})
		return
	}

	if user.Password == "" || !utils.CheckPasswordHash(req.Password, user.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Неверный email или пароль"})
		return
	}

	token, err := utils.GenerateToken(user.ID, user.Email, string(user.Role))
	if err != nil {
		h.logger.Error("Ошибка генерации токена", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка генерации токена"})
		return
	}

	ipAddress := c.ClientIP()
	go h.emailService.SendLoginNotification(user.Email, user.FullName, ipAddress)

	c.JSON(http.StatusOK, dto.AuthResponse{
		Token: token,
		User: dto.UserInfo{
			ID:    user.ID.String(),
			Email: user.Email,
			Role:  string(user.Role),
		},
	})
}

// Logout godoc
// @Summary Выход из системы
// @Description Выход из системы (на клиенте необходимо удалить токен)
// @Tags Авторизация
// @Accept json
// @Produce json
// @Success 200 {object} map[string]string "Выход выполнен"
// @Router /auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Выход выполнен успешно"})
}

// ForgotPassword godoc
// @Summary Запрос восстановления пароля
// @Description Отправка письма со ссылкой для сброса пароля
// @Tags Авторизация
// @Accept json
// @Produce json
// @Param request body dto.ForgotPasswordRequest true "Email для восстановления"
// @Success 200 {object} map[string]string "Письмо отправлено (или пользователь не найден)"
// @Failure 400 {object} map[string]string "Ошибка валидации"
// @Failure 500 {object} map[string]string "Внутренняя ошибка сервера"
// @Router /auth/forgot-password [post]
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req dto.ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверные данные"})
		return
	}

	if !utils.ValidateEmail(req.Email) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат электронной почты"})
		return
	}

	var user models.User
	if err := database.DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "Если пользователь с такой почтой существует, письмо со ссылкой для сброса пароля отправлено на указанную почту"})
		return
	}

	token, err := utils.GenerateResetToken()
	if err != nil {
		h.logger.Error("Ошибка генерации токена сброса", zap.Error(err))
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
		h.logger.Error("Ошибка создания записи сброса пароля", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при создании запроса на сброс пароля"})
		return
	}

	if err := h.emailService.SendPasswordResetLink(req.Email, token); err != nil {
		h.logger.Error("Ошибка отправки ссылки сброса", zap.String("email", req.Email), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка отправки письма"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Если пользователь с такой почтой существует, письмо со ссылкой для сброса пароля отправлено на указанную почту"})
}

// ResetPassword godoc
// @Summary Сброс пароля
// @Description Сброс пароля по токену из письма
// @Tags Авторизация
// @Accept json
// @Produce json
// @Param request body dto.ResetPasswordRequest true "Токен и новый пароль"
// @Success 200 {object} map[string]string "Пароль успешно изменен"
// @Failure 400 {object} map[string]interface{} "Неверный или истекший токен, ошибка валидации"
// @Failure 404 {object} map[string]string "Пользователь не найден"
// @Failure 500 {object} map[string]string "Внутренняя ошибка сервера"
// @Router /auth/reset-password [post]
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req dto.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверные данные"})
		return
	}

	if !utils.ValidatePassword(req.Password) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Пароль должен содержать минимум 8 символов, латинские буквы, цифры и специальные символы"})
		return
	}

	if req.Password != req.PasswordConfirm {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Пароли не совпадают"})
		return
	}

	var passwordReset models.PasswordReset
	if err := database.DB.Where("token = ? AND expires_at > ? AND used = ?", req.Token, time.Now(), false).First(&passwordReset).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Неверная или истекшая ссылка для сброса пароля",
			"message": "Ссылка для сброса пароля неверна или истекла. Вы можете запросить новую ссылку.",
		})
		return
	}

	var user models.User
	if err := database.DB.Where("email = ?", passwordReset.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Пользователь не найден"})
		return
	}

	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		h.logger.Error("Ошибка хеширования пароля", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при обновлении пароля"})
		return
	}

	user.Password = hashedPassword
	if err := database.DB.Save(&user).Error; err != nil {
		h.logger.Error("Ошибка сохранения нового пароля", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при обновлении пароля"})
		return
	}

	passwordReset.Used = true
	database.DB.Save(&passwordReset)

	go func() {
		if err := h.emailService.SendPasswordChangedNotification(user.Email); err != nil {
			h.logger.Error("Ошибка отправки уведомления о смене пароля", zap.String("email", user.Email), zap.Error(err))
		}
	}()

	c.JSON(http.StatusOK, gin.H{
		"message": "Пароль успешно изменен. Теперь вы можете войти в систему, используя новый пароль.",
	})
}
