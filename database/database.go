package database

import (
	"fmt"
	"strings"

	"bekend/config"
	"bekend/logger"
	"bekend/models"

	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
)

var DB *gorm.DB

func createDatabaseIfNotExists() error {
	logger.GetLogger().Info("Проверка и создание базы данных", 
		zap.String("host", config.AppConfig.DBHost),
		zap.String("port", config.AppConfig.DBPort),
		zap.String("database", config.AppConfig.DBName))

	dsnPostgres := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=postgres port=%s sslmode=disable TimeZone=UTC",
		config.AppConfig.DBHost,
		config.AppConfig.DBUser,
		config.AppConfig.DBPassword,
		config.AppConfig.DBPort,
	)

	db, err := gorm.Open(postgres.Open(dsnPostgres), &gorm.Config{
		Logger: gormLogger.Default.LogMode(gormLogger.Silent),
	})
	if err != nil {
		return fmt.Errorf("не удалось подключиться к PostgreSQL: %w", err)
	}
	defer func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}()

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("ошибка получения sql.DB: %w", err)
	}

	var count int64
	err = sqlDB.QueryRow("SELECT COUNT(*) FROM pg_database WHERE datname = $1", config.AppConfig.DBName).Scan(&count)
	if err != nil {
		return fmt.Errorf("ошибка проверки существования базы данных: %w", err)
	}

	if count == 0 {
		logger.GetLogger().Info("База данных не найдена, создание новой базы", zap.String("database", config.AppConfig.DBName))
		
		createSQL := fmt.Sprintf(`CREATE DATABASE "%s"`, config.AppConfig.DBName)
		if _, err := sqlDB.Exec(createSQL); err != nil {
			return fmt.Errorf("ошибка создания базы данных: %w", err)
		}
		
		logger.GetLogger().Info("✅ База данных успешно создана", zap.String("database", config.AppConfig.DBName))
	} else {
		logger.GetLogger().Info("База данных уже существует", zap.String("database", config.AppConfig.DBName))
	}

	return nil
}

func Connect() {
	logger.GetLogger().Info("Инициализация подключения к базе данных")
	
	if err := createDatabaseIfNotExists(); err != nil {
		if strings.Contains(err.Error(), "не удалось подключиться") {
			logger.GetLogger().Fatal("Ошибка подключения к PostgreSQL. Убедитесь, что PostgreSQL запущен.", zap.Error(err))
		} else {
			logger.GetLogger().Warn("Не удалось создать базу данных автоматически, попытка подключения к существующей", zap.Error(err))
		}
	}

	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=UTC",
		config.AppConfig.DBHost,
		config.AppConfig.DBUser,
		config.AppConfig.DBPassword,
		config.AppConfig.DBName,
		config.AppConfig.DBPort,
	)

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormLogger.Default.LogMode(gormLogger.Info),
	})

	if err != nil {
		logger.GetLogger().Fatal("Ошибка подключения к базе данных", zap.Error(err))
	}

	logger.GetLogger().Info("Подключение к базе данных установлено")

	if err := models.AutoMigrate(DB); err != nil {
		logger.GetLogger().Fatal("Ошибка миграции базы данных", zap.Error(err))
	}

	logger.GetLogger().Info("Миграция базы данных выполнена успешно")
}

