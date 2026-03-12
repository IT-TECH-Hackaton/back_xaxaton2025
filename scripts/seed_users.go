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

var testUsers = []struct {
	Email    string
	Password string
	FullName string
	Role     models.UserRole
}{
	{"user1@test.local", "User123!", "Иван Петров", models.RoleUser},
	{"user2@test.local", "User123!", "Мария Сидорова", models.RoleUser},
	{"user3@test.local", "User123!", "Алексей Козлов", models.RoleUser},
	{"moderator@test.local", "Mod123!", "Елена Модератор", models.RoleAdmin},
}

func main() {
	config.LoadConfig()
	database.Connect()

	created := 0
	skipped := 0

	for _, u := range testUsers {
		var existing models.User
		err := database.DB.Unscoped().Where("email = ?", u.Email).First(&existing).Error
		if err == nil {
			if existing.Status == models.UserStatusDeleted {
				database.DB.Unscoped().Delete(&existing)
			} else {
				fmt.Printf("⏭ Пропуск (уже существует): %s\n", u.Email)
				skipped++
				continue
			}
		}

		hashedPassword, err := utils.HashPassword(u.Password)
		if err != nil {
			log.Printf("Ошибка хеширования пароля для %s: %v", u.Email, err)
			continue
		}

		user := models.User{
			ID:            uuid.New(),
			FullName:      u.FullName,
			Email:         u.Email,
			Password:      hashedPassword,
			Role:          u.Role,
			Status:        models.UserStatusActive,
			EmailVerified: true,
			AuthProvider:  "email",
		}

		if err := database.DB.Create(&user).Error; err != nil {
			log.Printf("Ошибка создания %s: %v", u.Email, err)
			continue
		}

		fmt.Printf("✅ Создан: %s / %s (роль: %s)\n", u.Email, u.Password, u.Role)
		created++
	}

	fmt.Printf("\nГотово. Создано: %d, пропущено: %d\n", created, skipped)
	fmt.Println("\nТестовые пользователи:")
	fmt.Println("  user1@test.local / User123! — Иван Петров")
	fmt.Println("  user2@test.local / User123! — Мария Сидорова")
	fmt.Println("  user3@test.local / User123! — Алексей Козлов")
	fmt.Println("  moderator@test.local / Mod123! — Елена Модератор (админ)")
	fmt.Println("  admin@admin.com / Admin123! — администратор по умолчанию")
}
