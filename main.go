// @title Bekend Backend API
// @version 1.0.0
// @description Бекенд для системы электронной афиши на Go с полной функциональностью авторизации, управления событиями и администрирования.
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8081
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
	var adminCount int64
	database.DB.Model(&models.User{}).Where("role = ?", models.RoleAdmin).Count(&adminCount)

	if adminCount > 0 {
		return
	}

	defaultAdminEmail := "admin@system.local"
	var existingUser models.User
	if err := database.DB.Where("email = ?", defaultAdminEmail).First(&existingUser).Error; err == nil {
		return
	}

	hashedPassword, err := utils.HashPassword("admin123")
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
		logger.GetLogger().Error("Ошибка при создании администратора по умолчанию", zap.Error(err))
		return
	}

	logger.GetLogger().Info("Создан администратор по умолчанию",
		zap.String("email", defaultAdminEmail),
		zap.String("warning", "Не забудьте изменить пароль по умолчанию!"),
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

