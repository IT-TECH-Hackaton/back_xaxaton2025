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

	fmt.Println("🔍 Проверка событий в базе данных...")
	fmt.Println()

	var totalEvents int64
	database.DB.Model(&models.Event{}).Count(&totalEvents)
	fmt.Printf("📊 Всего событий: %d\n", totalEvents)

	var activeEvents int64
	database.DB.Model(&models.Event{}).Where("status = ?", models.EventStatusActive).Count(&activeEvents)
	fmt.Printf("🟢 Активных событий: %d\n", activeEvents)

	var pastEvents int64
	database.DB.Model(&models.Event{}).Where("status = ?", models.EventStatusPast).Count(&pastEvents)
	fmt.Printf("⚫ Прошедших событий: %d\n", pastEvents)

	var rejectedEvents int64
	database.DB.Model(&models.Event{}).Where("status = ?", models.EventStatusRejected).Count(&rejectedEvents)
	fmt.Printf("🔴 Отклоненных событий: %d\n", rejectedEvents)

	fmt.Println()
	fmt.Println("📋 Список активных событий:")
	var events []models.Event
	if err := database.DB.Where("status = ?", models.EventStatusActive).
		Order("start_date ASC").
		Limit(10).
		Find(&events).Error; err != nil {
		log.Printf("Ошибка получения событий: %v", err)
	} else {
		if len(events) == 0 {
			fmt.Println("  ❌ Активных событий не найдено")
		} else {
			for i, event := range events {
				fmt.Printf("  %d. %s (ID: %s, Start: %s, End: %s)\n",
					i+1, event.Title, event.ID.String()[:8], event.StartDate.Format("2006-01-02 15:04"), event.EndDate.Format("2006-01-02 15:04"))
			}
		}
	}

	fmt.Println()
	fmt.Println("📋 Список всех событий (первые 10):")
	var allEvents []models.Event
	if err := database.DB.Order("created_at DESC").
		Limit(10).
		Find(&allEvents).Error; err != nil {
		log.Printf("Ошибка получения событий: %v", err)
	} else {
		if len(allEvents) == 0 {
			fmt.Println("  ❌ Событий не найдено")
		} else {
			for i, event := range allEvents {
				fmt.Printf("  %d. %s (Статус: %s, Start: %s)\n",
					i+1, event.Title, event.Status, event.StartDate.Format("2006-01-02 15:04"))
			}
		}
	}
}

