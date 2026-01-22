package server

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"

	_ "github.com/joho/godotenv/autoload"
	"github.com/pressly/goose/v3"

	"canasta-server/internal/database"
)

type Server struct {
	port               int
	db                 database.Service
	connectionManager  *ConnectionManager
	gameManager        *GameManager
	sessionManager     *SessionManager
	persistenceManager *PersistenceManager
	rateLimiter        *RateLimiter      // Rate limiting per connection
	connectionHealth   *ConnectionHealth // Connection health tracking
}

// NewServer creates and initializes the server
// Returns both the custom Server (for shutdown logic) and http.Server (for serving)
// Why both: Need access to Server methods while http.Server handles HTTP lifecycle
func NewServer() (*Server, *http.Server) {
	port := 8080

	// Initialize database
	dbService := database.New()

	// Run migrations
	if err := runMigrations(dbService.DB()); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Initialize persistence manager
	persistenceManager := NewPersistenceManager(dbService.DB())

	// Initialize game and session managers
	gameManager := NewGameManager()
	sessionManager := NewSessionManager()

	// Load persisted state from database
	if err := loadPersistedState(persistenceManager, gameManager, sessionManager); err != nil {
		log.Printf("Warning: Failed to load persisted state: %v", err)
		// Don't fatal - allow server to start with empty state
	}

	NewServer := &Server{
		port:               port,
		db:                 dbService,
		connectionManager:  NewConnectionManager(),
		gameManager:        gameManager,
		sessionManager:     sessionManager,
		persistenceManager: persistenceManager,
		rateLimiter:        NewRateLimiter(10, time.Second), // 10 messages per second
		connectionHealth:   NewConnectionHealth(),
	}

	// Start background tasks
	go NewServer.periodicSaveTask()
	go NewServer.cleanupTask()
	go NewServer.checkInactiveConnections() // Phase 7: Monitor connection health

	// Declare Server config
	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", NewServer.port),
		Handler:      NewServer.RegisterRoutes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	return NewServer, httpServer
}

// runMigrations applies database migrations using goose
func runMigrations(db *sql.DB) error {
	// Set SQLite dialect
	if err := goose.SetDialect("sqlite3"); err != nil {
		return fmt.Errorf("failed to set goose dialect: %w", err)
	}

	// Run migrations from db/migrations directory
	if err := goose.Up(db, "./db/migrations"); err != nil {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	log.Println("Database migrations applied successfully")
	return nil
}

// loadPersistedState restores games and sessions from the database
func loadPersistedState(pm *PersistenceManager, gm *GameManager, sm *SessionManager) error {
	// Load all active games
	games, err := pm.LoadAllActiveGames()
	if err != nil {
		return fmt.Errorf("failed to load games: %w", err)
	}

	// Restore games to GameManager
	gm.mu.Lock()
	for _, game := range games {
		gm.games[game.RoomCode] = game
		log.Printf("Restored game: %s (status: %s)", game.RoomCode, game.Status)
	}
	gm.mu.Unlock()

	// Load room codes
	usedCodes, err := pm.LoadUsedRoomCodes()
	if err != nil {
		return fmt.Errorf("failed to load room codes: %w", err)
	}

	gm.mu.Lock()
	gm.usedCodes = usedCodes
	gm.mu.Unlock()

	// Load all sessions
	sessions, err := pm.LoadAllSessions()
	if err != nil {
		return fmt.Errorf("failed to load sessions: %w", err)
	}

	// Restore sessions to SessionManager
	sm.mu.Lock()
	for _, session := range sessions {
		sm.sessions[session.Token] = session
		// Safe token display (handle short/corrupted tokens)
		tokenDisplay := session.Token
		if len(tokenDisplay) > 8 {
			tokenDisplay = tokenDisplay[:8]
		}
		log.Printf("Restored session: %s -> %s (player %d)", tokenDisplay, session.RoomCode, session.PlayerID)
	}
	sm.mu.Unlock()

	log.Printf("Loaded %d games, %d room codes, %d sessions", len(games), len(usedCodes), len(sessions))
	return nil
}

// periodicSaveTask runs every 30 seconds and persists all active games
// Why periodic saves: Catches any state changes that might not have been persisted (e.g., disconnects, lobby changes)
func (s *Server) periodicSaveTask() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// Hold read lock during entire save to prevent race conditions
		// Why: If we release lock before SaveGame, concurrent handlers can modify
		// the game struct while json.Marshal is reading it, causing data corruption
		s.gameManager.mu.RLock()

		savedCount := 0
		for _, game := range s.gameManager.games {
			if err := s.persistenceManager.SaveGame(game); err != nil {
				log.Printf("Periodic save failed for game %s: %v", game.RoomCode, err)
			} else {
				savedCount++
			}
		}

		s.gameManager.mu.RUnlock()

		log.Printf("Periodic save completed: %d games persisted", savedCount)
	}
}

// cleanupTask runs every hour and deletes completed games older than 24 hours
// Why cleanup: Prevents database from growing indefinitely
// Why 24 hours: Gives players time to review final scores before game disappears
func (s *Server) cleanupTask() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		deleted, err := s.persistenceManager.CleanupOldGames(24 * time.Hour)
		if err != nil {
			log.Printf("Cleanup task failed: %v", err)
			continue
		}

		if deleted > 0 {
			log.Printf("Cleanup task: deleted %d old completed games", deleted)
		}
	}
}

// Shutdown performs graceful shutdown operations
// Why separate method: Encapsulate all shutdown logic in Server
// Why return error: Allow caller to handle failures appropriately
func (s *Server) Shutdown(ctx context.Context) error {
	log.Println("Beginning graceful shutdown...")

	// Step 1: Notify all connected players
	// Why first: Give players immediate feedback that server is shutting down
	s.notifyAllPlayers("server_shutdown", ServerShutdownNotification{
		Message: "Server is shutting down for maintenance. Your game will be saved.",
	})
	log.Println("Notified all connected players of shutdown")

	// Step 2: Save all active games to database
	// Why: Ensure no game state is lost during shutdown
	s.gameManager.mu.RLock()
	gameCount := len(s.gameManager.games)
	savedCount := 0

	for _, game := range s.gameManager.games {
		if err := s.persistenceManager.SaveGame(game); err != nil {
			log.Printf("Failed to save game %s during shutdown: %v", game.RoomCode, err)
		} else {
			savedCount++
		}
	}
	s.gameManager.mu.RUnlock()

	log.Printf("Saved %d/%d games to database", savedCount, gameCount)

	// Step 3: Close database connection
	// Why last: Ensure all saves complete before closing DB
	if err := s.db.Close(); err != nil {
		log.Printf("Error closing database: %v", err)
		return err
	}

	log.Println("Database closed successfully")
	log.Println("Graceful shutdown complete")
	return nil
}

// notifyAllPlayers sends a message to all connected players
// Why: Centralize broadcast logic, reusable for different scenarios
func (s *Server) notifyAllPlayers(messageType string, payload interface{}) {
	s.connectionManager.mu.RLock()
	defer s.connectionManager.mu.RUnlock()

	for connID, conn := range s.connectionManager.connections {
		msg := ServerMessage{
			Type:    messageType,
			Payload: payload,
		}
		// Use background context for broadcasts (don't block)
		if err := s.sendMessage(conn, context.Background(), msg); err != nil {
			log.Printf("Failed to notify connection %s: %v", connID, err)
		}
	}
}
