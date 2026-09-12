// Package server wires internal/room's Manager to net/http: it exposes the
// HTTP endpoint to create a room and the websocket endpoint to join one.
package server

import (
	"net/http"
	"os"
	"time"

	"canasta-server/internal/room"
)

// Server holds the dependencies shared by the HTTP handlers.
type Server struct {
	mgr           *room.Manager
	allowedOrigin string
}

// New builds a Server backed by mgr. allowedOrigin is the CORS/websocket
// origin to accept (from the CLIENT_URL env var); an empty string allows
// any origin, which is only appropriate for local development.
func New(mgr *room.Manager, allowedOrigin string) *Server {
	return &Server{mgr: mgr, allowedOrigin: allowedOrigin}
}

// Routes builds the HTTP handler for the whole server.
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /rooms", s.handleCreateRoom)
	mux.HandleFunc("GET /rooms/{code}/ws", s.handleWebsocket)
	return s.withCORS(mux)
}

// NewHTTPServer builds a *http.Server configured from PORT/CLIENT_URL env
// vars, matching the timeouts used by the project's prior scaffolding.
func NewHTTPServer(mgr *room.Manager) *http.Server {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := New(mgr, os.Getenv("CLIENT_URL"))

	return &http.Server{
		Addr:         ":" + port,
		Handler:      srv.Routes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
}

func (s *Server) withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.allowedOrigin != "" {
			w.Header().Set("Access-Control-Allow-Origin", s.allowedOrigin)
		} else {
			// No CLIENT_URL configured: allow any origin. Intended for
			// local development only (see New's doc comment); staging and
			// production must always set CLIENT_URL to their one real
			// client origin.
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
