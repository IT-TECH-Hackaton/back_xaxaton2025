package handlers

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"bekend/database"
	"bekend/models"
	"bekend/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type UserHandler struct {
	logger *zap.Logger
}

func NewUserHandler() *UserHandler {
	return &UserHandler{
		logger: utils.GetLogger(),
	}
}

// GetProfile godoc
// @Summary Получить профиль пользователя
// @Description Получение информации о текущем авторизованном пользователе
// @Tags Пользователь
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "Профиль пользователя"
// @Failure 401 {object} map[string]string "Требуется авторизация"
// @Failure 404 {object} map[string]string "Пользователь не найден"
// @Router /user/profile [get]
func (h *UserHandler) GetProfile(c *gin.Context) {
	userIDValue, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Требуется авторизация"})
		return
	}

	userID, ok := userIDValue.(uuid.UUID)
	if !ok {
		h.logger.Error("Неверный тип userID в контексте", zap.Any("userID", userIDValue))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка авторизации"})
		return
	}

	var user models.User
	if err := database.DB.Where("id = ?", userID).First(&user).Error; err != nil {
		h.logger.Error("Пользователь не найден", zap.String("userID", userID.String()), zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "Пользователь не найден"})
		return
	}

	if user.Status == models.UserStatusDeleted {
		h.logger.Warn("Попытка доступа к профилю удаленного пользователя", zap.Any("userID", userID))
		c.JSON(http.StatusForbidden, gin.H{"error": "Доступ запрещен: пользователь удален"})
		return
	}

	var userInterests []models.UserInterest
	if err := database.DB.Preload("Interest").Where("user_id = ?", userID).Find(&userInterests).Error; err != nil {
		h.logger.Warn("Ошибка получения интересов пользователя при получении профиля", zap.String("userID", userID.String()), zap.Error(err))
	}
	
	tags := make([]string, 0, len(userInterests))
	for _, ui := range userInterests {
		if ui.Interest.ID != uuid.Nil && ui.Interest.Name != "" {
			tags = append(tags, ui.Interest.Name)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"uid":        user.ID.String(),
		"firstName":  "",
		"secondName": "",
		"mail":       user.Email,
		"phone":      "",
		"tags":       tags,
		"birthDate":  "",
		"image":      user.AvatarURL,
		"telegram":   user.Telegram,
		"role":       string(user.Role),
		"fullName":   user.FullName,
	})
}

// UpdateProfile godoc
// @Summary Обновить профиль пользователя
// @Description Обновление профиля пользователя (ФИО, Telegram, аватар)
// @Tags Пользователь
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param fullName formData string false "ФИО пользователя (только русские буквы, 2-100 символов)"
// @Param telegram formData string false "Telegram username (5-32 символа, латиница, цифры, подчеркивания)"
// @Param avatar formData file false "Аватар пользователя (jpg, jpeg, png, gif, webp, до 10MB)"
// @Success 200 {object} map[string]interface{} "Профиль обновлен"
// @Failure 400 {object} map[string]string "Ошибка валидации"
// @Failure 401 {object} map[string]string "Требуется авторизация"
// @Failure 404 {object} map[string]string "Пользователь не найден"
// @Failure 500 {object} map[string]string "Внутренняя ошибка сервера"
// @Router /user/profile [put]
func (h *UserHandler) UpdateProfile(c *gin.Context) {
	userIDValue, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Требуется авторизация"})
		return
	}

	userID, ok := userIDValue.(uuid.UUID)
	if !ok {
		h.logger.Error("Неверный тип userID в контексте", zap.Any("userID", userIDValue))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка авторизации"})
		return
	}

	var user models.User
	if err := database.DB.Where("id = ?", userID).First(&user).Error; err != nil {
		h.logger.Error("Пользователь не найден для обновления", zap.String("userID", userID.String()), zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "Пользователь не найден"})
		return
	}

	updated := false

	fullName := c.PostForm("fullName")
	if fullName != "" {
		if !utils.ValidateFullName(fullName) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ФИО должно содержать только русские буквы"})
			return
		}
		if !utils.ValidateStringLength(fullName, utils.MinFullNameLength, utils.MaxFullNameLength) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ФИО должно быть от 2 до 100 символов"})
			return
		}
		user.FullName = fullName
		updated = true
	}

	telegram := c.PostForm("telegram")
	if telegram != "" {
		telegramValue := telegram
		if len(telegramValue) > 0 && telegramValue[0] == '@' {
			telegramValue = telegramValue[1:]
		}
		if !utils.ValidateStringLength(telegramValue, 5, 32) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Telegram username должен быть от 5 до 32 символов"})
			return
		}
		if !utils.ValidateTelegramUsername(telegramValue) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат Telegram username. Используйте только латинские буквы, цифры и подчеркивания"})
			return
		}
		user.Telegram = telegramValue
		updated = true
	}

	fileHeader, err := c.FormFile("avatar")
	if err == nil {
		if fileHeader.Size > utils.MaxAvatarFileSize {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Размер файла не должен превышать 10MB"})
			return
		}

		if fileHeader.Size == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Файл пустой"})
			return
		}

		ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
		allowedExts := []string{".jpg", ".jpeg", ".png", ".gif", ".webp"}
		allowed := false
		for _, e := range allowedExts {
			if ext == e {
				allowed = true
				break
			}
		}

		if !allowed {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Недопустимый формат файла. Разрешены: jpg, jpeg, png, gif, webp"})
			return
		}

		src, err := fileHeader.Open()
		if err != nil {
			h.logger.Error("Ошибка открытия файла аватарки", zap.Error(err))
			c.JSON(http.StatusBadRequest, gin.H{"error": "Ошибка при открытии файла"})
			return
		}
		defer src.Close()

		buffer := make([]byte, 512)
		if _, err := src.Read(buffer); err != nil && err != io.EOF {
			h.logger.Error("Ошибка чтения файла аватарки", zap.Error(err))
			c.JSON(http.StatusBadRequest, gin.H{"error": "Ошибка при чтении файла"})
			return
		}

		mimeType := http.DetectContentType(buffer)
		if !h.isValidImageFile(buffer, mimeType, ext) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Недопустимый тип файла. Файл должен быть изображением (JPEG, PNG, GIF, WebP)"})
			return
		}

		if _, err := src.Seek(0, io.SeekStart); err != nil {
			h.logger.Error("Ошибка сброса указателя файла", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при чтении файла"})
			return
		}

		uploadDir := "uploads/avatars"
		if err := os.MkdirAll(uploadDir, 0755); err != nil {
			h.logger.Error("Ошибка создания директории для аватарок", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при создании директории"})
			return
		}

		if user.AvatarURL != "" && strings.HasPrefix(user.AvatarURL, "/uploads/avatars/") {
			oldPath := strings.TrimPrefix(user.AvatarURL, "/")
			if _, err := os.Stat(oldPath); err == nil {
				if err := os.Remove(oldPath); err != nil {
					h.logger.Warn("Не удалось удалить старый файл аватарки", zap.String("path", oldPath), zap.Error(err))
				}
			}
		}

		filename := fmt.Sprintf("%s_%d%s", uuid.New().String(), time.Now().Unix(), ext)
		filePath := filepath.Join(uploadDir, filename)

		dst, err := os.Create(filePath)
		if err != nil {
			h.logger.Error("Ошибка создания файла аватарки", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при создании файла"})
			return
		}
		defer dst.Close()

		if _, err := io.Copy(dst, src); err != nil {
			h.logger.Error("Ошибка сохранения файла аватарки", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при сохранении файла"})
			return
		}

		user.AvatarURL = fmt.Sprintf("/uploads/avatars/%s", filename)
		updated = true
	}

	if !updated {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Не указаны поля для обновления"})
		return
	}

	if err := database.DB.Save(&user).Error; err != nil {
		h.logger.Error("Ошибка сохранения профиля", zap.Any("userID", userID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при обновлении профиля"})
		return
	}

	var userInterests []models.UserInterest
	if err := database.DB.Preload("Interest").Where("user_id = ?", userID).Find(&userInterests).Error; err != nil {
		h.logger.Warn("Ошибка получения интересов пользователя при обновлении профиля", zap.String("userID", userID.String()), zap.Error(err))
	}
	
	tags := make([]string, 0, len(userInterests))
	for _, ui := range userInterests {
		if ui.Interest.ID != uuid.Nil && ui.Interest.Name != "" {
			tags = append(tags, ui.Interest.Name)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Профиль обновлен",
		"user": gin.H{
			"uid":        user.ID.String(),
			"firstName":  "",
			"secondName": "",
			"mail":       user.Email,
			"phone":      "",
			"tags":       tags,
			"birthDate":  "",
			"image":      user.AvatarURL,
			"telegram":   user.Telegram,
			"role":       string(user.Role),
			"fullName":   user.FullName,
		},
	})
}

func (h *UserHandler) isValidImageFile(buffer []byte, mimeType string, ext string) bool {
	allowedMimeTypes := []string{
		"image/jpeg",
		"image/png",
		"image/gif",
		"image/webp",
	}

	mimeAllowed := false
	for _, mime := range allowedMimeTypes {
		if strings.HasPrefix(mimeType, mime) {
			mimeAllowed = true
			break
		}
	}

	if !mimeAllowed {
		return false
	}

	extLower := strings.ToLower(ext)
	allowedExts := []string{".jpg", ".jpeg", ".png", ".gif", ".webp"}
	extAllowed := false
	for _, e := range allowedExts {
		if extLower == e {
			extAllowed = true
			break
		}
	}

	if !extAllowed {
		return false
	}

	if len(buffer) < 4 {
		return false
	}

	jpegMagic := []byte{0xFF, 0xD8, 0xFF}
	pngMagic := []byte{0x89, 0x50, 0x4E, 0x47}
	gifMagic := []byte{0x47, 0x49, 0x46, 0x38}
	webpMagic := []byte{0x52, 0x49, 0x46, 0x46}

	if bytes.HasPrefix(buffer, jpegMagic) {
		return extLower == ".jpg" || extLower == ".jpeg"
	}
	if bytes.HasPrefix(buffer, pngMagic) {
		return extLower == ".png"
	}
	if bytes.HasPrefix(buffer, gifMagic) {
		return extLower == ".gif"
	}
	if bytes.HasPrefix(buffer, webpMagic) && len(buffer) >= 12 {
		if bytes.Equal(buffer[8:12], []byte{0x57, 0x45, 0x42, 0x50}) {
			return extLower == ".webp"
		}
	}

	return false
}
