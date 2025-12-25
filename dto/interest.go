package dto

import "github.com/google/uuid"

type InterestResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Category    string `json:"category"`
	Description string `json:"description"`
}

type UserInterestRequest struct {
	InterestID uuid.UUID `json:"interestID" binding:"required"`
	Weight     int       `json:"weight"`
}

type UserInterestResponse struct {
	ID        string           `json:"id"`
	Interest  InterestResponse `json:"interest"`
	Weight    int              `json:"weight"`
	CreatedAt string           `json:"createdAt"`
}

