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

// AddConnectionWithToken associates a token with a connection
// Used during: initial connection and reconnection
// Returns: old connectionID if token was already connected (device switch scenario)
// Why return old connection: Allows caller to send "disconnected_elsewhere" message
// Note: This method REPLACES the old connection mapping. Caller must handle old connection cleanup.
func (cm *ConnectionManager) AddConnectionWithToken(connectionID string, conn *websocket.Conn, token string) string {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Check if token already connected (device switch scenario)
	// Why check: Enforce single connection per token
	var oldConnectionId string
	for id, player := range cm.players {
		if player.Token == token {
			oldConnectionId = id
			break
		}
	}

	// If old connection exists and it's different from new connection, remove old mapping
	// Why remove: Only one connection per token allowed
	if oldConnectionId != "" && oldConnectionId != connectionID {
		delete(cm.players, oldConnectionId)
		// Note: We keep old connection in connections map so caller can send message to it
		// Caller should call RemoveConnection(oldConnectionId) after sending message
	}

	// Store new connection
	cm.connections[connectionID] = conn
	cm.players[connectionID] = PlayerConnection{
		Token: token,
	}

	// Return old connection ID if found (empty string if no previous connection)
	// Caller should:
	// 1. Get old connection via GetConnection(oldConnectionId)
	// 2. Send "disconnected_elsewhere" message
	// 3. Close old connection
	// 4. Call RemoveConnection(oldConnectionID) to cleanup
	return oldConnectionId
}

func (cm *ConnectionManager) RemoveConnection(id string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	delete(cm.connections, id)
	delete(cm.players, id)
}

// GetConnection returns websocket for connectionID
func (cm *ConnectionManager) GetConnection(connectionID string) *websocket.Conn {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return cm.connections[connectionID]
}

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

func (cm *ConnectionManager) GetTokenByConnection(connectionID string) string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if player, exists := cm.players[connectionID]; exists {
		return player.Token
	}
	return ""
}

// MapToken stores token → connectionID mapping (legacy method)
// Kept for backward compatibility with existing code
// Prefer using AddConnectionWithToken for new code
func (cm *ConnectionManager) MapToken(token, connectionID string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Store in players map
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
