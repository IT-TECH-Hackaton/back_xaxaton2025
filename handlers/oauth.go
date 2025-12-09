package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"bekend/config"
	"bekend/database"
	"bekend/models"
	"bekend/utils"

	"github.com/gin-gonic/gin"
)

type YandexUserInfo struct {
	ID           string `json:"id"`
	Login        string `json:"login"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	DisplayName  string `json:"display_name"`
	DefaultEmail string `json:"default_email"`
	Emails       []string `json:"emails"`
}

type YandexTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

func (h *AuthHandler) YandexAuth(c *gin.Context) {
	// Если включена фейковая авторизация, возвращаем специальный ответ
	if config.AppConfig.FakeYandexAuth {
		c.JSON(http.StatusOK, gin.H{
			"fake": true,
			"message": "Используйте POST /api/auth/yandex/fake для фейковой авторизации",
		})
		return
	}

	if config.AppConfig.YandexClientID == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Яндекс OAuth не настроен"})
		return
	}

	state := utils.GenerateRandomString(32)
	
	authURL := fmt.Sprintf(
		"https://oauth.yandex.ru/authorize?response_type=code&client_id=%s&redirect_uri=%s&state=%s",
		config.AppConfig.YandexClientID,
		url.QueryEscape(config.AppConfig.YandexRedirectURI),
		state,
	)

	c.JSON(http.StatusOK, gin.H{
		"authUrl": authURL,
		"state":   state,
	})
}

func (h *AuthHandler) YandexCallback(c *gin.Context) {
	code := c.Query("code")
	_ = c.Query("state") // State можно использовать для проверки CSRF

	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Код авторизации не получен"})
		return
	}

	// Обмениваем код на токен
	token, err := h.exchangeCodeForToken(code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка получения токена: " + err.Error()})
		return
	}

	// Получаем информацию о пользователе
	userInfo, err := h.getYandexUserInfo(token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка получения информации о пользователе: " + err.Error()})
		return
	}

	// Определяем email
	email := userInfo.DefaultEmail
	if email == "" && len(userInfo.Emails) > 0 {
		email = userInfo.Emails[0]
	}
	if email == "" {
		email = userInfo.Login + "@yandex.ru"
	}

	// Определяем имя
	fullName := userInfo.DisplayName
	if fullName == "" {
		parts := []string{}
		if userInfo.FirstName != "" {
			parts = append(parts, userInfo.FirstName)
		}
		if userInfo.LastName != "" {
			parts = append(parts, userInfo.LastName)
		}
		if len(parts) > 0 {
			fullName = strings.Join(parts, " ")
		} else {
			fullName = userInfo.Login
		}
	}

	// Проверяем валидность имени (только русские буквы)
	if !utils.ValidateFullName(fullName) {
		// Если имя не на русском, используем логин
		fullName = userInfo.Login
	}

	// Ищем существующего пользователя по Яндекс ID или email
	var user models.User
	err = database.DB.Where("yandex_id = ? OR email = ?", userInfo.ID, email).First(&user).Error

	if err != nil {
		// Пользователь не найден - создаем нового
		user = models.User{
			FullName:      fullName,
			Email:         email,
			YandexID:      userInfo.ID,
			Password:      "", // Пароль не нужен для OAuth
			Role:          models.RoleUser,
			Status:        models.UserStatusActive,
			EmailVerified: true, // Email уже подтвержден через Яндекс
			AuthProvider:  "yandex",
		}

		if err := database.DB.Create(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка создания пользователя"})
			return
		}

		// Отправляем приветственное письмо
		go h.emailService.SendWelcomeEmail(user.Email, user.FullName)
	} else {
		// Пользователь найден - обновляем информацию
		if user.YandexID == "" {
			user.YandexID = userInfo.ID
		}
		if user.AuthProvider != "yandex" {
			user.AuthProvider = "yandex"
		}
		user.EmailVerified = true
		user.FullName = fullName
		database.DB.Save(&user)
	}

	// Генерируем JWT токен
	tokenJWT, err := utils.GenerateToken(user.ID, user.Email, string(user.Role))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка генерации токена"})
		return
	}

	// Перенаправляем на фронтенд с токеном
	redirectURL := fmt.Sprintf("%s/auth/callback?token=%s&provider=yandex", config.AppConfig.FrontendURL, tokenJWT)
	c.Redirect(http.StatusFound, redirectURL)
}

func (h *AuthHandler) exchangeCodeForToken(code string) (string, error) {
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("client_id", config.AppConfig.YandexClientID)
	data.Set("client_secret", config.AppConfig.YandexClientSecret)

	req, err := http.NewRequest("POST", "https://oauth.yandex.ru/token", strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ошибка получения токена: %s", string(body))
	}

	var tokenResp YandexTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", err
	}

	return tokenResp.AccessToken, nil
}

func (h *AuthHandler) getYandexUserInfo(accessToken string) (*YandexUserInfo, error) {
	req, err := http.NewRequest("GET", "https://login.yandex.ru/info", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "OAuth "+accessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ошибка получения информации: %s", string(body))
	}

	var userInfo YandexUserInfo
	if err := json.Unmarshal(body, &userInfo); err != nil {
		return nil, err
	}

	return &userInfo, nil
}

type FakeYandexAuthRequest struct {
	YandexID  string `json:"yandexId" binding:"required"`
	Email     string `json:"email" binding:"required,email"`
	FullName  string `json:"fullName" binding:"required"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
}

func (h *AuthHandler) FakeYandexAuth(c *gin.Context) {
	if !config.AppConfig.FakeYandexAuth {
		c.JSON(http.StatusForbidden, gin.H{"error": "Фейковая авторизация отключена"})
		return
	}

	var req FakeYandexAuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверные данные"})
		return
	}

	// Валидация
	if !utils.ValidateEmail(req.Email) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат email"})
		return
	}

	// Определяем имя
	fullName := req.FullName
	if fullName == "" {
		parts := []string{}
		if req.FirstName != "" {
			parts = append(parts, req.FirstName)
		}
		if req.LastName != "" {
			parts = append(parts, req.LastName)
		}
		if len(parts) > 0 {
			fullName = strings.Join(parts, " ")
		} else {
			fullName = "Пользователь Яндекс"
		}
	}

	// Проверяем валидность имени (только русские буквы)
	if !utils.ValidateFullName(fullName) {
		// Если имя не на русском, используем дефолтное
		fullName = "Пользователь Яндекс"
	}

	if !utils.ValidateStringLength(fullName, 2, 100) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ФИО должно быть от 2 до 100 символов"})
		return
	}

	// Ищем существующего пользователя по Яндекс ID или email
	var user models.User
	err := database.DB.Where("yandex_id = ? OR email = ?", req.YandexID, req.Email).First(&user).Error

	if err != nil {
		// Пользователь не найден - создаем нового
		user = models.User{
			FullName:      fullName,
			Email:         req.Email,
			YandexID:      req.YandexID,
			Password:      "", // Пароль не нужен для OAuth
			Role:          models.RoleUser,
			Status:        models.UserStatusActive,
			EmailVerified: true, // Email считается подтвержденным
			AuthProvider:  "yandex",
		}

		if err := database.DB.Create(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка создания пользователя"})
			return
		}

		// Отправляем приветственное письмо
		go h.emailService.SendWelcomeEmail(user.Email, user.FullName)
	} else {
		// Пользователь найден - обновляем информацию
		if user.YandexID == "" {
			user.YandexID = req.YandexID
		}
		if user.AuthProvider != "yandex" {
			user.AuthProvider = "yandex"
		}
		user.EmailVerified = true
		user.FullName = fullName
		database.DB.Save(&user)
	}

	// Генерируем JWT токен
	tokenJWT, err := utils.GenerateToken(user.ID, user.Email, string(user.Role))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка генерации токена"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": tokenJWT,
		"user": gin.H{
			"id":    user.ID,
			"email": user.Email,
			"role":  user.Role,
			"fullName": user.FullName,
		},
		"message": "Авторизация через Яндекс выполнена успешно (фейковый режим)",
	})
}

