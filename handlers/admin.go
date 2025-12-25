package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"bekend/database"
	"bekend/dto"
	"bekend/models"
	"bekend/services"
	"bekend/utils"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
	"go.uber.org/zap"
)

type AdminHandler struct {
	emailService *services.EmailService
	logger       *zap.Logger
}

func NewAdminHandler() *AdminHandler {
	return &AdminHandler{
		emailService: services.NewEmailService(),
		logger:       utils.GetLogger(),
	}
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
		if !utils.ValidateStringLength(fullName, 1, 100) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ФИО должно быть от 1 до 100 символов"})
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
			query = query.Where("created_at <= ?", t.Add(24*time.Hour))
		}
	}

	var total int64
	query.Model(&models.User{}).Count(&total)

	if err := query.Offset(offset).Limit(limitInt).Order("created_at DESC").Find(&users).Error; err != nil {
		h.logger.Error("Ошибка при получении пользователей", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении пользователей"})
		return
	}

	result := make([]dto.UserResponse, len(users))
	for i, user := range users {
		result[i] = dto.UserResponse{
			ID:        user.ID.String(),
			FullName:  user.FullName,
			Email:     user.Email,
			Role:      string(user.Role),
			Status:    string(user.Status),
			CreatedAt: user.CreatedAt.Format("2006-01-02"),
		}
	}

	totalPages := int((total + int64(limitInt) - 1) / int64(limitInt))
	c.JSON(http.StatusOK, dto.PaginationResponse{
		Data: result,
		Pagination: dto.Pagination{
			Page:       pageInt,
			Limit:      limitInt,
			Total:      total,
			TotalPages: totalPages,
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
		h.logger.Error("Пользователь не найден админом", zap.String("userID", userID), zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "Пользователь не найден"})
		return
	}

	c.JSON(http.StatusOK, dto.UserResponse{
		ID:        user.ID.String(),
		FullName:  user.FullName,
		Email:     user.Email,
		Role:      string(user.Role),
		Status:    string(user.Status),
		CreatedAt: user.CreatedAt.Format("2006-01-02"),
	})
}

func (h *AdminHandler) UpdateUser(c *gin.Context) {
	userID := c.Param("id")

	if !utils.ValidateUUID(userID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат ID пользователя"})
		return
	}

	var req dto.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Неверные данные при обновлении пользователя админом", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверные данные"})
		return
	}

	var user models.User
	if err := database.DB.Where("id = ?", userID).First(&user).Error; err != nil {
		h.logger.Error("Пользователь не найден для обновления админом", zap.String("userID", userID), zap.Error(err))
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
		h.logger.Error("Ошибка сохранения обновленного пользователя админом", zap.String("userID", userID), zap.Error(err))
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

	var req dto.ResetUserPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Неверные данные при сбросе пароля админом", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверные данные"})
		return
	}

	if !utils.ValidatePassword(req.Password) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Пароль должен содержать минимум 8 символов, латинские буквы, цифры и специальные символы"})
		return
	}

	var user models.User
	if err := database.DB.Where("id = ?", userID).First(&user).Error; err != nil {
		h.logger.Error("Пользователь не найден для сброса пароля админом", zap.String("userID", userID), zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "Пользователь не найден"})
		return
	}

	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		h.logger.Error("Ошибка хеширования пароля админом", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при обновлении пароля"})
		return
	}

	user.Password = hashedPassword
	if err := database.DB.Save(&user).Error; err != nil {
		h.logger.Error("Ошибка сохранения нового пароля админом", zap.String("userID", userID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при обновлении пароля"})
		return
	}

	go func() {
		if err := h.emailService.SendPasswordToUser(user.Email, user.FullName, req.Password); err != nil {
			h.logger.Error("Ошибка отправки пароля пользователю", zap.String("email", user.Email), zap.Error(err))
		}
	}()

	c.JSON(http.StatusOK, gin.H{
		"message": "Пароль успешно изменен и отправлен на почту пользователя",
	})
}

func (h *AdminHandler) DeleteUser(c *gin.Context) {
	userID := c.Param("id")

	if !utils.ValidateUUID(userID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат ID пользователя"})
		return
	}

	var user models.User
	if err := database.DB.Where("id = ?", userID).First(&user).Error; err != nil {
		h.logger.Error("Пользователь не найден для удаления админом", zap.String("userID", userID), zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "Пользователь не найден"})
		return
	}

	user.Status = models.UserStatusDeleted
	if err := database.DB.Save(&user).Error; err != nil {
		h.logger.Error("Ошибка удаления пользователя админом из БД", zap.String("userID", userID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при удалении пользователя"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Пользователь помечен как удаленный"})
}

func (h *AdminHandler) CreateUser(c *gin.Context) {
	var req dto.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Неверные данные при создании пользователя админом", zap.Error(err))
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
		h.logger.Info("Попытка создания пользователя админом с существующей почтой", zap.String("email", req.Email))
		c.JSON(http.StatusConflict, gin.H{"error": "Пользователь с таким email уже существует"})
		return
	}

	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		h.logger.Error("Ошибка хеширования пароля при создании пользователя админом", zap.Error(err))
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
		h.logger.Error("Ошибка создания пользователя админом в БД", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при создании пользователя"})
		return
	}

	go func() {
		if err := h.emailService.SendWelcomeEmail(user.Email, user.FullName); err != nil {
			h.logger.Error("Ошибка отправки приветственного письма при создании пользователя админом", zap.String("email", user.Email), zap.Error(err))
		}
	}()

	c.JSON(http.StatusCreated, gin.H{
		"message": "Пользователь успешно создан",
		"user": dto.UserResponse{
			ID:        user.ID.String(),
			Email:     user.Email,
			FullName:  user.FullName,
			Role:      string(user.Role),
			Status:    string(user.Status),
			CreatedAt: user.CreatedAt.Format("2006-01-02"),
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
		h.logger.Error("Ошибка при получении событий для админа", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении событий"})
		return
	}

	result := make([]dto.EventResponse, len(events))
	for i, event := range events {
		result[i] = dto.EventResponse{
			ID:               event.ID.String(),
			Title:            event.Title,
			ShortDescription: event.ShortDescription,
			FullDescription:  event.FullDescription,
			StartDate:        event.StartDate,
			EndDate:          event.EndDate,
			ImageURL:         event.ImageURL,
			PaymentInfo:      event.PaymentInfo,
			MaxParticipants:  event.MaxParticipants,
			Status:           string(event.Status),
			ParticipantsCount: event.GetParticipantsCount(),
			Address:          event.Address,
			Latitude:         event.Latitude,
			Longitude:        event.Longitude,
			YandexMapLink:    event.YandexMapLink,
			Organizer: dto.UserInfo{
				ID:       event.Organizer.ID.String(),
				FullName: event.Organizer.FullName,
				Email:    event.Organizer.Email,
			},
		}
	}

	totalPages := int((total + int64(limitInt) - 1) / int64(limitInt))
	c.JSON(http.StatusOK, dto.PaginationResponse{
		Data: result,
		Pagination: dto.Pagination{
			Page:       pageInt,
			Limit:      limitInt,
			Total:      total,
			TotalPages: totalPages,
		},
	})
}

func (h *AdminHandler) ExportUsers(c *gin.Context) {
	format := c.DefaultQuery("format", "xlsx")

	var users []models.User
	query := database.DB.Unscoped() // Include soft-deleted users for admin export

	fullName := c.Query("fullName")
	if fullName != "" {
		if !utils.ValidateStringLength(fullName, 1, 100) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ФИО должно быть от 1 до 100 символов"})
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
			query = query.Where("created_at <= ?", t.Add(24*time.Hour))
		}
	}

	if err := query.Order("created_at DESC").Find(&users).Error; err != nil {
		h.logger.Error("Ошибка при получении пользователей для экспорта", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении пользователей"})
		return
	}

	h.logger.Info("Экспорт пользователей", zap.Int("count", len(users)))

	if format == "csv" {
		c.Header("Content-Type", "text/csv; charset=utf-8")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=users_%s.csv", time.Now().Format("2006-01-02")))
		c.Writer.WriteString("\xEF\xBB\xBF") // BOM for UTF-8 in Excel
		c.Writer.WriteString("ФИО,Email,Роль,Статус,Дата регистрации\n")
		for _, u := range users {
			c.Writer.WriteString(fmt.Sprintf("%s,%s,%s,%s,%s\n",
				u.FullName,
				u.Email,
				string(u.Role),
				string(u.Status),
				u.CreatedAt.Format("2006-01-02")))
		}
		return
	}

	// Default to XLSX
	f := excelize.NewFile()
	defer func() {
		if err := f.Close(); err != nil {
			h.logger.Error("Ошибка при закрытии файла Excel", zap.Error(err))
		}
	}()

	sheetName := "Пользователи"
	index, err := f.NewSheet(sheetName)
	if err != nil {
		h.logger.Error("Ошибка при создании листа Excel", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при создании файла"})
		return
	}

	// Set active sheet
	f.SetActiveSheet(index)

	// Delete default Sheet1 if it exists
	if sheetIndex, _ := f.GetSheetIndex("Sheet1"); sheetIndex != -1 {
		f.DeleteSheet("Sheet1")
	}

	// Set headers
	f.SetCellValue(sheetName, "A1", "ФИО")
	f.SetCellValue(sheetName, "B1", "Email")
	f.SetCellValue(sheetName, "C1", "Роль")
	f.SetCellValue(sheetName, "D1", "Статус")
	f.SetCellValue(sheetName, "E1", "Дата регистрации")

	// Populate data
	for i, u := range users {
		row := i + 2 // Start from row 2
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), u.FullName)
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), u.Email)
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), string(u.Role))
		f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), string(u.Status))
		f.SetCellValue(sheetName, fmt.Sprintf("E%d", row), u.CreatedAt.Format("2006-01-02"))
	}

	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=users_%s.xlsx", time.Now().Format("2006-01-02")))

	if err := f.Write(c.Writer); err != nil {
		h.logger.Error("Ошибка при записи файла Excel", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при экспорте"})
		return
	}
}
