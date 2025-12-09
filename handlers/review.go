package handlers

import (
	"net/http"

	"bekend/database"
	"bekend/models"

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

	var reviews []models.EventReview
	if err := database.DB.Preload("User").Where("event_id = ?", eventID).Order("created_at DESC").Find(&reviews).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении отзывов"})
		return
	}

	result := make([]gin.H, len(reviews))
	var totalRating int
	for i, review := range reviews {
		totalRating += review.Rating
		result[i] = gin.H{
			"id": review.ID,
			"rating": review.Rating,
			"comment": review.Comment,
			"user": gin.H{
				"id": review.User.ID,
				"fullName": review.User.FullName,
			},
			"createdAt": review.CreatedAt,
		}
	}

	var avgRating float64
	if len(reviews) > 0 {
		avgRating = float64(totalRating) / float64(len(reviews))
	}

	c.JSON(http.StatusOK, gin.H{
		"reviews": result,
		"averageRating": avgRating,
		"totalReviews": len(reviews),
	})
}

func (h *ReviewHandler) UpdateReview(c *gin.Context) {
	reviewID := c.Param("reviewId")
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

