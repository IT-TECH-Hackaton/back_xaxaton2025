// Скрипт для создания тестовых рекламных баннеров
// Запуск: go run ./scripts/seed_ads.go
// Требуется: хотя бы один пользователь с ролью Рекламодатель (назначьте через админку)
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

var testAds = []struct {
	imageURL string
	linkURL  string
	title    string
	target   int
	price    float64
}{
	{
		imageURL: "https://images.unsplash.com/photo-1557804506-669a67965ba0?w=800&h=400&fit=crop",
		linkURL:  "https://example.com/business",
		title:    "Бизнес-конференция 2026",
		target:   5000,
		price:    1500,
	},
	{
		imageURL: "https://images.unsplash.com/photo-1511578314322-379afb476865?w=800&h=400&fit=crop",
		linkURL:  "https://example.com/restaurant",
		title:    "Ресторан «Вкусный вечер» — скидка 20%",
		target:   3000,
		price:    800,
	},
	{
		imageURL: "https://images.unsplash.com/photo-1522071820081-009f0129c71c?w=800&h=400&fit=crop",
		linkURL:  "https://example.com/coworking",
		title:    "Коворкинг в центре города",
		target:   2000,
		price:    500,
	},
	{
		imageURL: "https://images.unsplash.com/photo-1460925895917-afdab827c52f?w=800&h=400&fit=crop",
		linkURL:  "https://example.com/analytics",
		title:    "Курсы по аналитике данных",
		target:   4000,
		price:    1200,
	},
	{
		imageURL: "https://images.unsplash.com/photo-1542744173-8e7e53415bb0?w=800&h=400&fit=crop",
		linkURL:  "https://example.com/events",
		title:    "Платформа для организаторов мероприятий",
		target:   6000,
		price:    2000,
	},
}

func main() {
	config.LoadConfig()
	logger.InitLogger(config.AppConfig.AppEnv)
	defer logger.Sync()
	database.Connect()

	log.Println("=== Сид рекламных баннеров ===")

	// Ищем рекламодателя
	var advertiser models.User
	if err := database.DB.Where("role = ?", models.RoleAdvertiser).First(&advertiser).Error; err != nil {
		// Создаём рекламодателя, если нет
		hash, _ := utils.HashPassword("Advertiser123!")
		advertiser = models.User{
			ID:           uuid.New(),
			FullName:     "Тест Рекламодатель",
			Email:        "advertiser@test.local",
			Password:     hash,
			Role:         models.RoleAdvertiser,
			Status:       models.UserStatusActive,
			EmailVerified: true,
			AuthProvider: "email",
		}
		if err := database.DB.Create(&advertiser).Error; err != nil {
			log.Fatalf("Не удалось создать рекламодателя: %v", err)
		}
		log.Printf("Создан рекламодатель: %s (%s)", advertiser.Email, advertiser.FullName)
	} else {
		log.Printf("Найден рекламодатель: %s", advertiser.Email)
	}

	created := 0
	for _, ad := range testAds {
		var existing models.AdBanner
		if database.DB.Where("advertiser_id = ? AND title = ?", advertiser.ID, ad.title).First(&existing).Error == nil {
			continue
		}
		banner := models.AdBanner{
			ID:                uuid.New(),
			AdvertiserID:      advertiser.ID,
			ImageURL:          ad.imageURL,
			LinkURL:           ad.linkURL,
			Title:             ad.title,
			TargetImpressions: ad.target,
			Price:             ad.price,
			Status:            models.AdBannerStatusActive,
		}
		if err := database.DB.Create(&banner).Error; err != nil {
			log.Printf("Ошибка создания баннера %s: %v", ad.title, err)
			continue
		}
		created++
		log.Printf("Создан баннер: %s", ad.title)
	}

	log.Printf("Создано баннеров: %d (всего тестовых: %d)", created, len(testAds))
	log.Println("=== Сид рекламы завершён ===")
}
