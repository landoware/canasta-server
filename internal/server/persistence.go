package server

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// PersistenceManager handles saving and loading game state to/from database
type PersistenceManager struct {
	db *sql.DB
}

// NewPersistenceManager creates a new persistence manager
func NewPersistenceManager(db *sql.DB) *PersistenceManager {
	return &PersistenceManager{
		db: db,
	}
}

// SaveGame persists an ActiveGame to the database
// Uses UPSERT (INSERT OR REPLACE) to handle both new games and updates
func (pm *PersistenceManager) SaveGame(game *ActiveGame) error {
	// Serialize the entire ActiveGame struct to JSON
	gameData, err := json.Marshal(game)
	if err != nil {
		return fmt.Errorf("failed to serialize game: %w", err)
	}

	// Use INSERT OR REPLACE to handle both new and existing games
	query := `
		INSERT OR REPLACE INTO games (room_code, status, game_data, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
	`

	_, err = pm.db.Exec(
		query,
		game.RoomCode,
		string(game.Status),
		string(gameData),
		game.CreatedAt,
		game.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to save game %s: %w", game.RoomCode, err)
	}

	return nil
}

// LoadGame retrieves an ActiveGame from the database by room code
func (pm *PersistenceManager) LoadGame(roomCode string) (*ActiveGame, error) {
	query := `
		SELECT game_data FROM games WHERE room_code = ?
	`

	var gameData string
	err := pm.db.QueryRow(query, roomCode).Scan(&gameData)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("game not found: %s", roomCode)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to load game %s: %w", roomCode, err)
	}

	// Deserialize JSON back to ActiveGame
	var game ActiveGame
	if err := json.Unmarshal([]byte(gameData), &game); err != nil {
		return nil, fmt.Errorf("failed to deserialize game %s: %w", roomCode, err)
	}

	return &game, nil
}

// LoadAllActiveGames retrieves all games that are not completed
// Used on server startup to restore in-memory state
func (pm *PersistenceManager) LoadAllActiveGames() ([]*ActiveGame, error) {
	query := `
		SELECT game_data FROM games 
		WHERE status != ?
		ORDER BY updated_at DESC
	`

	rows, err := pm.db.Query(query, string(StatusCompleted))
	if err != nil {
		return nil, fmt.Errorf("failed to query active games: %w", err)
	}
	defer rows.Close()

	var games []*ActiveGame
	for rows.Next() {
		var gameData string
		if err := rows.Scan(&gameData); err != nil {
			return nil, fmt.Errorf("failed to scan game row: %w", err)
		}

		var game ActiveGame
		if err := json.Unmarshal([]byte(gameData), &game); err != nil {
			// Log the error but continue with other games
			fmt.Printf("Warning: failed to deserialize game: %v\n", err)
			continue
		}

		games = append(games, &game)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating game rows: %w", err)
	}

	return games, nil
}

// DeleteGame removes a game from the database
// Cascades to sessions table due to foreign key constraint
// Also marks the room code as available for reuse
func (pm *PersistenceManager) DeleteGame(roomCode string) error {
	query := `DELETE FROM games WHERE room_code = ?`

	result, err := pm.db.Exec(query, roomCode)
	if err != nil {
		return fmt.Errorf("failed to delete game %s: %w", roomCode, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check deletion result: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("game not found: %s", roomCode)
	}

	// Mark room code as available for reuse
	// Why: Prevents room code exhaustion over time
	if err := pm.SaveRoomCode(roomCode, false); err != nil {
		// Log but don't fail - game is already deleted
		fmt.Printf("Warning: failed to mark room code %s as unused: %v\n", roomCode, err)
	}

	return nil
}

// SaveSession persists a player session to the database
func (pm *PersistenceManager) SaveSession(session SessionInfo) error {
	query := `
		INSERT OR REPLACE INTO sessions (token, room_code, player_id, username, created_at)
		VALUES (?, ?, ?, ?, ?)
	`

	_, err := pm.db.Exec(
		query,
		session.Token,
		session.RoomCode,
		session.PlayerID,
		session.Username,
		time.Now(),
	)

	if err != nil {
		return fmt.Errorf("failed to save session %s: %w", session.Token, err)
	}

	return nil
}

// LoadSession retrieves a session by token
func (pm *PersistenceManager) LoadSession(token string) (*SessionInfo, error) {
	query := `
		SELECT token, room_code, player_id, username FROM sessions WHERE token = ?
	`

	var session SessionInfo
	err := pm.db.QueryRow(query, token).Scan(
		&session.Token,
		&session.RoomCode,
		&session.PlayerID,
		&session.Username,
	)

	if err == sql.ErrNoRows {
		return nil, errors.New("TOKEN_NOT_FOUND: Invalid session token")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to load session %s: %w", token, err)
	}

	return &session, nil
}

// LoadAllSessions retrieves all sessions from the database
// Used on server startup to restore SessionManager state
func (pm *PersistenceManager) LoadAllSessions() ([]SessionInfo, error) {
	query := `SELECT token, room_code, player_id, username FROM sessions`

	rows, err := pm.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query sessions: %w", err)
	}
	defer rows.Close()

	var sessions []SessionInfo
	for rows.Next() {
		var session SessionInfo
		if err := rows.Scan(&session.Token, &session.RoomCode, &session.PlayerID, &session.Username); err != nil {
			return nil, fmt.Errorf("failed to scan session row: %w", err)
		}
		sessions = append(sessions, session)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating session rows: %w", err)
	}

	return sessions, nil
}

// DeleteSession removes a session from the database
func (pm *PersistenceManager) DeleteSession(token string) error {
	query := `DELETE FROM sessions WHERE token = ?`

	_, err := pm.db.Exec(query, token)
	if err != nil {
		return fmt.Errorf("failed to delete session %s: %w", token, err)
	}

	return nil
}

// SaveRoomCode marks a room code as in use in the database
func (pm *PersistenceManager) SaveRoomCode(code string, inUse bool) error {
	query := `
		INSERT OR REPLACE INTO room_codes (code, in_use, created_at)
		VALUES (?, ?, ?)
	`

	_, err := pm.db.Exec(query, code, inUse, time.Now())
	if err != nil {
		return fmt.Errorf("failed to save room code %s: %w", code, err)
	}

	return nil
}

// LoadUsedRoomCodes retrieves all room codes that are currently in use
// Used on server startup to restore GameManager.usedCodes map
func (pm *PersistenceManager) LoadUsedRoomCodes() (map[string]bool, error) {
	query := `SELECT code, in_use FROM room_codes`

	rows, err := pm.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query room codes: %w", err)
	}
	defer rows.Close()

	usedCodes := make(map[string]bool)
	for rows.Next() {
		var code string
		var inUse bool
		if err := rows.Scan(&code, &inUse); err != nil {
			return nil, fmt.Errorf("failed to scan room code row: %w", err)
		}
		usedCodes[code] = inUse
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating room code rows: %w", err)
	}

	return usedCodes, nil
}

// CleanupOldGames deletes completed games older than the specified duration
// Also marks their room codes as available for reuse
func (pm *PersistenceManager) CleanupOldGames(olderThan time.Duration) (int, error) {
	cutoff := time.Now().Add(-olderThan)

	// First, get the room codes that will be deleted so we can free them
	selectQuery := `SELECT room_code FROM games WHERE status = ? AND updated_at < ?`
	rows, err := pm.db.Query(selectQuery, string(StatusCompleted), cutoff)
	if err != nil {
		return 0, fmt.Errorf("failed to query old games: %w", err)
	}

	var roomCodes []string
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			rows.Close()
			return 0, fmt.Errorf("failed to scan room code: %w", err)
		}
		roomCodes = append(roomCodes, code)
	}
	rows.Close()

	// Delete the games
	deleteQuery := `DELETE FROM games WHERE status = ? AND updated_at < ?`
	result, err := pm.db.Exec(deleteQuery, string(StatusCompleted), cutoff)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup old games: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to check cleanup result: %w", err)
	}

	// Mark room codes as available for reuse
	for _, code := range roomCodes {
		if err := pm.SaveRoomCode(code, false); err != nil {
			// Log but continue - don't fail cleanup because of room code update
			fmt.Printf("Warning: failed to mark room code %s as unused: %v\n", code, err)
		}
	}

	return int(rowsAffected), nil
}
