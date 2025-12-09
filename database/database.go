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

	var count int64
	err = db.Raw("SELECT COUNT(*) FROM pg_database WHERE datname = ?", config.AppConfig.DBName).Scan(&count).Error
	if err != nil {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
		return fmt.Errorf("ошибка проверки существования базы данных: %w", err)
	}

	if count == 0 {
		sqlDB, err := db.DB()
		if err != nil {
			return fmt.Errorf("ошибка получения sql.DB: %w", err)
		}
		defer sqlDB.Close()

		createSQL := fmt.Sprintf(`CREATE DATABASE "%s"`, config.AppConfig.DBName)
		if _, err := sqlDB.Exec(createSQL); err != nil {
			return fmt.Errorf("ошибка создания базы данных: %w", err)
		}
		logger.GetLogger().Info("База данных создана", zap.String("database", config.AppConfig.DBName))
	} else {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}

	return nil
}

func Connect() {
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

