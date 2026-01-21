package server

import (
	"sync"

	"github.com/coder/websocket"
)

type PlayerConnection struct {
	GameID   string
	PlayerID int
	Username string
	Token    string
}

type ConnectionManager struct {
	connections map[string]*websocket.Conn  // connectionID → socket
	players     map[string]PlayerConnection // connectionID → player info
	tokens      map[string]string           // token → connectionID
	mu          sync.RWMutex
}

func NewConnectionManager() *ConnectionManager {
	return &ConnectionManager{
		connections: make(map[string]*websocket.Conn),
		players:     make(map[string]PlayerConnection),
		tokens:      make(map[string]string),
	}
}

func (cm *ConnectionManager) AddConnection(id string, conn *websocket.Conn) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.connections[id] = conn
}

func (cm *ConnectionManager) RemoveConnection(id string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	delete(cm.connections, id)
	delete(cm.players, id)
}

// MapToken stores token → connectionID mapping
func (cm *ConnectionManager) MapToken(token, connectionID string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Store in players map (already exists)
	if player, exists := cm.players[connectionID]; exists {
		player.Token = token
		cm.players[connectionID] = player
	} else {
		cm.players[connectionID] = PlayerConnection{
			Token: token,
		}
	}
}

// UnmapToken removes token mapping
func (cm *ConnectionManager) UnmapToken(token string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Find and remove
	for connID, player := range cm.players {
		if player.Token == token {
			delete(cm.players, connID)
			break
		}
	}
}

// GetTokenByConnection returns token for a connection
func (cm *ConnectionManager) GetTokenByConnection(connectionID string) string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if player, exists := cm.players[connectionID]; exists {
		return player.Token
	}
	return ""
}

// GetConnectionByToken returns connectionID for a token
func (cm *ConnectionManager) GetConnectionByToken(token string) string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	for connID, player := range cm.players {
		if player.Token == token {
			return connID
		}
	}
	return ""
}

// GetConnection returns websocket for connectionID
func (cm *ConnectionManager) GetConnection(connectionID string) *websocket.Conn {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return cm.connections[connectionID]
}
