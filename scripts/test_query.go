package main

import (
	"fmt"
	"log"

	"bekend/config"
	"bekend/database"
	"bekend/models"
)

func main() {
	config.LoadConfig()
	database.Connect()

	fmt.Println("🔍 Тестирование запроса событий...")
	fmt.Println()

	// Тестируем запрос, аналогичный тому, что используется в API
	query := database.DB.Model(&models.Event{}).
		Preload("Organizer").
		Preload("Participants").
		Preload("Categories").
		Where("status = ?", models.EventStatusActive)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		log.Fatalf("Ошибка подсчета: %v", err)
	}

	fmt.Printf("📊 Всего активных событий (по запросу): %d\n", total)

	var events []models.Event
	if err := query.Order("start_date ASC").Limit(12).Find(&events).Error; err != nil {
		log.Fatalf("Ошибка получения событий: %v", err)
	}

	fmt.Printf("✅ Найдено событий: %d\n", len(events))
	fmt.Println()

	for i, event := range events {
		fmt.Printf("%d. %s\n", i+1, event.Title)
		fmt.Printf("   ID: %s\n", event.ID.String())
		fmt.Printf("   Статус: %s\n", event.Status)
		fmt.Printf("   Организатор ID: %s\n", event.OrganizerID.String())
		organizerLoaded := event.Organizer.ID.String() != "00000000-0000-0000-0000-000000000000"
		fmt.Printf("   Организатор загружен: %v\n", organizerLoaded)
		if organizerLoaded {
			fmt.Printf("   Организатор: %s (%s)\n", event.Organizer.FullName, event.Organizer.Email)
		}
		fmt.Printf("   Категорий: %d\n", len(event.Categories))
		fmt.Printf("   Тегов: %d\n", len(event.Tags))
		fmt.Printf("   StartDate: %s\n", event.StartDate.Format("2006-01-02 15:04:05"))
		fmt.Println()
	}

	// Проверяем, что происходит при преобразовании в JSON
	fmt.Println("🧪 Тестирование преобразования в JSON...")
	
	type TestResponse struct {
		ID       string   `json:"id"`
		Title    string   `json:"title"`
		Status   string   `json:"status"`
		Tags     []string `json:"tags"`
	}

	testEvents := make([]TestResponse, 0, len(events))
	for _, event := range events {
		testEvents = append(testEvents, TestResponse{
			ID:     event.ID.String(),
			Title:  event.Title,
			Status: string(event.Status),
			Tags:   []string(event.Tags),
		})
	}

	fmt.Printf("✅ Успешно преобразовано %d событий для JSON\n", len(testEvents))
}

