package storage

import (
	"sync"
	"time"
)

type SessionManager struct {
	storage       *SessionStorage
	cleanupTicker *time.Ticker
	stopChan      chan bool
	sessionTTL    time.Duration
	mu            sync.RWMutex
}

func NewSessionManager(ttl time.Duration) *SessionManager {
	sm := &SessionManager{
		storage:    NewSessionStorage(),
		sessionTTL: ttl,
		stopChan:   make(chan bool),
	}

	sm.cleanupTicker = time.NewTicker(5 * time.Minute)
	go sm.cleanupRoutine()

	return sm
}

func (sm *SessionManager) cleanupRoutine() {
	for {
		select {
		case <-sm.cleanupTicker.C:
			sm.cleanupExpiredSessions()
		case <-sm.stopChan:
			sm.cleanupTicker.Stop()
			return
		}
	}
}

func (sm *SessionManager) cleanupExpiredSessions() {
	sm.storage.mu.Lock()
	defer sm.storage.mu.Unlock()

	now := time.Now()
	for id, session := range sm.storage.Sessions {
		if now.Sub(session.UpdatedAt) > sm.sessionTTL {
			delete(sm.storage.Sessions, id)
		}
	}
}

func (sm *SessionManager) GetStorage() *SessionStorage {
	return sm.storage
}

func (sm *SessionManager) Stop() {
	close(sm.stopChan)
}
