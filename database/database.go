package database

import (
	"fmt"

	"bekend/config"
	"bekend/logger"
	"bekend/models"

	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
)

var DB *gorm.DB

func Connect() {
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

