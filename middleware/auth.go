package middleware

import (
	"net/http"
	"strings"

	"bekend/logger"
	"bekend/utils"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			logger.GetLogger().Warn("Попытка доступа без токена",
				zap.String("ip", c.ClientIP()),
				zap.String("path", c.Request.URL.Path),
			)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Требуется авторизация"})
			c.Abort()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			logger.GetLogger().Warn("Неверный формат токена",
				zap.String("ip", c.ClientIP()),
				zap.String("path", c.Request.URL.Path),
			)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Неверный формат токена"})
			c.Abort()
			return
		}

		claims, err := utils.ValidateToken(parts[1])
		if err != nil {
			logger.GetLogger().Warn("Недействительный токен",
				zap.String("ip", c.ClientIP()),
				zap.String("path", c.Request.URL.Path),
				zap.Error(err),
			)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Недействительный токен"})
			c.Abort()
			return
		}

		c.Set("userID", claims.UserID)
		c.Set("email", claims.Email)
		c.Set("role", claims.Role)
		c.Next()
	}
}

// OptionalAuthMiddleware - middleware, который не требует токена, но если он есть - проверяет его
func OptionalAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			parts := strings.Split(authHeader, " ")
			if len(parts) == 2 && parts[0] == "Bearer" {
				claims, err := utils.ValidateToken(parts[1])
				if err == nil {
					c.Set("userID", claims.UserID)
					c.Set("email", claims.Email)
					c.Set("role", claims.Role)
				} else {
					logger.GetLogger().Debug("Недействительный токен в OptionalAuthMiddleware",
						zap.String("path", c.Request.URL.Path),
						zap.Error(err),
					)
				}
			}
		}
		c.Next()
	}
}

func AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists || role != "Администратор" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Требуются права администратора"})
			c.Abort()
			return
		}
		c.Next()
	}
}

