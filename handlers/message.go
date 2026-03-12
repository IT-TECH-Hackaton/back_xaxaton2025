package handlers

import (
	"net/http"
	"time"

	"bekend/database"
	"bekend/dto"
	"bekend/models"
	"bekend/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type MessageHandler struct {
	logger *zap.Logger
}

func NewMessageHandler() *MessageHandler {
	return &MessageHandler{
		logger: utils.GetLogger(),
	}
}

// GetConversations godoc
// @Summary Список диалогов
// @Description Получение списка диалогов текущего пользователя
// @Tags Сообщения
// @Produce json
// @Security BearerAuth
// @Success 200 {array} dto.ConversationResponse
// @Router /messages [get]
func (h *MessageHandler) GetConversations(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}

	var subq []struct {
		PeerID uuid.UUID
		LastAt interface{}
	}

	// Найти всех собеседников и последнее сообщение
	rows, err := database.DB.Raw(`
		SELECT 
			CASE WHEN sender_id = ? THEN receiver_id ELSE sender_id END AS peer_id,
			MAX(created_at) AS last_at
		FROM messages
		WHERE sender_id = ? OR receiver_id = ?
		GROUP BY peer_id
		ORDER BY last_at DESC
	`, userID, userID, userID).Rows()
	if err != nil {
		h.logger.Error("Ошибка получения диалогов", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении диалогов"})
		return
	}
	defer rows.Close()

	for rows.Next() {
		var peerID uuid.UUID
		var lastAt interface{}
		if err := rows.Scan(&peerID, &lastAt); err != nil {
			continue
		}
		subq = append(subq, struct {
			PeerID uuid.UUID
			LastAt interface{}
		}{PeerID: peerID, LastAt: lastAt})
	}

	result := make([]dto.ConversationResponse, 0, len(subq))
	for _, s := range subq {
		var peer models.User
		if err := database.DB.Where("id = ?", s.PeerID).First(&peer).Error; err != nil {
			continue
		}
		if peer.Status == models.UserStatusDeleted {
			continue
		}

		var lastMsg models.Message
		err := database.DB.Where("(sender_id = ? AND receiver_id = ?) OR (sender_id = ? AND receiver_id = ?)",
			userID, s.PeerID, s.PeerID, userID).
			Order("created_at DESC").
			First(&lastMsg).Error
		if err != nil {
			lastMsg.Content = ""
		}

		var unread int64
		database.DB.Model(&models.Message{}).
			Where("sender_id = ? AND receiver_id = ? AND read_at IS NULL", s.PeerID, userID).
			Count(&unread)

		result = append(result, dto.ConversationResponse{
			UserID:      peer.ID.String(),
			FullName:    peer.FullName,
			AvatarURL:   peer.AvatarURL,
			LastMessage: truncate(lastMsg.Content, 80),
			LastAt:      lastMsg.CreatedAt,
			UnreadCount: unread,
		})
	}

	c.JSON(http.StatusOK, result)
}

// GetMessagesWithUser godoc
// @Summary Сообщения с пользователем
// @Description Получение переписки с указанным пользователем
// @Tags Сообщения
// @Param userId path string true "ID пользователя"
// @Produce json
// @Security BearerAuth
// @Success 200 {array} dto.MessageResponse
// @Router /messages/{userId} [get]
func (h *MessageHandler) GetMessagesWithUser(c *gin.Context) {
	myID, ok := h.getUserID(c)
	if !ok {
		return
	}

	peerIDStr := c.Param("userId")
	if !utils.ValidateUUID(peerIDStr) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат ID пользователя"})
		return
	}
	peerID, _ := uuid.Parse(peerIDStr)

	if peerID == myID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Нельзя писать самому себе"})
		return
	}

	var peer models.User
	if err := database.DB.Where("id = ?", peerID).First(&peer).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Пользователь не найден"})
		return
	}
	if peer.Status == models.UserStatusDeleted {
		c.JSON(http.StatusNotFound, gin.H{"error": "Пользователь не найден"})
		return
	}

	var messages []models.Message
	if err := database.DB.Where(
		"(sender_id = ? AND receiver_id = ?) OR (sender_id = ? AND receiver_id = ?)",
		myID, peerID, peerID, myID,
	).Order("created_at ASC").Find(&messages).Error; err != nil {
		h.logger.Error("Ошибка получения сообщений", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении сообщений"})
		return
	}

	// Пометить прочитанными входящие
	now := time.Now()
	database.DB.Model(&models.Message{}).
		Where("sender_id = ? AND receiver_id = ? AND read_at IS NULL", peerID, myID).
		Update("read_at", now)

	result := make([]dto.MessageResponse, len(messages))
	for i, m := range messages {
		result[i] = dto.MessageResponse{
			ID:         m.ID.String(),
			SenderID:   m.SenderID.String(),
			ReceiverID: m.ReceiverID.String(),
			Content:    m.Content,
			ReadAt:     m.ReadAt,
			CreatedAt:  m.CreatedAt,
			IsMine:     m.SenderID == myID,
		}
	}

	c.JSON(http.StatusOK, result)
}

// SendMessage godoc
// @Summary Отправить сообщение
// @Description Отправка сообщения пользователю
// @Tags Сообщения
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body dto.SendMessageRequest true "Сообщение"
// @Success 201 {object} dto.MessageResponse
// @Router /messages [post]
func (h *MessageHandler) SendMessage(c *gin.Context) {
	myID, ok := h.getUserID(c)
	if !ok {
		return
	}

	var req dto.SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат запроса"})
		return
	}

	if !utils.ValidateUUID(req.ReceiverID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат ID получателя"})
		return
	}
	receiverID, _ := uuid.Parse(req.ReceiverID)

	if receiverID == myID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Нельзя отправить сообщение самому себе"})
		return
	}

	var receiver models.User
	if err := database.DB.Where("id = ?", receiverID).First(&receiver).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Пользователь не найден"})
		return
	}
	if receiver.Status == models.UserStatusDeleted {
		c.JSON(http.StatusNotFound, gin.H{"error": "Пользователь не найден"})
		return
	}

	msg := models.Message{
		SenderID:   myID,
		ReceiverID: receiverID,
		Content:    req.Content,
	}
	if err := database.DB.Create(&msg).Error; err != nil {
		h.logger.Error("Ошибка сохранения сообщения", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при отправке сообщения"})
		return
	}

	c.JSON(http.StatusCreated, dto.MessageResponse{
		ID:         msg.ID.String(),
		SenderID:   msg.SenderID.String(),
		ReceiverID: msg.ReceiverID.String(),
		Content:    msg.Content,
		ReadAt:     msg.ReadAt,
		CreatedAt:  msg.CreatedAt,
		IsMine:     true,
	})
}

func (h *MessageHandler) getUserID(c *gin.Context) (uuid.UUID, bool) {
	v, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Требуется авторизация"})
		return uuid.Nil, false
	}
	uid, ok := v.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка авторизации"})
		return uuid.Nil, false
	}
	return uid, true
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
