// Скрипт для создания тестовых данных коммерции: заявки на рекламодателя и баннеры на модерации
// Запуск: go run ./scripts/seed_commerce.go
package main

import (
	"log"

	"bekend/config"
	"bekend/database"
	"bekend/logger"
	"bekend/services"
)

func main() {
	config.LoadConfig()
	logger.InitLogger(config.AppConfig.AppEnv)
	defer logger.Sync()
	database.Connect()

	log.Println("=== Сид коммерции (заявки + баннеры на модерации) ===")
	services.RunSeedCommerce()
	log.Println("=== Сид коммерции завершён ===")
}
