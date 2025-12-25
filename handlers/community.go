package handlers

import (
	"net/http"
	"strconv"
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

	var interests []models.Interest
	if len(req.InterestIDs) > 0 {
		var interestUUIDs []uuid.UUID
		for _, idStr := range req.InterestIDs {
			if !utils.ValidateUUID(idStr) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат ID интереса: " + idStr})
				return
			}
			interestUUIDs = append(interestUUIDs, uuid.MustParse(idStr))
		}

		if err := database.DB.Where("id IN ?", interestUUIDs).Find(&interests).Error; err != nil {
			h.logger.Error("Ошибка получения интересов", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении интересов"})
			return
		}

		if len(interests) != len(req.InterestIDs) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Некоторые интересы не найдены"})
			return
		}
	}

	community := models.MicroCommunity{
		Name:         strings.TrimSpace(req.Name),
		Description:  req.Description,
		AdminID:      userID.(uuid.UUID),
		AutoNotify:   req.AutoNotify,
		MembersCount: 1,
		Interests:    interests,
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
	interestID := c.Query("interestID")

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

	query := database.DB.Model(&models.MicroCommunity{}).Preload("Admin").Preload("Interests")

	if search != "" {
		if !utils.ValidateStringLength(search, 1, 100) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Поисковый запрос должен быть от 1 до 100 символов"})
			return
		}
		query = query.Where("name ILIKE ? OR description ILIKE ?",
			"%"+search+"%", "%"+search+"%")
	}

	if category != "" {
		query = query.Joins("JOIN community_interests ON micro_communities.id = community_interests.community_id").
			Joins("JOIN interests ON community_interests.interest_id = interests.id").
			Where("interests.category = ?", category)
	}

	if interestID != "" {
		if !utils.ValidateUUID(interestID) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат ID интереса"})
			return
		}
		query = query.Joins("JOIN community_interests ON micro_communities.id = community_interests.community_id").
			Where("community_interests.interest_id = ?", interestID)
	}

	var total int64
	query.Count(&total)

	var communities []models.MicroCommunity
	if err := query.Offset(offset).Limit(limitInt).Order("members_count DESC, created_at DESC").Find(&communities).Error; err != nil {
		h.logger.Error("Ошибка получения сообществ", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении сообществ"})
		return
	}

	result := make([]dto.CommunityResponse, len(communities))
	for i, comm := range communities {
		result[i] = h.communityToResponse(comm)
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

func (h *CommunityHandler) GetCommunity(c *gin.Context) {
	communityID := c.Param("id")
	if !utils.ValidateUUID(communityID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат ID"})
		return
	}

	var community models.MicroCommunity
	if err := database.DB.Preload("Admin").Preload("Interests").
		Where("id = ?", communityID).First(&community).Error; err != nil {
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
	if err := database.DB.Preload("Community").Preload("Community.Admin").Preload("Community.Interests").
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
	interests := make([]dto.InterestInfo, len(community.Interests))
	for i, interest := range community.Interests {
		interests[i] = dto.InterestInfo{
			ID:          interest.ID.String(),
			Name:        interest.Name,
			Category:    interest.Category,
			Description: interest.Description,
		}
	}

	return dto.CommunityResponse{
		ID:           community.ID.String(),
		Name:         community.Name,
		Description:  community.Description,
		Interests:    interests,
		Admin: dto.UserInfo{
			ID:    community.Admin.ID.String(),
			Email: community.Admin.Email,
		},
		AutoNotify:   community.AutoNotify,
		MembersCount: community.MembersCount,
		CreatedAt:    community.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
}
