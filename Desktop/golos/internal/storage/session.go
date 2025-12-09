package storage

import (
	"sync"
	"time"
)

type Session struct {
	ID        string
	Messages  []Message
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Message struct {
	Role    string
	Content string
	Time    time.Time
}

type SessionStorage struct {
	Sessions map[string]*Session
	mu       sync.RWMutex
}

func (s *SessionStorage) GetMu() *sync.RWMutex {
	return &s.mu
}

func NewSessionStorage() *SessionStorage {
	return &SessionStorage{
		Sessions: make(map[string]*Session),
	}
}

func (s *SessionStorage) GetOrCreate(sessionID string) *Session {
	s.mu.Lock()
	defer s.mu.Unlock()

	if sessionID == "" {
		sessionID = generateSessionID()
	}

	session, exists := s.Sessions[sessionID]
	if !exists {
		session = &Session{
			ID:        sessionID,
			Messages:  make([]Message, 0),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		s.Sessions[sessionID] = session
	}

	return session
}

func (s *SessionStorage) AddMessage(sessionID string, role, content string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, exists := s.Sessions[sessionID]
	if !exists {
		return
	}

	session.Messages = append(session.Messages, Message{
		Role:    role,
		Content: content,
		Time:    time.Now(),
	})
	session.UpdatedAt = time.Now()

	if len(session.Messages) > 20 {
		session.Messages = session.Messages[len(session.Messages)-20:]
	}
}

func (s *SessionStorage) GetMessages(sessionID string) []Message {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, exists := s.Sessions[sessionID]
	if !exists {
		return nil
	}

	messages := make([]Message, len(session.Messages))
	copy(messages, session.Messages)
	return messages
}

func (s *SessionStorage) Clear(sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.Sessions, sessionID)
}

func generateSessionID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(8)
}

func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}
