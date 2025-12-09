package handlers

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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

// GetEvents godoc
// @Summary Получить список событий
// @Description Получение списка событий с фильтрацией, поиском и пагинацией
// @Tags События
// @Accept json
// @Produce json
// @Param tab query string false "Тип фильтрации: active, my, past"
// @Param page query int false "Номер страницы (по умолчанию: 1)"
// @Param limit query int false "Количество элементов на странице (по умолчанию: 20, максимум: 100)"
// @Param search query string false "Поиск по названию и описанию (1-200 символов)"
// @Param status query string false "Фильтр по статусу: Активное, Прошедшее, Отклоненное (для обычных пользователей доступны только Активное и Прошедшее)"
// @Param categoryIDs query []string false "Фильтр по категориям (массив UUID)"
// @Param tags query []string false "Фильтр по тегам (массив строк)"
// @Param dateFrom query string false "Фильтр по дате начала (YYYY-MM-DD)"
// @Param dateTo query string false "Фильтр по дате окончания (YYYY-MM-DD)"
// @Param sortBy query string false "Сортировка: startDate, createdAt, participantsCount (по умолчанию: startDate)"
// @Param sortOrder query string false "Порядок сортировки: ASC, DESC (по умолчанию: ASC)"
// @Success 200 {object} dto.PaginationResponse{data=[]dto.EventResponse} "Список событий с пагинацией"
// @Failure 400 {object} map[string]string "Ошибка валидации параметров"
// @Failure 500 {object} map[string]string "Внутренняя ошибка сервера"
// @Router /events [get]
func (h *EventHandler) GetEvents(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		userID = nil
	}
	tab := c.Query("tab")
	statusFilter := c.Query("status")
	search := c.Query("search")
	categoryIDs := c.QueryArray("categoryIDs")
	tags := c.QueryArray("tags")
	dateFrom := c.Query("dateFrom")
	dateTo := c.Query("dateTo")
	sortBy := c.DefaultQuery("sortBy", "startDate")
	sortOrder := c.DefaultQuery("sortOrder", "ASC")

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

	query := database.DB.Model(&models.Event{}).Preload("Organizer").Preload("Participants").Preload("Categories")

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

	if statusFilter != "" {
		if models.IsValidEventStatus(models.EventStatus(statusFilter)) {
			query = query.Where("status = ?", models.EventStatus(statusFilter))
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный статус. Допустимые значения: Активное, Прошедшее, Отклоненное"})
			return
		}
	} else {
		query = query.Where("status != ?", models.EventStatusRejected)
	}

	if search != "" {
		if !utils.ValidateStringLength(search, 1, 200) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Поисковый запрос должен быть от 1 до 200 символов"})
			return
		}
		query = query.Where("title ILIKE ? OR short_description ILIKE ? OR full_description ILIKE ?",
			"%"+search+"%", "%"+search+"%", "%"+search+"%")
	}

	if len(categoryIDs) > 0 {
		var validCategoryIDs []uuid.UUID
		for _, catIDStr := range categoryIDs {
			if !utils.ValidateUUID(catIDStr) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат ID категории: " + catIDStr})
				return
			}
			validCategoryIDs = append(validCategoryIDs, uuid.MustParse(catIDStr))
		}
		if len(validCategoryIDs) > 0 {
			query = query.Where("id IN (SELECT event_id FROM event_categories WHERE category_id IN ?)", validCategoryIDs)
		}
	}

	if len(tags) > 0 {
		for _, tag := range tags {
			if tag != "" {
				query = query.Where("? = ANY(tags)", tag)
			}
		}
	}

	if dateFrom != "" {
		if dateFromTime, err := time.Parse("2006-01-02", dateFrom); err == nil {
			query = query.Where("start_date >= ?", dateFromTime)
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат dateFrom. Используйте YYYY-MM-DD"})
			return
		}
	}

	if dateTo != "" {
		if dateToTime, err := time.Parse("2006-01-02", dateTo); err == nil {
			dateToTime = dateToTime.Add(24 * time.Hour).Add(-1 * time.Second)
			query = query.Where("end_date <= ?", dateToTime)
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат dateTo. Используйте YYYY-MM-DD"})
			return
		}
	}

	var total int64
	query.Count(&total)

	orderBy := "start_date ASC"
	if sortBy == "createdAt" {
		if sortOrder == "DESC" {
			orderBy = "created_at DESC"
		} else {
			orderBy = "created_at ASC"
		}
	} else if sortBy == "participantsCount" {
		if sortOrder == "DESC" {
			orderBy = "(SELECT COUNT(*) FROM event_participants WHERE event_id = events.id) DESC"
		} else {
			orderBy = "(SELECT COUNT(*) FROM event_participants WHERE event_id = events.id) ASC"
		}
	} else {
		if sortOrder == "DESC" {
			orderBy = "start_date DESC"
		} else {
			orderBy = "start_date ASC"
		}
	}

	var events []models.Event
	if err := query.Offset(offset).Limit(limitInt).Order(orderBy).Find(&events).Error; err != nil {
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
			ID:                event.ID.String(),
			Title:             event.Title,
			ShortDescription:  event.ShortDescription,
			FullDescription:   event.FullDescription,
			StartDate:         event.StartDate,
			EndDate:           event.EndDate,
			ImageURL:          event.ImageURL,
			PaymentInfo:       event.PaymentInfo,
			MaxParticipants:   event.MaxParticipants,
			Status:            string(event.Status),
			ParticipantsCount: event.GetParticipantsCount(),
			Categories:        categories,
			Tags:              event.Tags,
			Address:           event.Address,
			Latitude:          event.Latitude,
			Longitude:         event.Longitude,
			YandexMapLink:     event.YandexMapLink,
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

// GetEvent godoc
// @Summary Получить детальную информацию о событии
// @Description Получение полной информации о событии, включая участников, отзывы и рейтинг
// @Tags События
// @Accept json
// @Produce json
// @Param id path string true "UUID события"
// @Success 200 {object} dto.EventDetailResponse "Детальная информация о событии"
// @Failure 400 {object} map[string]string "Неверный формат ID"
// @Failure 403 {object} map[string]string "Доступ запрещен (отклоненное событие)"
// @Failure 404 {object} map[string]string "Событие не найдено"
// @Router /events/{id} [get]
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
		ID:                event.ID.String(),
		Title:             event.Title,
		ShortDescription:  event.ShortDescription,
		FullDescription:   event.FullDescription,
		StartDate:         event.StartDate,
		EndDate:           event.EndDate,
		ImageURL:          event.ImageURL,
		PaymentInfo:       event.PaymentInfo,
		MaxParticipants:   event.MaxParticipants,
		Status:            string(event.Status),
		ParticipantsCount: event.GetParticipantsCount(),
		IsParticipant:     isParticipant,
		AverageRating:     avgRating,
		TotalReviews:      totalReviews,
		Categories:        categories,
		Tags:              event.Tags,
		Address:           event.Address,
		Latitude:          event.Latitude,
		Longitude:         event.Longitude,
		YandexMapLink:     event.YandexMapLink,
		Organizer: dto.UserInfo{
			ID:       event.Organizer.ID.String(),
			FullName: event.Organizer.FullName,
			Email:    event.Organizer.Email,
		},
	})
}

// CreateEvent godoc
// @Summary Создать новое событие
// @Description Создание нового события с возможностью указания категорий, тегов и участников. Можно загрузить изображение файлом или указать URL
// @Tags События
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param title formData string true "Название события"
// @Param shortDescription formData string false "Краткое описание"
// @Param fullDescription formData string true "Полное описание"
// @Param startDate formData string true "Дата начала (RFC3339)"
// @Param endDate formData string true "Дата окончания (RFC3339)"
// @Param image formData file false "Изображение события (jpeg, jpg, png, gif, webp, svg, до 10MB)"
// @Param imageURL formData string false "URL изображения (если не загружается файл)"
// @Param paymentInfo formData string false "Информация об оплате"
// @Param maxParticipants formData int false "Максимальное количество участников"
// @Param categoryIDs formData []string false "ID категорий (массив UUID)"
// @Param tags formData []string false "Теги (массив строк)"
// @Param address formData string false "Адрес"
// @Param latitude formData number false "Широта"
// @Param longitude formData number false "Долгота"
// @Param yandexMapLink formData string false "Ссылка на Яндекс.Карты"
// @Success 200 {object} map[string]interface{} "Событие создано"
// @Failure 400 {object} map[string]string "Ошибка валидации"
// @Failure 401 {object} map[string]string "Требуется авторизация"
// @Failure 500 {object} map[string]string "Внутренняя ошибка сервера"
// @Router /events [post]
func (h *EventHandler) CreateEvent(c *gin.Context) {
	title := c.PostForm("title")
	fullDescription := c.PostForm("fullDescription")
	shortDescription := c.PostForm("shortDescription")
	startDateStr := c.PostForm("startDate")
	endDateStr := c.PostForm("endDate")
	imageURL := c.PostForm("imageURL")
	paymentInfo := c.PostForm("paymentInfo")
	address := c.PostForm("address")
	yandexMapLink := c.PostForm("yandexMapLink")

	if title == "" || fullDescription == "" || startDateStr == "" || endDateStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Необходимо указать: title, fullDescription, startDate, endDate"})
		return
	}

	startDate, err := time.Parse(time.RFC3339, startDateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат startDate. Используйте RFC3339 (например: 2024-12-10T10:00:00Z)"})
		return
	}

	endDate, err := time.Parse(time.RFC3339, endDateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат endDate. Используйте RFC3339 (например: 2024-12-10T18:00:00Z)"})
		return
	}

	var maxParticipants *int
	if mpStr := c.PostForm("maxParticipants"); mpStr != "" {
		if mp, err := strconv.Atoi(mpStr); err == nil && mp > 0 {
			maxParticipants = &mp
		}
	}

	var latitude *float64
	if latStr := c.PostForm("latitude"); latStr != "" {
		if lat, err := strconv.ParseFloat(latStr, 64); err == nil {
			latitude = &lat
		}
	}

	var longitude *float64
	if lonStr := c.PostForm("longitude"); lonStr != "" {
		if lon, err := strconv.ParseFloat(lonStr, 64); err == nil {
			longitude = &lon
		}
	}

	var categoryIDs []uuid.UUID
	if catIDsStr := c.PostFormArray("categoryIDs"); len(catIDsStr) > 0 {
		for _, catIDStr := range catIDsStr {
			if catID, err := uuid.Parse(catIDStr); err == nil {
				categoryIDs = append(categoryIDs, catID)
			}
		}
	}

	var tags []string
	if tagsStr := c.PostFormArray("tags"); len(tagsStr) > 0 {
		tags = tagsStr
	}

	fileHeader, err := c.FormFile("image")
	if err == nil {
		if fileHeader.Size > 10*1024*1024 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Размер файла не должен превышать 10MB"})
			return
		}

		ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
		allowedExts := []string{".jpg", ".jpeg", ".png", ".gif", ".webp", ".svg"}
		allowed := false
		for _, e := range allowedExts {
			if ext == e {
				allowed = true
				break
			}
		}

		if !allowed {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Недопустимый формат файла. Разрешены: jpg, jpeg, png, gif, webp, svg"})
			return
		}

		src, err := fileHeader.Open()
		if err != nil {
			h.logger.Error("Ошибка открытия файла изображения", zap.Error(err))
			c.JSON(http.StatusBadRequest, gin.H{"error": "Ошибка при открытии файла"})
			return
		}
		defer src.Close()

		buffer := make([]byte, 512)
		if _, err := src.Read(buffer); err != nil && err != io.EOF {
			h.logger.Error("Ошибка чтения файла изображения", zap.Error(err))
			c.JSON(http.StatusBadRequest, gin.H{"error": "Ошибка при чтении файла"})
			return
		}

		mimeType := http.DetectContentType(buffer)
		if !h.isValidImageFile(buffer, mimeType, ext) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Недопустимый тип файла. Файл должен быть изображением (JPEG, PNG, GIF, WebP, SVG)"})
			return
		}

		if _, err := src.Seek(0, io.SeekStart); err != nil {
			h.logger.Error("Ошибка сброса указателя файла", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при чтении файла"})
			return
		}

		uploadDir := "uploads/events"
		if err := os.MkdirAll(uploadDir, 0755); err != nil {
			h.logger.Error("Ошибка создания директории для изображений событий", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при создании директории"})
			return
		}

		filename := fmt.Sprintf("%s_%d%s", uuid.New().String(), time.Now().Unix(), ext)
		filePath := filepath.Join(uploadDir, filename)

		dst, err := os.Create(filePath)
		if err != nil {
			h.logger.Error("Ошибка создания файла изображения", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при создании файла"})
			return
		}
		defer dst.Close()

		if _, err := io.Copy(dst, src); err != nil {
			h.logger.Error("Ошибка сохранения файла изображения", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при сохранении файла"})
			return
		}

		imageURL = fmt.Sprintf("/uploads/events/%s", filename)
	} else if imageURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Необходимо указать imageURL или загрузить файл image"})
		return
	}

	if !utils.ValidateStringLength(title, 1, 200) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Название события должно быть от 1 до 200 символов"})
		return
	}

	if !utils.ValidateStringLength(fullDescription, 1, 5000) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Полное описание должно быть от 1 до 5000 символов"})
		return
	}

	if shortDescription != "" && !utils.ValidateStringLength(shortDescription, 1, 500) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Краткое описание должно быть от 1 до 500 символов"})
		return
	}

	if startDate.Before(time.Now()) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Дата начала должна быть в будущем"})
		return
	}

	if endDate.Before(startDate) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Дата окончания должна быть позже даты начала"})
		return
	}

	if maxParticipants != nil && *maxParticipants < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Максимальное количество участников должно быть больше 0"})
		return
	}

	if latitude != nil {
		if *latitude < -90 || *latitude > 90 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Широта должна быть от -90 до 90"})
			return
		}
	}
	if longitude != nil {
		if *longitude < -180 || *longitude > 180 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Долгота должна быть от -180 до 180"})
			return
		}
	}
	if address != "" && !utils.ValidateStringLength(address, 0, 500) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Адрес должен быть до 500 символов"})
		return
	}
	if yandexMapLink != "" && !utils.ValidateStringLength(yandexMapLink, 0, 1000) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ссылка на карту должна быть до 1000 символов"})
		return
	}

	if len(tags) > 0 {
		if valid, errMsg := utils.ValidateTags(tags); !valid {
			c.JSON(http.StatusBadRequest, gin.H{"error": errMsg})
			return
		}
	}

	userID, _ := c.Get("userID")
	organizerID := userID.(uuid.UUID)

	event := models.Event{
		Title:            title,
		ShortDescription: shortDescription,
		FullDescription:  fullDescription,
		StartDate:        startDate,
		EndDate:          endDate,
		ImageURL:         imageURL,
		PaymentInfo:      paymentInfo,
		MaxParticipants:  maxParticipants,
		Status:           models.EventStatusActive,
		OrganizerID:      organizerID,
		Tags:             tags,
		Address:          address,
		Latitude:         latitude,
		Longitude:        longitude,
		YandexMapLink:    yandexMapLink,
	}

	if err := database.DB.Create(&event).Error; err != nil {
		h.logger.Error("Ошибка при создании события в БД", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при создании события"})
		return
	}

	if len(categoryIDs) > 0 {
		if len(categoryIDs) > 10 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Максимальное количество категорий на событие - 10"})
			return
		}
		var categories []models.Category
		if err := database.DB.Where("id IN (?)", categoryIDs).Find(&categories).Error; err == nil {
			if len(categories) != len(categoryIDs) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Некоторые категории не найдены"})
				return
			}
			if err := database.DB.Model(&event).Association("Categories").Append(categories); err != nil {
				h.logger.Error("Ошибка добавления категорий к событию", zap.String("eventID", event.ID.String()), zap.Error(err))
			}
		} else {
			h.logger.Error("Ошибка получения категорий для события", zap.Error(err))
		}
	}

	communityService := services.NewCommunityService()
	go communityService.NotifyCommunitiesAboutEvent(&event)

	participantIDsStr := c.PostFormArray("participantIDs")
	if len(participantIDsStr) > 0 {
		var participantIDs []uuid.UUID
		for _, pidStr := range participantIDsStr {
			if pid, err := uuid.Parse(pidStr); err == nil {
				participantIDs = append(participantIDs, pid)
			}
		}

		if len(participantIDs) > 0 {
			var users []models.User
			if err := database.DB.Where("id IN ? AND status = ?", participantIDs, models.UserStatusActive).Find(&users).Error; err == nil {
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
	}

	c.JSON(http.StatusOK, gin.H{
		"id":      event.ID,
		"message": "Событие создано",
	})
}

// UpdateEvent godoc
// @Summary Обновить событие
// @Description Обновление данных события (все поля необязательны)
// @Tags События
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "UUID события"
// @Param request body dto.UpdateEventRequest true "Данные для обновления"
// @Success 200 {object} map[string]string "Событие обновлено"
// @Failure 400 {object} map[string]string "Ошибка валидации"
// @Failure 401 {object} map[string]string "Требуется авторизация"
// @Failure 404 {object} map[string]string "Событие не найдено"
// @Failure 500 {object} map[string]string "Внутренняя ошибка сервера"
// @Router /events/{id} [put]
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
		if valid, errMsg := utils.ValidateTags(req.Tags); !valid {
			c.JSON(http.StatusBadRequest, gin.H{"error": errMsg})
			return
		}
		event.Tags = req.Tags
	}

	if err := database.DB.Save(&event).Error; err != nil {
		h.logger.Error("Ошибка при обновлении события в БД", zap.String("eventID", eventID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при обновлении события"})
		return
	}

	if req.CategoryIDs != nil {
		if len(req.CategoryIDs) > 10 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Максимальное количество категорий на событие - 10"})
			return
		}
		var categories []models.Category
		if err := database.DB.Where("id IN (?)", req.CategoryIDs).Find(&categories).Error; err == nil {
			if len(categories) != len(req.CategoryIDs) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Некоторые категории не найдены"})
				return
			}
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

// DeleteEvent godoc
// @Summary Удалить событие
// @Description Мягкое удаление события (помечается как удаленное)
// @Tags События
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "UUID события"
// @Success 200 {object} map[string]string "Событие удалено"
// @Failure 400 {object} map[string]string "Неверный формат ID"
// @Failure 401 {object} map[string]string "Требуется авторизация"
// @Failure 500 {object} map[string]string "Внутренняя ошибка сервера"
// @Router /events/{id} [delete]
func (h *EventHandler) DeleteEvent(c *gin.Context) {
	eventID := c.Param("id")

	if !utils.ValidateUUID(eventID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат ID события"})
		return
	}

	var event models.Event
	if err := database.DB.Where("id = ?", eventID).First(&event).Error; err != nil {
		h.logger.Error("Событие не найдено для удаления", zap.String("eventID", eventID), zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "Событие не найдено"})
		return
	}

	if err := database.DB.Delete(&event).Error; err != nil {
		h.logger.Error("Ошибка при удалении события из БД", zap.String("eventID", eventID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при удалении события"})
		return
	}

	var participants []models.EventParticipant
	database.DB.Where("event_id = ?", eventID).Find(&participants)
	for _, p := range participants {
		var user models.User
		if err := database.DB.Where("id = ?", p.UserID).First(&user).Error; err == nil {
			go h.emailService.SendEventNotification(user.Email, event.Title, "Событие было отменено организатором.")
		} else {
			h.logger.Error("Ошибка получения пользователя для уведомления об отмене события", zap.Any("userID", p.UserID), zap.Error(err))
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "Событие удалено"})
}

// JoinEvent godoc
// @Summary Присоединиться к событию
// @Description Подтверждение участия в событии
// @Tags События
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "UUID события"
// @Success 200 {object} map[string]string "Участие подтверждено"
// @Failure 400 {object} map[string]interface{} "Событие не активное, лимит участников, уже участвуете"
// @Failure 401 {object} map[string]string "Требуется авторизация"
// @Failure 404 {object} map[string]string "Событие не найдено"
// @Failure 500 {object} map[string]string "Внутренняя ошибка сервера"
// @Router /events/{id}/join [post]
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

// LeaveEvent godoc
// @Summary Покинуть событие
// @Description Отмена участия в событии
// @Tags События
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "UUID события"
// @Success 200 {object} map[string]string "Участие отменено"
// @Failure 400 {object} map[string]string "Неверный формат ID"
// @Failure 401 {object} map[string]string "Требуется авторизация"
// @Failure 404 {object} map[string]string "Участие не найдено"
// @Failure 500 {object} map[string]string "Внутренняя ошибка сервера"
// @Router /events/{id}/leave [delete]
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

// ExportParticipants godoc
// @Summary Экспорт участников события
// @Description Экспорт списка участников события в CSV или XLSX формате
// @Tags События
// @Accept json
// @Produce application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Produce text/csv
// @Security BearerAuth
// @Param id path string true "UUID события"
// @Param format query string false "Формат экспорта: csv или xlsx (по умолчанию: xlsx)"
// @Success 200 {file} file "Файл с участниками"
// @Failure 400 {object} map[string]string "Неверный формат ID"
// @Failure 401 {object} map[string]string "Требуется авторизация"
// @Failure 404 {object} map[string]string "Событие не найдено"
// @Failure 500 {object} map[string]string "Внутренняя ошибка сервера"
// @Router /events/{id}/export [get]
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

func (h *EventHandler) isValidImageFile(buffer []byte, mimeType string, ext string) bool {
	extLower := strings.ToLower(ext)

	if extLower == ".svg" {
		svgContent := strings.ToLower(string(buffer[:min(len(buffer), 100)]))
		return strings.Contains(svgContent, "<svg") || strings.Contains(svgContent, "<?xml")
	}

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
