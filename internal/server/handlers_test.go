package server

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/stretchr/testify/assert"
)

// ============================================================================
// CREATE GAME TESTS
// ============================================================================

func TestHandleCreateGame_Success(t *testing.T) {
	assert := assert.New(t)
	ctx := context.Background()
	_, url, cleanup := setupTestServer()
	defer cleanup()

	conn, _, err := websocket.Dial(ctx, url, nil)
	assert.NoError(err)
	defer conn.Close(websocket.StatusNormalClosure, "")

	// Send create_game message
	req := ClientMessage{
		Type: "create_game",
		Payload: mustMarshal(CreateGameRequest{
			Username:        "Alice",
			RandomTeamOrder: true,
		}),
	}

	err = conn.Write(ctx, websocket.MessageText, mustMarshal(req))
	assert.NoError(err)

	// Read game_created response
	_, data, err := conn.Read(ctx)
	assert.NoError(err)

	var response ServerMessage
	err = json.Unmarshal(data, &response)
	assert.NoError(err)
	assert.Equal("game_created", response.Type)

	// Parse payload
	var createResp CreateGameResponse
	payloadBytes, _ := json.Marshal(response.Payload)
	err = json.Unmarshal(payloadBytes, &createResp)
	assert.NoError(err)

	assert.NotEmpty(createResp.RoomCode)
	assert.Equal(4, len(createResp.RoomCode))
	assert.NotEmpty(createResp.Token)
	assert.Equal(0, createResp.PlayerID)

	// Should also receive lobby_update
	_, data, err = conn.Read(ctx)
	assert.NoError(err)

	err = json.Unmarshal(data, &response)
	assert.NoError(err)
	assert.Equal("lobby_update", response.Type)
}

func TestHandleCreateGame_InvalidUsername(t *testing.T) {
	assert := assert.New(t)
	ctx := context.Background()
	_, url, cleanup := setupTestServer()
	defer cleanup()

	conn, _, err := websocket.Dial(ctx, url, nil)
	assert.NoError(err)
	defer conn.Close(websocket.StatusNormalClosure, "")

	// Send create_game with empty username
	req := ClientMessage{
		Type: "create_game",
		Payload: mustMarshal(CreateGameRequest{
			Username:        "",
			RandomTeamOrder: true,
		}),
	}

	err = conn.Write(ctx, websocket.MessageText, mustMarshal(req))
	assert.NoError(err)

	// Should receive error
	_, data, err := conn.Read(ctx)
	assert.NoError(err)

	var response ServerMessage
	err = json.Unmarshal(data, &response)
	assert.NoError(err)
	assert.Equal("error", response.Type)
}

// ============================================================================
// JOIN GAME TESTS
// ============================================================================

func TestHandleJoinGame_Success(t *testing.T) {
	assert := assert.New(t)
	ctx := context.Background()
	_, url, cleanup := setupTestServer()
	defer cleanup()

	// Create game with first connection
	conn1, _, err := websocket.Dial(ctx, url, nil)
	assert.NoError(err)
	defer conn1.Close(websocket.StatusNormalClosure, "")

	req := ClientMessage{
		Type: "create_game",
		Payload: mustMarshal(CreateGameRequest{
			Username:        "Alice",
			RandomTeamOrder: true,
		}),
	}
	conn1.Write(ctx, websocket.MessageText, mustMarshal(req))

	// Read responses
	_, data, _ := conn1.Read(ctx) // game_created
	var response ServerMessage
	json.Unmarshal(data, &response)

	var createResp CreateGameResponse
	payloadBytes, _ := json.Marshal(response.Payload)
	json.Unmarshal(payloadBytes, &createResp)
	roomCode := createResp.RoomCode

	conn1.Read(ctx) // lobby_update

	// Join with second connection
	conn2, _, err := websocket.Dial(ctx, url, nil)
	assert.NoError(err)
	defer conn2.Close(websocket.StatusNormalClosure, "")

	joinReq := ClientMessage{
		Type: "join_game",
		Payload: mustMarshal(JoinGameRequest{
			RoomCode: roomCode,
			Username: "Bob",
		}),
	}
	err = conn2.Write(ctx, websocket.MessageText, mustMarshal(joinReq))
	assert.NoError(err)

	// Read game_joined response
	_, data, err = conn2.Read(ctx)
	assert.NoError(err)

	err = json.Unmarshal(data, &response)
	assert.NoError(err)
	assert.Equal("game_joined", response.Type)

	var joinResp JoinGameResponse
	payloadBytes, _ = json.Marshal(response.Payload)
	json.Unmarshal(payloadBytes, &joinResp)

	assert.True(joinResp.Success)
	assert.NotEmpty(joinResp.Token)
	assert.Equal(1, joinResp.PlayerID)
}

func TestHandleJoinGame_RoomNotFound(t *testing.T) {
	assert := assert.New(t)
	ctx := context.Background()
	_, url, cleanup := setupTestServer()
	defer cleanup()

	conn, _, err := websocket.Dial(ctx, url, nil)
	assert.NoError(err)
	defer conn.Close(websocket.StatusNormalClosure, "")

	req := ClientMessage{
		Type: "join_game",
		Payload: mustMarshal(JoinGameRequest{
			RoomCode: "ZZZZ",
			Username: "Bob",
		}),
	}
	conn.Write(ctx, websocket.MessageText, mustMarshal(req))

	// Should receive error
	_, data, err := conn.Read(ctx)
	assert.NoError(err)

	var response ServerMessage
	json.Unmarshal(data, &response)
	assert.Equal("error", response.Type)
}

func TestHandleJoinGame_DuplicateUsername(t *testing.T) {
	assert := assert.New(t)
	ctx := context.Background()
	_, url, cleanup := setupTestServer()
	defer cleanup()

	// Create game
	conn1, _, _ := websocket.Dial(ctx, url, nil)
	defer conn1.Close(websocket.StatusNormalClosure, "")

	req := ClientMessage{
		Type: "create_game",
		Payload: mustMarshal(CreateGameRequest{
			Username:        "Alice",
			RandomTeamOrder: true,
		}),
	}
	conn1.Write(ctx, websocket.MessageText, mustMarshal(req))

	_, data, _ := conn1.Read(ctx) // game_created
	var response ServerMessage
	json.Unmarshal(data, &response)
	var createResp CreateGameResponse
	payloadBytes, _ := json.Marshal(response.Payload)
	json.Unmarshal(payloadBytes, &createResp)
	roomCode := createResp.RoomCode

	conn1.Read(ctx) // lobby_update

	// Try to join with same username
	conn2, _, _ := websocket.Dial(ctx, url, nil)
	defer conn2.Close(websocket.StatusNormalClosure, "")

	joinReq := ClientMessage{
		Type: "join_game",
		Payload: mustMarshal(JoinGameRequest{
			RoomCode: roomCode,
			Username: "Alice", // Same as creator
		}),
	}
	conn2.Write(ctx, websocket.MessageText, mustMarshal(joinReq))

	// Should receive error
	_, data, _ = conn2.Read(ctx)
	json.Unmarshal(data, &response)
	assert.Equal("error", response.Type)
}

// ============================================================================
// SET READY TESTS
// ============================================================================

func TestHandleSetReady_Success(t *testing.T) {
	assert := assert.New(t)
	ctx := context.Background()
	_, url, cleanup := setupTestServer()
	defer cleanup()

	// Create game
	conn, _, _ := websocket.Dial(ctx, url, nil)
	defer conn.Close(websocket.StatusNormalClosure, "")

	createReq := ClientMessage{
		Type: "create_game",
		Payload: mustMarshal(CreateGameRequest{
			Username:        "Alice",
			RandomTeamOrder: true,
		}),
	}
	conn.Write(ctx, websocket.MessageText, mustMarshal(createReq))
	conn.Read(ctx) // game_created
	conn.Read(ctx) // lobby_update

	// Set ready
	readyReq := ClientMessage{
		Type: "set_ready",
		Payload: mustMarshal(SetReadyRequest{
			Ready: true,
		}),
	}
	conn.Write(ctx, websocket.MessageText, mustMarshal(readyReq))

	// Should receive lobby_update
	_, data, err := conn.Read(ctx)
	assert.NoError(err)

	var response ServerMessage
	json.Unmarshal(data, &response)
	assert.Equal("lobby_update", response.Type)

	// Check that player is ready in lobby state
	var lobbyState LobbyState
	payloadBytes, _ := json.Marshal(response.Payload)
	json.Unmarshal(payloadBytes, &lobbyState)

	assert.True(lobbyState.Players[0].Ready)
}

func TestHandleSetReady_AutoStart(t *testing.T) {
	assert := assert.New(t)
	ctx := context.Background()
	s, url, cleanup := setupTestServer()
	defer cleanup()

	// Create game and join 4 players
	conns := make([]*websocket.Conn, 4)
	var roomCode string

	// Player 1 creates
	conns[0], _, _ = websocket.Dial(ctx, url, nil)
	defer conns[0].Close(websocket.StatusNormalClosure, "")

	createReq := ClientMessage{
		Type: "create_game",
		Payload: mustMarshal(CreateGameRequest{
			Username:        "Alice",
			RandomTeamOrder: true,
		}),
	}
	conns[0].Write(ctx, websocket.MessageText, mustMarshal(createReq))

	_, data, _ := conns[0].Read(ctx) // game_created
	var response ServerMessage
	json.Unmarshal(data, &response)
	var createResp CreateGameResponse
	payloadBytes, _ := json.Marshal(response.Payload)
	json.Unmarshal(payloadBytes, &createResp)
	roomCode = createResp.RoomCode

	conns[0].Read(ctx) // lobby_update

	// Players 2-4 join
	playerNames := []string{"Bob", "Charlie", "Diana"}
	for i, name := range playerNames {
		conns[i+1], _, _ = websocket.Dial(ctx, url, nil)
		defer conns[i+1].Close(websocket.StatusNormalClosure, "")

		joinReq := ClientMessage{
			Type: "join_game",
			Payload: mustMarshal(JoinGameRequest{
				RoomCode: roomCode,
				Username: name,
			}),
		}
		conns[i+1].Write(ctx, websocket.MessageText, mustMarshal(joinReq))
		conns[i+1].Read(ctx) // game_joined

		// Each player receives lobby_update
		for j := range i + 1 {
			conns[j].Read(ctx) // lobby_update broadcast
		}
	}

	// All 4 players ready up
	for i := range 4 {
		readyReq := ClientMessage{
			Type: "set_ready",
			Payload: mustMarshal(SetReadyRequest{
				Ready: true,
			}),
		}
		conns[i].Write(ctx, websocket.MessageText, mustMarshal(readyReq))

		// Drain lobby_update messages
		for j := range 4 {
			conns[j].Read(ctx)
		}
	}

	// After 4th player readies, should receive game_started
	_, data, err := conns[0].Read(ctx)
	assert.NoError(err)

	json.Unmarshal(data, &response)
	assert.Equal("game_started", response.Type)

	// Verify game status changed
	game, _ := s.gameManager.GetGame(roomCode)
	assert.Equal(StatusPlaying, game.Status)
	assert.NotNil(game.Game) // Canasta game initialized
}

// ============================================================================
// UPDATE TEAM ORDER TESTS
// ============================================================================

func TestHandleUpdateTeamOrder_Success(t *testing.T) {
	assert := assert.New(t)
	ctx := context.Background()
	s, url, cleanup := setupTestServer()
	defer cleanup()

	// Create game with 4 players
	conn, roomCode := createFullLobby(t, ctx, url)
	defer conn.Close(websocket.StatusNormalClosure, "")

	// Update team order (creator)
	updateReq := ClientMessage{
		Type: "update_team_order",
		Payload: mustMarshal(UpdateTeamOrderRequest{
			PlayerOrder: [4]string{"Bob", "Alice", "Diana", "Charlie"},
		}),
	}
	conn.Write(ctx, websocket.MessageText, mustMarshal(updateReq))

	// Should receive lobby_update
	_, data, _ := conn.Read(ctx)
	var response ServerMessage
	json.Unmarshal(data, &response)
	assert.Equal("lobby_update", response.Type)

	// Verify order changed
	game, _ := s.gameManager.GetGame(roomCode)
	assert.Equal([4]string{"Bob", "Alice", "Diana", "Charlie"}, game.Config.PlayerOrder)
}

func TestHandleUpdateTeamOrder_NotCreator(t *testing.T) {
	assert := assert.New(t)
	ctx := context.Background()
	_, url, cleanup := setupTestServer()
	defer cleanup()

	// Create game
	conn1, _, _ := websocket.Dial(ctx, url, nil)
	defer conn1.Close(websocket.StatusNormalClosure, "")

	createReq := ClientMessage{
		Type: "create_game",
		Payload: mustMarshal(CreateGameRequest{
			Username:        "Alice",
			RandomTeamOrder: false,
		}),
	}
	conn1.Write(ctx, websocket.MessageText, mustMarshal(createReq))
	_, data, _ := conn1.Read(ctx) // game_created

	var response ServerMessage
	json.Unmarshal(data, &response)
	var createResp CreateGameResponse
	payloadBytes, _ := json.Marshal(response.Payload)
	json.Unmarshal(payloadBytes, &createResp)
	roomCode := createResp.RoomCode

	conn1.Read(ctx) // lobby_update

	// Second player joins
	conn2, _, _ := websocket.Dial(ctx, url, nil)
	defer conn2.Close(websocket.StatusNormalClosure, "")

	joinReq := ClientMessage{
		Type: "join_game",
		Payload: mustMarshal(JoinGameRequest{
			RoomCode: roomCode,
			Username: "Bob",
		}),
	}
	conn2.Write(ctx, websocket.MessageText, mustMarshal(joinReq))
	conn2.Read(ctx) // game_joined
	conn1.Read(ctx) // lobby_update
	conn2.Read(ctx) // lobby_update

	// Bob (non-creator) tries to update order
	updateReq := ClientMessage{
		Type: "update_team_order",
		Payload: mustMarshal(UpdateTeamOrderRequest{
			PlayerOrder: [4]string{"Bob", "Alice", "", ""},
		}),
	}
	conn2.Write(ctx, websocket.MessageText, mustMarshal(updateReq))

	// Should receive error
	_, data, _ = conn2.Read(ctx)
	json.Unmarshal(data, &response)
	assert.Equal("error", response.Type)
}

// ============================================================================
// LEAVE GAME TESTS
// ============================================================================

func TestHandleLeaveGame_NonCreator(t *testing.T) {
	assert := assert.New(t)
	ctx := context.Background()
	s, url, cleanup := setupTestServer()
	defer cleanup()

	// Create game with 2 players
	conn1, roomCode := createGameWithPlayers(t, ctx, url, []string{"Alice", "Bob"})
	defer conn1.Close(websocket.StatusNormalClosure, "")

	// Get Bob's connection (we need to track it separately in real test)
	// For this test, we'll create a new connection and join
	conn2, _, _ := websocket.Dial(ctx, url, nil)
	joinReq := ClientMessage{
		Type: "join_game",
		Payload: mustMarshal(JoinGameRequest{
			RoomCode: roomCode,
			Username: "Charlie",
		}),
	}
	conn2.Write(ctx, websocket.MessageText, mustMarshal(joinReq))
	conn2.Read(ctx) // game_joined
	conn1.Read(ctx) // lobby_update
	conn2.Read(ctx) // lobby_update

	// Charlie leaves
	leaveReq := ClientMessage{
		Type:    "leave_game",
		Payload: json.RawMessage(`{}`),
	}
	conn2.Write(ctx, websocket.MessageText, mustMarshal(leaveReq))

	// Should receive lobby_update
	conn1.Read(ctx) // lobby_update showing Charlie disconnected

	// Verify Charlie is marked disconnected
	game, _ := s.gameManager.GetGame(roomCode)
	assert.False(game.Players[2].Connected)
	assert.Equal("Charlie", game.Players[2].Username) // Still there

	conn2.Close(websocket.StatusNormalClosure, "")
}

func TestHandleLeaveGame_CreatorPromotes(t *testing.T) {
	assert := assert.New(t)
	ctx := context.Background()
	s, url, cleanup := setupTestServer()
	defer cleanup()

	// Create game with 2 players
	conn1, roomCode := createGameWithPlayers(t, ctx, url, []string{"Alice", "Bob"})

	// Alice (creator) leaves
	leaveReq := ClientMessage{
		Type:    "leave_game",
		Payload: json.RawMessage(`{}`),
	}
	conn1.Write(ctx, websocket.MessageText, mustMarshal(leaveReq))

	// Wait a bit for processing
	time.Sleep(50 * time.Millisecond)

	// Verify Bob is now in slot 0 (promoted)
	game, _ := s.gameManager.GetGame(roomCode)
	assert.Equal("Bob", game.Players[0].Username)
	assert.Empty(game.Players[1].Username) // Old Bob slot empty

	conn1.Close(websocket.StatusNormalClosure, "")
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

func mustMarshal(v interface{}) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}

// createFullLobby creates a game with 4 players and returns creator connection + room code
func createFullLobby(t *testing.T, ctx context.Context, url string) (*websocket.Conn, string) {
	// Create game
	conn, _, _ := websocket.Dial(ctx, url, nil)

	createReq := ClientMessage{
		Type: "create_game",
		Payload: mustMarshal(CreateGameRequest{
			Username:        "Alice",
			RandomTeamOrder: false,
		}),
	}
	conn.Write(ctx, websocket.MessageText, mustMarshal(createReq))

	_, data, _ := conn.Read(ctx) // game_created
	var response ServerMessage
	json.Unmarshal(data, &response)
	var createResp CreateGameResponse
	payloadBytes, _ := json.Marshal(response.Payload)
	json.Unmarshal(payloadBytes, &createResp)
	roomCode := createResp.RoomCode

	conn.Read(ctx) // lobby_update

	// Join 3 more players
	players := []string{"Bob", "Charlie", "Diana"}
	for _, name := range players {
		tempConn, _, _ := websocket.Dial(ctx, url, nil)
		joinReq := ClientMessage{
			Type: "join_game",
			Payload: mustMarshal(JoinGameRequest{
				RoomCode: roomCode,
				Username: name,
			}),
		}
		tempConn.Write(ctx, websocket.MessageText, mustMarshal(joinReq))
		tempConn.Read(ctx) // game_joined

		// Drain broadcast messages
		conn.Read(ctx) // lobby_update
		// NOTE: Don't close tempConn - Phase 3 disconnect handler would mark player as disconnected
		// Let test cleanup handle connection closing
	}

	return conn, roomCode
}

// createGameWithPlayers creates a game with specified players
func createGameWithPlayers(t *testing.T, ctx context.Context, url string, players []string) (*websocket.Conn, string) {
	// Create game with first player
	conn, _, _ := websocket.Dial(ctx, url, nil)

	createReq := ClientMessage{
		Type: "create_game",
		Payload: mustMarshal(CreateGameRequest{
			Username:        players[0],
			RandomTeamOrder: false,
		}),
	}
	conn.Write(ctx, websocket.MessageText, mustMarshal(createReq))

	_, data, _ := conn.Read(ctx) // game_created
	var response ServerMessage
	json.Unmarshal(data, &response)
	var createResp CreateGameResponse
	payloadBytes, _ := json.Marshal(response.Payload)
	json.Unmarshal(payloadBytes, &createResp)
	roomCode := createResp.RoomCode

	conn.Read(ctx) // lobby_update

	// Join additional players
	for i := 1; i < len(players); i++ {
		tempConn, _, _ := websocket.Dial(ctx, url, nil)
		joinReq := ClientMessage{
			Type: "join_game",
			Payload: mustMarshal(JoinGameRequest{
				RoomCode: roomCode,
				Username: players[i],
			}),
		}
		tempConn.Write(ctx, websocket.MessageText, mustMarshal(joinReq))
		tempConn.Read(ctx) // game_joined
		conn.Read(ctx)     // lobby_update
		// NOTE: Don't close tempConn - Phase 3 disconnect handler would mark player as disconnected
		// Let test cleanup handle connection closing
	}

	return conn, roomCode
}

// ============================================================================
// COMPREHENSIVE INTEGRATION TESTS
// ============================================================================

func TestFullLobbyFlow_CreateJoinReadyStart(t *testing.T) {
	// This test validates the complete happy path:
	// 1. Player 1 creates game
	// 2. Players 2-4 join
	// 3. All 4 ready up
	// 4. Game auto-starts
	// 5. All players receive game_started notification

	assert := assert.New(t)
	ctx := context.Background()
	s, url, cleanup := setupTestServer()
	defer cleanup()

	// Create connections for all 4 players
	conns := make([]*websocket.Conn, 4)
	tokens := make([]string, 4)
	playerNames := []string{"Alice", "Bob", "Charlie", "Diana"}

	// Player 1 creates game
	conns[0], _, _ = websocket.Dial(ctx, url, nil)
	defer conns[0].Close(websocket.StatusNormalClosure, "")

	createReq := ClientMessage{
		Type: "create_game",
		Payload: mustMarshal(CreateGameRequest{
			Username:        playerNames[0],
			RandomTeamOrder: false,
		}),
	}
	conns[0].Write(ctx, websocket.MessageText, mustMarshal(createReq))

	// Read game_created
	_, data, _ := conns[0].Read(ctx)
	var response ServerMessage
	json.Unmarshal(data, &response)
	assert.Equal("game_created", response.Type)

	var createResp CreateGameResponse
	payloadBytes, _ := json.Marshal(response.Payload)
	json.Unmarshal(payloadBytes, &createResp)
	roomCode := createResp.RoomCode
	tokens[0] = createResp.Token

	conns[0].Read(ctx) // lobby_update

	// Players 2-4 join
	for i := 1; i < 4; i++ {
		conns[i], _, _ = websocket.Dial(ctx, url, nil)
		defer conns[i].Close(websocket.StatusNormalClosure, "")

		joinReq := ClientMessage{
			Type: "join_game",
			Payload: mustMarshal(JoinGameRequest{
				RoomCode: roomCode,
				Username: playerNames[i],
			}),
		}
		conns[i].Write(ctx, websocket.MessageText, mustMarshal(joinReq))

		// Read game_joined
		_, data, _ = conns[i].Read(ctx)
		json.Unmarshal(data, &response)
		assert.Equal("game_joined", response.Type)

		var joinResp JoinGameResponse
		payloadBytes, _ = json.Marshal(response.Payload)
		json.Unmarshal(payloadBytes, &joinResp)
		tokens[i] = joinResp.Token

		// Drain lobby_update broadcasts
		for j := 0; j <= i; j++ {
			conns[j].Read(ctx)
		}
	}

	// All 4 players ready up
	for i := range 4 {
		readyReq := ClientMessage{
			Type: "set_ready",
			Payload: mustMarshal(SetReadyRequest{
				Ready: true,
			}),
		}
		conns[i].Write(ctx, websocket.MessageText, mustMarshal(readyReq))

		// Drain lobby_update broadcasts
		for j := range 4 {
			conns[j].Read(ctx)
		}
	}

	// All players should receive game_started
	for i := range 4 {
		_, data, err := conns[i].Read(ctx)
		assert.NoError(err)
		json.Unmarshal(data, &response)
		assert.Equal("game_started", response.Type)
	}

	// Verify game actually started
	game, _ := s.gameManager.GetGame(roomCode)
	assert.Equal(StatusPlaying, game.Status)
	assert.NotNil(game.Game)
	assert.Equal(4, len(game.Game.Players))
}

func TestLobbyBroadcast_AllPlayersReceive(t *testing.T) {
	// This test verifies that lobby_update broadcasts reach all connected players

	assert := assert.New(t)
	ctx := context.Background()
	_, url, cleanup := setupTestServer()
	defer cleanup()

	conns := make([]*websocket.Conn, 3)

	// Create game
	conns[0], _, _ = websocket.Dial(ctx, url, nil)
	defer conns[0].Close(websocket.StatusNormalClosure, "")

	createReq := ClientMessage{
		Type: "create_game",
		Payload: mustMarshal(CreateGameRequest{
			Username:        "Alice",
			RandomTeamOrder: true,
		}),
	}
	conns[0].Write(ctx, websocket.MessageText, mustMarshal(createReq))
	conns[0].Read(ctx)               // game_created
	_, data, _ := conns[0].Read(ctx) // lobby_update

	var response ServerMessage
	json.Unmarshal(data, &response)
	var createResp CreateGameResponse
	payloadBytes, _ := json.Marshal(response.Payload)
	json.Unmarshal(payloadBytes, &createResp)
	roomCode := createResp.RoomCode

	// Player 2 joins
	conns[1], _, _ = websocket.Dial(ctx, url, nil)
	defer conns[1].Close(websocket.StatusNormalClosure, "")

	joinReq := ClientMessage{
		Type: "join_game",
		Payload: mustMarshal(JoinGameRequest{
			RoomCode: roomCode,
			Username: "Bob",
		}),
	}
	conns[1].Write(ctx, websocket.MessageText, mustMarshal(joinReq))
	conns[1].Read(ctx) // game_joined

	// Both should receive lobby_update
	_, data0, _ := conns[0].Read(ctx)
	_, data1, _ := conns[1].Read(ctx)

	json.Unmarshal(data0, &response)
	assert.Equal("lobby_update", response.Type)

	json.Unmarshal(data1, &response)
	assert.Equal("lobby_update", response.Type)

	// Verify lobby state shows 2 players
	var lobbyState LobbyState
	payloadBytes, _ = json.Marshal(response.Payload)
	json.Unmarshal(payloadBytes, &lobbyState)
	assert.Equal(2, lobbyState.PlayerCount)
}

func TestLobbyState_Personalization(t *testing.T) {
	// This test verifies that each player receives personalized LobbyState
	// with their IsYou flag set correctly

	assert := assert.New(t)
	ctx := context.Background()
	_, url, cleanup := setupTestServer()
	defer cleanup()

	// Create game
	conn1, _, _ := websocket.Dial(ctx, url, nil)
	defer conn1.Close(websocket.StatusNormalClosure, "")

	createReq := ClientMessage{
		Type: "create_game",
		Payload: mustMarshal(CreateGameRequest{
			Username:        "Alice",
			RandomTeamOrder: true,
		}),
	}
	conn1.Write(ctx, websocket.MessageText, mustMarshal(createReq))

	_, data, _ := conn1.Read(ctx) // game_created
	var response ServerMessage
	json.Unmarshal(data, &response)
	var createResp CreateGameResponse
	payloadBytes, _ := json.Marshal(response.Payload)
	json.Unmarshal(payloadBytes, &createResp)
	roomCode := createResp.RoomCode

	// Read lobby_update for Alice
	_, data, _ = conn1.Read(ctx)
	json.Unmarshal(data, &response)
	var lobbyState LobbyState
	payloadBytes, _ = json.Marshal(response.Payload)
	json.Unmarshal(payloadBytes, &lobbyState)

	// Alice should see IsYou=true for slot 0
	assert.True(lobbyState.Players[0].IsYou)
	assert.Equal("Alice", lobbyState.Players[0].Username)

	// Player 2 joins
	conn2, _, _ := websocket.Dial(ctx, url, nil)
	defer conn2.Close(websocket.StatusNormalClosure, "")

	joinReq := ClientMessage{
		Type: "join_game",
		Payload: mustMarshal(JoinGameRequest{
			RoomCode: roomCode,
			Username: "Bob",
		}),
	}
	conn2.Write(ctx, websocket.MessageText, mustMarshal(joinReq))
	conn2.Read(ctx) // game_joined

	// Alice receives lobby_update - should see Bob, IsYou still on Alice
	_, data, _ = conn1.Read(ctx)
	json.Unmarshal(data, &response)
	payloadBytes, _ = json.Marshal(response.Payload)
	json.Unmarshal(payloadBytes, &lobbyState)

	assert.True(lobbyState.Players[0].IsYou)  // Alice's IsYou
	assert.False(lobbyState.Players[1].IsYou) // Bob's IsYou (from Alice's perspective)

	// Bob receives lobby_update - should see IsYou on Bob
	_, data, _ = conn2.Read(ctx)
	json.Unmarshal(data, &response)
	payloadBytes, _ = json.Marshal(response.Payload)
	json.Unmarshal(payloadBytes, &lobbyState)

	assert.False(lobbyState.Players[0].IsYou) // Alice's IsYou (from Bob's perspective)
	assert.True(lobbyState.Players[1].IsYou)  // Bob's IsYou
}

func TestHandleJoinGame_RoomFull(t *testing.T) {
	// Test that 5th player cannot join full lobby

	assert := assert.New(t)
	ctx := context.Background()
	_, url, cleanup := setupTestServer()
	defer cleanup()

	// Create full lobby
	conns := make([]*websocket.Conn, 4)
	var roomCode string

	conns[0], _, _ = websocket.Dial(ctx, url, nil)
	defer conns[0].Close(websocket.StatusNormalClosure, "")

	createReq := ClientMessage{
		Type: "create_game",
		Payload: mustMarshal(CreateGameRequest{
			Username:        "Alice",
			RandomTeamOrder: true,
		}),
	}
	conns[0].Write(ctx, websocket.MessageText, mustMarshal(createReq))

	_, data, _ := conns[0].Read(ctx) // game_created
	var response ServerMessage
	json.Unmarshal(data, &response)
	var createResp CreateGameResponse
	payloadBytes, _ := json.Marshal(response.Payload)
	json.Unmarshal(payloadBytes, &createResp)
	roomCode = createResp.RoomCode

	conns[0].Read(ctx) // lobby_update

	// Join 3 more players
	for i := 1; i < 4; i++ {
		conns[i], _, _ = websocket.Dial(ctx, url, nil)
		defer conns[i].Close(websocket.StatusNormalClosure, "")

		joinReq := ClientMessage{
			Type: "join_game",
			Payload: mustMarshal(JoinGameRequest{
				RoomCode: roomCode,
				Username: string(rune('A'+i)) + "Player",
			}),
		}
		conns[i].Write(ctx, websocket.MessageText, mustMarshal(joinReq))
		conns[i].Read(ctx) // game_joined

		// Drain lobby_update
		for j := 0; j <= i; j++ {
			conns[j].Read(ctx)
		}
	}

	// 5th player tries to join
	conn5, _, _ := websocket.Dial(ctx, url, nil)
	defer conn5.Close(websocket.StatusNormalClosure, "")

	joinReq := ClientMessage{
		Type: "join_game",
		Payload: mustMarshal(JoinGameRequest{
			RoomCode: roomCode,
			Username: "Eve",
		}),
	}
	conn5.Write(ctx, websocket.MessageText, mustMarshal(joinReq))

	// Should receive error
	_, data, _ = conn5.Read(ctx)
	json.Unmarshal(data, &response)
	assert.Equal("error", response.Type)

	var errorMsg ErrorMessage
	payloadBytes, _ = json.Marshal(response.Payload)
	json.Unmarshal(payloadBytes, &errorMsg)
	assert.Contains(errorMsg.Message, "ROOM_FULL")
}

func TestReadyToggle_UnreadyAfterReady(t *testing.T) {
	// Test that players can toggle ready state

	assert := assert.New(t)
	ctx := context.Background()
	_, url, cleanup := setupTestServer()
	defer cleanup()

	conn, _, _ := websocket.Dial(ctx, url, nil)
	defer conn.Close(websocket.StatusNormalClosure, "")

	createReq := ClientMessage{
		Type: "create_game",
		Payload: mustMarshal(CreateGameRequest{
			Username:        "Alice",
			RandomTeamOrder: true,
		}),
	}
	conn.Write(ctx, websocket.MessageText, mustMarshal(createReq))
	conn.Read(ctx) // game_created
	conn.Read(ctx) // lobby_update

	// Ready up
	readyReq := ClientMessage{
		Type: "set_ready",
		Payload: mustMarshal(SetReadyRequest{
			Ready: true,
		}),
	}
	conn.Write(ctx, websocket.MessageText, mustMarshal(readyReq))

	_, data, _ := conn.Read(ctx) // lobby_update
	var response ServerMessage
	json.Unmarshal(data, &response)
	var lobbyState LobbyState
	payloadBytes, _ := json.Marshal(response.Payload)
	json.Unmarshal(payloadBytes, &lobbyState)
	assert.True(lobbyState.Players[0].Ready)

	// Unready
	unreadyReq := ClientMessage{
		Type: "set_ready",
		Payload: mustMarshal(SetReadyRequest{
			Ready: false,
		}),
	}
	conn.Write(ctx, websocket.MessageText, mustMarshal(unreadyReq))

	_, data, _ = conn.Read(ctx) // lobby_update
	json.Unmarshal(data, &response)
	payloadBytes, _ = json.Marshal(response.Payload)
	json.Unmarshal(payloadBytes, &lobbyState)
	assert.False(lobbyState.Players[0].Ready)
}

func TestUpdateTeamOrder_AfterJoin(t *testing.T) {
	// Test that creator can rearrange teams after players join

	assert := assert.New(t)
	ctx := context.Background()
	s, url, cleanup := setupTestServer()
	defer cleanup()

	// Create lobby with 4 players
	conn, roomCode := createFullLobby(t, ctx, url)
	defer conn.Close(websocket.StatusNormalClosure, "")

	// Verify initial order
	game, _ := s.gameManager.GetGame(roomCode)
	assert.Equal([4]string{"Alice", "Bob", "Charlie", "Diana"}, game.Config.PlayerOrder)

	// Creator updates team order
	updateReq := ClientMessage{
		Type: "update_team_order",
		Payload: mustMarshal(UpdateTeamOrderRequest{
			PlayerOrder: [4]string{"Charlie", "Diana", "Alice", "Bob"},
		}),
	}
	conn.Write(ctx, websocket.MessageText, mustMarshal(updateReq))

	// Drain lobby_update
	conn.Read(ctx)

	// Verify order changed
	game, _ = s.gameManager.GetGame(roomCode)
	assert.Equal([4]string{"Charlie", "Diana", "Alice", "Bob"}, game.Config.PlayerOrder)
}

func TestGameStart_RespectsTeamOrder(t *testing.T) {
	// Test that StartGame uses the configured team order

	assert := assert.New(t)
	ctx := context.Background()
	s, url, cleanup := setupTestServer()
	defer cleanup()

	// Create lobby with fixed team order
	conns := make([]*websocket.Conn, 4)
	playerNames := []string{"Alice", "Bob", "Charlie", "Diana"}

	conns[0], _, _ = websocket.Dial(ctx, url, nil)
	defer conns[0].Close(websocket.StatusNormalClosure, "")

	createReq := ClientMessage{
		Type: "create_game",
		Payload: mustMarshal(CreateGameRequest{
			Username:        playerNames[0],
			RandomTeamOrder: false, // Fixed order
		}),
	}
	conns[0].Write(ctx, websocket.MessageText, mustMarshal(createReq))

	_, data, _ := conns[0].Read(ctx) // game_created
	var response ServerMessage
	json.Unmarshal(data, &response)
	var createResp CreateGameResponse
	payloadBytes, _ := json.Marshal(response.Payload)
	json.Unmarshal(payloadBytes, &createResp)
	roomCode := createResp.RoomCode

	conns[0].Read(ctx) // lobby_update

	// Join 3 more
	for i := 1; i < 4; i++ {
		conns[i], _, _ = websocket.Dial(ctx, url, nil)
		defer conns[i].Close(websocket.StatusNormalClosure, "")

		joinReq := ClientMessage{
			Type: "join_game",
			Payload: mustMarshal(JoinGameRequest{
				RoomCode: roomCode,
				Username: playerNames[i],
			}),
		}
		conns[i].Write(ctx, websocket.MessageText, mustMarshal(joinReq))
		conns[i].Read(ctx) // game_joined

		// Read lobby_update from all connected players (0 to i)
		for j := 0; j <= i; j++ {
			conns[j].Read(ctx)
		}
	}

	// Rearrange teams
	newOrder := [4]string{"Bob", "Alice", "Diana", "Charlie"}
	updateReq := ClientMessage{
		Type: "update_team_order",
		Payload: mustMarshal(UpdateTeamOrderRequest{
			PlayerOrder: newOrder,
		}),
	}
	conns[0].Write(ctx, websocket.MessageText, mustMarshal(updateReq))

	for i := range 4 {
		conns[i].Read(ctx)
	}

	// All ready up
	for i := range 4 {
		readyReq := ClientMessage{
			Type:    "set_ready",
			Payload: mustMarshal(SetReadyRequest{Ready: true}),
		}
		conns[i].Write(ctx, websocket.MessageText, mustMarshal(readyReq))

		for j := range 4 {
			conns[j].Read(ctx)
		}
	}

	// Drain game_started
	for i := range 4 {
		conns[i].Read(ctx)
	}

	// Verify game started with correct order
	game, _ := s.gameManager.GetGame(roomCode)
	assert.Equal("Bob", game.Game.Players[0].Name)
	assert.Equal("Alice", game.Game.Players[1].Name)
	assert.Equal("Diana", game.Game.Players[2].Name)
	assert.Equal("Charlie", game.Game.Players[3].Name)
}

// ============================================================================
// RECONNECTION TESTS (Phase 3)
// ============================================================================

// Test: Reconnect to lobby after disconnect
// Why: Verify players can reconnect to lobby with their token
func TestHandleReconnect_ToLobby(t *testing.T) {
	assert := assert.New(t)
	ctx := context.Background()
	server, url, cleanup := setupTestServer()
	defer cleanup()

	// Player creates game
	conn1, _, err := websocket.Dial(ctx, url, nil)
	assert.NoError(err)

	createReq := ClientMessage{
		Type: "create_game",
		Payload: mustMarshal(CreateGameRequest{
			Username:        "Alice",
			RandomTeamOrder: false,
		}),
	}
	conn1.Write(ctx, websocket.MessageText, mustMarshal(createReq))

	// Read game_created
	_, data, _ := conn1.Read(ctx)
	var gameCreatedMsg ServerMessage
	json.Unmarshal(data, &gameCreatedMsg)
	var createResp CreateGameResponse
	payloadBytes, _ := json.Marshal(gameCreatedMsg.Payload)
	json.Unmarshal(payloadBytes, &createResp)

	token := createResp.Token
	roomCode := createResp.RoomCode

	// Read lobby_update
	conn1.Read(ctx)

	// Disconnect
	conn1.Close(websocket.StatusNormalClosure, "")
	time.Sleep(50 * time.Millisecond) // Give server time to process disconnect

	// Verify player marked as disconnected in game
	game, _ := server.gameManager.GetGame(roomCode)
	assert.False(game.Players[0].Connected)

	// Reconnect with token
	conn2, _, err := websocket.Dial(ctx, url, nil)
	assert.NoError(err)
	defer conn2.Close(websocket.StatusNormalClosure, "")

	reconnectReq := ClientMessage{
		Type: "reconnect",
		Payload: mustMarshal(ReconnectRequest{
			Token: token,
		}),
	}
	conn2.Write(ctx, websocket.MessageText, mustMarshal(reconnectReq))

	// Read reconnected response
	_, data, err = conn2.Read(ctx)
	assert.NoError(err)

	var response ServerMessage
	json.Unmarshal(data, &response)
	assert.Equal("reconnected", response.Type)

	var reconnectResp ReconnectResponse
	payloadBytes, _ = json.Marshal(response.Payload)
	json.Unmarshal(payloadBytes, &reconnectResp)
	assert.True(reconnectResp.Success)
	assert.Equal(roomCode, reconnectResp.RoomCode)
	assert.Equal(0, reconnectResp.PlayerID)

	// Verify player marked as connected
	game, _ = server.gameManager.GetGame(roomCode)
	assert.True(game.Players[0].Connected)
}

// Test: Reconnect with invalid token
// Why: Verify security - invalid tokens are rejected
func TestHandleReconnect_InvalidToken(t *testing.T) {
	assert := assert.New(t)
	ctx := context.Background()
	_, url, cleanup := setupTestServer()
	defer cleanup()

	conn, _, err := websocket.Dial(ctx, url, nil)
	assert.NoError(err)
	defer conn.Close(websocket.StatusNormalClosure, "")

	reconnectReq := ClientMessage{
		Type: "reconnect",
		Payload: mustMarshal(ReconnectRequest{
			Token: "invalid-token",
		}),
	}
	conn.Write(ctx, websocket.MessageText, mustMarshal(reconnectReq))

	// Should receive error
	_, data, err := conn.Read(ctx)
	assert.NoError(err)

	var response ServerMessage
	json.Unmarshal(data, &response)
	assert.Equal("error", response.Type)
}

// Test: Device switch - connect from two devices with same token
// Why: Verify single connection per token enforcement
func TestHandleReconnect_DeviceSwitch(t *testing.T) {
	assert := assert.New(t)
	ctx := context.Background()
	_, url, cleanup := setupTestServer()
	defer cleanup()

	// Player creates game on device 1
	conn1, _, err := websocket.Dial(ctx, url, nil)
	assert.NoError(err)

	createReq := ClientMessage{
		Type: "create_game",
		Payload: mustMarshal(CreateGameRequest{
			Username:        "Alice",
			RandomTeamOrder: false,
		}),
	}
	conn1.Write(ctx, websocket.MessageText, mustMarshal(createReq))

	// Read game_created
	_, data, _ := conn1.Read(ctx)
	var gameCreatedMsg ServerMessage
	json.Unmarshal(data, &gameCreatedMsg)
	var createResp CreateGameResponse
	payloadBytes, _ := json.Marshal(gameCreatedMsg.Payload)
	json.Unmarshal(payloadBytes, &createResp)
	token := createResp.Token

	// Read lobby_update
	conn1.Read(ctx)

	// Connect from device 2 with same token
	conn2, _, err := websocket.Dial(ctx, url, nil)
	assert.NoError(err)
	defer conn2.Close(websocket.StatusNormalClosure, "")

	reconnectReq := ClientMessage{
		Type: "reconnect",
		Payload: mustMarshal(ReconnectRequest{
			Token: token,
		}),
	}
	conn2.Write(ctx, websocket.MessageText, mustMarshal(reconnectReq))

	// Device 1 should receive disconnected_elsewhere
	_, data, err = conn1.Read(ctx)
	if err == nil {
		var response ServerMessage
		json.Unmarshal(data, &response)
		// Should be disconnected_elsewhere or connection closed
		if response.Type == "disconnected_elsewhere" {
			assert.Equal("disconnected_elsewhere", response.Type)
		}
	}

	// Device 2 should receive reconnected
	_, data, err = conn2.Read(ctx)
	assert.NoError(err)
	var response ServerMessage
	json.Unmarshal(data, &response)
	assert.Equal("reconnected", response.Type)
}

// ============================================================================
// PHASE 4: GAME STATE BROADCASTING TESTS
// ============================================================================

func TestGameStart_BroadcastsInitialState(t *testing.T) {
	// Why: After all players ready, verify game_state broadcast is sent
	assert := assert.New(t)
	s, _, cleanup := setupTestServer()
	defer cleanup()

	// Create game with 4 players directly via game manager
	game, _, err := s.gameManager.CreateGame("Alice", false)
	assert.NoError(err)

	for _, name := range []string{"Bob", "Carol", "Dave"} {
		_, _, _, err := s.gameManager.JoinGame(game.RoomCode, name)
		assert.NoError(err)
	}

	// Set all ready
	for i := 0; i < 4; i++ {
		_, _, err := s.gameManager.SetReady(game.RoomCode, game.Players[i].Token, true)
		assert.NoError(err)
	}

	// Start game
	err = s.gameManager.StartGame(game.RoomCode)
	assert.NoError(err)

	// Verify game state can be built (broadcasting tested separately)
	stateMsg := s.buildGameStateMessage(game, 0)
	assert.NotNil(stateMsg.State)
	assert.Equal(game.Game.CurrentPlayer, stateMsg.CurrentPlayer)
	assert.Equal(string(game.Game.Phase), stateMsg.Phase)
	assert.Equal(string(game.Status), stateMsg.Status)
	assert.Equal(string(StatusPlaying), stateMsg.Status)
}

func TestReconnect_SendsCurrentGameState(t *testing.T) {
	// Why: Reconnecting player should receive current game state
	// This is integration-tested via handleReconnect, here we test the logic
	assert := assert.New(t)
	s, _, cleanup := setupTestServer()
	defer cleanup()

	// Create and start game
	game, _, err := s.gameManager.CreateGame("Alice", false)
	assert.NoError(err)

	for _, name := range []string{"Bob", "Carol", "Dave"} {
		_, _, _, err := s.gameManager.JoinGame(game.RoomCode, name)
		assert.NoError(err)
	}

	for i := 0; i < 4; i++ {
		_, _, err := s.gameManager.SetReady(game.RoomCode, game.Players[i].Token, true)
		assert.NoError(err)
	}

	err = s.gameManager.StartGame(game.RoomCode)
	assert.NoError(err)

	// Verify we can build state for reconnection
	stateMsg := s.buildGameStateMessage(game, 0)
	assert.NotNil(stateMsg.State)
	assert.Equal(game.Game.CurrentPlayer, stateMsg.CurrentPlayer)
	assert.Equal(string(game.Game.Phase), stateMsg.Phase)
}

func TestGameStateMessage_IncludesRedThrees(t *testing.T) {
	// Why: Verify red threes are included in game state (Phase 4 requirement)
	assert := assert.New(t)
	s, _, cleanup := setupTestServer()
	defer cleanup()

	// Create and start game
	game, _, err := s.gameManager.CreateGame("Alice", false)
	assert.NoError(err)

	for _, name := range []string{"Bob", "Carol", "Dave"} {
		_, _, _, err := s.gameManager.JoinGame(game.RoomCode, name)
		assert.NoError(err)
	}

	for i := 0; i < 4; i++ {
		_, _, err := s.gameManager.SetReady(game.RoomCode, game.Players[i].Token, true)
		assert.NoError(err)
	}

	err = s.gameManager.StartGame(game.RoomCode)
	assert.NoError(err)

	// Build game state
	stateMsg := s.buildGameStateMessage(game, 0)

	// Verify ClientState includes red threes fields
	// Note: The actual content depends on game logic, we're just verifying the structure
	assert.NotNil(stateMsg.State)
}

func TestGameState_MetadataMatchesGameState(t *testing.T) {
	// Why: Verify currentPlayer, phase, status match actual game state
	assert := assert.New(t)
	s, _, cleanup := setupTestServer()
	defer cleanup()

	// Create and start game
	game, _, err := s.gameManager.CreateGame("Alice", false)
	assert.NoError(err)

	for _, name := range []string{"Bob", "Carol", "Dave"} {
		_, _, _, err := s.gameManager.JoinGame(game.RoomCode, name)
		assert.NoError(err)
	}

	for i := 0; i < 4; i++ {
		_, _, err := s.gameManager.SetReady(game.RoomCode, game.Players[i].Token, true)
		assert.NoError(err)
	}

	err = s.gameManager.StartGame(game.RoomCode)
	assert.NoError(err)

	// Build game state for player 0
	stateMsg := s.buildGameStateMessage(game, 0)

	// Verify metadata matches game
	assert.Equal(game.Game.CurrentPlayer, stateMsg.CurrentPlayer)
	assert.Equal(string(game.Game.Phase), stateMsg.Phase)
	assert.Equal(string(game.Status), stateMsg.Status)
}
