package database

import (
	"fmt"
	"log"

	"golos/internal/config"
	"golos/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func Connect(cfg *config.DatabaseConfig) error {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		cfg.Host, cfg.User, cfg.Password, cfg.DBName, cfg.Port, cfg.SSLMode,
	)

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})

	if err != nil {
		return fmt.Errorf("не удалось подключиться к базе данных: %w", err)
	}

	log.Println("Подключение к базе данных установлено")
	return nil
}

func Migrate() error {
	if DB == nil {
		return fmt.Errorf("база данных не инициализирована")
	}

	err := DB.AutoMigrate(
		&models.User{},
		&models.EmailVerification{},
		&models.PasswordReset{},
		&models.Event{},
		&models.EventParticipant{},
	)

	if err != nil {
		return fmt.Errorf("ошибка миграции: %w", err)
	}

	log.Println("Миграции выполнены успешно")
	return nil
}

func Close() error {
	if DB == nil {
		return nil
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}

	return sqlDB.Close()
}

func GetDB() *gorm.DB {
	return DB
}
