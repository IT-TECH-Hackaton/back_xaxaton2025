package dto

import (
	"time"

	"github.com/google/uuid"
)

type CreateEventRequest struct {
	Title           string     `json:"title" binding:"required"`
	ShortDescription string    `json:"shortDescription"`
	FullDescription string     `json:"fullDescription" binding:"required"`
	StartDate       time.Time  `json:"startDate" binding:"required"`
	EndDate         time.Time  `json:"endDate" binding:"required"`
	ImageURL        string     `json:"imageURL" binding:"required"`
	PaymentInfo     string     `json:"paymentInfo"`
	MaxParticipants *int       `json:"maxParticipants"`
	ParticipantIDs  []uuid.UUID `json:"participantIDs"`
	Address         string     `json:"address"`
	Latitude        *float64   `json:"latitude"`
	Longitude       *float64   `json:"longitude"`
	YandexMapLink   string     `json:"yandexMapLink"`
}

type UpdateEventRequest struct {
	Title           string    `json:"title"`
	ShortDescription string   `json:"shortDescription"`
	FullDescription string    `json:"fullDescription"`
	StartDate       time.Time `json:"startDate"`
	EndDate         time.Time `json:"endDate"`
	ImageURL        string    `json:"imageURL"`
	PaymentInfo     string    `json:"paymentInfo"`
	MaxParticipants *int      `json:"maxParticipants"`
	Status          string    `json:"status"`
	Address         string    `json:"address"`
	Latitude        *float64  `json:"latitude"`
	Longitude       *float64  `json:"longitude"`
	YandexMapLink   string    `json:"yandexMapLink"`
}

type PaginationResponse struct {
	Data       interface{} `json:"data"`
	Pagination Pagination  `json:"pagination"`
}

type Pagination struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"totalPages"`
}

