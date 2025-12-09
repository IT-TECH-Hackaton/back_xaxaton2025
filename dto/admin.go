package dto

import "time"

type CreateUserRequest struct {
	FullName string `json:"fullName" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
	Role     string `json:"role"`
}

type UpdateUserRequest struct {
	FullName string `json:"fullName"`
	Role     string `json:"role"`
	Status   string `json:"status"`
}

type ResetUserPasswordRequest struct {
	Password string `json:"password" binding:"required"`
}

type UserFilterRequest struct {
	FullName string    `json:"fullName"`
	Role     string    `json:"role"`
	Status   string    `json:"status"`
	DateFrom time.Time `json:"dateFrom"`
	DateTo   time.Time `json:"dateTo"`
}

