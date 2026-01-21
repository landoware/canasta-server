package server

import (
	"testing"
	"time"
)

// TestRateLimiter_Allow tests basic rate limiting functionality
func TestRateLimiter_Allow(t *testing.T) {
	limiter := NewRateLimiter(10, time.Second) // 10 requests per second
	connID := "test-conn-1"

	// First 10 requests should be allowed
	for i := 0; i < 10; i++ {
		if !limiter.Allow(connID) {
			t.Errorf("Request %d should be allowed", i+1)
		}
	}

	// 11th request should be denied
	if limiter.Allow(connID) {
		t.Error("11th request should be denied")
	}
}

// TestRateLimiter_WindowReset tests that rate limit window resets after duration
func TestRateLimiter_WindowReset(t *testing.T) {
	limiter := NewRateLimiter(2, 100*time.Millisecond) // 2 requests per 100ms
	connID := "test-conn-2"

	// Use up the limit
	if !limiter.Allow(connID) {
		t.Error("First request should be allowed")
	}
	if !limiter.Allow(connID) {
		t.Error("Second request should be allowed")
	}
	if limiter.Allow(connID) {
		t.Error("Third request should be denied")
	}

	// Wait for window to reset
	time.Sleep(150 * time.Millisecond)

	// Should be allowed again
	if !limiter.Allow(connID) {
		t.Error("Request after window reset should be allowed")
	}
}

// TestRateLimiter_MultipleConnections tests that limits are per-connection
func TestRateLimiter_MultipleConnections(t *testing.T) {
	limiter := NewRateLimiter(5, time.Second)
	conn1 := "conn-1"
	conn2 := "conn-2"

	// Exhaust conn1's limit
	for i := 0; i < 5; i++ {
		limiter.Allow(conn1)
	}
	if limiter.Allow(conn1) {
		t.Error("conn1 should be rate limited")
	}

	// conn2 should still have full limit
	for i := 0; i < 5; i++ {
		if !limiter.Allow(conn2) {
			t.Errorf("conn2 request %d should be allowed", i+1)
		}
	}
}

// TestRateLimiter_Cleanup tests that old connection data is cleaned up
func TestRateLimiter_Cleanup(t *testing.T) {
	limiter := NewRateLimiter(10, 100*time.Millisecond)

	// Add requests for multiple connections
	for i := 0; i < 5; i++ {
		connID := "conn-" + string(rune('0'+i))
		limiter.Allow(connID)
	}

	// Verify we have 5 connections tracked
	limiter.mu.Lock()
	if len(limiter.requests) != 5 {
		t.Errorf("Expected 5 connections, got %d", len(limiter.requests))
	}
	limiter.mu.Unlock()

	// Wait for cleanup
	time.Sleep(200 * time.Millisecond)
	limiter.Cleanup()

	// All connections should be cleaned up since no recent activity
	limiter.mu.Lock()
	if len(limiter.requests) != 0 {
		t.Errorf("Expected 0 connections after cleanup, got %d", len(limiter.requests))
	}
	limiter.mu.Unlock()
}

// TestConnectionHealth_UpdateActivity tests activity tracking
func TestConnectionHealth_UpdateActivity(t *testing.T) {
	health := NewConnectionHealth()
	connID := "test-conn"

	// Update activity
	health.UpdateActivity(connID)

	// Verify last activity was recorded
	health.mu.RLock()
	lastActivity, exists := health.lastActivity[connID]
	health.mu.RUnlock()

	if !exists {
		t.Error("Activity should be recorded")
	}

	if time.Since(lastActivity) > time.Second {
		t.Error("Activity should be recent")
	}
}

// TestConnectionHealth_IsInactive tests timeout detection
func TestConnectionHealth_IsInactive(t *testing.T) {
	health := NewConnectionHealth()
	connID := "test-conn"

	// Brand new connection should not be inactive
	if health.IsInactive(connID, time.Minute) {
		t.Error("New connection should not be inactive")
	}

	// Record activity
	health.UpdateActivity(connID)

	// Still not inactive
	if health.IsInactive(connID, time.Minute) {
		t.Error("Recently active connection should not be inactive")
	}

	// Manually set old activity time
	health.mu.Lock()
	health.lastActivity[connID] = time.Now().Add(-2 * time.Minute)
	health.mu.Unlock()

	// Now should be inactive
	if !health.IsInactive(connID, time.Minute) {
		t.Error("Connection with old activity should be inactive")
	}
}

// TestConnectionHealth_GetInactiveConnections tests batch inactive detection
func TestConnectionHealth_GetInactiveConnections(t *testing.T) {
	health := NewConnectionHealth()

	// Create connections with different activity times
	health.UpdateActivity("active-1")
	health.UpdateActivity("active-2")

	health.mu.Lock()
	health.lastActivity["inactive-1"] = time.Now().Add(-6 * time.Minute)
	health.lastActivity["inactive-2"] = time.Now().Add(-10 * time.Minute)
	health.mu.Unlock()

	// Get inactive connections (5 minute timeout)
	inactive := health.GetInactiveConnections(5 * time.Minute)

	if len(inactive) != 2 {
		t.Errorf("Expected 2 inactive connections, got %d", len(inactive))
	}

	// Verify inactive-1 and inactive-2 are in the list
	found1, found2 := false, false
	for _, id := range inactive {
		if id == "inactive-1" {
			found1 = true
		}
		if id == "inactive-2" {
			found2 = true
		}
	}

	if !found1 || !found2 {
		t.Error("Should find both inactive connections")
	}
}

// TestConnectionHealth_RemoveConnection tests cleanup on disconnect
func TestConnectionHealth_RemoveConnection(t *testing.T) {
	health := NewConnectionHealth()
	connID := "test-conn"

	// Add activity
	health.UpdateActivity(connID)

	health.mu.RLock()
	_, exists := health.lastActivity[connID]
	health.mu.RUnlock()
	if !exists {
		t.Error("Connection should exist")
	}

	// Remove connection
	health.RemoveConnection(connID)

	health.mu.RLock()
	_, exists = health.lastActivity[connID]
	health.mu.RUnlock()
	if exists {
		t.Error("Connection should be removed")
	}
}

// TestValidateMessageType tests message type validation
func TestValidateMessageType(t *testing.T) {
	// Valid types
	validTypes := []string{"ping", "create_game", "join_game", "reconnect",
		"set_ready", "update_team_order", "leave_game", "execute_move"}

	for _, msgType := range validTypes {
		if err := ValidateMessageType(msgType); err != nil {
			t.Errorf("Valid message type '%s' should not error", msgType)
		}
	}

	// Invalid types
	invalidTypes := []string{"invalid", "create", "PING", ""}
	for _, msgType := range invalidTypes {
		if err := ValidateMessageType(msgType); err == nil {
			t.Errorf("Invalid message type '%s' should error", msgType)
		}
	}
}

// TestValidateUsername tests username validation
func TestValidateUsername(t *testing.T) {
	// Valid usernames
	validNames := []string{"Alice", "Bob123", "Player 1", "用户"}
	for _, name := range validNames {
		if err := ValidateUsername(name); err != nil {
			t.Errorf("Valid username '%s' should not error: %v", name, err)
		}
	}

	// Invalid usernames
	if err := ValidateUsername(""); err == nil {
		t.Error("Empty username should error")
	}
	if err := ValidateUsername("ThisUsernameIsWayTooLongAndShouldFail"); err == nil {
		t.Error("Username >20 chars should error")
	}
}
