package server

import (
	"canasta-server/internal/canasta"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

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
		// Environment-based CORS configuration
		// Why environment-based: Development needs flexible CORS, production should be restrictive
		// Why specific domain: Prevents unauthorized sites from using our API
		origin := s.getAllowedOrigin()

		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-CSRF-Token")
		w.Header().Set("Access-Control-Allow-Credentials", "false")

		// Handle preflight OPTIONS requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// Proceed with the next handler
		next.ServeHTTP(w, r)
	})
}

// getAllowedOrigin returns the CORS origin based on environment
// Why: Centralize CORS logic for easy testing and modification
func (s *Server) getAllowedOrigin() string {
	env := os.Getenv("ENVIRONMENT")

	// Production: Only allow specific domain
	if env == "production" {
		return "https://landoware.com/canasta"
	}

	// Development/Test: Allow all origins
	// Why wildcard in dev: Easier local development with different ports
	return "*"
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
	// Environment-based WebSocket origin patterns
	// Why match CORS: Consistent security policy
	originPatterns := []string{"*"}
	if os.Getenv("ENVIRONMENT") == "production" {
		originPatterns = []string{"landoware.com"}
	}

	socket, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: originPatterns,
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

	// Start heartbeat goroutine
	// Why: Detect dead connections by sending periodic pings
	// Why 30 seconds: Balance between quick detection and not overwhelming with traffic
	heartbeatDone := make(chan struct{})
	go s.heartbeatLoop(ctx, socket, connectionID, heartbeatDone)

	defer func() {
		// Stop heartbeat
		close(heartbeatDone)
		token := s.connectionManager.GetTokenByConnection(connectionID)

		// Remove connection
		s.connectionManager.RemoveConnection(connectionID)

		// Clean up rate limiter and health tracker
		// Why: Prevent memory leaks from disconnected connections
		s.rateLimiter.RemoveConnection(connectionID)
		s.connectionHealth.RemoveConnection(connectionID)

		log.Printf("Connection closed: %s", connectionID)

		// If player had a token, mark as disconnected
		if token != "" {
			gamePaused, game, playerID, err := s.gameManager.MarkPlayerDisconnected(token)
			if err != nil {
				// This can happen if player left via leave_game before disconnect
				// It's not an error, just log at debug level
				if err.Error() != "TOKEN_NOT_FOUND: Invalid session token" {
					log.Printf("Error marking player disconnected: %v", err)
				}
				return
			}

			log.Printf("Player %d (%s) disconnected from game %s",
				playerID, game.Players[playerID].Username, game.RoomCode)

			// Persist disconnect state
			// Why: If server crashes after disconnect but before periodic save, game would reload incorrectly
			if err := s.persistenceManager.SaveGame(game); err != nil {
				log.Printf("Failed to persist game %s after disconnect: %v", game.RoomCode, err)
			}

			// Broadcast disconnect notification
			s.broadcastToLobby(game, "player_disconnected", PlayerStatusNotification{
				PlayerID:  playerID,
				Username:  game.Players[playerID].Username,
				Connected: false,
			})

			// If game was paused, broadcast that
			if gamePaused {
				s.broadcastToLobby(game, "game_paused", GamePausedNotification{
					Message: fmt.Sprintf("%s disconnected. Game paused.",
						game.Players[playerID].Username),
				})
			}
		}
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

		var msg ClientMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			log.Printf("Invalid JSON from %s: %v", connectionID, err)
			s.sendError(socket, ctx, "Invalid JSON")
			continue
		}

		// Update activity tracking
		// Why: Track last message time for connection health monitoring
		s.connectionHealth.UpdateActivity(connectionID)

		// Check rate limit
		// Why: Prevent abuse - limit to 10 messages per second per connection
		// Why here: After JSON parse but before processing (don't count invalid messages against limit)
		if !s.rateLimiter.Allow(connectionID) {
			log.Printf("Rate limit exceeded for connection %s", connectionID)
			s.sendError(socket, ctx, "RATE_LIMIT_EXCEEDED: Too many messages, please slow down")
			continue
		}

		log.Printf("Message Type '%s' from %s", msg.Type, connectionID)

		// Route the message
		switch msg.Type {
		case "ping":
			s.handlePing(socket, ctx, connectionID, msg.Payload)

		case "create_game":
			s.handleCreateGame(socket, ctx, connectionID, msg.Payload)

		case "join_game":
			s.handleJoinGame(socket, ctx, connectionID, msg.Payload)

		case "reconnect":
			s.handleReconnect(socket, ctx, connectionID, msg.Payload)

		case "set_ready":
			s.handleSetReady(socket, ctx, connectionID, msg.Payload)

		case "update_team_order":
			s.handleUpdateTeamOrder(socket, ctx, connectionID, msg.Payload)

		case "leave_game":
			s.handleLeaveGame(socket, ctx, connectionID, msg.Payload)

		case "execute_move":
			s.handleExecuteMove(socket, ctx, connectionID, msg.Payload)
		default:
			log.Printf("Unknown message type '%s' from %s", msg.Type, connectionID)
			s.sendError(socket, ctx, fmt.Sprintf("Unknown message type: %s", msg.Type))
		}
	}
}

// heartbeatLoop sends periodic pings to detect dead connections
// Why separate goroutine: Don't block message processing
// Why use context: Gracefully stop when connection closes
func (s *Server) heartbeatLoop(ctx context.Context, socket *websocket.Conn, connectionID string, done chan struct{}) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			// Connection closing, stop heartbeat
			return
		case <-ctx.Done():
			// Context cancelled, stop heartbeat
			return
		case <-ticker.C:
			// Send ping using websocket protocol (not application message)
			// Why websocket ping: More efficient than application-level ping
			// The library will automatically handle pong responses
			if err := socket.Ping(ctx); err != nil {
				log.Printf("Heartbeat ping failed for %s: %v", connectionID, err)
				return
			}
			log.Printf("Heartbeat ping sent to %s", connectionID)
		}
	}
}

// checkInactiveConnections runs periodically to close connections that haven't responded
// Why: Detect zombie connections that don't respond to heartbeats
func (s *Server) checkInactiveConnections() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		// Find connections inactive for > 5 minutes
		inactive := s.connectionHealth.GetInactiveConnections(5 * time.Minute)

		if len(inactive) > 0 {
			log.Printf("Found %d inactive connections, closing them", len(inactive))
		}

		for _, connID := range inactive {
			// Get the connection
			conn := s.connectionManager.GetConnection(connID)
			if conn != nil {
				// Close with timeout status
				// Why StatusGoingAway: Indicates server-initiated close due to inactivity
				conn.Close(websocket.StatusGoingAway, "Connection inactive")
				log.Printf("Closed inactive connection: %s", connID)
			}
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

func (s *Server) broadcastToLobby(game *ActiveGame, messageType string, payload interface{}) {
	for _, slot := range game.Players {
		if slot.Token == "" {
			continue // Empty slot
		}

		// Find connection for this token
		connID := s.connectionManager.GetConnectionByToken(slot.Token)
		if connID == "" {
			continue // Player not connected
		}

		conn := s.connectionManager.GetConnection(connID)
		if conn == nil {
			continue
		}

		// Send message
		msg := ServerMessage{
			Type:    messageType,
			Payload: payload,
		}
		// Use background context for broadcasts
		s.sendMessage(conn, context.Background(), msg)
	}
}

func (s *Server) handleCreateGame(socket *websocket.Conn, ctx context.Context, connectionID string, payload json.RawMessage) {
	// Parse request
	var req CreateGameRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		s.sendError(socket, ctx, "Invalid create_game payload")
		return
	}

	// Call game manager
	game, token, err := s.gameManager.CreateGame(req.Username, req.RandomTeamOrder)
	if err != nil {
		s.sendError(socket, ctx, err.Error())
		return
	}

	// Store session and token mapping
	session := SessionInfo{
		Token:    token,
		RoomCode: game.RoomCode,
		PlayerID: 0,
		Username: req.Username,
	}
	s.sessionManager.StoreSession(session)
	s.connectionManager.AddConnectionWithToken(connectionID, socket, token)

	// Persist to database
	if err := s.persistenceManager.SaveGame(game); err != nil {
		log.Printf("Failed to persist game %s: %v", game.RoomCode, err)
	}
	if err := s.persistenceManager.SaveSession(session); err != nil {
		log.Printf("Failed to persist session %s: %v", token, err)
	}
	if err := s.persistenceManager.SaveRoomCode(game.RoomCode, true); err != nil {
		log.Printf("Failed to persist room code %s: %v", game.RoomCode, err)
	}

	// Step 4: Send response to creator
	response := ServerMessage{
		Type: "game_created",
		Payload: CreateGameResponse{
			RoomCode: game.RoomCode,
			Token:    token,
			PlayerID: 0,
		},
	}
	if err := s.sendMessage(socket, ctx, response); err != nil {
		log.Printf("Failed to send game_created: %v", err)
		return
	}

	// Step 5: Broadcast lobby state
	// Why: Creator should see initial lobby state
	s.broadcastLobbyUpdate(game)
}

func (s *Server) handleJoinGame(socket *websocket.Conn, ctx context.Context, connectionID string, payload json.RawMessage) {
	// Parse request
	var req JoinGameRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		s.sendError(socket, ctx, "Invalid join_game payload")
		return
	}

	// Call game manager
	game, token, slotID, err := s.gameManager.JoinGame(req.RoomCode, req.Username)
	if err != nil {
		s.sendError(socket, ctx, err.Error())
		return
	}

	// Store token mapping
	session := SessionInfo{
		Token:    token,
		RoomCode: game.RoomCode,
		PlayerID: slotID,
		Username: req.Username,
	}
	s.sessionManager.StoreSession(session)
	s.connectionManager.AddConnectionWithToken(connectionID, socket, token)

	// Persist to database
	if err := s.persistenceManager.SaveGame(game); err != nil {
		log.Printf("Failed to persist game %s: %v", game.RoomCode, err)
	}
	if err := s.persistenceManager.SaveSession(session); err != nil {
		log.Printf("Failed to persist session %s: %v", token, err)
	}

	// Send response to joiner
	response := ServerMessage{
		Type: "game_joined",
		Payload: JoinGameResponse{
			Success:  true,
			Token:    token,
			PlayerID: slotID,
		},
	}
	if err := s.sendMessage(socket, ctx, response); err != nil {
		log.Printf("Failed to send game_joined: %v", err)
		return
	}

	// Broadcast lobby state to ALL players
	s.broadcastLobbyUpdate(game)
}

func (s *Server) handleReconnect(socket *websocket.Conn, ctx context.Context, connectionID string, payload json.RawMessage) {
	// Parse payload
	var req ReconnectRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		s.sendError(socket, ctx, "Invalid reconnect payload")
		return
	}

	// Validate session
	session, err := s.sessionManager.GetSession(req.Token)
	if err != nil {
		s.sendError(socket, ctx, err.Error())
		return
	}

	// Has this token already connected?
	oldConnectionID := s.connectionManager.AddConnectionWithToken(connectionID, socket, req.Token)

	if oldConnectionID != "" && oldConnectionID != connectionID {
		// Disconnect the old token
		oldConn := s.connectionManager.GetConnection(oldConnectionID)
		if oldConn != nil {
			s.sendMessage(oldConn, context.Background(), ServerMessage{
				Type: "disconnected_elsewhere",
				Payload: struct {
					Message string `json:"message"`
				}{
					Message: "You connected on another device",
				},
			})
			oldConn.Close(websocket.StatusNormalClosure, "Connected from another device")
		}
		s.connectionManager.RemoveConnection(oldConnectionID)
	}

	// Reconnect in gameManager
	game, err := s.gameManager.ReconnectPlayer(req.Token, session.RoomCode, session.PlayerID)
	if err != nil {
		s.sendError(socket, ctx, err.Error())
		return
	}

	// Persist reconnection state
	// Why: If server crashes after reconnect but before periodic save, game would reload as paused
	if err := s.persistenceManager.SaveGame(game); err != nil {
		log.Printf("Failed to persist game %s after reconnect: %v", game.RoomCode, err)
	}

	// Respond to the player
	response := ServerMessage{
		Type: "reconnected",
		Payload: ReconnectResponse{
			Success:  true,
			RoomCode: session.RoomCode,
			PlayerID: session.PlayerID,
			Message:  "Successfully reconnected",
		},
	}
	if err := s.sendMessage(socket, ctx, response); err != nil {
		log.Printf("Failed to send reconnected response: %v", err)
	}

	// Broadcast to others
	s.broadcastToLobby(game, "player_reconnected", PlayerStatusNotification{
		PlayerID:  session.PlayerID,
		Username:  session.Username,
		Connected: true,
	})

	// If game resumed, broadcast that too
	if game.Status == StatusPlaying {
		s.broadcastToLobby(game, "game_resumed", struct {
			Message string `json:"message"`
		}{
			Message: "Game resumed!",
		})
	}

	// Send current state to reconnected player
	// Why: Player needs to see current game state after reconnecting
	if game.Status == StatusPlaying || game.Status == StatusPaused {
		// Send game state for active games
		state := s.buildGameStateMessage(game, session.PlayerID)
		s.sendMessage(socket, ctx, ServerMessage{
			Type:    "game_state",
			Payload: state,
		})
	} else if game.Status == StatusLobby {
		// Send lobby state if still in lobby
		lobbyState := s.buildLobbyState(game, req.Token)
		s.sendMessage(socket, ctx, ServerMessage{
			Type:    "lobby_update",
			Payload: lobbyState,
		})
	}
}

func (s *Server) handleSetReady(socket *websocket.Conn, ctx context.Context, connectionID string, payload json.RawMessage) {
	// Why we need this handler:
	// - Player marks themselves ready/unready
	// - When all 4 ready, auto-start game

	// Step 1: Parse request
	var req SetReadyRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		s.sendError(socket, ctx, "Invalid set_ready payload")
		return
	}

	// Step 2: Get player's token
	token := s.connectionManager.GetTokenByConnection(connectionID)
	if token == "" {
		s.sendError(socket, ctx, "NOT_IN_GAME: No active game session")
		return
	}

	// Step 3: Get game by token
	game, _, err := s.gameManager.GetGameByToken(token)
	if err != nil {
		s.sendError(socket, ctx, err.Error())
		return
	}

	// Step 4: Set ready state
	game, allReady, err := s.gameManager.SetReady(game.RoomCode, token, req.Ready)
	if err != nil {
		s.sendError(socket, ctx, err.Error())
		return
	}

	// Step 5: Persist ready state change
	if err := s.persistenceManager.SaveGame(game); err != nil {
		log.Printf("Failed to persist game %s: %v", game.RoomCode, err)
	}

	// Step 6: Broadcast lobby update
	s.broadcastLobbyUpdate(game)

	// Step 7: If all ready, start game!
	if allReady {
		if err := s.gameManager.StartGame(game.RoomCode); err != nil {
			log.Printf("Failed to start game: %v", err)
			return
		}

		// Persist game start
		if err := s.persistenceManager.SaveGame(game); err != nil {
			log.Printf("Failed to persist game %s after start: %v", game.RoomCode, err)
		}

		// Broadcast game started notification
		s.broadcastToLobby(game, "game_started", GameStartedNotification{
			Message: "Game is starting! Get ready to play.",
		})

		// Broadcast initial game state to all players
		// Why: Players need to see their starting hands and game state
		// Why after notification: Notification tells UI game is starting, state shows the cards
		s.broadcastGameState(game)
	}
}

func (s *Server) handleUpdateTeamOrder(socket *websocket.Conn, ctx context.Context, connectionID string, payload json.RawMessage) {
	// Why we need this handler:
	// - Room creator rearranges player positions
	// - Only creator (slot 0) can do this

	// Step 1: Parse request
	var req UpdateTeamOrderRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		s.sendError(socket, ctx, "Invalid update_team_order payload")
		return
	}

	// Step 2: Get player's token
	token := s.connectionManager.GetTokenByConnection(connectionID)
	if token == "" {
		s.sendError(socket, ctx, "NOT_IN_GAME: No active game session")
		return
	}

	// Step 3: Get game by token
	game, _, err := s.gameManager.GetGameByToken(token)
	if err != nil {
		s.sendError(socket, ctx, err.Error())
		return
	}

	// Step 4: Update team order (permission check inside)
	game, err = s.gameManager.UpdateTeamOrder(game.RoomCode, token, req.PlayerOrder)
	if err != nil {
		s.sendError(socket, ctx, err.Error())
		return
	}

	// Step 5: Persist team order change
	if err := s.persistenceManager.SaveGame(game); err != nil {
		log.Printf("Failed to persist game %s: %v", game.RoomCode, err)
	}

	// Step 6: Broadcast lobby update
	// Why: Everyone sees new arrangement
	s.broadcastLobbyUpdate(game)
}

func (s *Server) handleLeaveGame(socket *websocket.Conn, ctx context.Context, connectionID string, payload json.RawMessage) {
	// Get player's token
	token := s.connectionManager.GetTokenByConnection(connectionID)
	if token == "" {
		s.sendError(socket, ctx, "NOT_IN_GAME: No active game session")
		return
	}

	// Get game by token
	game, _, err := s.gameManager.GetGameByToken(token)
	if err != nil {
		s.sendError(socket, ctx, err.Error())
		return
	}

	// Leave game
	game, err = s.gameManager.LeaveGame(game.RoomCode, token)
	if err != nil {
		s.sendError(socket, ctx, err.Error())
		return
	}

	// Persist leave (game state changed) and remove session
	if err := s.persistenceManager.SaveGame(game); err != nil {
		log.Printf("Failed to persist game %s: %v", game.RoomCode, err)
	}
	if err := s.persistenceManager.DeleteSession(token); err != nil {
		log.Printf("Failed to delete session %s: %v", token, err)
	}

	// Broadcast lobby update
	s.broadcastLobbyUpdate(game)
}

// broadcastLobbyUpdate sends personalized lobby state to all players
func (s *Server) broadcastLobbyUpdate(game *ActiveGame) {
	for _, slot := range game.Players {
		if slot.Token == "" || !slot.Connected {
			continue
		}

		// Build personalized state
		lobbyState := s.buildLobbyState(game, slot.Token)

		// Find connection
		connID := s.connectionManager.GetConnectionByToken(slot.Token)
		if connID == "" {
			continue
		}

		conn := s.connectionManager.GetConnection(connID)
		if conn == nil {
			continue
		}

		// Send
		msg := ServerMessage{
			Type:    "lobby_update",
			Payload: lobbyState,
		}

		// Use background context for broadcasts
		if err := s.sendMessage(conn, context.Background(), msg); err != nil {
			log.Printf("Failed to broadcast to %s: %v", slot.Username, err)
		}
	}
}

// buildLobbyState creates personalized lobby state for a specific player
func (s *Server) buildLobbyState(game *ActiveGame, forToken string) LobbyState {
	players := [4]LobbyPlayer{}
	playerCount := 0

	for i, slot := range game.Players {
		if slot.Username == "" {
			players[i] = LobbyPlayer{} // Empty slot
			continue
		}

		playerCount++
		players[i] = LobbyPlayer{
			Username:  slot.Username,
			Ready:     slot.Ready,
			Connected: slot.Connected,
			IsYou:     slot.Token == forToken,
		}
	}

	// Check if all ready
	allReady := true
	if playerCount < 4 {
		allReady = false
	} else {
		for _, slot := range game.Players {
			if slot.Username != "" && !slot.Ready {
				allReady = false
				break
			}
		}
	}

	return LobbyState{
		RoomCode:        game.RoomCode,
		Players:         players,
		PlayerCount:     playerCount,
		RandomTeamOrder: game.Config.RandomTeamOrder,
		Status:          string(game.Status),
		AllReady:        allReady,
	}
}

// broadcastGameState sends personalized game state to all connected players
// Why: Each player needs to see their own hand, but only hand counts for others
// Why personalized: ClientState.GetClientState(playerID) returns player-specific view
// Why skip disconnected: No point sending to players who aren't connected
func (s *Server) broadcastGameState(game *ActiveGame) {
	// Don't broadcast if game not started yet
	if game.Game == nil {
		log.Printf("Warning: Attempted to broadcast game state before game started")
		return
	}

	for i, slot := range game.Players {
		// Skip empty slots
		if slot.Token == "" {
			continue
		}

		// Skip disconnected players
		if !slot.Connected {
			continue
		}

		// Build personalized state for this player
		state := s.buildGameStateMessage(game, i)

		// Find connection
		connID := s.connectionManager.GetConnectionByToken(slot.Token)
		if connID == "" {
			continue
		}

		conn := s.connectionManager.GetConnection(connID)
		if conn == nil {
			continue
		}

		// Send with background context (non-blocking)
		msg := ServerMessage{
			Type:    "game_state",
			Payload: state,
		}

		if err := s.sendMessage(conn, context.Background(), msg); err != nil {
			log.Printf("Failed to broadcast game state to %s: %v", slot.Username, err)
		}
	}
}

// buildGameStateMessage creates personalized game state for a specific player
// Why separate from broadcast: Makes testing easier, cleaner separation of concerns
// Why playerID parameter: Each player gets different ClientState (shows their hand)
func (s *Server) buildGameStateMessage(game *ActiveGame, playerID int) GameStateMessage {
	// Safety check: game must be started
	if game.Game == nil {
		return GameStateMessage{
			Status: string(game.Status),
		}
	}

	// Get personalized client state from canasta package
	// Why GetClientState: Handles card visibility, team info, etc.
	clientState := game.Game.GetClientState(playerID)

	return GameStateMessage{
		State:         clientState,
		CurrentPlayer: game.Game.CurrentPlayer,
		Phase:         string(game.Game.Phase),
		Status:        string(game.Status),
	}
}

// handleExecuteMove processes game moves from players
// Why in routes.go: Follows same pattern as other handlers, keeps all WebSocket logic together
func (s *Server) handleExecuteMove(socket *websocket.Conn, ctx context.Context, connectionID string, payload json.RawMessage) {
	// Step 1: Parse request
	var req MoveRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		s.sendError(socket, ctx, "INVALID_PAYLOAD: Invalid move request")
		return
	}

	// Step 2: Get player's token
	token := s.connectionManager.GetTokenByConnection(connectionID)
	if token == "" {
		s.sendError(socket, ctx, "NOT_IN_GAME: No active game session")
		return
	}

	// Step 3: Get game by token
	game, playerID, err := s.gameManager.GetGameByToken(token)
	if err != nil {
		s.sendError(socket, ctx, err.Error())
		return
	}

	// Step 4: Validate game status
	// Why: Can't make moves in lobby, paused, or completed games
	if game.Status != StatusPlaying {
		if game.Status == StatusLobby {
			s.sendError(socket, ctx, "GAME_NOT_STARTED: Game hasn't started yet")
			return
		} else if game.Status == StatusPaused {
			s.sendError(socket, ctx, "GAME_PAUSED: Game is paused due to disconnection")
			return
		} else if game.Status == StatusCompleted {
			s.sendError(socket, ctx, "GAME_COMPLETED: Game has ended")
			return
		}
	}

	// Step 5: Lock game state for move execution
	// Why: Prevent concurrent moves from corrupting game state
	// Why here: After all validation but before any state changes
	// Critical section: From capturing state through move execution
	game.mu.Lock()
	defer game.mu.Unlock()

	// Step 6: Capture state before move for hand end detection
	// Why inside lock: Ensure we see consistent state
	previousHandNumber := game.Game.HandNumber

	// Step 7: Build and execute move
	// Why construct here: We need to add playerID from server context
	move := canasta.Move{
		PlayerId: playerID,
		Type:     canasta.MoveType(req.Type),
		Ids:      req.Ids,
		Id:       req.Id,
	}

	response := game.Game.ExecuteMove(move)

	// Step 8: Handle move failure
	if !response.Success {
		s.sendMessage(socket, ctx, ServerMessage{
			Type: "move_result",
			Payload: MoveResultResponse{
				Success: false,
				Message: response.Message,
			},
		})
		return
	}

	// Step 9: Move succeeded - update timestamp
	game.UpdatedAt = time.Now()

	// Step 9a: Persist game state after move
	// Why here: After move succeeds but before broadcasting
	// If persistence fails, we still broadcast (prioritize gameplay over persistence)
	if err := s.persistenceManager.SaveGame(game); err != nil {
		log.Printf("Failed to persist game %s after move: %v", game.RoomCode, err)
	}

	// Step 9b: Handle special move broadcasting (partner permissions)
	// Why: Some moves require notifying specific players beyond normal game state
	if req.Type == "ask_to_go_out" && game.Game.GoOutRequestPending {
		// Broadcast permission request to partner only
		partnerSlot := game.Players[game.Game.GoOutPartner]
		if partnerSlot.Token != "" {
			connID := s.connectionManager.GetConnectionByToken(partnerSlot.Token)
			if connID != "" {
				conn := s.connectionManager.GetConnection(connID)
				if conn != nil {
					s.sendMessage(conn, context.Background(), ServerMessage{
						Type: "permission_requested",
						Payload: PermissionRequestNotification{
							RequestingPlayer: game.Game.GoOutRequester,
							RequestingName:   game.Players[game.Game.GoOutRequester].Username,
						},
					})
					log.Printf("Permission request sent from player %d to partner %d in game %s",
						game.Game.GoOutRequester, game.Game.GoOutPartner, game.RoomCode)
				}
			}
		}
	} else if req.Type == "respond_go_out" {
		// Broadcast response to requester and partner
		approved := req.Id == 1
		s.broadcastToLobby(game, "permission_response", PermissionResponseNotification{
			Approved: approved,
		})
		log.Printf("Permission %s by player %d in game %s",
			map[bool]string{true: "approved", false: "denied"}[approved], playerID, game.RoomCode)
	}

	// Step 10: Detect hand/game end
	// Why check: Hand end triggers scoring, game end triggers completion
	handEnded := game.Game.HandNumber != previousHandNumber

	if handEnded {
		// Hand ended - send notification with scores
		s.broadcastToLobby(game, "hand_ended", HandEndedNotification{
			HandNumber:    previousHandNumber, // The hand that just ended
			TeamAScore:    game.Game.TeamA.Score,
			TeamBScore:    game.Game.TeamB.Score,
			NextHandReady: game.Game.HandNumber < 4, // false if game ended (hand 4 complete)
		})

		// Check if game ended (4 hands complete)
		if game.Game.HandNumber >= 4 {
			// Game completed
			game.Status = StatusCompleted

			// Determine winner (handle ties)
			winnerTeam := "Tie"
			if game.Game.TeamA.Score > game.Game.TeamB.Score {
				winnerTeam = "TeamA"
			} else if game.Game.TeamB.Score > game.Game.TeamA.Score {
				winnerTeam = "TeamB"
			}

			s.broadcastToLobby(game, "game_ended", GameEndedNotification{
				TeamAScore: game.Game.TeamA.Score,
				TeamBScore: game.Game.TeamB.Score,
				WinnerTeam: winnerTeam,
			})

			log.Printf("Game %s completed. Winner: %s (A:%d B:%d)",
				game.RoomCode, winnerTeam, game.Game.TeamA.Score, game.Game.TeamB.Score)
		}
	}

	// Step 11: Broadcast game state to all players
	// Why: All players need to see the updated state
	// Note: Still holding lock during broadcast to ensure consistency
	s.broadcastGameState(game)

	// Step 12: Send success response to requesting player
	// Why after broadcast: Ensures player gets confirmation after state is consistent
	s.sendMessage(socket, ctx, ServerMessage{
		Type: "move_result",
		Payload: MoveResultResponse{
			Success: true,
		},
	})

	// Mutex automatically released by defer after this function returns
}
