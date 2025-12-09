package routes

import (
	"strings"

	"bekend/config"
	"bekend/handlers"
	"bekend/middleware"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func SetupRoutes() *gin.Engine {
	r := gin.Default()

	r.Static("/uploads", "./uploads")

	corsConfig := cors.DefaultConfig()
	origins := strings.Split(config.AppConfig.CORSAllowOrigins, ",")
	for i := range origins {
		origins[i] = strings.TrimSpace(origins[i])
	}
	corsConfig.AllowOrigins = origins
	corsConfig.AllowCredentials = true
	corsConfig.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization"}
	corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"}
	r.Use(cors.New(corsConfig))

	authHandler := handlers.NewAuthHandler()
	userHandler := handlers.NewUserHandler()
	eventHandler := handlers.NewEventHandler()
	adminHandler := handlers.NewAdminHandler()
	uploadHandler := handlers.NewUploadHandler()

	healthHandler := handlers.NewHealthHandler()
	r.GET("/health", healthHandler.HealthCheck)

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	api := r.Group("/api")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/register", middleware.RateLimitMiddleware("3-H"), authHandler.Register)
			auth.POST("/verify-email", middleware.RateLimitMiddleware("10-M"), authHandler.VerifyEmail)
			auth.POST("/resend-code", middleware.RateLimitMiddleware("3-H"), authHandler.ResendCode)
			auth.POST("/login", middleware.RateLimitMiddleware("5-M"), authHandler.Login)
			auth.POST("/logout", authHandler.Logout)
			auth.POST("/forgot-password", middleware.RateLimitMiddleware("3-H"), authHandler.ForgotPassword)
			auth.POST("/reset-password", middleware.RateLimitMiddleware("5-M"), authHandler.ResetPassword)
			auth.GET("/yandex", middleware.RateLimitMiddleware("10-M"), authHandler.YandexAuth)
			auth.GET("/yandex/callback", authHandler.YandexCallback)
			auth.POST("/yandex/fake", middleware.RateLimitMiddleware("10-M"), authHandler.FakeYandexAuth)
		}

		upload := api.Group("/upload")
		upload.Use(middleware.AuthMiddleware())
		upload.Use(middleware.RateLimitMiddleware("20-H"))
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

		geocoderHandler := handlers.NewGeocoderHandler()
		geocoder := api.Group("/geocoder")
		geocoder.Use(middleware.RateLimitMiddleware("100-H"))
		{
			geocoder.POST("/geocode", geocoderHandler.GeocodeAddress)
			geocoder.POST("/reverse", geocoderHandler.ReverseGeocode)
			geocoder.POST("/map-link", geocoderHandler.GenerateMapLink)
		}

		interestHandler := handlers.NewInterestHandler()
		interests := api.Group("/interests")
		{
			interests.GET("", interestHandler.GetInterests)
			interests.GET("/categories", interestHandler.GetCategories)
			interests.POST("", middleware.AuthMiddleware(), interestHandler.CreateInterest)
			interests.GET("/my", middleware.AuthMiddleware(), interestHandler.GetUserInterests)
			interests.POST("/my", middleware.AuthMiddleware(), interestHandler.AddUserInterest)
			interests.DELETE("/my/:id", middleware.AuthMiddleware(), interestHandler.RemoveUserInterest)
			interests.PUT("/my/:id/weight", middleware.AuthMiddleware(), interestHandler.UpdateUserInterestWeight)
		}

		matchingHandler := handlers.NewMatchingHandler()
		matching := api.Group("/events/:id/matching")
		matching.Use(middleware.AuthMiddleware())
		{
			matching.POST("", matchingHandler.CreateEventMatching)
			matching.GET("", matchingHandler.GetMatches)
			matching.DELETE("", matchingHandler.RemoveEventMatching)
			matching.POST("/request", matchingHandler.CreateMatchRequest)
			matching.GET("/requests", matchingHandler.GetMyMatchRequests)
			matching.POST("/requests/:id/accept", matchingHandler.AcceptMatchRequest)
			matching.POST("/requests/:id/reject", matchingHandler.RejectMatchRequest)
		}

		communityHandler := handlers.NewCommunityHandler()
		communities := api.Group("/communities")
		{
			communities.GET("", communityHandler.GetCommunities)
			communities.GET("/:id", communityHandler.GetCommunity)
			communities.GET("/:id/members", communityHandler.GetCommunityMembers)
			communities.POST("", middleware.AuthMiddleware(), communityHandler.CreateCommunity)
			communities.POST("/:id/join", middleware.AuthMiddleware(), communityHandler.JoinCommunity)
			communities.DELETE("/:id/leave", middleware.AuthMiddleware(), communityHandler.LeaveCommunity)
			communities.GET("/my", middleware.AuthMiddleware(), communityHandler.GetMyCommunities)
		}

		admin := api.Group("/admin")
		admin.Use(middleware.AuthMiddleware())
		admin.Use(middleware.AdminMiddleware())
		{
			adminUsers := admin.Group("/users")
			{
				adminUsers.POST("", adminHandler.CreateUser)
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

		categoryHandler := handlers.NewCategoryHandler()
		adminCategories := admin.Group("/categories")
		{
			adminCategories.GET("", categoryHandler.GetCategories)
			adminCategories.POST("", categoryHandler.CreateCategory)
			adminCategories.PUT("/:id", categoryHandler.UpdateCategory)
			adminCategories.DELETE("/:id", categoryHandler.DeleteCategory)
		}
	}

	categoryHandler := handlers.NewCategoryHandler()
	categories := api.Group("/categories")
	{
		categories.GET("", categoryHandler.GetCategories)
	}
	}

	return r
}

