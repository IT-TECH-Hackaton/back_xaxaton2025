package dto

type UpdateProfileRequest struct {
	FullName *string `json:"fullName"` // Указатель для возможности очистки поля
	Telegram *string `json:"telegram"` // Указатель для возможности очистки поля
	// Avatar загружается как файл через multipart/form-data с полем "avatar"
}

type UserResponse struct {
	ID        string `json:"id"`
	FullName  string `json:"fullName"`
	Email     string `json:"email"`
	Telegram  string `json:"telegram"`
	AvatarURL string `json:"avatarURL"`
	Role      string `json:"role"`
	Status    string `json:"status"`
	CreatedAt string `json:"createdAt"`
}

