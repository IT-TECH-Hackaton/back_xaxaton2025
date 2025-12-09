package routes

import (
	"bekend/handlers"
	"bekend/middleware"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func SetupRoutes() *gin.Engine {
	r := gin.Default()

	r.Static("/uploads", "./uploads")

	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"http://localhost:5173"}
	config.AllowCredentials = true
	config.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization"}
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"}
	r.Use(cors.New(config))

	authHandler := handlers.NewAuthHandler()
	userHandler := handlers.NewUserHandler()
	eventHandler := handlers.NewEventHandler()
	adminHandler := handlers.NewAdminHandler()
	uploadHandler := handlers.NewUploadHandler()

	healthHandler := handlers.NewHealthHandler()
	r.GET("/health", healthHandler.HealthCheck)

	api := r.Group("/api")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/verify-email", authHandler.VerifyEmail)
			auth.POST("/resend-code", authHandler.ResendCode)
			auth.POST("/login", authHandler.Login)
			auth.POST("/logout", authHandler.Logout)
			auth.POST("/forgot-password", authHandler.ForgotPassword)
			auth.POST("/reset-password", authHandler.ResetPassword)
			auth.GET("/yandex", authHandler.YandexAuth)
			auth.GET("/yandex/callback", authHandler.YandexCallback)
			auth.POST("/yandex/fake", authHandler.FakeYandexAuth)
		}

		upload := api.Group("/upload")
		upload.Use(middleware.AuthMiddleware())
		{
			upload.POST("/image", uploadHandler.UploadImage)
		}

		user := api.Group("/user")
		user.Use(middleware.AuthMiddleware())
		{
			user.GET("/profile", userHandler.GetProfile)
			user.PUT("/profile", userHandler.UpdateProfile)
		}

		events := api.Group("/events")
		{
			events.GET("", eventHandler.GetEvents)
			events.GET("/:id", eventHandler.GetEvent)
			events.POST("", middleware.AuthMiddleware(), eventHandler.CreateEvent)
			events.PUT("/:id", middleware.AuthMiddleware(), eventHandler.UpdateEvent)
			events.DELETE("/:id", middleware.AuthMiddleware(), eventHandler.DeleteEvent)
			events.POST("/:id/join", middleware.AuthMiddleware(), eventHandler.JoinEvent)
			events.DELETE("/:id/leave", middleware.AuthMiddleware(), eventHandler.LeaveEvent)
			events.GET("/:id/export", middleware.AuthMiddleware(), eventHandler.ExportParticipants)
		}

		reviewHandler := handlers.NewReviewHandler()
		reviews := api.Group("/events/:id/reviews")
		reviews.Use(middleware.AuthMiddleware())
		{
			reviews.GET("", reviewHandler.GetEventReviews)
			reviews.POST("", reviewHandler.CreateReview)
			reviews.PUT("/:reviewId", reviewHandler.UpdateReview)
			reviews.DELETE("/:reviewId", reviewHandler.DeleteReview)
		}

		admin := api.Group("/admin")
		admin.Use(middleware.AuthMiddleware())
		admin.Use(middleware.AdminMiddleware())
		{
			adminUsers := admin.Group("/users")
			{
				adminUsers.GET("", adminHandler.GetUsers)
				adminUsers.GET("/:id", adminHandler.GetUser)
				adminUsers.PUT("/:id", adminHandler.UpdateUser)
				adminUsers.POST("/:id/reset-password", adminHandler.ResetUserPassword)
				adminUsers.DELETE("/:id", adminHandler.DeleteUser)
			}

			adminEvents := admin.Group("/events")
			{
				adminEvents.GET("", adminHandler.GetAdminEvents)
			}
		}
	}

	return r
}

