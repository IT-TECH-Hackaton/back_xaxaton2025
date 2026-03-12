package main

import (
	"fmt"
	"log"

	"bekend/config"
	"bekend/database"
	"bekend/models"
	"bekend/utils"

	"github.com/google/uuid"
)

func main() {
	config.LoadConfig()
	
	fmt.Println("=== Создание администратора ===")
	fmt.Printf("Host: %s\n", config.AppConfig.DBHost)
	fmt.Printf("Port: %s\n", config.AppConfig.DBPort)
	fmt.Printf("User: %s\n", config.AppConfig.DBUser)
	fmt.Printf("Database: %s\n", config.AppConfig.DBName)
	fmt.Println()

	database.Connect()
	defer func() {
		sqlDB, _ := database.DB.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}()

	// Данные администратора
	adminEmail := "admin@admin.com"
	adminPassword := "Admin123!"
	adminFullName := "Администратор"

	fmt.Printf("Создание администратора:\n")
	fmt.Printf("  Email: %s\n", adminEmail)
	fmt.Printf("  Пароль: %s\n", adminPassword)
	fmt.Println()

	// Проверяем, существует ли уже такой пользователь
	var existingUser models.User
	if err := database.DB.Where("email = ?", adminEmail).First(&existingUser).Error; err == nil {
		fmt.Printf("Пользователь с email %s уже существует. Удаление...\n", adminEmail)
		if err := database.DB.Unscoped().Delete(&existingUser).Error; err != nil {
			log.Fatalf("Ошибка удаления существующего пользователя: %v", err)
		}
		fmt.Println("✅ Старый пользователь удален")
	}

	// Хешируем пароль
	hashedPassword, err := utils.HashPassword(adminPassword)
	if err != nil {
		log.Fatalf("Ошибка при хешировании пароля: %v", err)
	}

	// Создаем администратора
	admin := models.User{
		ID:            uuid.New(),
		FullName:      adminFullName,
		Email:         adminEmail,
		Password:      hashedPassword,
		YandexID:      nil,
		Role:          models.RoleAdmin,
		Status:        models.UserStatusActive,
		EmailVerified: true,
		AuthProvider:  "email",
	}

	if err := database.DB.Create(&admin).Error; err != nil {
		log.Fatalf("Ошибка при создании администратора: %v", err)
	}

	fmt.Println("✅ Администратор успешно создан!")
	fmt.Println()
	fmt.Println("=== ДАННЫЕ ДЛЯ ВХОДА ===")
	fmt.Printf("Email:    %s\n", adminEmail)
	fmt.Printf("Пароль:   %s\n", adminPassword)
	fmt.Println("========================")
}

