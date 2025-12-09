package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) metricsHandler(c *gin.Context) {
	storage := s.sessionManager.GetStorage()
	mu := storage.GetMu()
	mu.RLock()
	sessionCount := int64(len(storage.Sessions))
	mu.RUnlock()

	globalMetrics.SetActiveSessions(sessionCount)
	metrics := globalMetrics.Get()

	c.JSON(http.StatusOK, metrics)
}
