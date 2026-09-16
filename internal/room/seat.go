package room

import "time"

// Seat is one of a room's four fixed player slots. It outlives any single
// websocket connection, so a dropped connection can reconnect (by
// reconnecting with the same name) and resume without losing game state.
type Seat struct {
	Index          int
	Name           string
	Conn           Conn
	Connected      bool
	DisconnectedAt time.Time
	Ready          bool
	IsHost         bool
}
