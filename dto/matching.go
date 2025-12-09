package dto

import "github.com/google/uuid"

type CreateEventMatchingRequest struct {
	Status      string `json:"status"`
	Preferences string `json:"preferences"`
}

type MatchRequestCreate struct {
	ToUserID uuid.UUID `json:"toUserID" binding:"required"`
	Message  string    `json:"message"`
}

type MatchRequestResponse struct {
	ID        string `json:"id"`
	FromUser  UserInfo `json:"fromUser"`
	ToUser    UserInfo `json:"toUser"`
	Event     EventInfo `json:"event"`
	Status    string `json:"status"`
	Message   string `json:"message"`
	CreatedAt string `json:"createdAt"`
}

type MatchResponse struct {
	User      UserMatchInfo `json:"user"`
	Score     float64      `json:"score"`
	CommonInterests []string `json:"commonInterests"`
}

type UserMatchInfo struct {
	ID       string `json:"id"`
	FullName string `json:"fullName"`
	Email    string `json:"email"`
}

type EventInfo struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

