package dto

// PromotionPackageResponse — пакет продвижения для API
type PromotionPackageResponse struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Price        float64 `json:"price"`
	DurationDays int     `json:"durationDays"`
	Description  string  `json:"description"`
}

// PurchasePromotionRequest — запрос на покупку продвижения
type PurchasePromotionRequest struct {
	EventID   string `json:"eventId" binding:"required"`
	PackageID string `json:"packageId" binding:"required"`
	ReturnURL string `json:"returnUrl"`
	CancelURL string `json:"cancelUrl"`
}
