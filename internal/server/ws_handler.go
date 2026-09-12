package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/coder/websocket"

	"canasta-server/internal/protocol"
	"canasta-server/internal/room"
)

// handleWebsocket resolves the name query parameter to a seat, then
// upgrades the connection and attaches it to that seat. The name is
// resolved before upgrading so a rejected join (missing name, full room)
// gets a plain HTTP error rather than a socket that opens and immediately
// closes. The handler blocks for the lifetime of the connection, running
// the read loop itself.
func (s *Server) handleWebsocket(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	name := r.URL.Query().Get("name")

	rm, ok := s.mgr.Get(code)
	if !ok {
		http.Error(w, "room not found", http.StatusNotFound)
		return
	}

	seatIdx, err := rm.Join(name)
	if err != nil {
		switch {
		case errors.Is(err, room.ErrNameRequired):
			http.Error(w, err.Error(), http.StatusBadRequest)
		case errors.Is(err, room.ErrRoomFull):
			http.Error(w, err.Error(), http.StatusConflict)
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	acceptOpts := &websocket.AcceptOptions{}
	if s.allowedOrigin != "" {
		acceptOpts.OriginPatterns = []string{s.allowedOrigin}
	} else {
		acceptOpts.InsecureSkipVerify = true // local development only
	}

	c, err := websocket.Accept(w, r, acceptOpts)
	if err != nil {
		return
	}

	conn := room.NewConn(c)
	rm.Attach(seatIdx, conn)

	readLoop(r.Context(), rm, seatIdx, conn)
}

func readLoop(ctx context.Context, rm *room.Room, seatIdx int, conn room.Conn) {
	defer rm.Disconnect(seatIdx, conn)
	defer conn.Close(websocket.StatusNormalClosure, "")

	for {
		data, err := conn.Read(ctx)
		if err != nil {
			return
		}

		var msg protocol.ClientMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			// Malformed frame: nothing to dispatch and no seat-specific
			// error channel exists outside the room actor, so drop it.
			continue
		}

		rm.Submit(seatIdx, msg)
	}
}
