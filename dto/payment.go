package dto

// CreatePaymentRequest — запрос на создание платежа
type CreatePaymentRequest struct {
	Amount      float64 `json:"amount" binding:"required,gt=0"`
	PaymentType string  `json:"paymentType" binding:"required,oneof=subscription ticket promotion"`
	Description string  `json:"description"`
	RelatedID   string  `json:"relatedID"`   // UUID связанной сущности
	ReturnURL   string  `json:"returnUrl"`
	CancelURL   string  `json:"cancelUrl"`
}

// CreatePaymentResponse — ответ с ссылкой на оплату
type CreatePaymentResponse struct {
	PaymentID       string `json:"paymentId"`
	ConfirmationURL string `json:"confirmationUrl"`
	Status          string `json:"status"`
}
