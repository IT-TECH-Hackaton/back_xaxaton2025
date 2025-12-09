package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"bekend/config"
	"bekend/database"
	"bekend/models"
)

func main() {
	fmt.Println("🔍 ПОЛНАЯ ДИАГНОСТИКА ЗАПРОСА СОБЫТИЙ")
	fmt.Println(strings.Repeat("=", 50))
	fmt.Println()

	// 1. Загрузка конфигурации
	fmt.Println("1️⃣ Загрузка конфигурации...")
	config.LoadConfig()
	fmt.Printf("   ✅ Конфиг загружен\n")
	fmt.Printf("   DB Host: %s\n", config.AppConfig.DBHost)
	fmt.Printf("   DB Port: %s\n", config.AppConfig.DBPort)
	fmt.Printf("   DB Name: %s\n", config.AppConfig.DBName)
	fmt.Println()

	// 2. Подключение к БД
	fmt.Println("2️⃣ Подключение к базе данных...")
	database.Connect()
	fmt.Println("   ✅ Подключено")
	fmt.Println()

	// 3. Проверка наличия событий
	fmt.Println("3️⃣ Проверка наличия событий в БД...")
	var totalCount int64
	if err := database.DB.Model(&models.Event{}).Count(&totalCount).Error; err != nil {
		log.Fatalf("   ❌ Ошибка подсчета: %v", err)
	}
	fmt.Printf("   📊 Всего событий: %d\n", totalCount)

	var activeCount int64
	if err := database.DB.Model(&models.Event{}).
		Where("status = ?", models.EventStatusActive).
		Count(&activeCount).Error; err != nil {
		log.Fatalf("   ❌ Ошибка подсчета активных: %v", err)
	}
	fmt.Printf("   🟢 Активных событий: %d\n", activeCount)
	fmt.Println()

	if activeCount == 0 {
		fmt.Println("   ⚠️  ВНИМАНИЕ: Активных событий нет в базе!")
		fmt.Println("   💡 Запустите: go run scripts/seed_events.go")
		return
	}

	// 4. Тест запроса БЕЗ Preload
	fmt.Println("4️⃣ Тест запроса БЕЗ Preload...")
	var eventsSimple []models.Event
	querySimple := database.DB.Model(&models.Event{}).
		Where("status = ?", models.EventStatusActive).
		Order("start_date ASC").
		Limit(5)

	if err := querySimple.Find(&eventsSimple).Error; err != nil {
		log.Fatalf("   ❌ Ошибка запроса: %v", err)
	}
	fmt.Printf("   ✅ Найдено событий: %d\n", len(eventsSimple))
	for i, e := range eventsSimple {
		fmt.Printf("      %d. %s (ID: %s)\n", i+1, e.Title, e.ID.String()[:8])
	}
	fmt.Println()

	// 5. Тест запроса С Preload
	fmt.Println("5️⃣ Тест запроса С Preload...")
	var eventsWithPreload []models.Event
	queryPreload := database.DB.Model(&models.Event{}).
		Preload("Organizer").
		Preload("Participants").
		Preload("Categories").
		Where("status = ?", models.EventStatusActive).
		Order("start_date ASC").
		Limit(5)

	if err := queryPreload.Find(&eventsWithPreload).Error; err != nil {
		log.Fatalf("   ❌ Ошибка запроса с Preload: %v", err)
	}
	fmt.Printf("   ✅ Найдено событий: %d\n", len(eventsWithPreload))
	
	for i, e := range eventsWithPreload {
		organizerLoaded := e.Organizer.ID.String() != "00000000-0000-0000-0000-000000000000"
		fmt.Printf("      %d. %s\n", i+1, e.Title)
		fmt.Printf("         Организатор загружен: %v\n", organizerLoaded)
		if organizerLoaded {
			fmt.Printf("         Организатор: %s\n", e.Organizer.FullName)
		}
		fmt.Printf("         Категорий: %d\n", len(e.Categories))
		fmt.Printf("         Тегов: %d\n", len(e.Tags))
	}
	fmt.Println()

	// 6. Тест преобразования в JSON
	fmt.Println("6️⃣ Тест преобразования в JSON...")
	type TestEvent struct {
		ID       string   `json:"id"`
		Title    string   `json:"title"`
		Status   string   `json:"status"`
		Tags     []string `json:"tags"`
		Organizer struct {
			ID       string `json:"id"`
			FullName string `json:"fullName"`
		} `json:"organizer"`
	}

	testEvents := make([]TestEvent, 0, len(eventsWithPreload))
	for _, e := range eventsWithPreload {
		te := TestEvent{
			ID:     e.ID.String(),
			Title:  e.Title,
			Status: string(e.Status),
			Tags:   []string(e.Tags),
		}
		if e.Organizer.ID.String() != "00000000-0000-0000-0000-000000000000" {
			te.Organizer.ID = e.Organizer.ID.String()
			te.Organizer.FullName = e.Organizer.FullName
		}
		testEvents = append(testEvents, te)
	}

	jsonData, err := json.MarshalIndent(testEvents, "", "  ")
	if err != nil {
		log.Fatalf("   ❌ Ошибка сериализации JSON: %v", err)
	}
	fmt.Printf("   ✅ JSON сериализация успешна\n")
	fmt.Printf("   📄 Размер JSON: %d байт\n", len(jsonData))
	fmt.Println()

	// 7. Тест полного запроса как в API
	fmt.Println("7️⃣ Тест полного запроса (как в API)...")
	var total int64
	queryFull := database.DB.Model(&models.Event{}).
		Preload("Organizer").
		Preload("Participants").
		Preload("Categories").
		Where("status = ?", models.EventStatusActive)

	if err := queryFull.Count(&total).Error; err != nil {
		log.Fatalf("   ❌ Ошибка подсчета: %v", err)
	}
	fmt.Printf("   📊 Total: %d\n", total)

	var eventsFull []models.Event
	if err := queryFull.Order("start_date ASC").Limit(12).Find(&eventsFull).Error; err != nil {
		log.Fatalf("   ❌ Ошибка получения: %v", err)
	}
	fmt.Printf("   ✅ Найдено: %d\n", len(eventsFull))
	fmt.Println()

	// 8. Итоги
	fmt.Println(strings.Repeat("=", 50))
	fmt.Println("📋 ИТОГИ:")
	fmt.Printf("   - Всего событий в БД: %d\n", totalCount)
	fmt.Printf("   - Активных событий: %d\n", activeCount)
	fmt.Printf("   - Запрос БЕЗ Preload: ✅ (%d событий)\n", len(eventsSimple))
	fmt.Printf("   - Запрос С Preload: ✅ (%d событий)\n", len(eventsWithPreload))
	fmt.Printf("   - JSON сериализация: ✅\n")
	fmt.Printf("   - Полный запрос (как в API): ✅ (%d событий)\n", len(eventsFull))
	fmt.Println()

	if len(eventsFull) > 0 {
		fmt.Println("✅ ВСЕ ТЕСТЫ ПРОЙДЕНЫ!")
		fmt.Println("💡 Если API все еще возвращает пустой массив, проверьте:")
		fmt.Println("   1. Логи сервера при запросе")
		fmt.Println("   2. Правильно ли работает JSON сериализация в handlers")
		fmt.Println("   3. Нет ли ошибок при обработке данных")
	} else {
		fmt.Println("❌ ПРОБЛЕМА: Запрос не возвращает события!")
	}
}

