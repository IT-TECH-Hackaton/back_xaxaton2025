package api

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type HealthStatus struct {
	Status      string                 `json:"status"`
	Service     string                 `json:"service"`
	Version     string                 `json:"version"`
	Uptime      string                 `json:"uptime"`
	Timestamp   string                 `json:"timestamp"`
	Dependencies HealthDependencies   `json:"dependencies"`
}

type HealthDependencies struct {
	GigaChat    DependencyStatus `json:"gigachat"`
	AudioService DependencyStatus `json:"audio_service"`
	Database    DependencyStatus `json:"database"`
}

type DependencyStatus struct {
	Status      string `json:"status"`
	ResponseTime string `json:"response_time,omitempty"`
	Error       string `json:"error,omitempty"`
}

var startTime = time.Now()

func (s *Server) healthHandler(c *gin.Context) {
	status := HealthStatus{
		Status:    "ok",
		Service:   "golos-api",
		Version:   "1.0.0",
		Uptime:    time.Since(startTime).String(),
		Timestamp: time.Now().Format(time.RFC3339),
		Dependencies: HealthDependencies{
			GigaChat:     s.checkGigaChat(),
			AudioService: s.checkAudioService(),
			Database:     DependencyStatus{Status: "ok"},
		},
	}

	overallStatus := http.StatusOK
	if status.Dependencies.GigaChat.Status == "down" || 
	   status.Dependencies.AudioService.Status == "down" {
		status.Status = "degraded"
		overallStatus = http.StatusServiceUnavailable
	}

	c.JSON(overallStatus, status)
}

func (s *Server) checkGigaChat() DependencyStatus {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	start := time.Now()
	
	if s.config.GigaChat.ClientID == "" && s.config.GigaChat.AuthorizationKey == "" {
		return DependencyStatus{
			Status: "down",
			Error:  "credentials not configured",
		}
	}

	select {
	case <-ctx.Done():
		return DependencyStatus{
			Status:      "down",
			ResponseTime: time.Since(start).String(),
			Error:       "timeout",
		}
	default:
		return DependencyStatus{
			Status:      "ok",
			ResponseTime: time.Since(start).String(),
		}
	}
}

func (s *Server) checkAudioService() DependencyStatus {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, "GET", s.config.AudioService.URL+"/health", nil)
	if err != nil {
		return DependencyStatus{
			Status:      "down",
			ResponseTime: time.Since(start).String(),
			Error:       err.Error(),
		}
	}

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return DependencyStatus{
			Status:      "down",
			ResponseTime: time.Since(start).String(),
			Error:       err.Error(),
		}
	}
	defer resp.Body.Close()

	responseTime := time.Since(start).String()

	if resp.StatusCode != http.StatusOK {
		return DependencyStatus{
			Status:      "down",
			ResponseTime: responseTime,
			Error:       "non-200 status code",
		}
	}

	return DependencyStatus{
		Status:      "ok",
		ResponseTime: responseTime,
	}
}
