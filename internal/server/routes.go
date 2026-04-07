package server

import (
	// "context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/coder/websocket"
)

func (s *Server) RegisterRoutes() http.Handler {
	mux := http.NewServeMux()

	// Register routes
	mux.HandleFunc("/ws", s.websocketHandler)
	mux.HandleFunc("/new", s.newGameHandler)

	// Wrap the mux with CORS middleware
	return s.corsMiddleware(mux)
}

func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", os.Getenv("CLIENT_URL"))
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-CSRF-Token")
		w.Header().Set("Access-Control-Allow-Credentials", "false") // Set to "true" if credentials are required

		// Handle preflight OPTIONS requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// Proceed with the next handler
		next.ServeHTTP(w, r)
	})
}

func (s *Server) newGameHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	code := newRoomCode()

	s.hub.GetOrCreateRoom(code)

	resp := struct {
		Code string `json:"code"`
	}{Code: code}

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, "{}", http.StatusInternalServerError)
		return
	}
}

func (s *Server) websocketHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: []string{os.Getenv("CLIENT_URL")},
	})
	if err != nil {
		fmt.Print(err)
		return
	}
	defer conn.Close(websocket.StatusGoingAway, "Server closing websocket")
}
