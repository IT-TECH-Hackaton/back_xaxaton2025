package dto

type FakeYandexAuthRequest struct {
	YandexID  string `json:"yandexId" binding:"required"`
	Email     string `json:"email" binding:"required,email"`
	FullName  string `json:"fullName" binding:"required"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
}

