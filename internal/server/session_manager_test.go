package server

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test 1: Basic session storage and retrieval
// Why: Foundation of session management - must work reliably
func TestSessionManager_StoreAndRetrieve(t *testing.T) {
	sm := NewSessionManager()

	// Store session
	session := SessionInfo{
		Token:    "test-token-123",
		RoomCode: "ABCD",
		PlayerID: 0,
		Username: "Alice",
	}
	sm.StoreSession(session)

	// Retrieve session
	retrieved, err := sm.GetSession("test-token-123")
	assert.NoError(t, err)
	assert.Equal(t, session, retrieved)
}

// Test 2: Get non-existent session returns error
// Why: Security - invalid tokens must be rejected
func TestSessionManager_GetNonExistentSession(t *testing.T) {
	sm := NewSessionManager()

	// Try to get session that doesn't exist
	_, err := sm.GetSession("non-existent-token")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "TOKEN_NOT_FOUND")
}

// Test 3: Remove session
// Why: Need to clean up when players permanently leave
func TestSessionManager_RemoveSession(t *testing.T) {
	sm := NewSessionManager()

	// Store session
	session := SessionInfo{
		Token:    "temp-token",
		RoomCode: "WXYZ",
		PlayerID: 1,
		Username: "Bob",
	}
	sm.StoreSession(session)

	// Verify it exists
	_, err := sm.GetSession("temp-token")
	assert.NoError(t, err)

	// Remove session
	sm.RemoveSession("temp-token")

	// Verify it's gone
	_, err = sm.GetSession("temp-token")
	assert.Error(t, err)
}

// Test 4: Update session (overwrite)
// Why: Session info might need updating (though rare in practice)
func TestSessionManager_UpdateSession(t *testing.T) {
	sm := NewSessionManager()

	// Store initial session
	session := SessionInfo{
		Token:    "update-token",
		RoomCode: "ROOM",
		PlayerID: 0,
		Username: "Charlie",
	}
	sm.StoreSession(session)

	// Update with different player ID (same token)
	updatedSession := SessionInfo{
		Token:    "update-token",
		RoomCode: "ROOM",
		PlayerID: 2, // Changed
		Username: "Charlie",
	}
	sm.StoreSession(updatedSession)

	// Retrieve and verify update
	retrieved, err := sm.GetSession("update-token")
	assert.NoError(t, err)
	assert.Equal(t, 2, retrieved.PlayerID)
}

// Test 5: Get all sessions
// Why: Phase 6 needs to persist all sessions to DB
func TestSessionManager_GetAllSessions(t *testing.T) {
	sm := NewSessionManager()

	// Store multiple sessions
	sessions := []SessionInfo{
		{Token: "token1", RoomCode: "AAA1", PlayerID: 0, Username: "Player1"},
		{Token: "token2", RoomCode: "BBB2", PlayerID: 1, Username: "Player2"},
		{Token: "token3", RoomCode: "CCC3", PlayerID: 2, Username: "Player3"},
	}

	for _, session := range sessions {
		sm.StoreSession(session)
	}

	// Get all sessions
	allSessions := sm.GetAllSessions()
	assert.Equal(t, 3, len(allSessions))

	// Verify all tokens present (order not guaranteed)
	tokenMap := make(map[string]bool)
	for _, s := range allSessions {
		tokenMap[s.Token] = true
	}

	for _, expected := range sessions {
		assert.True(t, tokenMap[expected.Token], "Expected to find token %s", expected.Token)
	}
}

// Test 6: Empty session manager returns empty list
// Why: Edge case - should handle gracefully
func TestSessionManager_GetAllSessionsEmpty(t *testing.T) {
	sm := NewSessionManager()

	allSessions := sm.GetAllSessions()
	assert.Equal(t, 0, len(allSessions))
}

// Test 7: Concurrent session operations (thread safety)
// Why: Multiple goroutines will access SessionManager simultaneously
func TestSessionManager_ConcurrentOperations(t *testing.T) {
	sm := NewSessionManager()

	var wg sync.WaitGroup
	numGoroutines := 100

	// Concurrent stores
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			session := SessionInfo{
				Token:    fmt.Sprintf("token-%d", id),
				RoomCode: "CONC",
				PlayerID: id % 4,
				Username: fmt.Sprintf("User%d", id),
			}
			sm.StoreSession(session)
		}(i)
	}
	wg.Wait()

	// Verify all stored
	allSessions := sm.GetAllSessions()
	assert.Equal(t, numGoroutines, len(allSessions))

	// Concurrent reads
	wg.Add(numGoroutines)
	errorsChan := make(chan error, numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			_, err := sm.GetSession(fmt.Sprintf("token-%d", id))
			if err != nil {
				errorsChan <- err
			}
		}(i)
	}
	wg.Wait()
	close(errorsChan)

	// Check for any errors
	for err := range errorsChan {
		t.Errorf("Concurrent read error: %v", err)
	}

	// Concurrent deletes
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			sm.RemoveSession(fmt.Sprintf("token-%d", id))
		}(i)
	}
	wg.Wait()

	// Verify all removed
	allSessions = sm.GetAllSessions()
	assert.Equal(t, 0, len(allSessions))
}

// Test 8: Multiple sessions for same room
// Why: Multiple players can be in same game
func TestSessionManager_MultipleSessionsSameRoom(t *testing.T) {
	sm := NewSessionManager()

	roomCode := "SAME"
	sessions := []SessionInfo{
		{Token: "player1-token", RoomCode: roomCode, PlayerID: 0, Username: "Player1"},
		{Token: "player2-token", RoomCode: roomCode, PlayerID: 1, Username: "Player2"},
		{Token: "player3-token", RoomCode: roomCode, PlayerID: 2, Username: "Player3"},
		{Token: "player4-token", RoomCode: roomCode, PlayerID: 3, Username: "Player4"},
	}

	// Store all sessions
	for _, session := range sessions {
		sm.StoreSession(session)
	}

	// Verify each can be retrieved independently
	for i, expected := range sessions {
		retrieved, err := sm.GetSession(expected.Token)
		assert.NoError(t, err, "Session %d should be retrievable", i)
		assert.Equal(t, roomCode, retrieved.RoomCode)
		assert.Equal(t, expected.PlayerID, retrieved.PlayerID)
	}
}
