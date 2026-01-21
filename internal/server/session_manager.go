package server

import (
	"errors"
	"sync"
)

type SessionInfo struct {
	Token    string
	RoomCode string
	PlayerID int
	Username string
}

type SessionManager struct {
	sessions map[string]SessionInfo // Token -> SessionInfo
	mu       sync.RWMutex
}

func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[string]SessionInfo),
	}
}

func (sm *SessionManager) StoreSession(info SessionInfo) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.sessions[info.Token] = info
}

func (sm *SessionManager) GetSession(token string) (SessionInfo, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	session, exists := sm.sessions[token]
	if !exists {
		return SessionInfo{}, errors.New("TOKEN_NOT_FOUND: Invalid session token")
	}

	return session, nil
}

// Used for players who intentionally leave
func (sm *SessionManager) RemoveSession(token string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.sessions, token)
}

func (sm *SessionManager) GetAllSessions() []SessionInfo {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	sessions := make([]SessionInfo, 0, len(sm.sessions))
	for _, session := range sm.sessions {
		sessions = append(sessions, session)
	}

	return sessions
}
