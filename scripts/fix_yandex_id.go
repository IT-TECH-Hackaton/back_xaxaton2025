package main

import (
	"fmt"
	"log"

	"bekend/config"
	"bekend/database"
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

	fmt.Println("Исправление типа yandex_id...")

	sqlDB, err := database.DB.DB()
	if err != nil {
		log.Fatalf("Ошибка получения sql.DB: %v", err)
	}

	sql := `
		ALTER TABLE users 
		ALTER COLUMN yandex_id DROP NOT NULL;
		
		UPDATE users 
		SET yandex_id = NULL 
		WHERE yandex_id = '';
	`

	if _, err := sqlDB.Exec(sql); err != nil {
		log.Fatalf("Ошибка выполнения миграции: %v", err)
	}

	fmt.Println("✅ Тип yandex_id исправлен на nullable")
}

