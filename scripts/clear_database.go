package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"bekend/config"
	"bekend/database"
	"bekend/models"
)

func main() {
	config.LoadConfig()
	database.Connect()
	defer func() {
		sqlDB, _ := database.DB.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}()

	fmt.Println("⚠️  ОЧИСТКА БАЗЫ ДАННЫХ")
	fmt.Println("========================")
	fmt.Println()
	fmt.Println("Это действие удалит ВСЕ данные из базы данных!")
	fmt.Println()
	fmt.Println("Будут удалены все записи из:")
	fmt.Println("  - Отзывы (event_reviews)")
	fmt.Println("  - Участники событий (event_participants)")
	fmt.Println("  - Запросы на матчинг (match_requests)")
	fmt.Println("  - Матчинги событий (event_matchings)")
	fmt.Println("  - Участники сообществ (community_members)")
	fmt.Println("  - Интересы сообществ (community_interests)")
	fmt.Println("  - Сообщества (micro_communities)")
	fmt.Println("  - Интересы пользователей (user_interests)")
	fmt.Println("  - События-категории (event_categories)")
	fmt.Println("  - События (events)")
	fmt.Println("  - Категории (categories)")
	fmt.Println("  - Интересы (interests)")
	fmt.Println("  - Сбросы паролей (password_resets)")
	fmt.Println("  - Ожидающие регистрации (registration_pendings)")
	fmt.Println("  - Верификации email (email_verifications)")
	fmt.Println("  - Пользователи (users)")
	fmt.Println()
	fmt.Print("Вы уверены? Введите 'DELETE ALL' для подтверждения: ")

	reader := bufio.NewReader(os.Stdin)
	confirmation, _ := reader.ReadString('\n')
	confirmation = strings.TrimSpace(confirmation)

	if confirmation != "DELETE ALL" {
		fmt.Println("❌ Отменено. Данные не были удалены.")
		return
	}

	fmt.Println()
	fmt.Println("🗑️  Начало очистки базы данных...")
	fmt.Println()

	totalDeleted := 0

	// Удаление в правильном порядке (сначала дочерние таблицы)
	tables := []struct {
		name        string
		model       interface{}
		description string
	}{
		{"event_reviews", &models.EventReview{}, "Отзывы"},
		{"match_requests", &models.MatchRequest{}, "Запросы на матчинг"},
		{"event_matchings", &models.EventMatching{}, "Матчинги событий"},
		{"event_participants", &models.EventParticipant{}, "Участники событий"},
		{"community_members", &models.CommunityMember{}, "Участники сообществ"},
		{"community_interests", &models.CommunityInterest{}, "Интересы сообществ"},
		{"micro_communities", &models.MicroCommunity{}, "Сообщества"},
		{"user_interests", &models.UserInterest{}, "Интересы пользователей"},
		{"event_categories", &models.EventCategory{}, "Связи событий и категорий"},
		{"events", &models.Event{}, "События"},
		{"categories", &models.Category{}, "Категории"},
		{"interests", &models.Interest{}, "Интересы"},
		{"password_resets", &models.PasswordReset{}, "Сбросы паролей"},
		{"registration_pendings", &models.RegistrationPending{}, "Ожидающие регистрации"},
		{"email_verifications", &models.EmailVerification{}, "Верификации email"},
		{"users", &models.User{}, "Пользователи"},
	}

	for _, table := range tables {
		var count int64
		if err := database.DB.Model(table.model).Count(&count).Error; err != nil {
			fmt.Printf("  ⚠️  Ошибка подсчета записей в %s: %v\n", table.description, err)
			continue
		}

		if count == 0 {
			fmt.Printf("  ℹ️  %s: пусто (0 записей)\n", table.description)
			continue
		}

		if err := database.DB.Unscoped().Where("1 = 1").Delete(table.model).Error; err != nil {
			fmt.Printf("  ❌ Ошибка удаления из %s: %v\n", table.description, err)
			continue
		}

		fmt.Printf("  ✅ %s: удалено %d записей\n", table.description, count)
		totalDeleted += int(count)
	}

	fmt.Println()
	fmt.Println(strings.Repeat("=", 40))
	fmt.Printf("✅ Очистка завершена! Удалено записей: %d\n", totalDeleted)
	fmt.Println()

	fmt.Println("📊 Текущее состояние базы данных:")
	fmt.Println()

	statsTables := []struct {
		table string
		model interface{}
	}{
		{"users", &models.User{}},
		{"events", &models.Event{}},
		{"categories", &models.Category{}},
		{"event_participants", &models.EventParticipant{}},
		{"event_reviews", &models.EventReview{}},
		{"interests", &models.Interest{}},
		{"micro_communities", &models.MicroCommunity{}},
	}

	for _, s := range statsTables {
		var count int64
		database.DB.Model(s.model).Count(&count)
		fmt.Printf("  %s: %d записей\n", s.table, count)
	}

	fmt.Println()
	fmt.Println("💡 Теперь вы можете запустить: go run scripts/seed_events.go")
}

