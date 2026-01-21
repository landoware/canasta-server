package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test: Add connection with token (first connection)
// Why: Basic functionality - first time a token connects
func TestConnectionManager_AddConnectionWithToken_FirstConnection(t *testing.T) {
	cm := NewConnectionManager()

	token := "test-token"
	connID := "conn-1"

	// Add connection (no previous connection for this token)
	oldConnID := cm.AddConnectionWithToken(connID, nil, token)

	assert.Empty(t, oldConnID, "Should return empty string for first connection")

	// Verify connection mapped
	foundConnID := cm.GetConnectionByToken(token)
	assert.Equal(t, connID, foundConnID)

	// Verify token mapped
	foundToken := cm.GetTokenByConnection(connID)
	assert.Equal(t, token, foundToken)
}

// Test: Add connection with token (device switch - same token, new connection)
// Why: Core device switching detection - player connects from different device
func TestConnectionManager_AddConnectionWithToken_DeviceSwitch(t *testing.T) {
	cm := NewConnectionManager()

	token := "test-token"
	conn1ID := "conn-1"
	conn2ID := "conn-2"

	// First connection from device 1
	oldConnID := cm.AddConnectionWithToken(conn1ID, nil, token)
	assert.Empty(t, oldConnID, "First connection should return empty")

	// Second connection from device 2 (same token)
	oldConnID = cm.AddConnectionWithToken(conn2ID, nil, token)
	assert.Equal(t, conn1ID, oldConnID, "Should return old connection ID")

	// Verify new connection is now mapped
	foundConnID := cm.GetConnectionByToken(token)
	assert.Equal(t, conn2ID, foundConnID, "Should map to new connection")

	// Verify new connection has token
	foundToken := cm.GetTokenByConnection(conn2ID)
	assert.Equal(t, token, foundToken)
}

// Test: Add connection with token (same connection ID - no-op)
// Why: Edge case - reconnecting to same connection ID should be no-op
func TestConnectionManager_AddConnectionWithToken_SameConnectionID(t *testing.T) {
	cm := NewConnectionManager()

	token := "test-token"
	connID := "conn-1"

	// First connection
	oldConnID := cm.AddConnectionWithToken(connID, nil, token)
	assert.Empty(t, oldConnID)

	// Same connection ID again (treat as no-op per requirements)
	// This is an edge case that shouldn't happen in practice
	oldConnID = cm.AddConnectionWithToken(connID, nil, token)
	assert.Equal(t, connID, oldConnID, "Should return same connection ID")

	// Verify mapping unchanged
	foundConnID := cm.GetConnectionByToken(token)
	assert.Equal(t, connID, foundConnID)
}

// Test: Get connection by token
// Why: Used for broadcasting messages to specific player
func TestConnectionManager_GetConnectionByToken(t *testing.T) {
	cm := NewConnectionManager()

	token1 := "token-1"
	token2 := "token-2"
	conn1ID := "conn-1"
	conn2ID := "conn-2"

	// Add two connections
	cm.AddConnectionWithToken(conn1ID, nil, token1)
	cm.AddConnectionWithToken(conn2ID, nil, token2)

	// Verify each token maps to correct connection
	assert.Equal(t, conn1ID, cm.GetConnectionByToken(token1))
	assert.Equal(t, conn2ID, cm.GetConnectionByToken(token2))

	// Non-existent token returns empty
	assert.Empty(t, cm.GetConnectionByToken("fake-token"))
}

// Test: Get token by connection
// Why: Used for disconnect handling - need to know which player disconnected
func TestConnectionManager_GetTokenByConnection(t *testing.T) {
	cm := NewConnectionManager()

	token := "test-token"
	connID := "conn-1"

	cm.AddConnectionWithToken(connID, nil, token)

	// Verify connection maps to token
	assert.Equal(t, token, cm.GetTokenByConnection(connID))

	// Non-existent connection returns empty
	assert.Empty(t, cm.GetTokenByConnection("fake-conn"))
}

// Test: Remove connection
// Why: Cleanup when websocket closes
func TestConnectionManager_RemoveConnection(t *testing.T) {
	cm := NewConnectionManager()

	token := "test-token"
	connID := "conn-1"

	cm.AddConnectionWithToken(connID, nil, token)

	// Verify connection exists
	assert.Equal(t, connID, cm.GetConnectionByToken(token))
	assert.Equal(t, token, cm.GetTokenByConnection(connID))

	// Remove connection
	cm.RemoveConnection(connID)

	// Verify mappings removed
	assert.Empty(t, cm.GetConnectionByToken(token))
	assert.Empty(t, cm.GetTokenByConnection(connID))
}

// Test: MapToken (legacy method)
// Why: Backward compatibility with Phase 2 code
func TestConnectionManager_MapToken(t *testing.T) {
	cm := NewConnectionManager()

	token := "test-token"
	connID := "conn-1"

	// Add connection first
	cm.AddConnection(connID, nil)

	// Map token
	cm.MapToken(token, connID)

	// Verify mapping
	assert.Equal(t, token, cm.GetTokenByConnection(connID))
	assert.Equal(t, connID, cm.GetConnectionByToken(token))
}

// Test: UnmapToken
// Why: Remove token mapping when player leaves
func TestConnectionManager_UnmapToken(t *testing.T) {
	cm := NewConnectionManager()

	token := "test-token"
	connID := "conn-1"

	cm.AddConnectionWithToken(connID, nil, token)

	// Verify mapping exists
	assert.Equal(t, connID, cm.GetConnectionByToken(token))

	// Unmap token
	cm.UnmapToken(token)

	// Verify token mapping removed
	assert.Empty(t, cm.GetConnectionByToken(token))
	// Note: Connection may or may not exist depending on implementation
	// The important thing is token is unmapped
}

// Test: Multiple device switches
// Why: Player switches devices multiple times
func TestConnectionManager_MultipleDeviceSwitches(t *testing.T) {
	cm := NewConnectionManager()

	token := "test-token"
	conn1ID := "conn-1"
	conn2ID := "conn-2"
	conn3ID := "conn-3"

	// Device 1
	oldID := cm.AddConnectionWithToken(conn1ID, nil, token)
	assert.Empty(t, oldID)
	assert.Equal(t, conn1ID, cm.GetConnectionByToken(token))

	// Device 2 (old connection 1 should be returned and removed from players map)
	oldID = cm.AddConnectionWithToken(conn2ID, nil, token)
	assert.Equal(t, conn1ID, oldID)
	assert.Equal(t, conn2ID, cm.GetConnectionByToken(token))
	// Old connection removed from players map but still in connections (for sending message)
	assert.Empty(t, cm.GetTokenByConnection(conn1ID))

	// Clean up old connection (caller's responsibility)
	cm.RemoveConnection(conn1ID)

	// Device 3 (old connection 2 should be returned)
	oldID = cm.AddConnectionWithToken(conn3ID, nil, token)
	assert.Equal(t, conn2ID, oldID)
	assert.Equal(t, conn3ID, cm.GetConnectionByToken(token))
	assert.Empty(t, cm.GetTokenByConnection(conn2ID))
}

// Test: Different tokens, different connections
// Why: Normal multi-player scenario
func TestConnectionManager_MultiplePlayers(t *testing.T) {
	cm := NewConnectionManager()

	tokens := []string{"token-1", "token-2", "token-3", "token-4"}
	connIDs := []string{"conn-1", "conn-2", "conn-3", "conn-4"}

	// Add all players
	for i := 0; i < 4; i++ {
		cm.AddConnectionWithToken(connIDs[i], nil, tokens[i])
	}

	// Verify each mapping
	for i := 0; i < 4; i++ {
		assert.Equal(t, connIDs[i], cm.GetConnectionByToken(tokens[i]))
		assert.Equal(t, tokens[i], cm.GetTokenByConnection(connIDs[i]))
	}
}
