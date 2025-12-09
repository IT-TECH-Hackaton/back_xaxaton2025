package handlers

import (
	"net/http"
	"strings"

	"bekend/database"
	"bekend/dto"
	"bekend/models"
	"bekend/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type CommunityHandler struct {
	logger *zap.Logger
}

func NewCommunityHandler() *CommunityHandler {
	return &CommunityHandler{
		logger: utils.GetLogger(),
	}
}

func (h *CommunityHandler) CreateCommunity(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Требуется авторизация"})
		return
	}

	var req dto.CreateCommunityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверные данные"})
		return
	}

	if !utils.ValidateStringLength(req.Name, 1, 100) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Название должно быть от 1 до 100 символов"})
		return
	}

	tagsStr := strings.Join(req.InterestTags, ",")
	if len(tagsStr) > 500 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Слишком много тегов"})
		return
	}

	community := models.MicroCommunity{
		Name:         strings.TrimSpace(req.Name),
		Description:  req.Description,
		InterestTags: tagsStr,
		AdminID:      userID.(uuid.UUID),
		AutoNotify:   req.AutoNotify,
		MembersCount: 1,
	}

	if err := database.DB.Create(&community).Error; err != nil {
		h.logger.Error("Ошибка создания сообщества", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при создании сообщества"})
		return
	}

	member := models.CommunityMember{
		UserID:      userID.(uuid.UUID),
		CommunityID: community.ID,
	}

	if err := database.DB.Create(&member).Error; err != nil {
		h.logger.Error("Ошибка добавления администратора в сообщество", zap.Error(err))
	}

	c.JSON(http.StatusCreated, h.communityToResponse(community))
}

func (h *CommunityHandler) GetCommunities(c *gin.Context) {
	search := c.Query("search")
	category := c.Query("category")

	var communities []models.MicroCommunity
	query := database.DB.Preload("Admin")

	if search != "" {
		query = query.Where("name ILIKE ? OR description ILIKE ? OR interest_tags ILIKE ?",
			"%"+search+"%", "%"+search+"%", "%"+search+"%")
	}

	if category != "" {
		query = query.Where("interest_tags ILIKE ?", "%"+category+"%")
	}

	if err := query.Order("members_count DESC, created_at DESC").Find(&communities).Error; err != nil {
		h.logger.Error("Ошибка получения сообществ", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении сообществ"})
		return
	}

	result := make([]dto.CommunityResponse, len(communities))
	for i, comm := range communities {
		result[i] = h.communityToResponse(comm)
	}

	c.JSON(http.StatusOK, result)
}

func (h *CommunityHandler) GetCommunity(c *gin.Context) {
	communityID := c.Param("id")
	if !utils.ValidateUUID(communityID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат ID"})
		return
	}

	var community models.MicroCommunity
	if err := database.DB.Preload("Admin").Where("id = ?", communityID).First(&community).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Сообщество не найдено"})
		return
	}

	c.JSON(http.StatusOK, h.communityToResponse(community))
}

func (h *CommunityHandler) JoinCommunity(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Требуется авторизация"})
		return
	}

	communityID := c.Param("id")
	if !utils.ValidateUUID(communityID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат ID"})
		return
	}

	var community models.MicroCommunity
	if err := database.DB.Where("id = ?", communityID).First(&community).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Сообщество не найдено"})
		return
	}

	var existing models.CommunityMember
	if err := database.DB.Where("user_id = ? AND community_id = ?", userID, communityID).First(&existing).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Вы уже состоите в этом сообществе"})
		return
	}

	member := models.CommunityMember{
		UserID:      userID.(uuid.UUID),
		CommunityID: uuid.MustParse(communityID),
	}

	if err := database.DB.Create(&member).Error; err != nil {
		h.logger.Error("Ошибка вступления в сообщество", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при вступлении в сообщество"})
		return
	}

	community.MembersCount++
	database.DB.Save(&community)

	c.JSON(http.StatusOK, gin.H{"message": "Вы присоединились к сообществу"})
}

func (h *CommunityHandler) LeaveCommunity(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Требуется авторизация"})
		return
	}

	communityID := c.Param("id")
	if !utils.ValidateUUID(communityID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат ID"})
		return
	}

	var community models.MicroCommunity
	if err := database.DB.Where("id = ?", communityID).First(&community).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Сообщество не найдено"})
		return
	}

	if community.AdminID == userID.(uuid.UUID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Администратор не может покинуть сообщество"})
		return
	}

	if err := database.DB.Where("user_id = ? AND community_id = ?", userID, communityID).Delete(&models.CommunityMember{}).Error; err != nil {
		h.logger.Error("Ошибка выхода из сообщества", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при выходе из сообщества"})
		return
	}

	community.MembersCount--
	if community.MembersCount < 0 {
		community.MembersCount = 0
	}
	database.DB.Save(&community)

	c.JSON(http.StatusOK, gin.H{"message": "Вы покинули сообщество"})
}

func (h *CommunityHandler) GetMyCommunities(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Требуется авторизация"})
		return
	}

	var memberships []models.CommunityMember
	if err := database.DB.Preload("Community").Preload("Community.Admin").
		Where("user_id = ?", userID).Find(&memberships).Error; err != nil {
		h.logger.Error("Ошибка получения сообществ пользователя", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении сообществ"})
		return
	}

	result := make([]dto.CommunityResponse, len(memberships))
	for i, mem := range memberships {
		result[i] = h.communityToResponse(mem.Community)
	}

	c.JSON(http.StatusOK, result)
}

func (h *CommunityHandler) GetCommunityMembers(c *gin.Context) {
	communityID := c.Param("id")
	if !utils.ValidateUUID(communityID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат ID"})
		return
	}

	var members []models.CommunityMember
	if err := database.DB.Preload("User").Where("community_id = ?", communityID).
		Order("joined_at ASC").Find(&members).Error; err != nil {
		h.logger.Error("Ошибка получения участников", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении участников"})
		return
	}

	result := make([]dto.CommunityMemberResponse, len(members))
	for i, mem := range members {
		result[i] = dto.CommunityMemberResponse{
			ID: mem.ID.String(),
			User: dto.UserInfo{
				ID:    mem.User.ID.String(),
				Email: mem.User.Email,
			},
			JoinedAt: mem.JoinedAt.Format("2006-01-02T15:04:05Z"),
		}
	}

	c.JSON(http.StatusOK, result)
}

func (h *CommunityHandler) communityToResponse(community models.MicroCommunity) dto.CommunityResponse {
	tags := []string{}
	if community.InterestTags != "" {
		tags = strings.Split(community.InterestTags, ",")
		for i, tag := range tags {
			tags[i] = strings.TrimSpace(tag)
		}
	}

	return dto.CommunityResponse{
		ID:           community.ID.String(),
		Name:         community.Name,
		Description:  community.Description,
		InterestTags: tags,
		Admin: dto.UserInfo{
			ID:    community.Admin.ID.String(),
			Email: community.Admin.Email,
		},
		AutoNotify:   community.AutoNotify,
		MembersCount: community.MembersCount,
		CreatedAt:    community.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

