package dto

type GeocodeRequest struct {
	Address string `json:"address" binding:"required"`
}

type ReverseGeocodeRequest struct {
	Latitude  float64 `json:"latitude" binding:"required"`
	Longitude float64 `json:"longitude" binding:"required"`
}

type GeocodeResponse struct {
	Latitude      float64 `json:"latitude"`
	Longitude     float64 `json:"longitude"`
	Address       string  `json:"address"`
	YandexMapLink string  `json:"yandexMapLink"`
}

type MapLinkRequest struct {
	Address   string   `json:"address"`
	Latitude  *float64 `json:"latitude"`
	Longitude *float64 `json:"longitude"`
}

type MapLinkResponse struct {
	YandexMapLink string `json:"yandexMapLink"`
}

