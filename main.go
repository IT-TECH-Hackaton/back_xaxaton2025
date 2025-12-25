// @title Bekend Backend API
// @version 1.0.0
// @description Бекенд для системы электронной афиши на Go с полной функциональностью авторизации, управления событиями и администрирования.

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8080
// @BasePath /api
// @schemes http

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Введите 'Bearer {токен}' для авторизации. Токен получается при входе или регистрации.

package main

import (
	_ "bekend/docs"
	"bekend/config"
	"bekend/database"
	"bekend/logger"
	"bekend/models"
	"bekend/routes"
	"bekend/services"
	"bekend/utils"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

func initDefaultAdmin() {
	defaultAdminEmail := "admin@system.local"
	defaultPassword := "Admin123!"
	
	logger.GetLogger().Info("Проверка наличия администратора по умолчанию",
		zap.String("email", defaultAdminEmail),
	)

	var existingUser models.User
	err := database.DB.Where("email = ?", defaultAdminEmail).First(&existingUser).Error
	if err == nil {
		if existingUser.Status == models.UserStatusDeleted {
			logger.GetLogger().Info("Найден удаленный администратор, пересоздание",
				zap.String("email", defaultAdminEmail),
			)
			if err := database.DB.Unscoped().Delete(&existingUser).Error; err != nil {
				logger.GetLogger().Error("Ошибка при удалении старого администратора", zap.Error(err))
			}
		} else {
			logger.GetLogger().Info("Администратор по умолчанию уже существует",
				zap.String("email", defaultAdminEmail),
				zap.String("status", string(existingUser.Status)),
				zap.String("role", string(existingUser.Role)),
			)
			return
		}
	} else {
		logger.GetLogger().Info("Администратор по умолчанию не найден, создание нового",
			zap.String("email", defaultAdminEmail),
		)
	}

	hashedPassword, err := utils.HashPassword(defaultPassword)
	if err != nil {
		logger.GetLogger().Error("Ошибка при хешировании пароля админа", zap.Error(err))
		return
	}

	admin := models.User{
		ID:            uuid.New(),
		FullName:      "Администратор",
		Email:         defaultAdminEmail,
		Password:      hashedPassword,
		Role:          models.RoleAdmin,
		Status:        models.UserStatusActive,
		EmailVerified: true,
		AuthProvider:  "email",
	}

	if err := database.DB.Create(&admin).Error; err != nil {
		logger.GetLogger().Error("Ошибка при создании администратора по умолчанию",
			zap.Error(err),
			zap.String("email", defaultAdminEmail),
		)
		return
	}

	logger.GetLogger().Info("✅ Создан администратор по умолчанию",
		zap.String("email", defaultAdminEmail),
		zap.String("password", defaultPassword),
		zap.String("id", admin.ID.String()),
		zap.String("warning", "⚠️ Не забудьте изменить пароль по умолчанию!"),
	)
}

func main() {
	config.LoadConfig()
	logger.InitLogger(config.AppConfig.AppEnv)
	defer logger.Sync()

	database.Connect()
	initDefaultAdmin()

	cronService := services.NewCronService()
	cronService.Start()

	r := routes.SetupRoutes()

	port := config.AppConfig.AppPort
	logger.GetLogger().Info("Сервер запускается",
		zap.String("port", port),
		zap.String("env", config.AppConfig.AppEnv),
	)

	if err := r.Run(":" + port); err != nil {
		logger.GetLogger().Fatal("Ошибка запуска сервера", zap.Error(err))
	}
}

