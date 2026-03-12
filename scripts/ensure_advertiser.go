// Скрипт для создания/обновления тестового рекламодателя с известным паролем
// Запуск: go run ./scripts/ensure_advertiser.go
// Логин: advertiser@test.local
// Пароль: Advertiser123!
package main

import (
	"log"

	"bekend/config"
	"bekend/database"
	"bekend/logger"
	"bekend/models"
	"bekend/utils"

	"github.com/google/uuid"
)

func main() {
	config.LoadConfig()
	logger.InitLogger(config.AppConfig.AppEnv)
	defer logger.Sync()
	database.Connect()

	email := "advertiser@test.local"
	password := "Advertiser123!"

	var user models.User
	err := database.DB.Where("email = ?", email).First(&user).Error
	if err == nil {
		hash, err := utils.HashPassword(password)
		if err != nil {
			log.Fatalf("Ошибка хеширования: %v", err)
		}
		user.Password = hash
		user.Role = models.RoleAdvertiser
		user.Status = models.UserStatusActive
		user.EmailVerified = true
		if err := database.DB.Save(&user).Error; err != nil {
			log.Fatalf("Ошибка обновления: %v", err)
		}
		log.Printf("Пароль рекламодателя обновлён: %s", email)
	} else {
		hash, err := utils.HashPassword(password)
		if err != nil {
			log.Fatalf("Ошибка хеширования: %v", err)
		}
		user = models.User{
			ID:            uuid.New(),
			FullName:      "Тест Рекламодатель",
			Email:         email,
			Password:      hash,
			Role:          models.RoleAdvertiser,
			Status:        models.UserStatusActive,
			EmailVerified: true,
			AuthProvider:  "email",
		}
		if err := database.DB.Create(&user).Error; err != nil {
			log.Fatalf("Ошибка создания: %v", err)
		}
		log.Printf("Создан рекламодатель: %s", email)
	}

	log.Println("=== Готово ===")
	log.Printf("Логин:  %s", email)
	log.Printf("Пароль: %s", password)
}
