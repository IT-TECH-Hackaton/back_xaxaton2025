package main

import (
	"fmt"
	"log"
	"strings"

	"bekend/config"
	"bekend/database"
	"bekend/models"
)

func main() {
	config.LoadConfig()
	database.Connect()

	fmt.Println("🧪 ПРОСТОЙ ТЕСТ: Симуляция запроса API")
	fmt.Println()

	// Симулируем запрос GET /api/events?tab=active&page=1&limit=12
	tab := "active"
	pageInt := 1
	limitInt := 12
	offset := (pageInt - 1) * limitInt

	fmt.Printf("Параметры запроса:\n")
	fmt.Printf("  tab: %s\n", tab)
	fmt.Printf("  page: %d\n", pageInt)
	fmt.Printf("  limit: %d\n", limitInt)
	fmt.Printf("  offset: %d\n", offset)
	fmt.Println()

	// Создаем запрос точно как в handlers/event.go
	query := database.DB.Model(&models.Event{}).
		Preload("Organizer").
		Preload("Participants").
		Preload("Categories")

	switch tab {
	case "active":
		query = query.Where("status = ?", models.EventStatusActive)
	}

	// Подсчет
	var total int64
	if err := query.Count(&total).Error; err != nil {
		log.Fatalf("❌ Ошибка подсчета: %v", err)
	}
	fmt.Printf("📊 Total: %d\n", total)

	if total == 0 {
		fmt.Println("❌ ПРОБЛЕМА: Запрос не находит события!")
		fmt.Println("💡 Проверьте:")
		fmt.Println("   1. Есть ли события в базе: go run scripts/check_events.go")
		fmt.Println("   2. Правильно ли работает фильтр status = 'Активное'")
		return
	}

	// Получение событий
	orderBy := "start_date ASC"
	var events []models.Event
	if err := query.Offset(offset).Limit(limitInt).Order(orderBy).Find(&events).Error; err != nil {
		log.Fatalf("❌ Ошибка получения событий: %v", err)
	}

	fmt.Printf("✅ Найдено событий: %d\n", len(events))
	fmt.Println()

	// Проверка каждого события
	for i, event := range events {
		fmt.Printf("Событие %d:\n", i+1)
		fmt.Printf("  ID: %s\n", event.ID.String())
		fmt.Printf("  Title: %s\n", event.Title)
		fmt.Printf("  Status: %s\n", event.Status)
		
		// Проверка Organizer
		organizerOK := event.Organizer.ID.String() != "00000000-0000-0000-0000-000000000000"
		fmt.Printf("  Organizer загружен: %v\n", organizerOK)
		if !organizerOK {
			fmt.Printf("  ⚠️  ВНИМАНИЕ: Organizer не загружен! ID организатора: %s\n", event.OrganizerID.String())
		}
		
		// Проверка Tags
		fmt.Printf("  Tags: %v (тип: %T)\n", event.Tags, event.Tags)
		fmt.Printf("  Tags как []string: %v\n", []string(event.Tags))
		
		// Проверка Categories
		fmt.Printf("  Categories: %d\n", len(event.Categories))
		
		fmt.Println()
	}

	// Тест создания простого ответа
	fmt.Println("📤 Тест создания ответа...")
	type SimpleResponse struct {
		ID    string `json:"id"`
		Title string `json:"title"`
		Tags  []string `json:"tags"`
	}

	simpleResults := make([]SimpleResponse, 0, len(events))
	for _, event := range events {
		simpleResults = append(simpleResults, SimpleResponse{
			ID:    event.ID.String(),
			Title: event.Title,
			Tags:  []string(event.Tags),
		})
	}

	fmt.Printf("✅ Создано %d ответов\n", len(simpleResults))
	fmt.Println()

	// Итог
	fmt.Println(strings.Repeat("=", 50))
	if len(events) > 0 {
		fmt.Println("✅ ТЕСТ ПРОЙДЕН: События найдены и обработаны")
		fmt.Println("💡 Если API все еще не работает, проблема в:")
		fmt.Println("   1. Обработке данных в handlers/event.go")
		fmt.Println("   2. JSON сериализации")
		fmt.Println("   3. Логике создания DTO")
	} else {
		fmt.Println("❌ ТЕСТ НЕ ПРОЙДЕН: События не найдены")
	}
}

