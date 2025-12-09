package dto

import "github.com/google/uuid"

type CreateCommunityRequest struct {
	Name         string   `json:"name" binding:"required"`
	Description  string   `json:"description"`
	InterestTags []string  `json:"interestTags"`
	AutoNotify   bool     `json:"autoNotify"`
}

type UpdateCommunityRequest struct {
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	InterestTags []string `json:"interestTags"`
	AutoNotify   *bool    `json:"autoNotify"`
}

type CommunityResponse struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	InterestTags []string `json:"interestTags"`
	Admin        UserInfo `json:"admin"`
	AutoNotify   bool     `json:"autoNotify"`
	MembersCount int      `json:"membersCount"`
	CreatedAt    string   `json:"createdAt"`
}

type CommunityMemberResponse struct {
	ID        string   `json:"id"`
	User      UserInfo `json:"user"`
	JoinedAt  string   `json:"joinedAt"`
}

