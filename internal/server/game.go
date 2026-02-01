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
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case c := <-r.join:
			r.clients[c.playerID] = c
			r.lastActivity = time.Now()

			// Send snapshot to just this client
			c.sendJSON(ServerMsg{
				T:       "snapshot",
				Version: r.version,
				Data:    r.state.PublicViewFor(c.playerID), // implement projection if needed
			})

			// Notify others (optional)
			r.broadcast(ServerMsg{T: "event", Version: r.version, Data: map[string]any{
				"type":     "player_joined",
				"playerId": c.playerID,
				"name":     c.name,
			}}, nil)

		case c := <-r.leave:
			if _, ok := r.clients[c.playerID]; ok {
				delete(r.clients, c.playerID)
				r.lastActivity = time.Now()
				r.broadcast(ServerMsg{T: "event", Version: r.version, Data: map[string]any{
					"type":     "player_left",
					"playerId": c.playerID,
				}}, nil)
			}

		case in := <-r.in:
			r.lastActivity = time.Now()
			r.handleInbound(in.from, in.msg)

		case <-ticker.C:
			// housekeeping: if empty & idle, consider stop + delete from hub (done externally)
			// also a good place to trigger periodic persistence snapshots

		case <-r.stop:
			// Close all clients gracefully
			for _, c := range r.clients {
				c.close(errors.New("room closed"))
			}
			return
		}
	}
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
