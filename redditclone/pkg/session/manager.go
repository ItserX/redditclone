package session

import (
	"errors"
	"net/http"
	"sync"
	"time"
)

type SessionManager struct {
	data map[string]*Session
	mu   sync.RWMutex
}

var ErrSessionNotFound = errors.New("Session Not Found")

func NewSessionsManager() *SessionManager {
	return &SessionManager{
		data: make(map[string]*Session, 0),
		mu:   sync.RWMutex{},
	}
}

func (manager *SessionManager) CreateSession(w http.ResponseWriter, name, userID string) *Session {
	sess := NewSession(name, userID)

	manager.mu.Lock()
	manager.data[sess.ID] = sess
	manager.mu.Unlock()

	cookie := &http.Cookie{Name: "session_id",
		Value:   sess.ID,
		Expires: time.Now().Add(24 * time.Hour),
		Path:    "/"}

	http.SetCookie(w, cookie)
	return sess
}

func (manager *SessionManager) CheckSession(r *http.Request) (*Session, error) {
	sessID, err := r.Cookie("session_id")
	if err != nil {
		return nil, err
	}

	if sess, ok := manager.data[sessID.Value]; ok {
		return sess, nil
	}

	return nil, ErrSessionNotFound
}

func (manager *SessionManager) DestroySession(w http.ResponseWriter, r *http.Request) error {
	sessID, err := r.Cookie("session_id")
	if err != nil {
		return err
	}
	manager.mu.Lock()
	delete(manager.data, sessID.Value)
	manager.mu.Unlock()

	http.SetCookie(w, &http.Cookie{
		Name:    "session_id",
		Value:   "",
		Path:    "/",
		Expires: time.Now().Add(-1 * time.Hour),
	})

	return nil
}
