package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/coder/websocket"
	"github.com/google/uuid"
)

func (s *Server) RegisterRoutes() http.Handler {
	mux := http.NewServeMux()

	// Register routes
	mux.HandleFunc("/", s.HelloWorldHandler)

	mux.HandleFunc("/health", s.healthHandler)

	mux.HandleFunc("/websocket", s.websocketHandler)

	// Wrap the mux with CORS middleware
	return s.corsMiddleware(mux)
}

func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*") // Replace "*" with specific origins if needed
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

func (s *Server) HelloWorldHandler(w http.ResponseWriter, r *http.Request) {
	resp := map[string]string{"message": "Hello World"}
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, "Failed to marshal response", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(jsonResp); err != nil {
		log.Printf("Failed to write response: %v", err)
	}
}

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	resp, err := json.Marshal(s.db.Health())
	if err != nil {
		http.Error(w, "Failed to marshal health check response", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(resp); err != nil {
		log.Printf("Failed to write response: %v", err)
	}
}

func (s *Server) websocketHandler(w http.ResponseWriter, r *http.Request) {
	socket, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: []string{"*"}, // TODO: make environment-specific
	})
	if err != nil {
		http.Error(w, "Failed to open websocket", http.StatusInternalServerError)
		return
	}
	defer socket.Close(websocket.StatusGoingAway, "Server closing")

	ctx := r.Context()

	connectionID := uuid.New().String()
	log.Printf("New connection: %s", connectionID)
	s.connectionManager.AddConnection(connectionID, socket)
	defer func() {
		s.connectionManager.RemoveConnection(connectionID)
		log.Printf("Connection closed: %s", connectionID)
	}()

	for {
		// Read from client
		msgType, data, err := socket.Read(ctx)

		if err != nil {
			log.Printf("Connection %s read error: %v", connectionID, err)
			return
		}

		if msgType != websocket.MessageText {
			log.Printf("Non-text input from %s", connectionID)
			continue
		}

		var msg CLientMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			log.Printf("Invalid JSON from %s: %v", connectionID, err)
			s.sendError(socket, ctx, "Invalid JSON")
			continue
		}

		log.Printf("Message Type '%s' from %s", msg.Type, connectionID)

		// Route the message
		switch msg.Type {
		case "ping":
			s.handlePing(socket, ctx, connectionID, msg.Payload)
		// Future:
		// case "create_game":
		// case "join_game":
		// case "execute_move"

		default:
			log.Printf("Unknown message type '%s' from %s", msg.Type, connectionID)
		}
	}

}

func (s *Server) handlePing(socket *websocket.Conn, ctx context.Context, connectionID string, msg json.RawMessage) {
	log.Printf("Ping from %s", connectionID)

	// No payload to parse

	// Pong
	response := ServerMessage{
		Type:    "pong",
		Payload: struct{}{},
	}

	if err := s.sendMessage(socket, ctx, response); err != nil {
		log.Printf("Failed to send pong to %s: %v", connectionID, err)
	}
}

func (s *Server) sendMessage(socket *websocket.Conn, ctx context.Context, msg ServerMessage) any {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("Marshal error: %w", err)
	}

	return socket.Write(ctx, websocket.MessageText, data)
}

func (s *Server) sendError(socket *websocket.Conn, ctx context.Context, msg string) {
	response := ServerMessage{
		Type: "error",
		Payload: ErrorMessage{
			Message: msg,
		},
	}

	if err := s.sendMessage(socket, ctx, response); err != nil {
		log.Printf("Failed to send error message: %v", err)
	}
}
