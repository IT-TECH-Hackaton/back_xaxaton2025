package main

import (
	"fmt"
	"log"

	"bekend/config"
	"bekend/database"
)

func main() {
	config.LoadConfig()
	
	fmt.Println("=== Проверка базы данных ===")
	fmt.Printf("Host: %s\n", config.AppConfig.DBHost)
	fmt.Printf("Port: %s\n", config.AppConfig.DBPort)
	fmt.Printf("User: %s\n", config.AppConfig.DBUser)
	fmt.Printf("Database: %s\n", config.AppConfig.DBName)
	fmt.Println()

	database.Connect()
	defer func() {
		sqlDB, _ := database.DB.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}()

	fmt.Println("Проверка таблиц...")
	
	var tables []string
	if err := database.DB.Raw(`
		SELECT table_name 
		FROM information_schema.tables 
		WHERE table_schema = 'public'
		ORDER BY table_name
	`).Scan(&tables).Error; err != nil {
		log.Fatalf("Ошибка получения списка таблиц: %v", err)
	}

	fmt.Printf("Найдено таблиц: %d\n", len(tables))
	for _, table := range tables {
		fmt.Printf("  - %s\n", table)
	}

	fmt.Println("\nПроверка структуры таблицы users...")
	var userColumns []struct {
		ColumnName string
		DataType   string
		IsNullable string
	}
	if err := database.DB.Raw(`
		SELECT column_name, data_type, is_nullable
		FROM information_schema.columns
		WHERE table_name = 'users'
		ORDER BY ordinal_position
	`).Scan(&userColumns).Error; err != nil {
		log.Fatalf("Ошибка получения структуры таблицы users: %v", err)
	}

	for _, col := range userColumns {
		fmt.Printf("  - %s (%s, nullable: %s)\n", col.ColumnName, col.DataType, col.IsNullable)
	}

	fmt.Println("\nПроверка структуры таблицы registration_pendings...")
	var regColumns []struct {
		ColumnName string
		DataType   string
		IsNullable string
	}
	if err := database.DB.Raw(`
		SELECT column_name, data_type, is_nullable
		FROM information_schema.columns
		WHERE table_name = 'registration_pendings'
		ORDER BY ordinal_position
	`).Scan(&regColumns).Error; err != nil {
		log.Fatalf("Ошибка получения структуры таблицы registration_pendings: %v", err)
	}

	for _, col := range regColumns {
		fmt.Printf("  - %s (%s, nullable: %s)\n", col.ColumnName, col.DataType, col.IsNullable)
	}

	fmt.Println("\n✅ Проверка завершена")
}

