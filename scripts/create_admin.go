package main

import (
	"fmt"
	"log"
	"os"

	"bekend/config"
	"bekend/database"
	"bekend/models"
	"bekend/utils"

	"github.com/google/uuid"
)

func main() {
	config.LoadConfig()
	database.Connect()

	if len(os.Args) < 4 {
		log.Fatal("Использование: go run create_admin.go <email> <password> <fullName>")
	}

	email := os.Args[1]
	password := os.Args[2]
	fullName := os.Args[3]

	if !utils.ValidateEmail(email) {
		log.Fatal("Неверный формат email")
	}

	if !utils.ValidatePassword(password) {
		log.Fatal("Пароль должен содержать минимум 8 символов, латинские буквы, цифры и специальные символы")
	}

	if !utils.ValidateFullName(fullName) {
		log.Fatal("ФИО должно содержать только русские буквы")
	}

	var existingUser models.User
	if err := database.DB.Where("email = ?", email).First(&existingUser).Error; err == nil {
		log.Fatal("Пользователь с таким email уже существует")
	}

	hashedPassword, err := utils.HashPassword(password)
	if err != nil {
		log.Fatal("Ошибка при хешировании пароля:", err)
	}

	admin := models.User{
		ID:            uuid.New(),
		FullName:      fullName,
		Email:         email,
		Password:      hashedPassword,
		Role:          models.RoleAdmin,
		Status:        models.UserStatusActive,
		EmailVerified: true,
	}

	if err := database.DB.Create(&admin).Error; err != nil {
		log.Fatal("Ошибка при создании администратора:", err)
	}

	fmt.Printf("Администратор успешно создан:\n")
	fmt.Printf("Email: %s\n", email)
	fmt.Printf("ФИО: %s\n", fullName)
}

