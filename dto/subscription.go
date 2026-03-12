package dto

// UpgradeSubscriptionRequest — запрос на оформление подписки
type UpgradeSubscriptionRequest struct {
	Plan      string `json:"plan" binding:"required,oneof=Про Бизнес"`
	ReturnURL string `json:"returnUrl"`
	CancelURL string `json:"cancelUrl"`
}
