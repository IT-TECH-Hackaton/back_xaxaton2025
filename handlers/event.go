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
	"github.com/google/uuid"
	"github.com/xuri/excelize/v2"
	"go.uber.org/zap"
)

type EventHandler struct {
	emailService *services.EmailService
	logger       *zap.Logger
}

func NewEventHandler() *EventHandler {
	return &EventHandler{
		emailService: services.NewEmailService(),
		logger:       utils.GetLogger(),
	}
}

func (h *EventHandler) GetEvents(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		userID = nil
	}
	tab := c.Query("tab")

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

	var events []models.Event
	query := database.DB.Preload("Organizer").Preload("Participants").Preload("Categories")

	switch tab {
	case "active":
		query = query.Where("status = ?", models.EventStatusActive)
	case "my":
		if userID != nil {
			query = query.Joins("JOIN event_participants ON events.id = event_participants.event_id").
				Where("event_participants.user_id = ?", userID).
				Where("status IN ?", []models.EventStatus{models.EventStatusActive, models.EventStatusPast})
		} else {
			query = query.Where("1 = 0")
		}
	case "past":
		query = query.Where("status = ?", models.EventStatusPast)
	default:
		if userID != nil {
			query = query.Joins("JOIN event_participants ON events.id = event_participants.event_id").
				Where("event_participants.user_id = ?", userID).
				Where("status IN ?", []models.EventStatus{models.EventStatusActive, models.EventStatusPast})
		} else {
			query = query.Where("1 = 0")
		}
	}

	query = query.Where("status != ?", models.EventStatusRejected)

	var total int64
	query.Model(&models.Event{}).Count(&total)

	if err := query.Offset(offset).Limit(limitInt).Order("start_date ASC").Find(&events).Error; err != nil {
		h.logger.Error("Ошибка при получении событий", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении событий"})
		return
	}

	result := make([]dto.EventResponse, len(events))
	for i, event := range events {
		categories := make([]dto.CategoryInfo, len(event.Categories))
		for j, cat := range event.Categories {
			categories[j] = dto.CategoryInfo{
				ID:   cat.ID.String(),
				Name: cat.Name,
			}
		}
		
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
			Categories:       categories,
			Tags:             event.Tags,
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

func (h *EventHandler) GetEvent(c *gin.Context) {
	eventID := c.Param("id")

	if !utils.ValidateUUID(eventID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат ID события"})
		return
	}

	userID, _ := c.Get("userID")

	var event models.Event
	if err := database.DB.Preload("Organizer").Preload("Participants.User").Preload("Categories").Where("id = ?", eventID).First(&event).Error; err != nil {
		h.logger.Error("Событие не найдено", zap.String("eventID", eventID), zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "Событие не найдено"})
		return
	}

	if event.Status == models.EventStatusRejected && (userID == nil || c.GetString("role") != "Администратор") {
		c.JSON(http.StatusForbidden, gin.H{"error": "Доступ запрещен"})
		return
	}

	isParticipant := false
	if userID != nil {
		var participant models.EventParticipant
		if err := database.DB.Where("event_id = ? AND user_id = ?", eventID, userID).First(&participant).Error; err == nil {
			isParticipant = true
		}
	}

	var reviews []models.EventReview
	var avgRating float64
	var totalReviews int
	if err := database.DB.Where("event_id = ?", eventID).Find(&reviews).Error; err == nil {
		totalReviews = len(reviews)
		if totalReviews > 0 {
			var totalRating int
			for _, r := range reviews {
				totalRating += r.Rating
			}
			avgRating = float64(totalRating) / float64(totalReviews)
		}
	}

	categories := make([]dto.CategoryInfo, len(event.Categories))
	for i, cat := range event.Categories {
		categories[i] = dto.CategoryInfo{
			ID:   cat.ID.String(),
			Name: cat.Name,
		}
	}

	c.JSON(http.StatusOK, dto.EventDetailResponse{
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
		IsParticipant:    isParticipant,
		AverageRating:    avgRating,
		TotalReviews:     totalReviews,
		Categories:       categories,
		Tags:             event.Tags,
		Address:          event.Address,
		Latitude:         event.Latitude,
		Longitude:        event.Longitude,
		YandexMapLink:    event.YandexMapLink,
		Organizer: dto.UserInfo{
			ID:       event.Organizer.ID.String(),
			FullName: event.Organizer.FullName,
			Email:    event.Organizer.Email,
		},
	})
}

func (h *EventHandler) CreateEvent(c *gin.Context) {
	var req dto.CreateEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Неверные данные при создании события", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверные данные"})
		return
	}

	if !utils.ValidateStringLength(req.Title, 1, 200) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Название события должно быть от 1 до 200 символов"})
		return
	}

	if !utils.ValidateStringLength(req.FullDescription, 1, 5000) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Полное описание должно быть от 1 до 5000 символов"})
		return
	}

	if req.ShortDescription != "" && !utils.ValidateStringLength(req.ShortDescription, 1, 500) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Краткое описание должно быть от 1 до 500 символов"})
		return
	}

	if req.StartDate.Before(time.Now()) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Дата начала должна быть в будущем"})
		return
	}

	if req.EndDate.Before(req.StartDate) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Дата окончания должна быть позже даты начала"})
		return
	}

	if req.MaxParticipants != nil && *req.MaxParticipants < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Максимальное количество участников должно быть больше 0"})
		return
	}

	if req.Latitude != nil {
		if *req.Latitude < -90 || *req.Latitude > 90 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Широта должна быть от -90 до 90"})
			return
		}
	}
	if req.Longitude != nil {
		if *req.Longitude < -180 || *req.Longitude > 180 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Долгота должна быть от -180 до 180"})
			return
		}
	}
	if req.Address != "" && !utils.ValidateStringLength(req.Address, 0, 500) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Адрес должен быть до 500 символов"})
		return
	}
	if req.YandexMapLink != "" && !utils.ValidateStringLength(req.YandexMapLink, 0, 1000) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ссылка на карту должна быть до 1000 символов"})
		return
	}

	userID, _ := c.Get("userID")
	organizerID := userID.(uuid.UUID)

	event := models.Event{
		Title:            req.Title,
		ShortDescription: req.ShortDescription,
		FullDescription:  req.FullDescription,
		StartDate:        req.StartDate,
		EndDate:          req.EndDate,
		ImageURL:         req.ImageURL,
		PaymentInfo:      req.PaymentInfo,
		MaxParticipants:  req.MaxParticipants,
		Status:           models.EventStatusActive,
		OrganizerID:      organizerID,
		Tags:             req.Tags,
		Address:          req.Address,
		Latitude:         req.Latitude,
		Longitude:        req.Longitude,
		YandexMapLink:    req.YandexMapLink,
	}

	if err := database.DB.Create(&event).Error; err != nil {
		h.logger.Error("Ошибка при создании события в БД", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при создании события"})
		return
	}

	if len(req.CategoryIDs) > 0 {
		var categories []models.Category
		if err := database.DB.Where("id IN (?)", req.CategoryIDs).Find(&categories).Error; err == nil {
			if err := database.DB.Model(&event).Association("Categories").Append(categories); err != nil {
				h.logger.Error("Ошибка добавления категорий к событию", zap.String("eventID", event.ID.String()), zap.Error(err))
			}
		} else {
			h.logger.Error("Ошибка получения категорий для события", zap.Error(err))
		}
	}

	communityService := services.NewCommunityService()
	go communityService.NotifyCommunitiesAboutEvent(&event)

	if len(req.ParticipantIDs) > 0 {
		var users []models.User
		if err := database.DB.Where("id IN ? AND status = ?", req.ParticipantIDs, models.UserStatusActive).Find(&users).Error; err == nil {
			for _, user := range users {
				participant := models.EventParticipant{
					EventID: event.ID,
					UserID:  user.ID,
				}
				if err := database.DB.Create(&participant).Error; err == nil {
					go h.emailService.SendEventNotification(user.Email, event.Title, "Вы были добавлены в новое событие: "+event.Title+". Дата начала: "+event.StartDate.Format("02.01.2006 15:04"))
				} else {
					h.logger.Error("Ошибка добавления участника при создании события", zap.Any("userID", user.ID), zap.String("eventID", event.ID.String()), zap.Error(err))
				}
			}
		} else {
			h.logger.Error("Ошибка получения пользователей для добавления в событие", zap.Error(err))
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"id":      event.ID,
		"message": "Событие создано",
	})
}

func (h *EventHandler) UpdateEvent(c *gin.Context) {
	eventID := c.Param("id")

	if !utils.ValidateUUID(eventID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат ID события"})
		return
	}

	var req dto.UpdateEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Неверные данные при обновлении события", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверные данные"})
		return
	}

	var event models.Event
	if err := database.DB.Where("id = ?", eventID).First(&event).Error; err != nil {
		h.logger.Error("Событие не найдено для обновления", zap.String("eventID", eventID), zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "Событие не найдено"})
		return
	}

	if req.Title != "" {
		if !utils.ValidateStringLength(req.Title, 1, 200) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Название события должно быть от 1 до 200 символов"})
			return
		}
		event.Title = req.Title
	}
	if req.ShortDescription != "" {
		if !utils.ValidateStringLength(req.ShortDescription, 1, 500) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Краткое описание должно быть от 1 до 500 символов"})
			return
		}
		event.ShortDescription = req.ShortDescription
	}
	if req.FullDescription != "" {
		if !utils.ValidateStringLength(req.FullDescription, 1, 5000) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Полное описание должно быть от 1 до 5000 символов"})
			return
		}
		event.FullDescription = req.FullDescription
	}
	if !req.StartDate.IsZero() {
		if req.StartDate.Before(time.Now()) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Дата начала должна быть в будущем"})
			return
		}
		event.StartDate = req.StartDate
	}
	if !req.EndDate.IsZero() {
		if req.EndDate.Before(event.StartDate) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Дата окончания должна быть позже даты начала"})
			return
		}
		event.EndDate = req.EndDate
	}
	if req.ImageURL != "" {
		event.ImageURL = req.ImageURL
	}
	if req.PaymentInfo != "" {
		if !utils.ValidateStringLength(req.PaymentInfo, 0, 2000) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Информация об оплате должна быть до 2000 символов"})
			return
		}
		event.PaymentInfo = req.PaymentInfo
	}
	if req.MaxParticipants != nil {
		if *req.MaxParticipants < 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Максимальное количество участников должно быть больше 0"})
			return
		}
		event.MaxParticipants = req.MaxParticipants
	}
	if req.Status != "" {
		if !models.IsValidEventStatus(models.EventStatus(req.Status)) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный статус. Допустимые значения: Активное, Прошедшее, Отклоненное"})
			return
		}
		event.Status = models.EventStatus(req.Status)
	}
	if req.Address != "" {
		if !utils.ValidateStringLength(req.Address, 0, 500) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Адрес должен быть до 500 символов"})
			return
		}
		event.Address = req.Address
	}
	if req.Latitude != nil {
		if *req.Latitude < -90 || *req.Latitude > 90 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Широта должна быть от -90 до 90"})
			return
		}
		event.Latitude = req.Latitude
	}
	if req.Longitude != nil {
		if *req.Longitude < -180 || *req.Longitude > 180 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Долгота должна быть от -180 до 180"})
			return
		}
		event.Longitude = req.Longitude
	}
	if req.YandexMapLink != "" {
		if !utils.ValidateStringLength(req.YandexMapLink, 0, 1000) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Ссылка на карту должна быть до 1000 символов"})
			return
		}
		event.YandexMapLink = req.YandexMapLink
	}

	if req.Tags != nil {
		event.Tags = req.Tags
	}

	if err := database.DB.Save(&event).Error; err != nil {
		h.logger.Error("Ошибка при обновлении события в БД", zap.String("eventID", eventID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при обновлении события"})
		return
	}

	if req.CategoryIDs != nil {
		var categories []models.Category
		if err := database.DB.Where("id IN (?)", req.CategoryIDs).Find(&categories).Error; err == nil {
			if err := database.DB.Model(&event).Association("Categories").Replace(categories); err != nil {
				h.logger.Error("Ошибка обновления категорий события", zap.String("eventID", eventID), zap.Error(err))
			}
		} else {
			h.logger.Error("Ошибка получения категорий для обновления", zap.Error(err))
		}
	}

	var participants []models.EventParticipant
	database.DB.Where("event_id = ?", eventID).Find(&participants)
	for _, p := range participants {
		var user models.User
		if err := database.DB.Where("id = ?", p.UserID).First(&user).Error; err == nil {
			go h.emailService.SendEventNotification(user.Email, event.Title, "Данные события были изменены. Проверьте обновленную информацию о событии.")
		} else {
			h.logger.Error("Ошибка получения пользователя для уведомления об изменении события", zap.Any("userID", p.UserID), zap.Error(err))
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "Событие обновлено"})
}

func (h *EventHandler) DeleteEvent(c *gin.Context) {
	eventID := c.Param("id")

	if !utils.ValidateUUID(eventID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат ID события"})
		return
	}

	if err := database.DB.Where("id = ?", eventID).Delete(&models.Event{}).Error; err != nil {
		h.logger.Error("Ошибка при удалении события из БД", zap.String("eventID", eventID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при удалении события"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Событие удалено"})
}

func (h *EventHandler) JoinEvent(c *gin.Context) {
	eventID := c.Param("id")

	if !utils.ValidateUUID(eventID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат ID события"})
		return
	}

	userID, _ := c.Get("userID")

	var event models.Event
	if err := database.DB.Preload("Participants").Where("id = ?", eventID).First(&event).Error; err != nil {
		h.logger.Error("Событие не найдено для присоединения", zap.String("eventID", eventID), zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "Событие не найдено"})
		return
	}

	if event.Status != models.EventStatusActive {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Можно участвовать только в активных событиях"})
		return
	}

	if event.MaxParticipants != nil && event.GetParticipantsCount() >= *event.MaxParticipants {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Достигнут максимальный лимит участников",
			"message": "Достигнут максимальный лимит участников",
		})
		return
	}

	var existingParticipant models.EventParticipant
	if err := database.DB.Where("event_id = ? AND user_id = ?", eventID, userID).First(&existingParticipant).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Вы уже участвуете в этом событии"})
		return
	}

	participant := models.EventParticipant{
		EventID: event.ID,
		UserID:  userID.(uuid.UUID),
	}

	if err := database.DB.Create(&participant).Error; err != nil {
		h.logger.Error("Ошибка при подтверждении участия в БД", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при подтверждении участия"})
		return
	}

	var organizer models.User
	if err := database.DB.Where("id = ?", event.OrganizerID).First(&organizer).Error; err == nil {
		var user models.User
		if err := database.DB.Where("id = ?", userID).First(&user).Error; err == nil {
			go h.emailService.SendEventNotification(organizer.Email, event.Title, user.FullName+" подтвердил участие в событии")
		} else {
			h.logger.Error("Ошибка получения пользователя для уведомления организатора о присоединении", zap.Any("userID", userID), zap.Error(err))
		}
	} else {
		h.logger.Error("Ошибка получения организатора для уведомления о присоединении", zap.Any("organizerID", event.OrganizerID), zap.Error(err))
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Участие успешно подтверждено",
	})
}

func (h *EventHandler) LeaveEvent(c *gin.Context) {
	eventID := c.Param("id")

	if !utils.ValidateUUID(eventID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат ID события"})
		return
	}

	userID, _ := c.Get("userID")

	var participant models.EventParticipant
	if err := database.DB.Where("event_id = ? AND user_id = ?", eventID, userID).First(&participant).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Участие не найдено"})
		return
	}

	if err := database.DB.Delete(&participant).Error; err != nil {
		h.logger.Error("Ошибка при отмене участия в БД", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при отмене участия"})
		return
	}

	var event models.Event
	if err := database.DB.Where("id = ?", eventID).First(&event).Error; err == nil {
		var organizer models.User
		if err := database.DB.Where("id = ?", event.OrganizerID).First(&organizer).Error; err == nil {
			var user models.User
			if err := database.DB.Where("id = ?", userID).First(&user).Error; err == nil {
				go h.emailService.SendEventNotification(organizer.Email, event.Title, user.FullName+" отменил участие в событии")
			} else {
				h.logger.Error("Ошибка получения пользователя для уведомления организатора об отмене участия", zap.Any("userID", userID), zap.Error(err))
			}
		} else {
			h.logger.Error("Ошибка получения организатора для уведомления об отмене участия", zap.Any("organizerID", event.OrganizerID), zap.Error(err))
		}
	} else {
		h.logger.Error("Ошибка получения события для уведомления об отмене участия", zap.String("eventID", eventID), zap.Error(err))
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Участие успешно отменено",
	})
}

func (h *EventHandler) ExportParticipants(c *gin.Context) {
	eventID := c.Param("id")

	if !utils.ValidateUUID(eventID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат ID события"})
		return
	}

	format := c.Query("format")

	var participants []models.EventParticipant
	if err := database.DB.Preload("User").Preload("Event").Where("event_id = ?", eventID).Find(&participants).Error; err != nil {
		h.logger.Error("Ошибка при получении участников для экспорта", zap.String("eventID", eventID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении участников"})
		return
	}

	if format == "csv" {
		c.Header("Content-Type", "text/csv; charset=utf-8")
		c.Header("Content-Disposition", "attachment; filename=participants.csv")
		c.Writer.WriteString("\xEF\xBB\xBF")
		c.Writer.WriteString("ФИО,Email\n")
		for _, p := range participants {
			c.Writer.WriteString(fmt.Sprintf("%s,%s\n", p.User.FullName, p.User.Email))
		}
		return
	}

	f := excelize.NewFile()
	defer func() {
		if err := f.Close(); err != nil {
			h.logger.Error("Ошибка при закрытии файла Excel", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при создании файла"})
		}
	}()

	f.DeleteSheet("Sheet1")
	sheetName := "Участники"
	f.NewSheet(sheetName)
	f.SetCellValue(sheetName, "A1", "ФИО")
	f.SetCellValue(sheetName, "B1", "Email")

	for i, p := range participants {
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", i+2), p.User.FullName)
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", i+2), p.User.Email)
	}

	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", "attachment; filename=participants.xlsx")

	if err := f.Write(c.Writer); err != nil {
		h.logger.Error("Ошибка при записи файла Excel", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при экспорте"})
		return
	}
}
