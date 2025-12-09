package handlers

import (
	"net/http"
	"strconv"
	"time"

	"bekend/database"
	"bekend/dto"
	"bekend/models"
	"bekend/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type MatchingHandler struct {
	logger *zap.Logger
}

func NewMatchingHandler() *MatchingHandler {
	return &MatchingHandler{
		logger: utils.GetLogger(),
	}
}

func (h *MatchingHandler) CreateEventMatching(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Требуется авторизация"})
		return
	}

	eventID := c.Param("id")
	if !utils.ValidateUUID(eventID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат ID события"})
		return
	}

	var req dto.CreateEventMatchingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверные данные"})
		return
	}

	var event models.Event
	if err := database.DB.Where("id = ?", eventID).First(&event).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Событие не найдено"})
		return
	}

	status := models.MatchStatusLooking
	if req.Status != "" {
		status = models.MatchStatus(req.Status)
	}

	var existing models.EventMatching
	if err := database.DB.Where("user_id = ? AND event_id = ?", userID, eventID).First(&existing).Error; err == nil {
		existing.Status = status
		existing.Preferences = req.Preferences
		if err := database.DB.Save(&existing).Error; err != nil {
			h.logger.Error("Ошибка обновления матчинга", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при обновлении матчинга"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Матчинг обновлен"})
		return
	}

	matching := models.EventMatching{
		UserID:      userID.(uuid.UUID),
		EventID:     uuid.MustParse(eventID),
		Status:      status,
		Preferences: req.Preferences,
	}

	if err := database.DB.Create(&matching).Error; err != nil {
		h.logger.Error("Ошибка создания матчинга", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при создании матчинга"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Матчинг создан"})
}

func (h *MatchingHandler) GetMatches(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Требуется авторизация"})
		return
	}

	eventID := c.Param("id")
	if !utils.ValidateUUID(eventID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат ID события"})
		return
	}

	var userMatching models.EventMatching
	if err := database.DB.Where("user_id = ? AND event_id = ?", userID, eventID).First(&userMatching).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Вы не отметили, что ищете компанию для этого события"})
		return
	}

	if userMatching.Status != models.MatchStatusLooking {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Вы больше не ищете компанию для этого события"})
		return
	}

	matches := h.findMatches(userID.(uuid.UUID), uuid.MustParse(eventID))

	c.JSON(http.StatusOK, matches)
}

func (h *MatchingHandler) findMatches(userID, eventID uuid.UUID) []dto.MatchResponse {
	var userInterests []models.UserInterest
	database.DB.Preload("Interest").Where("user_id = ?", userID).Find(&userInterests)

	userInterestMap := make(map[uuid.UUID]int)
	for _, ui := range userInterests {
		userInterestMap[ui.InterestID] = ui.Weight
	}

	var otherMatchings []models.EventMatching
	database.DB.Preload("User").
		Where("event_id = ? AND user_id != ? AND status = ?", eventID, userID, models.MatchStatusLooking).
		Find(&otherMatchings)

	var matches []dto.MatchResponse
	for _, matching := range otherMatchings {
		var otherInterests []models.UserInterest
		database.DB.Preload("Interest").Where("user_id = ?", matching.UserID).Find(&otherInterests)

		commonInterests := []string{}
		totalScore := 0.0
		commonCount := 0

		for _, oi := range otherInterests {
			if weight, exists := userInterestMap[oi.InterestID]; exists {
				commonInterests = append(commonInterests, oi.Interest.Name)
				totalScore += float64(weight+oi.Weight) / 2.0
				commonCount++
			}
		}

		if commonCount > 0 {
			score := totalScore / float64(commonCount)
			matches = append(matches, dto.MatchResponse{
				User: dto.UserMatchInfo{
					ID:       matching.User.ID.String(),
					FullName: matching.User.FullName,
					Email:    matching.User.Email,
				},
				Score:            score,
				CommonInterests: commonInterests,
			})
		}
	}

	return matches
}

func (h *MatchingHandler) CreateMatchRequest(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Требуется авторизация"})
		return
	}

	eventID := c.Param("id")
	if !utils.ValidateUUID(eventID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат ID события"})
		return
	}

	var req dto.MatchRequestCreate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверные данные"})
		return
	}

	if userID.(uuid.UUID) == req.ToUserID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Нельзя отправить запрос самому себе"})
		return
	}

	var existingRequest models.MatchRequest
	if err := database.DB.Where("from_user_id = ? AND to_user_id = ? AND event_id = ?", userID, req.ToUserID, eventID).First(&existingRequest).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Запрос уже отправлен"})
		return
	}

	matchRequest := models.MatchRequest{
		FromUserID: userID.(uuid.UUID),
		ToUserID:   req.ToUserID,
		EventID:    uuid.MustParse(eventID),
		Status:     models.MatchRequestStatusPending,
		Message:    req.Message,
	}

	if err := database.DB.Create(&matchRequest).Error; err != nil {
		h.logger.Error("Ошибка создания запроса на матч", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при создании запроса"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Запрос отправлен"})
}

func (h *MatchingHandler) GetMyMatchRequests(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Требуется авторизация"})
		return
	}

	status := c.Query("status")

	var requests []models.MatchRequest
	query := database.DB.Preload("FromUser").Preload("ToUser").Preload("Event").
		Where("to_user_id = ? OR from_user_id = ?", userID, userID)

	if status != "" {
		query = query.Where("status = ?", status)
	}

	if err := query.Order("created_at DESC").Find(&requests).Error; err != nil {
		h.logger.Error("Ошибка получения запросов", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении запросов"})
		return
	}

	result := make([]dto.MatchRequestResponse, len(requests))
	for i, req := range requests {
		result[i] = dto.MatchRequestResponse{
			ID: req.ID.String(),
			FromUser: dto.UserInfo{
				ID:    req.FromUser.ID.String(),
				Email: req.FromUser.Email,
			},
			ToUser: dto.UserInfo{
				ID:    req.ToUser.ID.String(),
				Email: req.ToUser.Email,
			},
			Event: dto.EventInfo{
				ID:    req.Event.ID.String(),
				Title: req.Event.Title,
			},
			Status:    string(req.Status),
			Message:   req.Message,
			CreatedAt: req.CreatedAt.Format("2006-01-02T15:04:05Z"),
		}
	}

	c.JSON(http.StatusOK, result)
}

func (h *MatchingHandler) AcceptMatchRequest(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Требуется авторизация"})
		return
	}

	requestID := c.Param("id")
	if !utils.ValidateUUID(requestID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат ID"})
		return
	}

	var request models.MatchRequest
	if err := database.DB.Where("id = ? AND to_user_id = ?", requestID, userID).First(&request).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Запрос не найден"})
		return
	}

	if request.Status != models.MatchRequestStatusPending {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Запрос уже обработан"})
		return
	}

	request.Status = models.MatchRequestStatusAccepted
	if err := database.DB.Save(&request).Error; err != nil {
		h.logger.Error("Ошибка принятия запроса", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при принятии запроса"})
		return
	}

	var fromMatching models.EventMatching
	database.DB.Where("user_id = ? AND event_id = ?", request.FromUserID, request.EventID).First(&fromMatching)
	fromMatching.Status = models.MatchStatusFound
	database.DB.Save(&fromMatching)

	var toMatching models.EventMatching
	database.DB.Where("user_id = ? AND event_id = ?", request.ToUserID, request.EventID).First(&toMatching)
	toMatching.Status = models.MatchStatusFound
	database.DB.Save(&toMatching)

	c.JSON(http.StatusOK, gin.H{"message": "Запрос принят"})
}

func (h *MatchingHandler) RejectMatchRequest(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Требуется авторизация"})
		return
	}

	requestID := c.Param("id")
	if !utils.ValidateUUID(requestID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат ID"})
		return
	}

	var request models.MatchRequest
	if err := database.DB.Where("id = ? AND to_user_id = ?", requestID, userID).First(&request).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Запрос не найден"})
		return
	}

	if request.Status != models.MatchRequestStatusPending {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Запрос уже обработан"})
		return
	}

	request.Status = models.MatchRequestStatusRejected
	if err := database.DB.Save(&request).Error; err != nil {
		h.logger.Error("Ошибка отклонения запроса", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при отклонении запроса"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Запрос отклонен"})
}

func (h *MatchingHandler) RemoveEventMatching(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Требуется авторизация"})
		return
	}

	eventID := c.Param("id")
	if !utils.ValidateUUID(eventID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат ID события"})
		return
	}

	if err := database.DB.Where("user_id = ? AND event_id = ?", userID, eventID).Delete(&models.EventMatching{}).Error; err != nil {
		h.logger.Error("Ошибка удаления матчинга", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при удалении матчинга"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Матчинг удален"})
}

