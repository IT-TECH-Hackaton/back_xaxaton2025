package api

import (
	"golos/internal/service/auth"

	"github.com/gin-gonic/gin"
)

func (s *Server) setupRouter() *gin.Engine {
	router := gin.New()

	router.Use(loggingMiddleware())
	router.Use(recoveryMiddleware())
	router.Use(corsMiddleware())
	router.Use(requestSizeLimitMiddleware(10 * 1024 * 1024))
	router.Use(rateLimitMiddleware())

	api := router.Group("/api/v1")
	{
		api.GET("/health", s.healthHandler)
		api.GET("/metrics", s.metricsHandler)
		api.POST("/voice/process", s.voiceProcessHandler)
		api.POST("/chat/message", s.chatMessageHandler)
		api.DELETE("/session/:id", s.clearSessionHandler)

		authGroup := api.Group("/auth")
		{
			jwtService := auth.NewJWTService(&s.config.JWT)
			authGroup.POST("/login", s.authHandler.Login)
			authGroup.POST("/register", s.authHandler.Register)
			authGroup.POST("/verify-email", s.authHandler.VerifyEmail)
			authGroup.POST("/resend-verification-code", s.authHandler.ResendVerificationCode)
			authGroup.POST("/forgot-password", s.authHandler.ForgotPassword)
			authGroup.POST("/reset-password", s.authHandler.ResetPassword)
			authGroup.POST("/refresh", s.authHandler.RefreshToken)
		}
	}

	router.Static("/static", "./web/static")
	router.LoadHTMLGlob("web/templates/*")
	router.GET("/", s.indexHandler)

	return router
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
