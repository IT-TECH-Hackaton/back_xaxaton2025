package dto

import "time"

type SendMessageRequest struct {
	ReceiverID string `json:"receiverId" binding:"required"`
	Content   string `json:"content" binding:"required,max=2000"`
}

type MessageResponse struct {
	ID         string     `json:"id"`
	SenderID   string     `json:"senderId"`
	ReceiverID string     `json:"receiverId"`
	Content    string     `json:"content"`
	ReadAt     *time.Time `json:"readAt,omitempty"`
	CreatedAt  time.Time  `json:"createdAt"`
	IsMine     bool       `json:"isMine"`
}

type ConversationResponse struct {
	UserID      string     `json:"userId"`
	FullName    string     `json:"fullName"`
	AvatarURL   string     `json:"avatarURL,omitempty"`
	LastMessage string     `json:"lastMessage,omitempty"`
	LastAt      time.Time  `json:"lastAt"`
	UnreadCount int64      `json:"unreadCount"`
}
