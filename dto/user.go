package dto

type UpdateProfileRequest struct {
	FullName string `json:"fullName"`
}

type UserResponse struct {
	ID        string `json:"id"`
	FullName  string `json:"fullName"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	Status    string `json:"status"`
	CreatedAt string `json:"createdAt"`
}

