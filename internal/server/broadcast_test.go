package server

import (
	"canasta-server/internal/canasta"
	"testing"
)

// ============================================================================
// Test Group 1: buildGameStateMessage Tests
// ============================================================================

func TestBuildGameStateMessage_Structure(t *testing.T) {
	// Why: Verify all required fields are present in the message
	s, _, cleanup := setupTestServer()
	defer cleanup()

	// Create and start a game
	game, _, err := s.gameManager.CreateGame("Alice", false)
	if err != nil {
		t.Fatalf("Failed to create game: %v", err)
	}

	// Join 3 more players
	_, _, _, err = s.gameManager.JoinGame(game.RoomCode, "Bob")
	if err != nil {
		t.Fatalf("Failed to join game: %v", err)
	}
	_, _, _, err = s.gameManager.JoinGame(game.RoomCode, "Carol")
	if err != nil {
		t.Fatalf("Failed to join game: %v", err)
	}
	_, _, _, err = s.gameManager.JoinGame(game.RoomCode, "Dave")
	if err != nil {
		t.Fatalf("Failed to join game: %v", err)
	}

	// Set all ready
	for i := 0; i < 4; i++ {
		_, _, err := s.gameManager.SetReady(game.RoomCode, game.Players[i].Token, true)
		if err != nil {
			t.Fatalf("Failed to set ready: %v", err)
		}
	}

	// Start game
	err = s.gameManager.StartGame(game.RoomCode)
	if err != nil {
		t.Fatalf("Failed to start game: %v", err)
	}

	// Build state for player 0
	state := s.buildGameStateMessage(game, 0)

	// Verify structure
	if state.State == nil {
		t.Error("State should not be nil")
	}
	if state.CurrentPlayer < 0 || state.CurrentPlayer > 3 {
		t.Errorf("CurrentPlayer should be 0-3, got %d", state.CurrentPlayer)
	}
	if state.Phase == "" {
		t.Error("Phase should not be empty")
	}
	if state.Status == "" {
		t.Error("Status should not be empty")
	}
}

func TestBuildGameStateMessage_GameNotStarted(t *testing.T) {
	// Why: Handle edge case where game not yet started
	s, _, cleanup := setupTestServer()
	defer cleanup()

	game, _, err := s.gameManager.CreateGame("Alice", false)
	if err != nil {
		t.Fatalf("Failed to create game: %v", err)
	}

	// Build state before starting game
	state := s.buildGameStateMessage(game, 0)

	// Should return minimal state
	if state.Status != string(StatusLobby) {
		t.Errorf("Status should be lobby, got %s", state.Status)
	}
}

func TestBuildGameStateMessage_Personalization(t *testing.T) {
	// Why: Each player should see their own hand, not others
	s, _, cleanup := setupTestServer()
	defer cleanup()

	// Create and start game
	game := setupFullGameAndStart(t, s)

	// Build state for different players
	state0 := s.buildGameStateMessage(game, 0)
	state1 := s.buildGameStateMessage(game, 1)

	// Extract client states
	clientState0 := state0.State.(*canasta.ClientState)
	clientState1 := state1.State.(*canasta.ClientState)

	// Verify they see their own names
	if clientState0.Name != "Alice" {
		t.Errorf("Player 0 should see their name as Alice, got %s", clientState0.Name)
	}
	if clientState1.Name != "Bob" {
		t.Errorf("Player 1 should see their name as Bob, got %s", clientState1.Name)
	}

	// Verify they have hands (should have been dealt)
	if len(clientState0.Hand) == 0 {
		t.Error("Player 0 should have cards in hand")
	}
	if len(clientState1.Hand) == 0 {
		t.Error("Player 1 should have cards in hand")
	}

	// Note: We can't easily verify hands are different without exposing internals,
	// but GetClientState is tested in canasta package
}

func TestBuildGameStateMessage_MetadataCorrect(t *testing.T) {
	// Why: Verify metadata (currentPlayer, phase, status) is correct
	s, _, cleanup := setupTestServer()
	defer cleanup()

	game := setupFullGameAndStart(t, s)

	state := s.buildGameStateMessage(game, 0)

	// Verify metadata matches game state
	if state.CurrentPlayer != game.Game.CurrentPlayer {
		t.Errorf("CurrentPlayer mismatch: got %d, want %d", state.CurrentPlayer, game.Game.CurrentPlayer)
	}
	if state.Phase != string(game.Game.Phase) {
		t.Errorf("Phase mismatch: got %s, want %s", state.Phase, game.Game.Phase)
	}
	if state.Status != string(game.Status) {
		t.Errorf("Status mismatch: got %s, want %s", state.Status, game.Status)
	}
}

// ============================================================================
// Test Group 2: broadcastGameState Tests
// ============================================================================

func TestBroadcastGameState_GameNotStarted(t *testing.T) {
	// Why: Should handle gracefully if game not started
	s, _, cleanup := setupTestServer()
	defer cleanup()

	game, _, err := s.gameManager.CreateGame("Alice", false)
	if err != nil {
		t.Fatalf("Failed to create game: %v", err)
	}

	// Should not panic
	s.broadcastGameState(game)
}

func TestBroadcastGameState_DisconnectedPlayerSkipped(t *testing.T) {
	// Why: Don't send to disconnected players
	s, _, cleanup := setupTestServer()
	defer cleanup()

	game := setupFullGameAndStart(t, s)

	// Mark player 2 as disconnected
	game.Players[2].Connected = false

	// Broadcast should succeed without error
	s.broadcastGameState(game)
	// Note: In a real test with mock connections, we'd verify only 3 messages sent
}

func TestBroadcastGameState_EmptySlotSkipped(t *testing.T) {
	// Why: Don't try to send to empty slots
	s, _, cleanup := setupTestServer()
	defer cleanup()

	game, _, err := s.gameManager.CreateGame("Alice", false)
	if err != nil {
		t.Fatalf("Failed to create game: %v", err)
	}

	// Only 1 player joined, slots 1-3 empty
	// Should not panic
	s.broadcastGameState(game)
}

func TestBuildGameStateMessage_EmptyDiscardPile(t *testing.T) {
	// Why: Empty discard pile should send nil, not zero-value Card
	// This happens when a player picks up the entire discard pile
	s, _, cleanup := setupTestServer()
	defer cleanup()

	game := setupFullGameAndStart(t, s)

	// Manually empty the discard pile to simulate pickup
	game.Game.Hand.DiscardPile = []canasta.Card{}

	// Build state
	stateMsg := s.buildGameStateMessage(game, 0)
	clientState := stateMsg.State.(*canasta.ClientState)

	// Verify DiscardTopCard is nil (not zero-value Card)
	if clientState.DiscardTopCard != nil {
		t.Errorf("DiscardTopCard should be nil when pile is empty, got %+v", clientState.DiscardTopCard)
	}

	// Verify count is 0
	if clientState.DiscardCount != 0 {
		t.Errorf("DiscardCount should be 0, got %d", clientState.DiscardCount)
	}
}

// ============================================================================
// Helper Functions
// ============================================================================

func setupFullGameAndStart(t *testing.T, s *Server) *ActiveGame {
	// Create game
	game, _, err := s.gameManager.CreateGame("Alice", false)
	if err != nil {
		t.Fatalf("Failed to create game: %v", err)
	}

	// Join 3 more players
	names := []string{"Bob", "Carol", "Dave"}
	for _, name := range names {
		_, _, _, err := s.gameManager.JoinGame(game.RoomCode, name)
		if err != nil {
			t.Fatalf("Failed to join game: %v", err)
		}
	}

	// Set all ready
	for i := 0; i < 4; i++ {
		_, _, err := s.gameManager.SetReady(game.RoomCode, game.Players[i].Token, true)
		if err != nil {
			t.Fatalf("Failed to set ready: %v", err)
		}
	}

	// Start game
	err = s.gameManager.StartGame(game.RoomCode)
	if err != nil {
		t.Fatalf("Failed to start game: %v", err)
	}

	return game
}
