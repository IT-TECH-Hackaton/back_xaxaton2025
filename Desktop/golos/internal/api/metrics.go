package api

import (
	"sync"
	"time"
)

type Metrics struct {
	mu                sync.RWMutex
	TotalRequests     int64     `json:"total_requests"`
	SuccessfulRequests int64    `json:"successful_requests"`
	FailedRequests    int64     `json:"failed_requests"`
	GigaChatRequests  int64     `json:"gigachat_requests"`
	STTRequests       int64     `json:"stt_requests"`
	TTSRequests       int64     `json:"tts_requests"`
	ActiveSessions    int64     `json:"active_sessions"`
	StartTime         time.Time `json:"start_time"`
}

var globalMetrics = &Metrics{
	StartTime: time.Now(),
}

func (m *Metrics) IncrementTotal() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.TotalRequests++
}

func (m *Metrics) IncrementSuccess() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.SuccessfulRequests++
}

func (m *Metrics) IncrementFailed() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.FailedRequests++
}

func (m *Metrics) IncrementGigaChat() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.GigaChatRequests++
}

func (m *Metrics) IncrementSTT() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.STTRequests++
}

func (m *Metrics) IncrementTTS() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.TTSRequests++
}

func (m *Metrics) SetActiveSessions(count int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ActiveSessions = count
}

func (m *Metrics) Get() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return map[string]interface{}{
		"total_requests":      m.TotalRequests,
		"successful_requests": m.SuccessfulRequests,
		"failed_requests":     m.FailedRequests,
		"gigachat_requests":   m.GigaChatRequests,
		"stt_requests":        m.STTRequests,
		"tts_requests":        m.TTSRequests,
		"active_sessions":     m.ActiveSessions,
		"start_time":          m.StartTime,
	}
}
