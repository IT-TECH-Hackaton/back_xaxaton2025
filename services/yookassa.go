package services

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"bekend/config"
	"bekend/database"
	"bekend/models"
	"bekend/utils"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

const yooKassaAPIURL = "https://api.yookassa.ru/v3/payments"

// YooKassaCreateRequest — запрос на создание платежа
type YooKassaCreateRequest struct {
	Amount struct {
		Value    string `json:"value"`
		Currency string `json:"currency"`
	} `json:"amount"`
	Confirmation struct {
		Type      string `json:"type"`
		ReturnURL string `json:"return_url"`
		CancelURL string `json:"cancel_url"`
	} `json:"confirmation"`
	Description string            `json:"description"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	Capture     bool              `json:"capture"` // true = одностадийная оплата
}

// YooKassaPaymentResponse — ответ ЮKassa
type YooKassaPaymentResponse struct {
	ID           string `json:"id"`
	Status       string `json:"status"`
	Amount       struct { Value string `json:"value"` } `json:"amount"`
	Confirmation struct {
		Type            string `json:"type"`
		ConfirmationURL string `json:"confirmation_url"`
	} `json:"confirmation"`
}

// YooKassaWebhookPayload — тело webhook от ЮKassa
type YooKassaWebhookPayload struct {
	Type  string `json:"type"`  // payment.succeeded, payment.canceled, payment.waiting_for_capture
	Event struct {
		Type   string `json:"type"`
		Object struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"object"`
	} `json:"event"`
}

// YooKassaService — сервис работы с ЮKassa
type YooKassaService struct {
	client *http.Client
}

func NewYooKassaService() *YooKassaService {
	return &YooKassaService{
		client: &http.Client{},
	}
}

// IsConfigured — проверка наличия настроек
func (s *YooKassaService) IsConfigured() bool {
	return config.AppConfig.YooKassaShopID != "" && config.AppConfig.YooKassaSecretKey != ""
}

// CreatePayment создаёт платёж в ЮKassa и сохраняет запись в БД
// metadata — доп. данные для webhook (например package_id для продвижения)
func (s *YooKassaService) CreatePayment(userID uuid.UUID, amount float64, paymentType models.PaymentType, description string, relatedID *uuid.UUID, returnURL, cancelURL string, metadata map[string]string) (*models.Payment, string, error) {
	if !s.IsConfigured() {
		return nil, "", fmt.Errorf("ЮKassa не настроена: задайте YOOKASSA_SHOP_ID и YOOKASSA_SECRET_KEY")
	}

	// Создаём запись в БД
	payment := &models.Payment{
		UserID:      userID,
		Amount:      amount,
		Currency:    "RUB",
		Status:      models.PaymentStatusPending,
		Provider:    models.PaymentProviderYooKassa,
		PaymentType: paymentType,
		RelatedID:   relatedID,
		Description: description,
	}
	if metadata != nil {
		payment.Metadata = make(models.JSONMap)
		for k, v := range metadata {
			payment.Metadata[k] = v
		}
	}
	if err := database.DB.Create(payment).Error; err != nil {
		return nil, "", err
	}

	// Формируем запрос к ЮKassa
	amountStr := fmt.Sprintf("%.2f", amount)
	if returnURL == "" {
		returnURL = config.AppConfig.FrontendURL + "/payment/success"
	}
	if cancelURL == "" {
		cancelURL = config.AppConfig.FrontendURL + "/payment/cancel"
	}

	reqBody := YooKassaCreateRequest{
		Amount: struct {
			Value    string `json:"value"`
			Currency string `json:"currency"`
		}{Value: amountStr, Currency: "RUB"},
		Confirmation: struct {
			Type      string `json:"type"`
			ReturnURL string `json:"return_url"`
			CancelURL string `json:"cancel_url"`
		}{Type: "redirect", ReturnURL: returnURL, CancelURL: cancelURL},
		Description: description,
		Metadata: map[string]string{
			"payment_id":   payment.ID.String(),
			"payment_type": string(paymentType),
			"user_id":      userID.String(),
		},
		Capture: true,
	}
	if relatedID != nil {
		reqBody.Metadata["related_id"] = relatedID.String()
	}
	if metadata != nil {
		for k, v := range metadata {
			reqBody.Metadata[k] = v
		}
	}

	bodyBytes, _ := json.Marshal(reqBody)
	req, err := http.NewRequest("POST", yooKassaAPIURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return payment, "", err
	}

	auth := base64.StdEncoding.EncodeToString([]byte(config.AppConfig.YooKassaShopID + ":" + config.AppConfig.YooKassaSecretKey))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotence-Key", uuid.New().String())

	resp, err := s.client.Do(req)
	if err != nil {
		return payment, "", err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		utils.GetLogger().Error("Ошибка ЮKassa", zap.Int("status", resp.StatusCode), zap.String("body", string(respBody)))
		return payment, "", fmt.Errorf("ЮKassa вернула статус %d: %s", resp.StatusCode, string(respBody))
	}

	var ykResp YooKassaPaymentResponse
	if err := json.Unmarshal(respBody, &ykResp); err != nil {
		return payment, "", err
	}

	payment.ExternalID = ykResp.ID
	database.DB.Save(payment)

	confirmationURL := ""
	if ykResp.Confirmation.ConfirmationURL != "" {
		confirmationURL = ykResp.Confirmation.ConfirmationURL
	}

	return payment, confirmationURL, nil
}

// HandleWebhook обрабатывает webhook от ЮKassa
func (s *YooKassaService) HandleWebhook(body []byte) error {
	var payload struct {
		Type   string `json:"type"`
		Event  string `json:"event"` // payment.succeeded, payment.canceled, payment.waiting_for_capture
		Object struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"object"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return err
	}

	paymentID := payload.Object.ID
	status := payload.Object.Status

	// Ищем платёж по ExternalID
	var payment models.Payment
	if err := database.DB.Where("external_id = ? AND provider = ?", paymentID, models.PaymentProviderYooKassa).First(&payment).Error; err != nil {
		utils.GetLogger().Warn("Webhook: платёж не найден", zap.String("external_id", paymentID))
		return nil // 200 OK, чтобы ЮKassa не повторяла
	}

	switch status {
	case "succeeded", "waiting_for_capture":
		payment.Status = models.PaymentStatusSucceeded
		database.DB.Save(&payment)
		utils.GetLogger().Info("Платёж успешен", zap.String("id", payment.ID.String()), zap.String("external_id", paymentID))

		// Активация подписки при оплате
		if payment.PaymentType == models.PaymentTypeSubscription {
			plan := PlanFromAmount(payment.Amount)
			if plan != "" {
				CreateSubscriptionFromPayment(payment.UserID, plan, payment.ID)
			}
		}
		// Активация продвижения события при оплате
		if payment.PaymentType == models.PaymentTypePromotion {
			CreateEventPromotionFromPayment(&payment)
		}
	case "canceled":
		payment.Status = models.PaymentStatusCancelled
		database.DB.Save(&payment)
		utils.GetLogger().Info("Платёж отменён", zap.String("id", payment.ID.String()))
	}

	return nil
}
