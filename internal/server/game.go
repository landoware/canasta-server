package server

import (
	"canasta-server/internal/canasta"
	"math/rand"
	"sync"
	"time"

	"github.com/coder/websocket"
)

type Hub struct {
	mu    sync.RWMutex
	rooms map[string]*Room
}

func NewHub() *Hub {
	return &Hub{
		rooms: make(map[string]*Room),
	}
}

func (h *Hub) GetRoom(code string) (*Room, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	r, ok := h.rooms[code]
	return r, ok
}

func (h *Hub) GetOrCreateRoom(code string) *Room {
	h.mu.Lock()
	defer h.mu.Unlock()

	if r, ok := h.rooms[code]; ok {
		return r
	}

	r := NewRoom(code)
	h.rooms[code] = r
	go r.run()
	return r
}

type Room struct {
	code         string
	clients      map[string]*Client
	game         *canasta.Game
	lastActivity time.Time
}

func NewRoom(code string) *Room {
	return &Room{
		code:    code,
		clients: make(map[string]*Client),
	}
}

func (r *Room) run() {

}

type Client struct {
	conn  *websocket.Conn
	state canasta.ClientState
}

func NewClient(conn *websocket.Conn) *Client {
	return &Client{
		conn: conn,
	}
}

func newRoomCode() string {
	letters := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	b := make([]byte, 4)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
