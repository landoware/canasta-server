package room

import (
	"context"
	"strings"
	"sync"
	"time"
)

const (
	reapCheckInterval = 5 * time.Minute
	idleReapAfter     = 30 * time.Minute
	forceReapAfter    = 12 * time.Hour
)

type roomEntry struct {
	room   *Room
	cancel context.CancelFunc
}

// Manager owns the set of live rooms and reaps abandoned ones. Its mutex
// guards only the room map, never game state — each Room serializes its
// own mutation through its actor goroutine.
type Manager struct {
	mu    sync.RWMutex
	rooms map[string]*roomEntry

	ctx    context.Context
	cancel context.CancelFunc
}

// NewManager starts a Manager and its background reaper.
func NewManager() *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	m := &Manager{
		rooms:  make(map[string]*roomEntry),
		ctx:    ctx,
		cancel: cancel,
	}
	go m.reapLoop()
	return m
}

// CreateRoom allocates a room with a unique code and starts its actor.
func (m *Manager) CreateRoom() *Room {
	m.mu.Lock()
	defer m.mu.Unlock()

	var code string
	for {
		code = newRoomCode()
		if _, exists := m.rooms[code]; !exists {
			break
		}
	}

	r := NewRoom(code)
	ctx, cancel := context.WithCancel(m.ctx)
	m.rooms[code] = &roomEntry{room: r, cancel: cancel}
	go r.Run(ctx)
	return r
}

// Get looks up a room by its code. The lookup is case-insensitive since
// codes are meant to be typed by hand.
func (m *Manager) Get(code string) (*Room, bool) {
	code = strings.ToUpper(strings.TrimSpace(code))

	m.mu.RLock()
	defer m.mu.RUnlock()
	e, ok := m.rooms[code]
	if !ok {
		return nil, false
	}
	return e.room, true
}

// Close stops every room's actor goroutine and the reaper. Intended for
// graceful server shutdown.
func (m *Manager) Close() {
	m.cancel()
}

func (m *Manager) reapLoop() {
	ticker := time.NewTicker(reapCheckInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			m.reapOnce()
		case <-m.ctx.Done():
			return
		}
	}
}

func (m *Manager) reapOnce() {
	now := time.Now()

	m.mu.Lock()
	defer m.mu.Unlock()

	for code, e := range m.rooms {
		idle := now.Sub(e.room.LastActivity())
		if idle > forceReapAfter || (idle > idleReapAfter && e.room.ConnectedCount() == 0) {
			e.cancel()
			delete(m.rooms, code)
		}
	}
}
