package gigachat

import (
	"bytes"
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"golos/internal/config"
)

type Client struct {
	config      config.GigaChatConfig
	accessToken string
	tokenExpiry time.Time
	mu          sync.RWMutex
	httpClient  *http.Client
}

func NewClient(cfg config.GigaChatConfig) *Client {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
		DisableKeepAlives:   false,
	}
	return &Client{
		config: cfg,
		httpClient: &http.Client{
			Timeout:   90 * time.Second,
			Transport: tr,
		},
	}
}

func (c *Client) getAccessToken() (string, error) {
	c.mu.RLock()
	if c.accessToken != "" && time.Now().Before(c.tokenExpiry) {
		token := c.accessToken
		c.mu.RUnlock()
		return token, nil
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.accessToken != "" && time.Now().Before(c.tokenExpiry) {
		return c.accessToken, nil
	}

	data := url.Values{}
	data.Set("scope", c.config.Scope)

	req, err := http.NewRequest("POST", c.config.AuthURL, bytes.NewBufferString(data.Encode()))
	if err != nil {
		return "", err
	}

	basicAuth := c.getBasicAuth()
	if basicAuth == "" {
		return "", fmt.Errorf("credentials not configured: ClientID and AuthorizationKey are empty")
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	if strings.HasPrefix(basicAuth, "Basic ") {
		req.Header.Set("Authorization", basicAuth)
	} else {
		req.Header.Set("Authorization", "Basic "+basicAuth)
	}

	rqUID := generateUUID()
	req.Header.Set("RqUID", rqUID)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("ошибка выполнения запроса: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		errorMsg := string(body)
		if errorMsg == "" {
			errorMsg = "пустой ответ от сервера"
		}
		return "", fmt.Errorf("ошибка получения токена: статус %d, ответ: %s, URL: %s, заголовки: Authorization=Basic [скрыто], RqUID=%s", resp.StatusCode, errorMsg, c.config.AuthURL, rqUID)
	}

	var tokenResp TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("ошибка декодирования ответа: %w, тело ответа: %s", err, string(body))
	}

	c.accessToken = tokenResp.AccessToken
	c.tokenExpiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

	return c.accessToken, nil
}

func generateUUID() string {
	b := make([]byte, 16)
	rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

func (c *Client) getBasicAuth() string {
	if c.config.ClientID != "" && c.config.ClientSecret != "" {
		return base64.StdEncoding.EncodeToString([]byte(c.config.ClientID + ":" + c.config.ClientSecret))
	}
	if c.config.AuthorizationKey != "" {
		authKey := strings.TrimSpace(c.config.AuthorizationKey)
		if strings.HasPrefix(authKey, "Basic ") {
			return strings.TrimPrefix(authKey, "Basic ")
		}
		return authKey
	}
	if c.config.ClientID != "" {
		return base64.StdEncoding.EncodeToString([]byte(c.config.ClientID + ":"))
	}
	return ""
}

func (c *Client) SendMessage(message, sessionID string) (string, error) {
	messages := []Message{
		{
			Role:    "user",
			Content: message,
		},
	}
	return c.SendMessageWithContext(messages)
}

func (c *Client) SendMessageWithContext(messages []Message) (string, error) {
	const maxRetries = 5
	const retryDelay = 2 * time.Second

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(retryDelay * time.Duration(attempt))
		}

		result, err := c.sendMessageRequest(messages)
		if err == nil {
			return result, nil
		}

		lastErr = err
		if !isRetryableError(err) {
			return "", err
		}
	}

	return "", fmt.Errorf("превышено количество попыток: %w", lastErr)
}

func (c *Client) sendMessageRequest(messages []Message) (string, error) {
	token, err := c.getAccessToken()
	if err != nil {
		return "", fmt.Errorf("ошибка получения токена: %w", err)
	}

	if len(messages) == 0 {
		return "", fmt.Errorf("массив сообщений пустой")
	}

	reqBody := ChatRequest{
		Model:       "GigaChat-Pro",
		Messages:    messages,
		Temperature: 0.7,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("ошибка сериализации: %w", err)
	}

	log.Printf("Отправка запроса к GigaChat: модель=%s, сообщений=%d", reqBody.Model, len(messages))

	req, err := http.NewRequest("POST", c.config.APIURL+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("ошибка создания запроса: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Connection", "close")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("ошибка выполнения запроса: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		c.mu.Lock()
		c.accessToken = ""
		c.mu.Unlock()
		return "", fmt.Errorf("токен истек, требуется повторная аутентификация")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("ошибка GigaChat API: %d - %s, запрос: модель=%s, сообщений=%d", resp.StatusCode, string(body), reqBody.Model, len(reqBody.Messages))
	}

	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", fmt.Errorf("ошибка декодирования ответа: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("пустой ответ от GigaChat")
	}

	return chatResp.Choices[0].Message.Content, nil
}

func isRetryableError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "connection") ||
		strings.Contains(errStr, "network") ||
		strings.Contains(errStr, "EOF") ||
		strings.Contains(errStr, "broken pipe") ||
		strings.Contains(errStr, "connection reset") ||
		strings.Contains(errStr, "500") ||
		strings.Contains(errStr, "502") ||
		strings.Contains(errStr, "503") ||
		strings.Contains(errStr, "504")
}
