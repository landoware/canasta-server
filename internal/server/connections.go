package server

import (
	"sync"

	"github.com/coder/websocket"
)

type ConnectionManager struct {
	connections map[string]*websocket.Conn
	players     map[string]PlayerConnection
	mu          sync.RWMutex
}

func NewConnectionManager() *ConnectionManager {
	return &ConnectionManager{
		connections: make(map[string]*websocket.Conn),
		players:     make(map[string]PlayerConnection),
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

type PlayerConnection struct {
	GameID   string
	PlayerID int
	Username string
	Token    string
}
