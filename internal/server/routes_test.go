package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/coder/websocket"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/assert"
)

func TestHandler(t *testing.T) {
	s := &Server{}
	server := httptest.NewServer(http.HandlerFunc(s.HelloWorldHandler))
	defer server.Close()
	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("error making request to server. Err: %v", err)
	}
	defer resp.Body.Close()
	// Assertions
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status OK; got %v", resp.Status)
	}
	expected := "{\"message\":\"Hello World\"}"
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("error reading response body. Err: %v", err)
	}
	if expected != string(body) {
		t.Errorf("expected response body to be %v; got %v", expected, string(body))
	}
}

func TestWebSocketPingPing(t *testing.T) {
	assert := assert.New(t)

	ctx := context.Background()
	_, url, cleanup := setupTestServer()
	defer cleanup()

	conn, _, err := websocket.Dial(ctx, url, nil)
	assert.NoError(err)
	defer conn.Close(websocket.StatusNormalClosure, "")

	ping := ClientMessage{
		Type: "ping",
	}

	data, err := json.Marshal(ping)
	assert.NoError(err)

	// Send it
	err = conn.Write(ctx, websocket.MessageText, data)
	assert.NoErrorf(err, "Failed to send ping")

	_, responseData, err := conn.Read(ctx)
	assert.NoErrorf(err, "Failed to read response")

	var response ServerMessage
	err = json.Unmarshal(responseData, &response)
	assert.NoErrorf(err, "Failed to parse response")

	assert.Equal("pong", response.Type)
}

func TestWebSocketInvalidJSON(t *testing.T) {
	assert := assert.New(t)

	ctx := context.Background()
	_, url, cleanup := setupTestServer()
	defer cleanup()

	conn, _, err := websocket.Dial(ctx, url, nil)
	assert.NoError(err)
	defer conn.Close(websocket.StatusNormalClosure, "")

	// Send it
	err = conn.Write(ctx, websocket.MessageText, []byte("junk"))
	assert.NoErrorf(err, "Failed to send ping")

	_, responseData, err := conn.Read(ctx)
	assert.NoErrorf(err, "Failed to read response")

	var response ServerMessage
	err = json.Unmarshal(responseData, &response)
	assert.NoErrorf(err, "Failed to parse response")

	assert.Equal("error", response.Type)

	// Ping to ensure the connection didn't close
	ping := ClientMessage{
		Type: "ping",
	}

	data, err := json.Marshal(ping)
	assert.NoError(err)

	err = conn.Write(ctx, websocket.MessageText, data)
	assert.NoErrorf(err, "Failed to send ping")
}

func TestWebsocketConnectionRegistration(t *testing.T) {
	assert := assert.New(t)

	ctx := context.Background()
	s, url, cleanup := setupTestServer()
	defer cleanup()

	s.connectionManager.mu.RLock()
	initialCount := len(s.connectionManager.connections)
	s.connectionManager.mu.RUnlock()
	assert.Equal(0, initialCount)

	// Connect
	conn, _, err := websocket.Dial(ctx, url, nil)
	assert.NoError(err)

	s.connectionManager.mu.RLock()
	connectionCount := len(s.connectionManager.connections)
	s.connectionManager.mu.RUnlock()
	assert.Equal(1, connectionCount)

	// Disconnect
	conn.Close(websocket.StatusNormalClosure, "")

	s.connectionManager.mu.RLock()
	finalCount := len(s.connectionManager.connections)
	s.connectionManager.mu.RUnlock()
	assert.Equal(0, finalCount)

}

func TestWebSocketMultipleConnections(t *testing.T) {
	assert := assert.New(t)

	ctx := context.Background()
	s, url, cleanup := setupTestServer()
	defer cleanup()

	// Connect 4 clients
	connections := make([]*websocket.Conn, 4)
	for i := range 4 {
		conn, _, err := websocket.Dial(ctx, url, nil)
		assert.NoError(err)
		connections[i] = conn
		defer conn.Close(websocket.StatusNormalClosure, "")
	}

	s.connectionManager.mu.RLock()
	count := len(s.connectionManager.connections)
	s.connectionManager.mu.RUnlock()

	assert.Equal(4, count)

	for i, conn := range connections {
		pingMsg := ClientMessage{Type: "ping", Payload: json.RawMessage(`{}`)}
		data, _ := json.Marshal(pingMsg)

		err := conn.Write(ctx, websocket.MessageText, data)
		if err != nil {
			t.Errorf("Client %d failed to send ping: %v", i, err)
		}

		_, responseData, err := conn.Read(ctx)
		if err != nil {
			t.Errorf("Client %d failed to read response: %v", i, err)
		}

		var response ServerMessage
		json.Unmarshal(responseData, &response)

		assert.Equal("pong", response.Type)
	}
}

func setupTestServer() (*Server, string, func()) {
	// Create in-memory database for tests
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		panic(err)
	}

	// Set up goose and run migrations
	if err := goose.SetDialect("sqlite3"); err != nil {
		panic(err)
	}
	if err := goose.Up(db, "../../db/migrations"); err != nil {
		panic(err)
	}

	s := &Server{
		connectionManager:  NewConnectionManager(),
		gameManager:        NewGameManager(),
		sessionManager:     NewSessionManager(),
		persistenceManager: NewPersistenceManager(db), // Phase 6: Add PersistenceManager
	}

	server := httptest.NewServer(http.HandlerFunc(s.websocketHandler))
	url := "ws" + strings.TrimPrefix(server.URL, "http") + "/websocket"

	cleanup := func() {
		server.Close()
		db.Close()
		// Clean up any test database files
		os.Remove("test_persistence.db")
	}

	return s, url, cleanup
}
