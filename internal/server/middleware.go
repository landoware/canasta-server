package server

import (
	"fmt"
	"sync"
	"time"
)

// RateLimiter implements per-connection rate limiting using a sliding window algorithm
// Why sliding window: Prevents burst attacks while allowing consistent legitimate traffic
// Why per-connection: One abusive client shouldn't affect others
type RateLimiter struct {
	maxRequests int                    // Maximum requests allowed per window
	window      time.Duration          // Time window for rate limiting
	requests    map[string][]time.Time // connectionID -> timestamps of recent requests
	mu          sync.Mutex             // Protects concurrent access to requests map
}

// NewRateLimiter creates a new rate limiter
// maxRequests: number of requests allowed per window
// window: duration of the sliding window (e.g., 1 second for 10 req/sec)
func NewRateLimiter(maxRequests int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		maxRequests: maxRequests,
		window:      window,
		requests:    make(map[string][]time.Time),
	}
}

// Allow checks if a connection is allowed to send a message
// Returns true if allowed, false if rate limited
// Why sliding window: We remove old timestamps and count remaining ones
// This provides smoother rate limiting than fixed windows
func (r *RateLimiter) Allow(connectionID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-r.window)

	// Get or create timestamp list for this connection
	timestamps := r.requests[connectionID]

	// Remove timestamps outside the window
	// Why filter: Keep memory usage bounded and only count recent requests
	validTimestamps := make([]time.Time, 0, len(timestamps))
	for _, ts := range timestamps {
		if ts.After(cutoff) {
			validTimestamps = append(validTimestamps, ts)
		}
	}

	// Check if limit exceeded
	if len(validTimestamps) >= r.maxRequests {
		// Update with filtered list (cleanup old timestamps)
		r.requests[connectionID] = validTimestamps
		return false
	}

	// Add current timestamp and allow
	validTimestamps = append(validTimestamps, now)
	r.requests[connectionID] = validTimestamps
	return true
}

// Cleanup removes old connection data to prevent memory leaks
// Should be called periodically or when connections close
// Why: Disconnected connections leave data in the map
func (r *RateLimiter) Cleanup() {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-r.window)

	// Remove connections with no recent activity
	for connID, timestamps := range r.requests {
		// If all timestamps are old, remove the connection entirely
		allOld := true
		for _, ts := range timestamps {
			if ts.After(cutoff) {
				allOld = false
				break
			}
		}
		if allOld {
			delete(r.requests, connID)
		}
	}
}

// RemoveConnection immediately removes rate limit data for a connection
// Should be called when a websocket disconnects
func (r *RateLimiter) RemoveConnection(connectionID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.requests, connectionID)
}

// ConnectionHealth tracks last activity time for each connection
// Used for detecting dead/inactive connections
// Why separate from RateLimiter: Different concerns - health vs abuse prevention
type ConnectionHealth struct {
	lastActivity map[string]time.Time // connectionID -> last message time
	mu           sync.RWMutex         // Read-heavy workload, so RWMutex is better
}

// NewConnectionHealth creates a new connection health tracker
func NewConnectionHealth() *ConnectionHealth {
	return &ConnectionHealth{
		lastActivity: make(map[string]time.Time),
	}
}

// UpdateActivity records that a connection is active
// Should be called on every message received
// Why on every message: Most accurate way to detect inactive connections
func (h *ConnectionHealth) UpdateActivity(connectionID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.lastActivity[connectionID] = time.Now()
}

// IsInactive checks if a connection has been inactive for longer than timeout
// Returns true if connection should be considered dead
// Why timeout parameter: Different use cases may need different timeouts
func (h *ConnectionHealth) IsInactive(connectionID string, timeout time.Duration) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	lastActivity, exists := h.lastActivity[connectionID]
	if !exists {
		// Connection not tracked yet - not inactive
		return false
	}

	return time.Since(lastActivity) > timeout
}

// GetInactiveConnections returns all connections inactive longer than timeout
// Used for batch cleanup operations
// Why batch: More efficient than checking each connection individually
func (h *ConnectionHealth) GetInactiveConnections(timeout time.Duration) []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	inactive := make([]string, 0)
	now := time.Now()

	for connID, lastActivity := range h.lastActivity {
		if now.Sub(lastActivity) > timeout {
			inactive = append(inactive, connID)
		}
	}

	return inactive
}

// RemoveConnection removes health tracking for a connection
// Should be called when websocket disconnects
func (h *ConnectionHealth) RemoveConnection(connectionID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.lastActivity, connectionID)
}

// ValidateMessageType checks if a message type is recognized
// Why: Return clear error for typos/invalid message types
func ValidateMessageType(msgType string) error {
	validTypes := map[string]bool{
		"ping":              true,
		"create_game":       true,
		"join_game":         true,
		"reconnect":         true,
		"set_ready":         true,
		"update_team_order": true,
		"leave_game":        true,
		"execute_move":      true,
	}

	if !validTypes[msgType] {
		return fmt.Errorf("INVALID_MESSAGE_TYPE: Unknown message type '%s'", msgType)
	}
	return nil
}

// ValidateUsername checks username requirements
// Why: Centralize validation logic, consistent error messages
func ValidateUsername(username string) error {
	if len(username) == 0 {
		return fmt.Errorf("USERNAME_INVALID: Username cannot be empty")
	}
	if len(username) > 20 {
		return fmt.Errorf("USERNAME_INVALID: Username too long (max 20 characters)")
	}
	return nil
}
