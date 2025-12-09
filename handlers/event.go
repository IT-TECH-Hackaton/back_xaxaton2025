package handlers

import (
	"fmt"
	"net/http"
	"time"

	"bekend/database"
	"bekend/models"
	"bekend/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/xuri/excelize/v2"
)

type EventHandler struct {
	emailService *services.EmailService
}

func NewEventHandler() *EventHandler {
	return &EventHandler{
		emailService: services.NewEmailService(),
	}
}

type CreateEventRequest struct {
	Title           string    `json:"title" binding:"required"`
	ShortDescription string   `json:"shortDescription"`
	FullDescription string    `json:"fullDescription" binding:"required"`
	StartDate       time.Time `json:"startDate" binding:"required"`
	EndDate         time.Time `json:"endDate" binding:"required"`
	ImageURL        string    `json:"imageURL" binding:"required"`
	PaymentInfo     string    `json:"paymentInfo"`
	MaxParticipants *int      `json:"maxParticipants"`
	ParticipantIDs  []uuid.UUID `json:"participantIDs"`
}

type UpdateEventRequest struct {
	Title           string    `json:"title"`
	ShortDescription string   `json:"shortDescription"`
	FullDescription string    `json:"fullDescription"`
	StartDate       time.Time `json:"startDate"`
	EndDate         time.Time `json:"endDate"`
	ImageURL        string    `json:"imageURL"`
	PaymentInfo     string    `json:"paymentInfo"`
	MaxParticipants *int      `json:"maxParticipants"`
	Status          string    `json:"status"`
}

func (h *EventHandler) GetEvents(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		userID = nil
	}
	tab := c.Query("tab")

	var events []models.Event
	query := database.DB.Preload("Organizer").Preload("Participants")

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
		}
	}

	c.JSON(http.StatusOK, result)
}

func (h *EventHandler) GetEvent(c *gin.Context) {
	eventID := c.Param("id")
	userID, _ := c.Get("userID")

	var event models.Event
	if err := database.DB.Preload("Organizer").Preload("Participants.User").Where("id = ?", eventID).First(&event).Error; err != nil {
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

	c.JSON(http.StatusOK, gin.H{
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
		"isParticipant":    isParticipant,
		"averageRating":    avgRating,
		"totalReviews":    totalReviews,
		"organizer": gin.H{
			"id":   event.Organizer.ID,
			"name": event.Organizer.FullName,
		},
	})
}

func (h *EventHandler) CreateEvent(c *gin.Context) {
	var req CreateEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверные данные"})
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
	}

	if err := database.DB.Create(&event).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при создании события"})
		return
	}

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
				}
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"id":   event.ID,
		"message": "Событие создано",
	})
}

func (h *EventHandler) UpdateEvent(c *gin.Context) {
	eventID := c.Param("id")
	var req UpdateEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверные данные"})
		return
	}

	var event models.Event
	if err := database.DB.Where("id = ?", eventID).First(&event).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Событие не найдено"})
		return
	}

	if req.Title != "" {
		event.Title = req.Title
	}
	if req.ShortDescription != "" {
		event.ShortDescription = req.ShortDescription
	}
	if req.FullDescription != "" {
		event.FullDescription = req.FullDescription
	}
	if !req.StartDate.IsZero() {
		event.StartDate = req.StartDate
	}
	if !req.EndDate.IsZero() {
		event.EndDate = req.EndDate
	}
	if req.ImageURL != "" {
		event.ImageURL = req.ImageURL
	}
	if req.PaymentInfo != "" {
		event.PaymentInfo = req.PaymentInfo
	}
	if req.MaxParticipants != nil {
		event.MaxParticipants = req.MaxParticipants
	}
	if req.Status != "" {
		event.Status = models.EventStatus(req.Status)
	}

	if err := database.DB.Save(&event).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при обновлении события"})
		return
	}

	var participants []models.EventParticipant
	database.DB.Where("event_id = ?", eventID).Find(&participants)
	for _, p := range participants {
		var user models.User
		if err := database.DB.Where("id = ?", p.UserID).First(&user).Error; err == nil {
			go h.emailService.SendEventNotification(user.Email, event.Title, "Данные события были изменены. Проверьте обновленную информацию о событии.")
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "Событие обновлено"})
}

func (h *EventHandler) DeleteEvent(c *gin.Context) {
	eventID := c.Param("id")

	if err := database.DB.Where("id = ?", eventID).Delete(&models.Event{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при удалении события"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Событие удалено"})
}

func (h *EventHandler) JoinEvent(c *gin.Context) {
	eventID := c.Param("id")
	userID, _ := c.Get("userID")

	var event models.Event
	if err := database.DB.Preload("Participants").Where("id = ?", eventID).First(&event).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Событие не найдено"})
		return
	}

	if event.Status != models.EventStatusActive {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Можно участвовать только в активных событиях"})
		return
	}

	if event.MaxParticipants != nil && event.GetParticipantsCount() >= *event.MaxParticipants {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Достигнут максимальный лимит участников"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при подтверждении участия"})
		return
	}

	var organizer models.User
	if err := database.DB.Where("id = ?", event.OrganizerID).First(&organizer).Error; err == nil {
		var user models.User
		if err := database.DB.Where("id = ?", userID).First(&user).Error; err == nil {
			go h.emailService.SendEventNotification(organizer.Email, event.Title, user.FullName+" подтвердил участие в событии")
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "Участие подтверждено"})
}

func (h *EventHandler) LeaveEvent(c *gin.Context) {
	eventID := c.Param("id")
	userID, _ := c.Get("userID")

	var participant models.EventParticipant
	if err := database.DB.Where("event_id = ? AND user_id = ?", eventID, userID).First(&participant).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Участие не найдено"})
		return
	}

	if err := database.DB.Delete(&participant).Error; err != nil {
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
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "Участие отменено"})
}

func (h *EventHandler) ExportParticipants(c *gin.Context) {
	eventID := c.Param("id")
	format := c.Query("format")

	var participants []models.EventParticipant
	if err := database.DB.Preload("User").Preload("Event").Where("event_id = ?", eventID).Find(&participants).Error; err != nil {
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при экспорте"})
		return
	}
}

