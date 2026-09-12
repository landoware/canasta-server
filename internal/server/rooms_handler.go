package server

import (
	"encoding/json"
	"net/http"

	"canasta-server/internal/protocol"
)

// handleCreateRoom creates a new room in the Lobby state and returns its
// code — the only thing players need to share with each other. Each
// player then joins by connecting to the websocket endpoint with their own
// name (see handleWebsocket); there are no separate per-seat credentials
// to distribute.
func (s *Server) handleCreateRoom(w http.ResponseWriter, r *http.Request) {
	rm := s.mgr.CreateRoom()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(protocol.CreateRoomResponse{RoomCode: rm.Code})
}
