package handlers

import (
	"net/http"
	"strconv"

	"bekend/database"
	"bekend/dto"
	"bekend/models"
	"bekend/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ReviewHandler struct{}

func NewReviewHandler() *ReviewHandler {
	return &ReviewHandler{}
}

type CreateReviewRequest struct {
	Rating  int    `json:"rating" binding:"required,min=1,max=5"`
	Comment string `json:"comment"`
}

func (h *ReviewHandler) CreateReview(c *gin.Context) {
	eventID := c.Param("id")

	if !utils.ValidateUUID(eventID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат ID события"})
		return
	}

	userID, _ := c.Get("userID")

	var event models.Event
	if err := database.DB.Where("id = ?", eventID).First(&event).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Событие не найдено"})
		return
	}

	if event.Status != models.EventStatusPast {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Отзыв можно оставить только для прошедших событий"})
		return
	}

	var participant models.EventParticipant
	if err := database.DB.Where("event_id = ? AND user_id = ?", eventID, userID).First(&participant).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Вы не участвовали в этом событии"})
		return
	}

	var existingReview models.EventReview
	if err := database.DB.Where("event_id = ? AND user_id = ?", eventID, userID).First(&existingReview).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Вы уже оставили отзыв на это событие"})
		return
	}

	var req CreateReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверные данные"})
		return
	}

	if req.Rating < 1 || req.Rating > 5 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Рейтинг должен быть от 1 до 5"})
		return
	}

	if req.Comment != "" && !utils.ValidateStringLength(req.Comment, 0, 2000) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Комментарий должен быть до 2000 символов"})
		return
	}

	review := models.EventReview{
		EventID: uuid.MustParse(eventID),
		UserID:  userID.(uuid.UUID),
		Rating:  req.Rating,
		Comment: req.Comment,
	}

	if err := database.DB.Create(&review).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при создании отзыва"})
		return
	}

	database.DB.Preload("User").First(&review, review.ID)

	c.JSON(http.StatusOK, gin.H{
		"id": review.ID,
		"rating": review.Rating,
		"comment": review.Comment,
		"user": gin.H{
			"id": review.User.ID,
			"fullName": review.User.FullName,
		},
		"createdAt": review.CreatedAt,
	})
}

func (h *ReviewHandler) GetEventReviews(c *gin.Context) {
	eventID := c.Param("id")

	if !utils.ValidateUUID(eventID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат ID события"})
		return
	}

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

	var reviews []models.EventReview
	query := database.DB.Preload("User").Where("event_id = ?", eventID)

	var total int64
	query.Model(&models.EventReview{}).Count(&total)

	if err := query.Offset(offset).Limit(limitInt).Order("created_at DESC").Find(&reviews).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении отзывов"})
		return
	}

	result := make([]dto.ReviewResponse, len(reviews))
	var totalRating int
	for i, review := range reviews {
		totalRating += review.Rating
		result[i] = dto.ReviewResponse{
			ID:        review.ID.String(),
			EventID:   review.EventID.String(),
			UserID:    review.UserID.String(),
			Rating:    review.Rating,
			Comment:   review.Comment,
			User: dto.UserInfo{
				ID:       review.User.ID.String(),
				FullName: review.User.FullName,
				Email:    review.User.Email,
			},
			CreatedAt: review.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt: review.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	var avgRating float64
	if total > 0 {
		var allReviews []models.EventReview
		database.DB.Where("event_id = ?", eventID).Find(&allReviews)
		var allRating int
		for _, r := range allReviews {
			allRating += r.Rating
		}
		avgRating = float64(allRating) / float64(total)
	}

	totalPages := int((total + int64(limitInt) - 1) / int64(limitInt))
	c.JSON(http.StatusOK, dto.ReviewsResponse{
		Data:          result,
		AverageRating: avgRating,
		TotalReviews:  total,
		Pagination: dto.Pagination{
			Page:       pageInt,
			Limit:      limitInt,
			Total:      total,
			TotalPages: totalPages,
		},
	})
}

func (h *ReviewHandler) UpdateReview(c *gin.Context) {
	reviewID := c.Param("reviewId")

	if !utils.ValidateUUID(reviewID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат ID отзыва"})
		return
	}

	userID, _ := c.Get("userID")

	var review models.EventReview
	if err := database.DB.Where("id = ? AND user_id = ?", reviewID, userID).First(&review).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Отзыв не найден"})
		return
	}

	var req CreateReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверные данные"})
		return
	}

	if req.Rating < 1 || req.Rating > 5 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Рейтинг должен быть от 1 до 5"})
		return
	}

	if req.Comment != "" && !utils.ValidateStringLength(req.Comment, 0, 2000) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Комментарий должен быть до 2000 символов"})
		return
	}

	review.Rating = req.Rating
	review.Comment = req.Comment

	if err := database.DB.Save(&review).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при обновлении отзыва"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Отзыв обновлен"})
}

func (h *ReviewHandler) DeleteReview(c *gin.Context) {
	reviewID := c.Param("reviewId")

	if !utils.ValidateUUID(reviewID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат ID отзыва"})
		return
	}

	userID, _ := c.Get("userID")

	var review models.EventReview
	if err := database.DB.Where("id = ? AND user_id = ?", reviewID, userID).First(&review).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Отзыв не найден"})
		return
	}

	if err := database.DB.Delete(&review).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при удалении отзыва"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Отзыв удален"})
}

