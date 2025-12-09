package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ulule/limiter/v3"
	"github.com/ulule/limiter/v3/drivers/store/memory"
)

func RateLimitMiddleware(rate string) gin.HandlerFunc {
	store := memory.NewStore()
	
	defaultRate := limiter.Rate{
		Period: 1 * time.Minute,
		Limit:  10,
	}
	
	if rate != "" {
		parsed, err := limiter.NewRateFromFormatted(rate)
		if err == nil {
			instance := limiter.New(store, *parsed)
			return createRateLimitHandler(instance)
		}
	}
	
	instance := limiter.New(store, defaultRate)
	return createRateLimitHandler(instance)
}

func createRateLimitHandler(instance *limiter.Limiter) gin.HandlerFunc {

	return func(c *gin.Context) {
		key := c.ClientIP()
		context, err := instance.Get(c, key)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка проверки лимита"})
			c.Abort()
			return
		}

		c.Header("X-RateLimit-Limit", strconv.FormatInt(context.Limit, 10))
		c.Header("X-RateLimit-Remaining", strconv.FormatInt(context.Remaining, 10))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(context.Reset, 10))

		if context.Reached {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Превышен лимит запросов. Попробуйте позже.",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

