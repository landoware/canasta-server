package room

import (
	"context"

	"github.com/coder/websocket"
)

// Conn abstracts a single websocket connection so the room actor can be
// tested against an in-memory fake instead of a real socket.
type Conn interface {
	Read(ctx context.Context) ([]byte, error)
	Write(ctx context.Context, data []byte) error
	Close(code websocket.StatusCode, reason string) error
}

// wsConn adapts *websocket.Conn (JSON text frames only) to Conn.
type wsConn struct {
	c *websocket.Conn
}

// NewConn wraps an already-upgraded websocket connection.
func NewConn(c *websocket.Conn) Conn {
	return &wsConn{c: c}
}

func (w *wsConn) Read(ctx context.Context) ([]byte, error) {
	_, data, err := w.c.Read(ctx)
	return data, err
}

func (w *wsConn) Write(ctx context.Context, data []byte) error {
	return w.c.Write(ctx, websocket.MessageText, data)
}

func (w *wsConn) Close(code websocket.StatusCode, reason string) error {
	return w.c.Close(code, reason)
}
