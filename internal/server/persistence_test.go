package server

import (
	"canasta-server/internal/canasta"
	"database/sql"
	"log"
	"os"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pressly/goose/v3"
)

// setupTestDB creates a test database with migrations applied
func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	// Create a temporary database file
	dbPath := "test_persistence.db"

	// Clean up any existing test database
	os.Remove(dbPath)

	// Open database connection with foreign keys enabled
	db, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Set goose dialect for SQLite
	if err := goose.SetDialect("sqlite3"); err != nil {
		t.Fatalf("Failed to set goose dialect: %v", err)
	}

	// Run migrations
	if err := goose.Up(db, "../../db/migrations"); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Register cleanup
	t.Cleanup(func() {
		db.Close()
		os.Remove(dbPath)
	})

	return db
}

func TestPersistenceManager_SaveAndLoadGame_Lobby(t *testing.T) {
	db := setupTestDB(t)
	pm := NewPersistenceManager(db)

	// Create a test game in lobby state
	now := time.Now()
	game := &ActiveGame{
		Game:     nil, // Lobby games don't have Game initialized yet
		RoomCode: "TEST",
		Config: LobbyConfig{
			PlayerOrder:     [4]string{"Alice", "Bob", "", ""},
			RandomTeamOrder: false,
		},
		Status: StatusLobby,
		Players: [4]PlayerSlot{
			{Username: "Alice", Token: "token1", Connected: true, Ready: false, JoinedAt: now},
			{Username: "Bob", Token: "token2", Connected: true, Ready: false, JoinedAt: now},
			{Username: "", Token: "", Connected: false, Ready: false, JoinedAt: time.Time{}},
			{Username: "", Token: "", Connected: false, Ready: false, JoinedAt: time.Time{}},
		},
		CreatedAt:   now,
		UpdatedAt:   now,
		LobbyExpiry: now.Add(10 * time.Minute),
	}

	// Save the game
	if err := pm.SaveGame(game); err != nil {
		t.Fatalf("SaveGame failed: %v", err)
	}

	// Load the game back
	loaded, err := pm.LoadGame("TEST")
	if err != nil {
		t.Fatalf("LoadGame failed: %v", err)
	}

	// Verify all fields match
	if loaded.RoomCode != game.RoomCode {
		t.Errorf("RoomCode mismatch: got %s, want %s", loaded.RoomCode, game.RoomCode)
	}
	if loaded.Status != game.Status {
		t.Errorf("Status mismatch: got %s, want %s", loaded.Status, game.Status)
	}
	if loaded.Config.RandomTeamOrder != game.Config.RandomTeamOrder {
		t.Errorf("RandomTeamOrder mismatch: got %v, want %v", loaded.Config.RandomTeamOrder, game.Config.RandomTeamOrder)
	}
	if loaded.Players[0].Username != "Alice" {
		t.Errorf("Player 0 username mismatch: got %s, want Alice", loaded.Players[0].Username)
	}
	if loaded.Players[1].Username != "Bob" {
		t.Errorf("Player 1 username mismatch: got %s, want Bob", loaded.Players[1].Username)
	}
}

func TestPersistenceManager_SaveAndLoadGame_Active(t *testing.T) {
	db := setupTestDB(t)
	pm := NewPersistenceManager(db)

	// Create a test game with active canasta.Game
	now := time.Now()

	// Initialize a real game
	canastaGame := canasta.NewGame("test-game-1", []string{"Alice", "Bob", "Charlie", "Diana"})
	canastaGame.Deal()

	game := &ActiveGame{
		Game:     &canastaGame,
		RoomCode: "PLAY",
		Config: LobbyConfig{
			PlayerOrder:     [4]string{"Alice", "Bob", "Charlie", "Diana"},
			RandomTeamOrder: false,
		},
		Status: StatusPlaying,
		Players: [4]PlayerSlot{
			{Username: "Alice", Token: "token1", Connected: true, Ready: true, JoinedAt: now},
			{Username: "Bob", Token: "token2", Connected: true, Ready: true, JoinedAt: now},
			{Username: "Charlie", Token: "token3", Connected: true, Ready: true, JoinedAt: now},
			{Username: "Diana", Token: "token4", Connected: true, Ready: true, JoinedAt: now},
		},
		CreatedAt:   now,
		UpdatedAt:   now,
		LobbyExpiry: now.Add(10 * time.Minute),
	}

	// Save the game
	if err := pm.SaveGame(game); err != nil {
		t.Fatalf("SaveGame failed: %v", err)
	}

	// Load the game back
	loaded, err := pm.LoadGame("PLAY")
	if err != nil {
		t.Fatalf("LoadGame failed: %v", err)
	}

	// Verify game state is preserved
	if loaded.Game == nil {
		t.Fatal("Loaded game has nil Game field")
	}
	if loaded.Game.CurrentPlayer != canastaGame.CurrentPlayer {
		t.Errorf("CurrentPlayer mismatch: got %d, want %d", loaded.Game.CurrentPlayer, canastaGame.CurrentPlayer)
	}
	if loaded.Game.Phase != canastaGame.Phase {
		t.Errorf("Phase mismatch: got %s, want %s", loaded.Game.Phase, canastaGame.Phase)
	}
	if loaded.Status != StatusPlaying {
		t.Errorf("Status mismatch: got %s, want %s", loaded.Status, StatusPlaying)
	}
}

func TestPersistenceManager_LoadGame_NotFound(t *testing.T) {
	db := setupTestDB(t)
	pm := NewPersistenceManager(db)

	_, err := pm.LoadGame("NOEXIST")
	if err == nil {
		t.Fatal("Expected error for non-existent game, got nil")
	}
}

func TestPersistenceManager_SaveGame_Update(t *testing.T) {
	db := setupTestDB(t)
	pm := NewPersistenceManager(db)

	now := time.Now()
	game := &ActiveGame{
		Game:     nil,
		RoomCode: "UPDT",
		Config: LobbyConfig{
			PlayerOrder:     [4]string{"Alice", "", "", ""},
			RandomTeamOrder: false,
		},
		Status:      StatusLobby,
		Players:     [4]PlayerSlot{{Username: "Alice", Token: "token1", Connected: true, Ready: false, JoinedAt: now}},
		CreatedAt:   now,
		UpdatedAt:   now,
		LobbyExpiry: now.Add(10 * time.Minute),
	}

	// Save initially
	if err := pm.SaveGame(game); err != nil {
		t.Fatalf("Initial SaveGame failed: %v", err)
	}

	// Modify the game (add a player)
	game.Players[1] = PlayerSlot{Username: "Bob", Token: "token2", Connected: true, Ready: false, JoinedAt: now}
	game.Config.PlayerOrder[1] = "Bob"
	game.UpdatedAt = now.Add(1 * time.Minute)

	// Save again (should update, not insert)
	if err := pm.SaveGame(game); err != nil {
		t.Fatalf("Update SaveGame failed: %v", err)
	}

	// Load and verify the update
	loaded, err := pm.LoadGame("UPDT")
	if err != nil {
		t.Fatalf("LoadGame after update failed: %v", err)
	}

	if loaded.Players[1].Username != "Bob" {
		t.Errorf("Player 1 not updated: got %s, want Bob", loaded.Players[1].Username)
	}
}

func TestPersistenceManager_LoadAllActiveGames(t *testing.T) {
	db := setupTestDB(t)
	pm := NewPersistenceManager(db)

	now := time.Now()

	// Create multiple games with different statuses
	games := []*ActiveGame{
		{
			RoomCode:    "LOB1",
			Status:      StatusLobby,
			Config:      LobbyConfig{},
			Players:     [4]PlayerSlot{},
			CreatedAt:   now,
			UpdatedAt:   now,
			LobbyExpiry: now.Add(10 * time.Minute),
		},
		{
			RoomCode:    "PLAY",
			Status:      StatusPlaying,
			Config:      LobbyConfig{},
			Players:     [4]PlayerSlot{},
			CreatedAt:   now,
			UpdatedAt:   now,
			LobbyExpiry: now.Add(10 * time.Minute),
		},
		{
			RoomCode:    "PAUS",
			Status:      StatusPaused,
			Config:      LobbyConfig{},
			Players:     [4]PlayerSlot{},
			CreatedAt:   now,
			UpdatedAt:   now,
			LobbyExpiry: now.Add(10 * time.Minute),
		},
		{
			RoomCode:    "DONE",
			Status:      StatusCompleted,
			Config:      LobbyConfig{},
			Players:     [4]PlayerSlot{},
			CreatedAt:   now,
			UpdatedAt:   now,
			LobbyExpiry: now.Add(10 * time.Minute),
		},
	}

	// Save all games
	for _, g := range games {
		if err := pm.SaveGame(g); err != nil {
			t.Fatalf("SaveGame failed for %s: %v", g.RoomCode, err)
		}
	}

	// Load all active games (should exclude completed)
	loaded, err := pm.LoadAllActiveGames()
	if err != nil {
		t.Fatalf("LoadAllActiveGames failed: %v", err)
	}

	// Should have 3 games (lobby, playing, paused - NOT completed)
	if len(loaded) != 3 {
		t.Errorf("Expected 3 active games, got %d", len(loaded))
	}

	// Verify completed game is not included
	for _, g := range loaded {
		if g.Status == StatusCompleted {
			t.Errorf("LoadAllActiveGames should not return completed games")
		}
	}
}

func TestPersistenceManager_DeleteGame(t *testing.T) {
	db := setupTestDB(t)
	pm := NewPersistenceManager(db)

	now := time.Now()
	game := &ActiveGame{
		RoomCode:    "DELT",
		Status:      StatusLobby,
		Config:      LobbyConfig{},
		Players:     [4]PlayerSlot{},
		CreatedAt:   now,
		UpdatedAt:   now,
		LobbyExpiry: now.Add(10 * time.Minute),
	}

	// Save the game
	if err := pm.SaveGame(game); err != nil {
		t.Fatalf("SaveGame failed: %v", err)
	}

	// Delete the game
	if err := pm.DeleteGame("DELT"); err != nil {
		t.Fatalf("DeleteGame failed: %v", err)
	}

	// Verify it's gone
	_, err := pm.LoadGame("DELT")
	if err == nil {
		t.Fatal("Expected error after deletion, got nil")
	}
}

func TestPersistenceManager_SaveAndLoadSession(t *testing.T) {
	db := setupTestDB(t)
	pm := NewPersistenceManager(db)

	// Need a game first (foreign key constraint)
	now := time.Now()
	game := &ActiveGame{
		RoomCode:    "SESS",
		Status:      StatusLobby,
		Config:      LobbyConfig{},
		Players:     [4]PlayerSlot{},
		CreatedAt:   now,
		UpdatedAt:   now,
		LobbyExpiry: now.Add(10 * time.Minute),
	}
	if err := pm.SaveGame(game); err != nil {
		t.Fatalf("SaveGame failed: %v", err)
	}

	// Create a session
	session := SessionInfo{
		Token:    "test-token-123",
		RoomCode: "SESS",
		PlayerID: 0,
		Username: "Alice",
	}

	// Save the session
	if err := pm.SaveSession(session); err != nil {
		t.Fatalf("SaveSession failed: %v", err)
	}

	// Load the session back
	loaded, err := pm.LoadSession("test-token-123")
	if err != nil {
		t.Fatalf("LoadSession failed: %v", err)
	}

	// Verify fields match
	if loaded.Token != session.Token {
		t.Errorf("Token mismatch: got %s, want %s", loaded.Token, session.Token)
	}
	if loaded.RoomCode != session.RoomCode {
		t.Errorf("RoomCode mismatch: got %s, want %s", loaded.RoomCode, session.RoomCode)
	}
	if loaded.PlayerID != session.PlayerID {
		t.Errorf("PlayerID mismatch: got %d, want %d", loaded.PlayerID, session.PlayerID)
	}
	if loaded.Username != session.Username {
		t.Errorf("Username mismatch: got %s, want %s", loaded.Username, session.Username)
	}
}

func TestPersistenceManager_LoadSession_NotFound(t *testing.T) {
	db := setupTestDB(t)
	pm := NewPersistenceManager(db)

	_, err := pm.LoadSession("nonexistent")
	if err == nil {
		t.Fatal("Expected error for non-existent session, got nil")
	}
	if err.Error() != "TOKEN_NOT_FOUND: Invalid session token" {
		t.Errorf("Expected TOKEN_NOT_FOUND error, got: %v", err)
	}
}

func TestPersistenceManager_LoadAllSessions(t *testing.T) {
	db := setupTestDB(t)
	pm := NewPersistenceManager(db)

	// Create games
	now := time.Now()
	for _, code := range []string{"GM1", "GM2"} {
		game := &ActiveGame{
			RoomCode:    code,
			Status:      StatusLobby,
			Config:      LobbyConfig{},
			Players:     [4]PlayerSlot{},
			CreatedAt:   now,
			UpdatedAt:   now,
			LobbyExpiry: now.Add(10 * time.Minute),
		}
		if err := pm.SaveGame(game); err != nil {
			t.Fatalf("SaveGame failed: %v", err)
		}
	}

	// Create multiple sessions
	sessions := []SessionInfo{
		{Token: "tok1", RoomCode: "GM1", PlayerID: 0, Username: "Alice"},
		{Token: "tok2", RoomCode: "GM1", PlayerID: 1, Username: "Bob"},
		{Token: "tok3", RoomCode: "GM2", PlayerID: 0, Username: "Charlie"},
	}

	for _, s := range sessions {
		if err := pm.SaveSession(s); err != nil {
			t.Fatalf("SaveSession failed: %v", err)
		}
	}

	// Load all sessions
	loaded, err := pm.LoadAllSessions()
	if err != nil {
		t.Fatalf("LoadAllSessions failed: %v", err)
	}

	if len(loaded) != 3 {
		t.Errorf("Expected 3 sessions, got %d", len(loaded))
	}
}

func TestPersistenceManager_DeleteSession(t *testing.T) {
	db := setupTestDB(t)
	pm := NewPersistenceManager(db)

	// Create game
	now := time.Now()
	game := &ActiveGame{
		RoomCode:    "DEL",
		Status:      StatusLobby,
		Config:      LobbyConfig{},
		Players:     [4]PlayerSlot{},
		CreatedAt:   now,
		UpdatedAt:   now,
		LobbyExpiry: now.Add(10 * time.Minute),
	}
	if err := pm.SaveGame(game); err != nil {
		t.Fatalf("SaveGame failed: %v", err)
	}

	// Create session
	session := SessionInfo{Token: "del-tok", RoomCode: "DEL", PlayerID: 0, Username: "Alice"}
	if err := pm.SaveSession(session); err != nil {
		t.Fatalf("SaveSession failed: %v", err)
	}

	// Delete session
	if err := pm.DeleteSession("del-tok"); err != nil {
		t.Fatalf("DeleteSession failed: %v", err)
	}

	// Verify it's gone
	_, err := pm.LoadSession("del-tok")
	if err == nil {
		t.Fatal("Expected error after deletion, got nil")
	}
}

func TestPersistenceManager_CascadeDelete(t *testing.T) {
	db := setupTestDB(t)
	pm := NewPersistenceManager(db)

	// Create game with session
	now := time.Now()
	game := &ActiveGame{
		RoomCode:    "CASC",
		Status:      StatusLobby,
		Config:      LobbyConfig{},
		Players:     [4]PlayerSlot{},
		CreatedAt:   now,
		UpdatedAt:   now,
		LobbyExpiry: now.Add(10 * time.Minute),
	}
	if err := pm.SaveGame(game); err != nil {
		t.Fatalf("SaveGame failed: %v", err)
	}

	session := SessionInfo{Token: "casc-tok", RoomCode: "CASC", PlayerID: 0, Username: "Alice"}
	if err := pm.SaveSession(session); err != nil {
		t.Fatalf("SaveSession failed: %v", err)
	}

	// Delete the game (should cascade to session)
	if err := pm.DeleteGame("CASC"); err != nil {
		t.Fatalf("DeleteGame failed: %v", err)
	}

	// Verify session is also gone (cascaded)
	_, err := pm.LoadSession("casc-tok")
	if err == nil {
		t.Fatal("Expected session to be deleted via cascade, but it still exists")
	}
}

func TestPersistenceManager_SaveAndLoadRoomCodes(t *testing.T) {
	db := setupTestDB(t)
	pm := NewPersistenceManager(db)

	// Save multiple room codes
	codes := map[string]bool{
		"ABCD": true,
		"EFGH": true,
		"IJKL": false, // Not in use
	}

	for code, inUse := range codes {
		if err := pm.SaveRoomCode(code, inUse); err != nil {
			t.Fatalf("SaveRoomCode failed for %s: %v", code, err)
		}
	}

	// Load all room codes
	loaded, err := pm.LoadUsedRoomCodes()
	if err != nil {
		t.Fatalf("LoadUsedRoomCodes failed: %v", err)
	}

	// Verify all codes are present
	if len(loaded) != 3 {
		t.Errorf("Expected 3 room codes, got %d", len(loaded))
	}

	for code, expectedInUse := range codes {
		actualInUse, exists := loaded[code]
		if !exists {
			t.Errorf("Room code %s not found in loaded codes", code)
		}
		if actualInUse != expectedInUse {
			t.Errorf("Room code %s: expected inUse=%v, got %v", code, expectedInUse, actualInUse)
		}
	}
}

func TestPersistence_Integration_ServerRestart(t *testing.T) {
	// This test simulates a complete server lifecycle:
	// 1. Create game, players join
	// 2. Persist to DB
	// 3. Simulate server restart (new managers)
	// 4. Load state from DB
	// 5. Verify everything restored correctly

	db := setupTestDB(t)
	pm := NewPersistenceManager(db)

	// === Phase 1: Normal operation ===
	gm1 := NewGameManager()
	sm1 := NewSessionManager()

	// Create a game
	game, token1, err := gm1.CreateGame("Alice", false)
	if err != nil {
		t.Fatalf("CreateGame failed: %v", err)
	}

	// Alice's session
	session1 := SessionInfo{
		Token:    token1,
		RoomCode: game.RoomCode,
		PlayerID: 0,
		Username: "Alice",
	}
	sm1.StoreSession(session1)

	// Save to DB
	if err := pm.SaveGame(game); err != nil {
		t.Fatalf("SaveGame failed: %v", err)
	}
	if err := pm.SaveSession(session1); err != nil {
		t.Fatalf("SaveSession failed: %v", err)
	}
	if err := pm.SaveRoomCode(game.RoomCode, true); err != nil {
		t.Fatalf("SaveRoomCode failed: %v", err)
	}

	// Add more players
	_, token2, _, err := gm1.JoinGame(game.RoomCode, "Bob")
	if err != nil {
		t.Fatalf("JoinGame failed: %v", err)
	}
	session2 := SessionInfo{Token: token2, RoomCode: game.RoomCode, PlayerID: 1, Username: "Bob"}
	sm1.StoreSession(session2)

	_, token3, _, err := gm1.JoinGame(game.RoomCode, "Charlie")
	if err != nil {
		t.Fatalf("JoinGame failed: %v", err)
	}
	session3 := SessionInfo{Token: token3, RoomCode: game.RoomCode, PlayerID: 2, Username: "Charlie"}
	sm1.StoreSession(session3)

	_, token4, _, err := gm1.JoinGame(game.RoomCode, "Diana")
	if err != nil {
		t.Fatalf("JoinGame failed: %v", err)
	}
	session4 := SessionInfo{Token: token4, RoomCode: game.RoomCode, PlayerID: 3, Username: "Diana"}
	sm1.StoreSession(session4)

	// Persist all changes
	game, _ = gm1.GetGame(game.RoomCode)
	if err := pm.SaveGame(game); err != nil {
		t.Fatalf("SaveGame failed: %v", err)
	}
	// Save all sessions (including session1 again in case it needs update)
	for _, session := range []SessionInfo{session1, session2, session3, session4} {
		if err := pm.SaveSession(session); err != nil {
			t.Fatalf("SaveSession failed: %v", err)
		}
	}

	// === Phase 2: Server restart (simulate) ===
	// Create new managers (simulating fresh server start)
	gm2 := NewGameManager()
	sm2 := NewSessionManager()

	// Load state from DB
	games, err := pm.LoadAllActiveGames()
	if err != nil {
		t.Fatalf("LoadAllActiveGames failed: %v", err)
	}

	// Restore to new GameManager
	for _, g := range games {
		gm2.games[g.RoomCode] = g
	}

	// Load room codes
	usedCodes, err := pm.LoadUsedRoomCodes()
	if err != nil {
		t.Fatalf("LoadUsedRoomCodes failed: %v", err)
	}
	gm2.usedCodes = usedCodes

	// Load sessions
	sessions, err := pm.LoadAllSessions()
	if err != nil {
		t.Fatalf("LoadAllSessions failed: %v", err)
	}
	for _, s := range sessions {
		sm2.sessions[s.Token] = s
	}

	// === Phase 3: Verify restored state ===

	// Check game exists
	restoredGame, err := gm2.GetGame(game.RoomCode)
	if err != nil {
		t.Fatalf("GetGame after restart failed: %v", err)
	}

	// Verify all players
	if restoredGame.Players[0].Username != "Alice" {
		t.Errorf("Player 0 not restored: got %s", restoredGame.Players[0].Username)
	}
	if restoredGame.Players[1].Username != "Bob" {
		t.Errorf("Player 1 not restored: got %s", restoredGame.Players[1].Username)
	}
	if restoredGame.Players[2].Username != "Charlie" {
		t.Errorf("Player 2 not restored: got %s", restoredGame.Players[2].Username)
	}
	if restoredGame.Players[3].Username != "Diana" {
		t.Errorf("Player 3 not restored: got %s", restoredGame.Players[3].Username)
	}

	// Verify sessions
	for i, token := range []string{token1, token2, token3, token4} {
		session, err := sm2.GetSession(token)
		if err != nil {
			t.Errorf("Session %d not restored (token: %s): %v", i, token, err)
			t.Logf("Available sessions in sm2: %d", len(sm2.sessions))
			continue
		}
		t.Logf("Session %d restored: %s -> %s", i, session.Username, session.RoomCode)
	}

	// Verify room code is marked as used
	if !gm2.usedCodes[game.RoomCode] {
		t.Errorf("Room code not marked as used after restore")
	}

	log.Printf("Integration test passed: Server restart successfully restored game %s with 4 players", game.RoomCode)
}

func TestPersistenceManager_CleanupOldGames(t *testing.T) {
	db := setupTestDB(t)
	pm := NewPersistenceManager(db)

	now := time.Now()

	// Create games with different timestamps and statuses
	oldCompleted := &ActiveGame{
		RoomCode:    "OLD1",
		Status:      StatusCompleted,
		Config:      LobbyConfig{},
		Players:     [4]PlayerSlot{},
		CreatedAt:   now.Add(-48 * time.Hour),
		UpdatedAt:   now.Add(-48 * time.Hour),
		LobbyExpiry: now.Add(-38 * time.Hour),
	}

	recentCompleted := &ActiveGame{
		RoomCode:    "NEW1",
		Status:      StatusCompleted,
		Config:      LobbyConfig{},
		Players:     [4]PlayerSlot{},
		CreatedAt:   now.Add(-1 * time.Hour),
		UpdatedAt:   now.Add(-1 * time.Hour),
		LobbyExpiry: now.Add(9 * time.Hour),
	}

	oldActive := &ActiveGame{
		RoomCode:    "OLD2",
		Status:      StatusPlaying,
		Config:      LobbyConfig{},
		Players:     [4]PlayerSlot{},
		CreatedAt:   now.Add(-48 * time.Hour),
		UpdatedAt:   now.Add(-48 * time.Hour),
		LobbyExpiry: now.Add(-38 * time.Hour),
	}

	// Save all games
	for _, game := range []*ActiveGame{oldCompleted, recentCompleted, oldActive} {
		if err := pm.SaveGame(game); err != nil {
			t.Fatalf("SaveGame failed: %v", err)
		}
	}

	// Cleanup games completed more than 24 hours ago
	deleted, err := pm.CleanupOldGames(24 * time.Hour)
	if err != nil {
		t.Fatalf("CleanupOldGames failed: %v", err)
	}

	// Should delete only OLD1 (old + completed)
	if deleted != 1 {
		t.Errorf("Expected 1 game deleted, got %d", deleted)
	}

	// Verify OLD1 is gone
	_, err = pm.LoadGame("OLD1")
	if err == nil {
		t.Error("Expected OLD1 to be deleted")
	}

	// Verify NEW1 still exists (recent completed)
	_, err = pm.LoadGame("NEW1")
	if err != nil {
		t.Error("Expected NEW1 to still exist")
	}

	// Verify OLD2 still exists (old but playing)
	_, err = pm.LoadGame("OLD2")
	if err != nil {
		t.Error("Expected OLD2 to still exist (not completed)")
	}
}
