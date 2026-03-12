package utils

import (
	"net/http"

	"bekend/config"
	"github.com/gin-gonic/gin"
)

func SetRefreshCookie(c *gin.Context, token string) {
	maxAge := int(config.AppConfig.JWTRefreshExpiration.Seconds())
	c.SetSameSite(sameSiteMode(config.AppConfig.CookieSameSite))
	c.SetCookie(RefreshCookieName, token, maxAge, "/api", "", config.AppConfig.CookieSecure, true)
}

func ClearRefreshCookie(c *gin.Context) {
	c.SetSameSite(sameSiteMode(config.AppConfig.CookieSameSite))
	c.SetCookie(RefreshCookieName, "", -1, "/api", "", config.AppConfig.CookieSecure, true)
}

func sameSiteMode(s string) http.SameSite {
	switch s {
	case "strict":
		return http.SameSiteStrictMode
	case "none":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteLaxMode
	}
}
