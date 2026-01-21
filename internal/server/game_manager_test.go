package server

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewGameManager(t *testing.T) {
	assert := assert.New(t)

	gm := NewGameManager()

	// Assert manager is not nil
	assert.NotNil(gm)

	// Assert maps are initialized (not nil)
	assert.NotNil(gm.games)
	assert.NotNil(gm.usedCodes)

	// Assert maps start empty
	assert.Equal(0, len(gm.games))
	assert.Equal(0, len(gm.usedCodes))
}

func TestValidateUsernameFormat_Valid(t *testing.T) {
	assert := assert.New(t)
	gm := NewGameManager()

	validUsernames := []string{
		"Alice",                // Simple name
		"Bob123",               // Alphanumeric
		"Player One",           // With space
		"José",                 // Unicode/accented
		"日本語",                  // Non-latin characters
		"@user!",               // Special characters
		"12345678901234567890", // Exactly 20 chars
	}

	for _, username := range validUsernames {
		err := gm.validateUsernameFormat(username)
		assert.NoError(err, "Username '%s' should be valid", username)
	}
}

func TestValidateUsernameFormat_Empty(t *testing.T) {
	assert := assert.New(t)
	gm := NewGameManager()

	err := gm.validateUsernameFormat("")

	assert.Error(err)
}

func TestValidateUsernameFormat_TooLong(t *testing.T) {
	assert := assert.New(t)
	gm := NewGameManager()

	tooLongUsernames := []string{
		"123456789012345678901", // 21 chars (just over limit)
		"This is a very long username that exceeds twenty characters",
	}

	for _, username := range tooLongUsernames {
		err := gm.validateUsernameFormat(username)
		assert.Error(err, "Username '%s' should be invalid (too long)", username)
	}
}

func TestValidateUsernameFormat_EdgeCases(t *testing.T) {
	assert := assert.New(t)
	gm := NewGameManager()

	err := gm.validateUsernameFormat("A")
	assert.NoError(err, "Single character should be valid")

	err = gm.validateUsernameFormat(" ")
	assert.Error(err, "Whitespace-only should be invalid")
}

func TestCreateGame_Success(t *testing.T) {
	assert := assert.New(t)
	gm := NewGameManager()

	game, token, err := gm.CreateGame("Alice", true)

	// Assert no error
	assert.NoError(err)
	assert.NotNil(game)
	assert.NotEmpty(token)

	// Assert room code generated
	assert.NotEmpty(game.RoomCode)
	assert.Equal(4, len(game.RoomCode))

	// Assert game status is lobby
	assert.Equal(StatusLobby, game.Status)

	// Assert creator in slot 0
	assert.Equal("Alice", game.Players[0].Username)
	assert.Equal(token, game.Players[0].Token)
	assert.True(game.Players[0].Connected)
	assert.False(game.Players[0].Ready) // Not ready by default

	// Assert other slots empty
	assert.Empty(game.Players[1].Username)
	assert.Empty(game.Players[2].Username)
	assert.Empty(game.Players[3].Username)

	// Assert config set correctly
	assert.True(game.Config.RandomTeamOrder)
	assert.Equal("Alice", game.Config.PlayerOrder[0])

	// Assert timestamps set
	assert.False(game.CreatedAt.IsZero())
	assert.False(game.UpdatedAt.IsZero())
	assert.False(game.LobbyExpiry.IsZero())

	// Assert canasta.Game not initialized yet
	assert.Nil(game.Game)
}

func TestCreateGame_RoomCodeUnique(t *testing.T) {
	assert := assert.New(t)
	gm := NewGameManager()

	// Create 10 games
	roomCodes := make(map[string]bool)
	for i := range 10 {
		game, _, err := gm.CreateGame("Player"+string(rune('A'+i)), true)
		assert.NoError(err)

		// Assert code is unique
		assert.False(roomCodes[game.RoomCode], "Room code %s generated twice", game.RoomCode)
		roomCodes[game.RoomCode] = true
	}

	// Assert we got 10 unique codes
	assert.Equal(10, len(roomCodes))

	// Assert usedCodes map has 10 entries
	gm.mu.RLock()
	assert.Equal(10, len(gm.usedCodes))
	gm.mu.RUnlock()
}

func TestCreateGame_StoresInManager(t *testing.T) {
	assert := assert.New(t)
	gm := NewGameManager()

	game, _, err := gm.CreateGame("Bob", false)
	assert.NoError(err)

	// Retrieve game from manager
	gm.mu.RLock()
	storedGame, exists := gm.games[game.RoomCode]
	gm.mu.RUnlock()

	assert.True(exists)
	assert.Equal(game, storedGame)
	assert.Equal("Bob", storedGame.Players[0].Username)
}

func TestCreateGame_InvalidUsername(t *testing.T) {
	assert := assert.New(t)
	gm := NewGameManager()

	// Test empty username
	game, token, err := gm.CreateGame("", true)
	assert.Error(err)
	assert.Nil(game)
	assert.Empty(token)

	// Test too long username
	longName := "123456789012345678901" // 21 chars
	game, token, err = gm.CreateGame(longName, true)
	assert.Error(err)
	assert.Nil(game)
	assert.Empty(token)
}

func TestCreateGame_TokenGenerated(t *testing.T) {
	assert := assert.New(t)
	gm := NewGameManager()

	game1, token1, err := gm.CreateGame("Alice", true)
	assert.NoError(err)
	assert.NotEmpty(token1)

	game2, token2, err := gm.CreateGame("Bob", true)
	assert.NoError(err)
	assert.NotEmpty(token2)

	assert.NotEqual(token1, token2)

	// Token should match what's stored in game
	assert.Equal(token1, game1.Players[0].Token)
	assert.Equal(token2, game2.Players[0].Token)
}

func TestCreateGame_RandomTeamOrderFlag(t *testing.T) {
	assert := assert.New(t)
	gm := NewGameManager()

	// Create game with random=true
	game1, _, err := gm.CreateGame("Alice", true)
	assert.NoError(err)
	assert.True(game1.Config.RandomTeamOrder)

	// Create game with random=false
	game2, _, err := gm.CreateGame("Bob", false)
	assert.NoError(err)
	assert.False(game2.Config.RandomTeamOrder)
}

func TestCreateGame_LobbyExpiry(t *testing.T) {
	assert := assert.New(t)
	gm := NewGameManager()

	before := time.Now()
	game, _, err := gm.CreateGame("Alice", true)
	after := time.Now()

	assert.NoError(err)

	// Expiry should be ~10 minutes from now
	expectedExpiry := before.Add(10 * time.Minute)
	actualExpiry := game.LobbyExpiry

	// Allow 1 second variance for test execution time
	assert.True(actualExpiry.After(expectedExpiry.Add(-1 * time.Second)))
	assert.True(actualExpiry.Before(after.Add(10*time.Minute + 1*time.Second)))
}

func TestCreateGame_PlayerOrderInitialized(t *testing.T) {
	assert := assert.New(t)
	gm := NewGameManager()

	game, _, err := gm.CreateGame("Alice", true)
	assert.NoError(err)

	assert.Equal("Alice", game.Config.PlayerOrder[0])
	assert.Equal("", game.Config.PlayerOrder[1])
	assert.Equal("", game.Config.PlayerOrder[2])
	assert.Equal("", game.Config.PlayerOrder[3])
}

func TestCreateGame_ConcurrentCreation(t *testing.T) {
	assert := assert.New(t)
	gm := NewGameManager()

	const numGames = 50
	results := make(chan string, numGames)
	errors := make(chan error, numGames)

	// Create 50 games concurrently
	for i := range numGames {
		go func(id int) {
			game, _, err := gm.CreateGame(fmt.Sprintf("Player%d", id), true)
			if err != nil {
				errors <- err
			} else {
				results <- game.RoomCode
			}
		}(i)
	}

	// Collect results
	roomCodes := make(map[string]bool)
	for range numGames {
		select {
		case code := <-results:
			assert.False(roomCodes[code], "Duplicate room code: %s", code)
			roomCodes[code] = true
		case err := <-errors:
			t.Fatalf("Unexpected error: %v", err)
		}
	}

	// Assert all codes unique
	assert.Equal(numGames, len(roomCodes))
}

// TestJoinGame_Success verifies basic join functionality.
//
// Why this test:
// - Happy path - most common scenario
// - Validates player is added correctly
// - Checks token generation and slot assignment
func TestJoinGame_Success(t *testing.T) {
	assert := assert.New(t)
	gm := NewGameManager()

	// Create game
	game1, _, err := gm.CreateGame("Alice", true)
	assert.NoError(err)
	roomCode := game1.RoomCode

	// Join game
	game2, token, slotID, err := gm.JoinGame(roomCode, "Bob")

	// Assert no error
	assert.NoError(err)
	assert.NotNil(game2)
	assert.NotEmpty(token)
	assert.Equal(1, slotID) // Should get slot 1 (creator has 0)

	// Assert same game returned
	assert.Equal(game1, game2)

	// Assert Bob in slot 1
	assert.Equal("Bob", game2.Players[1].Username)
	assert.Equal(token, game2.Players[1].Token)
	assert.True(game2.Players[1].Connected)
	assert.False(game2.Players[1].Ready)

	// Assert PlayerOrder updated
	assert.Equal("Bob", game2.Config.PlayerOrder[1])

	// Assert other slots still empty
	assert.Empty(game2.Players[2].Username)
	assert.Empty(game2.Players[3].Username)
}

// TestJoinGame_FillsSequentialSlots verifies slots fill in order.
//
// Why this test:
// - Ensures predictable slot assignment (1, 2, 3)
// - Tests multiple joins to same lobby
// - Validates lobby can fill completely
func TestJoinGame_FillsSequentialSlots(t *testing.T) {
	assert := assert.New(t)
	gm := NewGameManager()

	// Create game
	game, _, err := gm.CreateGame("Alice", true)
	assert.NoError(err)
	roomCode := game.RoomCode

	// Join 3 more players
	_, _, slot1, err := gm.JoinGame(roomCode, "Bob")
	assert.NoError(err)
	assert.Equal(1, slot1)

	_, _, slot2, err := gm.JoinGame(roomCode, "Charlie")
	assert.NoError(err)
	assert.Equal(2, slot2)

	_, _, slot3, err := gm.JoinGame(roomCode, "Diana")
	assert.NoError(err)
	assert.Equal(3, slot3)

	// Verify all 4 slots filled
	assert.Equal("Alice", game.Players[0].Username)
	assert.Equal("Bob", game.Players[1].Username)
	assert.Equal("Charlie", game.Players[2].Username)
	assert.Equal("Diana", game.Players[3].Username)
}

// TestJoinGame_RoomNotFound verifies error for invalid room code.
//
// Why this test:
// - Most common error case (typos in room code)
// - Validates error message includes error code
func TestJoinGame_RoomNotFound(t *testing.T) {
	assert := assert.New(t)
	gm := NewGameManager()

	// Try to join non-existent room
	game, token, slotID, err := gm.JoinGame("ZZZZ", "Bob")

	assert.Error(err)
	assert.Nil(game)
	assert.Empty(token)
	assert.Equal(-1, slotID)
	assert.Contains(err.Error(), "ROOM_NOT_FOUND")
}

// TestJoinGame_DuplicateUsername verifies uniqueness check.
//
// Why this test:
// - Prevents confusing UI with duplicate names
// - Ensures validateUsername is called
// - Important for game logic (need to identify players uniquely)
func TestJoinGame_DuplicateUsername(t *testing.T) {
	assert := assert.New(t)
	gm := NewGameManager()

	// Create game with Alice
	game, _, err := gm.CreateGame("Alice", true)
	assert.NoError(err)

	// Try to join with same username
	_, _, _, err = gm.JoinGame(game.RoomCode, "Alice")

	assert.Error(err)
	assert.Contains(err.Error(), "already taken")

	// Verify slot 1 is still empty
	assert.Empty(game.Players[1].Username)
}

// TestJoinGame_RoomFull verifies 5th player is rejected.
//
// Why this test:
// - Canasta is exactly 4 players
// - Must prevent overfilling
// - Ensures clear error message
func TestJoinGame_RoomFull(t *testing.T) {
	assert := assert.New(t)
	gm := NewGameManager()

	// Create and fill lobby
	game, _, _ := gm.CreateGame("Alice", true)
	gm.JoinGame(game.RoomCode, "Bob")
	gm.JoinGame(game.RoomCode, "Charlie")
	gm.JoinGame(game.RoomCode, "Diana")

	// Try to add 5th player
	_, _, _, err := gm.JoinGame(game.RoomCode, "Eve")

	assert.Error(err)
	assert.Contains(err.Error(), "ROOM_FULL")
	assert.Contains(err.Error(), "4/4")
}

// TestJoinGame_GameAlreadyStarted verifies can't join playing game.
//
// Why this test:
// - Phase 2 doesn't support mid-game joins
// - Phase 3 will add reconnection
// - For now, must block joins to playing games
func TestJoinGame_GameAlreadyStarted(t *testing.T) {
	assert := assert.New(t)
	gm := NewGameManager()

	// Create game
	game, _, _ := gm.CreateGame("Alice", true)

	// Manually change status to playing (StartGame will do this normally)
	game.Status = StatusPlaying

	// Try to join
	_, _, _, err := gm.JoinGame(game.RoomCode, "Bob")

	assert.Error(err)
	assert.Contains(err.Error(), "GAME_ALREADY_STARTED")
}

// TestJoinGame_NormalizesRoomCode verifies case-insensitive join.
//
// Why this test:
// - UX feature - players can type lowercase
// - Important for mobile users (autocomplete often lowercases)
func TestJoinGame_NormalizesRoomCode(t *testing.T) {
	assert := assert.New(t)
	gm := NewGameManager()

	// Create game (code will be uppercase like "BEAR")
	game, _, _ := gm.CreateGame("Alice", true)
	roomCode := game.RoomCode

	// Join with lowercase
	lowercaseCode := strings.ToLower(roomCode)
	_, _, slotID, err := gm.JoinGame(lowercaseCode, "Bob")

	assert.NoError(err)
	assert.Equal(1, slotID)
	assert.Equal("Bob", game.Players[1].Username)
}

// TestJoinGame_InvalidUsername verifies username validation.
//
// Why this test:
// - Ensures validateUsername is called
// - Tests both format and uniqueness validation
func TestJoinGame_InvalidUsername(t *testing.T) {
	assert := assert.New(t)
	gm := NewGameManager()

	game, _, _ := gm.CreateGame("Alice", true)

	// Empty username
	_, _, _, err := gm.JoinGame(game.RoomCode, "")
	assert.Error(err)
	assert.Contains(err.Error(), "USERNAME_INVALID")

	// Too long username
	longName := "123456789012345678901" // 21 chars
	_, _, _, err = gm.JoinGame(game.RoomCode, longName)
	assert.Error(err)
	assert.Contains(err.Error(), "USERNAME_INVALID")
}

// TestJoinGame_TokensUnique verifies each player gets unique token.
//
// Why this test:
// - Tokens must be unique for security
// - UUID collision would be catastrophic (though negligible probability)
func TestJoinGame_TokensUnique(t *testing.T) {
	assert := assert.New(t)
	gm := NewGameManager()

	game, token0, _ := gm.CreateGame("Alice", true)
	_, token1, _, _ := gm.JoinGame(game.RoomCode, "Bob")
	_, token2, _, _ := gm.JoinGame(game.RoomCode, "Charlie")
	_, token3, _, _ := gm.JoinGame(game.RoomCode, "Diana")

	tokens := []string{token0, token1, token2, token3}

	// Check all different
	uniqueTokens := make(map[string]bool)
	for _, token := range tokens {
		assert.False(uniqueTokens[token], "Token %s used twice", token)
		uniqueTokens[token] = true
	}

	assert.Equal(4, len(uniqueTokens))
}

func TestJoinGame_UpdatesTimestamp(t *testing.T) {
	assert := assert.New(t)
	gm := NewGameManager()

	game, _, _ := gm.CreateGame("Alice", true)
	originalTime := game.UpdatedAt

	// Wait a tiny bit to ensure time difference
	time.Sleep(10 * time.Millisecond)

	gm.JoinGame(game.RoomCode, "Bob")

	// UpdatedAt should be newer
	assert.True(game.UpdatedAt.After(originalTime))
}

func TestJoinGame_InvalidRoomCodeFormat(t *testing.T) {
	assert := assert.New(t)
	gm := NewGameManager()

	// Too short
	_, _, _, err := gm.JoinGame("ABC", "Bob")
	assert.Error(err)
	assert.Contains(err.Error(), "exactly 4 characters")

	// Too long
	_, _, _, err = gm.JoinGame("ABCDE", "Bob")
	assert.Error(err)

	// Numbers
	_, _, _, err = gm.JoinGame("1234", "Bob")
	assert.Error(err)
	assert.Contains(err.Error(), "only letters")
}

// TestSetReady_Success verifies basic ready functionality.
func TestSetReady_Success(t *testing.T) {
	assert := assert.New(t)
	gm := NewGameManager()

	// Create game and get creator's token
	game, token0, _ := gm.CreateGame("Alice", true)
	roomCode := game.RoomCode

	// Creator readies up
	updatedGame, allReady, err := gm.SetReady(roomCode, token0, true)

	assert.NoError(err)
	assert.NotNil(updatedGame)
	assert.False(allReady) // Only 1/4 ready

	// Verify Alice is ready
	assert.True(game.Players[0].Ready)

	// Verify other slots not affected
	assert.Empty(game.Players[1].Username)
	assert.Empty(game.Players[2].Username)
	assert.Empty(game.Players[3].Username)
}

// TestSetReady_Toggle verifies players can unready.
func TestSetReady_Toggle(t *testing.T) {
	assert := assert.New(t)
	gm := NewGameManager()

	game, token, _ := gm.CreateGame("Alice", true)

	// Ready up
	gm.SetReady(game.RoomCode, token, true)
	assert.True(game.Players[0].Ready)

	// Unready
	gm.SetReady(game.RoomCode, token, false)
	assert.False(game.Players[0].Ready)

	// Ready again
	gm.SetReady(game.RoomCode, token, true)
	assert.True(game.Players[0].Ready)
}

// TestSetReady_InvalidToken verifies error for wrong token.
func TestSetReady_InvalidToken(t *testing.T) {
	assert := assert.New(t)
	gm := NewGameManager()

	game, _, _ := gm.CreateGame("Alice", true)

	// Try with fake token
	_, _, err := gm.SetReady(game.RoomCode, "invalid-token-12345", true)

	assert.Error(err)
}

// TestSetReady_RoomNotFound verifies error for invalid room.
func TestSetReady_RoomNotFound(t *testing.T) {
	assert := assert.New(t)
	gm := NewGameManager()

	_, _, err := gm.SetReady("ZZZZ", "token", true)

	assert.Error(err)
}

// TestSetReady_GameAlreadyStarted verifies can't ready after start.
func TestSetReady_GameAlreadyStarted(t *testing.T) {
	assert := assert.New(t)
	gm := NewGameManager()

	game, token, _ := gm.CreateGame("Alice", true)

	// Manually start game (or set status to Playing)
	game.Status = StatusPlaying

	// Try to ready up
	_, _, err := gm.SetReady(game.RoomCode, token, true)

	assert.Error(err)
}

// TestSetReady_AllReady verifies auto-start trigger condition.
func TestSetReady_AllReady(t *testing.T) {
	assert := assert.New(t)
	gm := NewGameManager()

	// Create and fill lobby
	game, token0, _ := gm.CreateGame("Alice", true)
	_, token1, _, _ := gm.JoinGame(game.RoomCode, "Bob")
	_, token2, _, _ := gm.JoinGame(game.RoomCode, "Charlie")
	_, token3, _, _ := gm.JoinGame(game.RoomCode, "Diana")

	// Ready up first 3 players
	_, allReady, _ := gm.SetReady(game.RoomCode, token0, true)
	assert.False(allReady) // 1/4

	_, allReady, _ = gm.SetReady(game.RoomCode, token1, true)
	assert.False(allReady) // 2/4

	_, allReady, _ = gm.SetReady(game.RoomCode, token2, true)
	assert.False(allReady) // 3/4

	// Ready up 4th player - should trigger
	_, allReady, _ = gm.SetReady(game.RoomCode, token3, true)
	assert.True(allReady) // 4/4 - trigger!
}

// TestSetReady_UnreadyBreaksAllReady tests unready after all ready.
func TestSetReady_UnreadyBreaksAllReady(t *testing.T) {
	assert := assert.New(t)
	gm := NewGameManager()

	// Create and fill lobby, all ready
	game, token0, _ := gm.CreateGame("Alice", true)
	_, token1, _, _ := gm.JoinGame(game.RoomCode, "Bob")
	_, token2, _, _ := gm.JoinGame(game.RoomCode, "Charlie")
	_, token3, _, _ := gm.JoinGame(game.RoomCode, "Diana")

	gm.SetReady(game.RoomCode, token0, true)
	gm.SetReady(game.RoomCode, token1, true)
	gm.SetReady(game.RoomCode, token2, true)
	_, allReady, _ := gm.SetReady(game.RoomCode, token3, true)
	assert.True(allReady) // All ready

	// Alice unreadies
	_, allReady, _ = gm.SetReady(game.RoomCode, token0, false)
	assert.False(allReady) // No longer all ready
}

// TestCheckAllReady_EmptySlots verifies false with <4 players.
func TestCheckAllReady_EmptySlots(t *testing.T) {
	assert := assert.New(t)
	gm := NewGameManager()

	// Create game with only 2 players
	game, token0, _ := gm.CreateGame("Alice", true)
	_, token1, _, _ := gm.JoinGame(game.RoomCode, "Bob")

	// Both ready
	gm.SetReady(game.RoomCode, token0, true)
	gm.SetReady(game.RoomCode, token1, true)

	// Check directly
	allReady := gm.checkAllReady(game)
	assert.False(allReady) // Only 2/4 players
}

// TestCheckAllReady_NotAllReady verifies false when some not ready.
func TestCheckAllReady_NotAllReady(t *testing.T) {
	assert := assert.New(t)
	gm := NewGameManager()

	// Create and fill lobby
	game, token0, _ := gm.CreateGame("Alice", true)
	_, token1, _, _ := gm.JoinGame(game.RoomCode, "Bob")
	gm.JoinGame(game.RoomCode, "Charlie")
	gm.JoinGame(game.RoomCode, "Diana")

	// Only 2 ready
	gm.SetReady(game.RoomCode, token0, true)
	gm.SetReady(game.RoomCode, token1, true)

	allReady := gm.checkAllReady(game)
	assert.False(allReady) // 2/4 ready
}

// TestCheckAllReady_AllReady verifies true when conditions met.
func TestCheckAllReady_AllReady(t *testing.T) {
	assert := assert.New(t)
	gm := NewGameManager()

	// Create and fill lobby
	game, token0, _ := gm.CreateGame("Alice", true)
	_, token1, _, _ := gm.JoinGame(game.RoomCode, "Bob")
	_, token2, _, _ := gm.JoinGame(game.RoomCode, "Charlie")
	_, token3, _, _ := gm.JoinGame(game.RoomCode, "Diana")

	// All ready
	gm.SetReady(game.RoomCode, token0, true)
	gm.SetReady(game.RoomCode, token1, true)
	gm.SetReady(game.RoomCode, token2, true)
	gm.SetReady(game.RoomCode, token3, true)

	allReady := gm.checkAllReady(game)
	assert.True(allReady) // 4/4 ready
}

// TestSetReady_UpdatesTimestamp verifies UpdatedAt changes.
func TestSetReady_UpdatesTimestamp(t *testing.T) {
	assert := assert.New(t)
	gm := NewGameManager()

	game, token, _ := gm.CreateGame("Alice", true)
	originalTime := game.UpdatedAt

	time.Sleep(10 * time.Millisecond)

	gm.SetReady(game.RoomCode, token, true)

	assert.True(game.UpdatedAt.After(originalTime))
}

func TestStartGame_Success(t *testing.T) {
	assert := assert.New(t)
	gm := NewGameManager()

	// Create full lobby, all ready
	game, token0, _ := gm.CreateGame("Alice", true)
	_, token1, _, _ := gm.JoinGame(game.RoomCode, "Bob")
	_, token2, _, _ := gm.JoinGame(game.RoomCode, "Charlie")
	_, token3, _, _ := gm.JoinGame(game.RoomCode, "Diana")

	gm.SetReady(game.RoomCode, token0, true)
	gm.SetReady(game.RoomCode, token1, true)
	gm.SetReady(game.RoomCode, token2, true)
	gm.SetReady(game.RoomCode, token3, true)

	// Start game
	err := gm.StartGame(game.RoomCode)

	assert.NoError(err)

	// Verify status changed
	assert.Equal(StatusPlaying, game.Status)

	// Verify canasta.Game created
	assert.NotNil(game.Game)
	assert.NotNil(game.Game.Players)
	assert.Equal(4, len(game.Game.Players))

	// Verify cards dealt (players should have cards in hand)
	// Deal() gives 15 cards to each player's hand
	for i, player := range game.Game.Players {
		assert.NotEmpty(player.Hand, "Player %d should have cards", i)
	}

	// Verify deck exists and has cards remaining
	assert.NotNil(game.Game.Hand.Deck)

	// Verify discard pile has one card (Deal flips top card)
	assert.Equal(1, len(game.Game.Hand.DiscardPile))
}

// TestStartGame_StatusChanges verifies lobby -> playing transition.
func TestStartGame_StatusChanges(t *testing.T) {
	assert := assert.New(t)
	gm := NewGameManager()

	game, token0, _ := gm.CreateGame("Alice", true)
	_, token1, _, _ := gm.JoinGame(game.RoomCode, "Bob")
	_, token2, _, _ := gm.JoinGame(game.RoomCode, "Charlie")
	_, token3, _, _ := gm.JoinGame(game.RoomCode, "Diana")

	// All ready
	gm.SetReady(game.RoomCode, token0, true)
	gm.SetReady(game.RoomCode, token1, true)
	gm.SetReady(game.RoomCode, token2, true)
	gm.SetReady(game.RoomCode, token3, true)

	// Verify initial status
	assert.Equal(StatusLobby, game.Status)

	// Start
	gm.StartGame(game.RoomCode)

	// Verify status changed
	assert.Equal(StatusPlaying, game.Status)
}

// TestStartGame_RespectsRandomTeamOrder verifies random shuffling.
func TestStartGame_RespectsRandomTeamOrder(t *testing.T) {
	assert := assert.New(t)

	// Run multiple game starts with RandomTeamOrder=true
	// We should see different player orderings due to shuffle
	playerOrders := make(map[string]bool)

	for range 10 {
		gm := NewGameManager()

		game, token0, _ := gm.CreateGame("Alice", true) // RandomTeamOrder=true
		_, token1, _, _ := gm.JoinGame(game.RoomCode, "Bob")
		_, token2, _, _ := gm.JoinGame(game.RoomCode, "Charlie")
		_, token3, _, _ := gm.JoinGame(game.RoomCode, "Diana")

		gm.SetReady(game.RoomCode, token0, true)
		gm.SetReady(game.RoomCode, token1, true)
		gm.SetReady(game.RoomCode, token2, true)
		gm.SetReady(game.RoomCode, token3, true)

		gm.StartGame(game.RoomCode)

		// Record order
		order := ""
		for _, player := range game.Game.Players {
			order += player.Name + ","
		}
		playerOrders[order] = true
	}

	// With random shuffling, we should see at least 2 different orders
	// (Could see same order by chance, but unlikely with 10 trials)
	assert.True(len(playerOrders) >= 2, "Expected variation in player order with random shuffling")
}

// TestStartGame_RespectsFixedTeamOrder verifies no shuffling.
func TestStartGame_RespectsFixedTeamOrder(t *testing.T) {
	assert := assert.New(t)
	gm := NewGameManager()

	// Create with RandomTeamOrder=false
	game, token0, _ := gm.CreateGame("Alice", false)
	_, token1, _, _ := gm.JoinGame(game.RoomCode, "Bob")
	_, token2, _, _ := gm.JoinGame(game.RoomCode, "Charlie")
	_, token3, _, _ := gm.JoinGame(game.RoomCode, "Diana")

	gm.SetReady(game.RoomCode, token0, true)
	gm.SetReady(game.RoomCode, token1, true)
	gm.SetReady(game.RoomCode, token2, true)
	gm.SetReady(game.RoomCode, token3, true)

	// Record lobby order
	lobbyOrder := game.Config.PlayerOrder

	// Start
	gm.StartGame(game.RoomCode)

	// Verify game order matches lobby order exactly
	for i, player := range game.Game.Players {
		assert.Equal(lobbyOrder[i], player.Name, "Player %d should match lobby order", i)
	}
}

// TestStartGame_NotAllReady verifies error when not all ready.
func TestStartGame_NotAllReady(t *testing.T) {
	assert := assert.New(t)
	gm := NewGameManager()

	game, token0, _ := gm.CreateGame("Alice", true)
	_, token1, _, _ := gm.JoinGame(game.RoomCode, "Bob")
	gm.JoinGame(game.RoomCode, "Charlie")
	gm.JoinGame(game.RoomCode, "Diana")

	// Only 2 ready
	gm.SetReady(game.RoomCode, token0, true)
	gm.SetReady(game.RoomCode, token1, true)

	// Try to start
	err := gm.StartGame(game.RoomCode)

	assert.Error(err)

	// Status should still be lobby
	assert.Equal(StatusLobby, game.Status)
	assert.Nil(game.Game)
}

// TestStartGame_AlreadyStarted verifies idempotency guard.
func TestStartGame_AlreadyStarted(t *testing.T) {
	assert := assert.New(t)
	gm := NewGameManager()

	// Create and start game
	game, token0, _ := gm.CreateGame("Alice", true)
	_, token1, _, _ := gm.JoinGame(game.RoomCode, "Bob")
	_, token2, _, _ := gm.JoinGame(game.RoomCode, "Charlie")
	_, token3, _, _ := gm.JoinGame(game.RoomCode, "Diana")

	gm.SetReady(game.RoomCode, token0, true)
	gm.SetReady(game.RoomCode, token1, true)
	gm.SetReady(game.RoomCode, token2, true)
	gm.SetReady(game.RoomCode, token3, true)

	gm.StartGame(game.RoomCode)

	// Try to start again
	err := gm.StartGame(game.RoomCode)

	assert.Error(err)
}

// TestStartGame_RoomNotFound verifies error for invalid room.
func TestStartGame_RoomNotFound(t *testing.T) {
	assert := assert.New(t)
	gm := NewGameManager()

	err := gm.StartGame("ZZZZ")

	assert.Error(err)
}

// TestStartGame_UsesPlayerOrder verifies team configuration.
func TestStartGame_UsesPlayerOrder(t *testing.T) {
	assert := assert.New(t)
	gm := NewGameManager()

	game, token0, _ := gm.CreateGame("Alice", false) // Fixed order
	_, token1, _, _ := gm.JoinGame(game.RoomCode, "Bob")
	_, token2, _, _ := gm.JoinGame(game.RoomCode, "Charlie")
	_, token3, _, _ := gm.JoinGame(game.RoomCode, "Diana")

	// Manually rearrange PlayerOrder (simulating UpdateTeamOrder)
	// Original: [Alice, Bob, Charlie, Diana]
	// New: [Bob, Alice, Diana, Charlie]
	game.Config.PlayerOrder = [4]string{"Bob", "Alice", "Diana", "Charlie"}

	gm.SetReady(game.RoomCode, token0, true)
	gm.SetReady(game.RoomCode, token1, true)
	gm.SetReady(game.RoomCode, token2, true)
	gm.SetReady(game.RoomCode, token3, true)

	gm.StartGame(game.RoomCode)

	// Verify game uses new order
	assert.Equal("Bob", game.Game.Players[0].Name)
	assert.Equal("Alice", game.Game.Players[1].Name)
	assert.Equal("Diana", game.Game.Players[2].Name)
	assert.Equal("Charlie", game.Game.Players[3].Name)
}

// TestStartGame_UpdatesTimestamp verifies UpdatedAt changes.
func TestStartGame_UpdatesTimestamp(t *testing.T) {
	assert := assert.New(t)
	gm := NewGameManager()

	game, token0, _ := gm.CreateGame("Alice", true)
	_, token1, _, _ := gm.JoinGame(game.RoomCode, "Bob")
	_, token2, _, _ := gm.JoinGame(game.RoomCode, "Charlie")
	_, token3, _, _ := gm.JoinGame(game.RoomCode, "Diana")

	gm.SetReady(game.RoomCode, token0, true)
	gm.SetReady(game.RoomCode, token1, true)
	gm.SetReady(game.RoomCode, token2, true)
	gm.SetReady(game.RoomCode, token3, true)

	originalTime := game.UpdatedAt
	time.Sleep(10 * time.Millisecond)

	gm.StartGame(game.RoomCode)

	assert.True(game.UpdatedAt.After(originalTime))
}

// TestUpdateTeamOrder_Success verifies creator can rearrange.
func TestUpdateTeamOrder_Success(t *testing.T) {
	assert := assert.New(t)
	gm := NewGameManager()

	// Create lobby with 4 players
	game, creatorToken, _ := gm.CreateGame("Alice", false)
	gm.JoinGame(game.RoomCode, "Bob")
	gm.JoinGame(game.RoomCode, "Charlie")
	gm.JoinGame(game.RoomCode, "Diana")

	// Original order: [Alice, Bob, Charlie, Diana]
	assert.Equal([4]string{"Alice", "Bob", "Charlie", "Diana"}, game.Config.PlayerOrder)

	// Rearrange: Team A = Bob & Charlie, Team B = Alice & Diana
	newOrder := [4]string{"Bob", "Alice", "Charlie", "Diana"}

	updatedGame, err := gm.UpdateTeamOrder(game.RoomCode, creatorToken, newOrder)

	assert.NoError(err)
	assert.NotNil(updatedGame)

	// Verify order updated
	assert.Equal(newOrder, game.Config.PlayerOrder)

	// Verify Players array unchanged (still tracks original join order)
	assert.Equal("Alice", game.Players[0].Username)
	assert.Equal("Bob", game.Players[1].Username)
}

// TestUpdateTeamOrder_NotCreator verifies permission check.
func TestUpdateTeamOrder_NotCreator(t *testing.T) {
	assert := assert.New(t)
	gm := NewGameManager()

	game, _, _ := gm.CreateGame("Alice", false)
	_, bobToken, _, _ := gm.JoinGame(game.RoomCode, "Bob")
	gm.JoinGame(game.RoomCode, "Charlie")
	gm.JoinGame(game.RoomCode, "Diana")

	// Bob tries to update (he's not creator)
	newOrder := [4]string{"Bob", "Alice", "Charlie", "Diana"}
	_, err := gm.UpdateTeamOrder(game.RoomCode, bobToken, newOrder)

	assert.Error(err)

	// Order should be unchanged
	assert.Equal([4]string{"Alice", "Bob", "Charlie", "Diana"}, game.Config.PlayerOrder)
}

// TestUpdateTeamOrder_InvalidNames verifies validation.
func TestUpdateTeamOrder_InvalidNames(t *testing.T) {
	assert := assert.New(t)
	gm := NewGameManager()

	game, creatorToken, _ := gm.CreateGame("Alice", false)
	gm.JoinGame(game.RoomCode, "Bob")
	gm.JoinGame(game.RoomCode, "Charlie")
	gm.JoinGame(game.RoomCode, "Diana")

	// Try to include "Eve" who isn't in lobby
	newOrder := [4]string{"Alice", "Bob", "Charlie", "Eve"}
	_, err := gm.UpdateTeamOrder(game.RoomCode, creatorToken, newOrder)

	assert.Error(err)
	assert.Contains(err.Error(), "Invalid player name")
}

// TestUpdateTeamOrder_DuplicateNames verifies no duplicates.
func TestUpdateTeamOrder_DuplicateNames(t *testing.T) {
	assert := assert.New(t)
	gm := NewGameManager()

	game, creatorToken, _ := gm.CreateGame("Alice", false)
	gm.JoinGame(game.RoomCode, "Bob")
	gm.JoinGame(game.RoomCode, "Charlie")
	gm.JoinGame(game.RoomCode, "Diana")

	// Try to put Alice twice
	newOrder := [4]string{"Alice", "Bob", "Alice", "Diana"}
	_, err := gm.UpdateTeamOrder(game.RoomCode, creatorToken, newOrder)

	assert.Error(err)
}

// TestUpdateTeamOrder_GameAlreadyStarted verifies status check.
func TestUpdateTeamOrder_GameAlreadyStarted(t *testing.T) {
	assert := assert.New(t)
	gm := NewGameManager()

	game, creatorToken, _ := gm.CreateGame("Alice", false)
	gm.JoinGame(game.RoomCode, "Bob")
	gm.JoinGame(game.RoomCode, "Charlie")
	gm.JoinGame(game.RoomCode, "Diana")

	// Start game (manually set status for test)
	game.Status = StatusPlaying

	// Try to update order
	newOrder := [4]string{"Bob", "Alice", "Diana", "Charlie"}
	_, err := gm.UpdateTeamOrder(game.RoomCode, creatorToken, newOrder)

	assert.Error(err)
}

// TestUpdateTeamOrder_RoomNotFound verifies error handling.
func TestUpdateTeamOrder_RoomNotFound(t *testing.T) {
	assert := assert.New(t)
	gm := NewGameManager()

	newOrder := [4]string{"Alice", "Bob", "Charlie", "Diana"}
	_, err := gm.UpdateTeamOrder("ZZZZ", "token", newOrder)

	assert.Error(err)
}

// TestUpdateTeamOrder_AfterCreatorPromotion verifies new creator can update.
// NOTE: This test depends on LeaveGame/promoteNewCreator implementation
// Can be added after implementing LeaveGame
func TestUpdateTeamOrder_AfterCreatorPromotion(t *testing.T) {
	assert := assert.New(t)
	gm := NewGameManager()

	game, creatorToken, _ := gm.CreateGame("Alice", false)
	_, bobToken, _, _ := gm.JoinGame(game.RoomCode, "Bob")
	gm.JoinGame(game.RoomCode, "Charlie")
	gm.JoinGame(game.RoomCode, "Diana")

	// Alice leaves (Bob promoted to slot 0)
	gm.LeaveGame(game.RoomCode, creatorToken)

	// Bob (new creator) should be able to update
	newOrder := [4]string{"Bob", "Charlie", "Diana", ""} // Alice's slot empty
	_, err := gm.UpdateTeamOrder(game.RoomCode, bobToken, newOrder)

	// Should succeed (Bob is now slot 0)
	assert.NoError(err)
}

// TestUpdateTeamOrder_PreservesEmptySlots verifies partial lobbies.
func TestUpdateTeamOrder_PreservesEmptySlots(t *testing.T) {
	assert := assert.New(t)
	gm := NewGameManager()

	// Only 3 players
	game, creatorToken, _ := gm.CreateGame("Alice", false)
	gm.JoinGame(game.RoomCode, "Bob")
	gm.JoinGame(game.RoomCode, "Charlie")

	// Rearrange the 3 players (slot 3 empty)
	newOrder := [4]string{"Bob", "Alice", "Charlie", ""}

	updatedGame, err := gm.UpdateTeamOrder(game.RoomCode, creatorToken, newOrder)

	assert.NoError(err)
	assert.Equal(newOrder, updatedGame.Config.PlayerOrder)
}

// TestUpdateTeamOrder_UpdatesTimestamp verifies timestamp tracking.
func TestUpdateTeamOrder_UpdatesTimestamp(t *testing.T) {
	assert := assert.New(t)
	gm := NewGameManager()

	game, creatorToken, _ := gm.CreateGame("Alice", false)
	gm.JoinGame(game.RoomCode, "Bob")
	gm.JoinGame(game.RoomCode, "Charlie")
	gm.JoinGame(game.RoomCode, "Diana")

	originalTime := game.UpdatedAt
	time.Sleep(10 * time.Millisecond)

	newOrder := [4]string{"Bob", "Alice", "Diana", "Charlie"}
	gm.UpdateTeamOrder(game.RoomCode, creatorToken, newOrder)

	assert.True(game.UpdatedAt.After(originalTime))
}

// TestUpdateTeamOrder_MultipleUpdates verifies can update multiple times.
func TestUpdateTeamOrder_MultipleUpdates(t *testing.T) {
	assert := assert.New(t)
	gm := NewGameManager()

	game, creatorToken, _ := gm.CreateGame("Alice", false)
	gm.JoinGame(game.RoomCode, "Bob")
	gm.JoinGame(game.RoomCode, "Charlie")
	gm.JoinGame(game.RoomCode, "Diana")

	// First update
	order1 := [4]string{"Bob", "Alice", "Diana", "Charlie"}
	gm.UpdateTeamOrder(game.RoomCode, creatorToken, order1)
	assert.Equal(order1, game.Config.PlayerOrder)

	// Second update
	order2 := [4]string{"Charlie", "Diana", "Bob", "Alice"}
	gm.UpdateTeamOrder(game.RoomCode, creatorToken, order2)
	assert.Equal(order2, game.Config.PlayerOrder)

	// Third update
	order3 := [4]string{"Alice", "Bob", "Charlie", "Diana"}
	gm.UpdateTeamOrder(game.RoomCode, creatorToken, order3)
	assert.Equal(order3, game.Config.PlayerOrder)
}

// TestLeaveGame_NonCreator verifies normal player leave.
func TestLeaveGame_NonCreator(t *testing.T) {
	assert := assert.New(t)
	gm := NewGameManager()

	game, _, _ := gm.CreateGame("Alice", true)
	_, bobToken, _, _ := gm.JoinGame(game.RoomCode, "Bob")
	gm.JoinGame(game.RoomCode, "Charlie")

	// Bob leaves
	updatedGame, err := gm.LeaveGame(game.RoomCode, bobToken)

	assert.NoError(err)
	assert.NotNil(updatedGame)

	// Verify Bob marked disconnected
	assert.Equal("Bob", game.Players[1].Username) // Still in slot 1
	assert.False(game.Players[1].Connected)       // But disconnected
	assert.False(game.Players[1].Ready)           // And not ready

	// Verify Alice still creator
	assert.Equal("Alice", game.Players[0].Username)
	assert.True(game.Players[0].Connected)
}

// TestLeaveGame_CreatorLeaves_PromotesSlot1 verifies promotion.
func TestLeaveGame_CreatorLeaves_PromotesSlot1(t *testing.T) {
	assert := assert.New(t)
	gm := NewGameManager()

	game, aliceToken, _ := gm.CreateGame("Alice", true)
	gm.JoinGame(game.RoomCode, "Bob")
	gm.JoinGame(game.RoomCode, "Charlie")

	// Alice leaves
	gm.LeaveGame(game.RoomCode, aliceToken)

	// Verify Bob promoted to slot 0
	assert.Equal("Bob", game.Players[0].Username)
	assert.True(game.Players[0].Connected)
	assert.False(game.Players[0].Ready) // Unreadied due to promotion

	// Verify slot 1 now empty
	assert.Empty(game.Players[1].Username)

	// Verify PlayerOrder updated
	assert.Equal("Bob", game.Config.PlayerOrder[0])
	assert.Equal("", game.Config.PlayerOrder[1])
}

// TestLeaveGame_CreatorLeaves_SkipsDisconnected verifies search logic.
func TestLeaveGame_CreatorLeaves_SkipsDisconnected(t *testing.T) {
	assert := assert.New(t)
	gm := NewGameManager()

	game, aliceToken, _ := gm.CreateGame("Alice", true)
	_, bobToken, _, _ := gm.JoinGame(game.RoomCode, "Bob")
	gm.JoinGame(game.RoomCode, "Charlie")
	gm.JoinGame(game.RoomCode, "Diana")

	// Bob leaves (now disconnected)
	gm.LeaveGame(game.RoomCode, bobToken)

	// Alice leaves (creator)
	gm.LeaveGame(game.RoomCode, aliceToken)

	// Verify Charlie promoted (slot 2), not Bob (disconnected)
	assert.Equal("Charlie", game.Players[0].Username)
	assert.Empty(game.Players[2].Username) // Charlie's old slot
}

// TestLeaveGame_CreatorLeaves_AllGone verifies expiry.
func TestLeaveGame_CreatorLeaves_AllGone(t *testing.T) {
	assert := assert.New(t)
	gm := NewGameManager()

	game, aliceToken, _ := gm.CreateGame("Alice", true)

	// Only Alice in lobby, she leaves
	gm.LeaveGame(game.RoomCode, aliceToken)

	// Verify lobby expired
	now := time.Now()
	assert.True(game.LobbyExpiry.Before(now) || game.LobbyExpiry.Equal(now))
}

// TestLeaveGame_UnreadiesPlayer verifies ready state reset.
func TestLeaveGame_UnreadiesPlayer(t *testing.T) {
	assert := assert.New(t)
	gm := NewGameManager()

	game, _, _ := gm.CreateGame("Alice", true)
	_, bobToken, _, _ := gm.JoinGame(game.RoomCode, "Bob")

	// Bob readies up
	gm.SetReady(game.RoomCode, bobToken, true)
	assert.True(game.Players[1].Ready)

	// Bob leaves
	gm.LeaveGame(game.RoomCode, bobToken)

	// Verify Bob unreadied
	assert.False(game.Players[1].Ready)
}

// TestLeaveGame_PromotedPlayerUnreadied verifies promotion unreadies.
func TestLeaveGame_PromotedPlayerUnreadied(t *testing.T) {
	assert := assert.New(t)
	gm := NewGameManager()

	game, aliceToken, _ := gm.CreateGame("Alice", true)
	_, bobToken, _, _ := gm.JoinGame(game.RoomCode, "Bob")

	// Bob readies up in slot 1
	gm.SetReady(game.RoomCode, bobToken, true)
	assert.True(game.Players[1].Ready)

	// Alice leaves (Bob promoted)
	gm.LeaveGame(game.RoomCode, aliceToken)

	// Verify Bob in slot 0 but not ready
	assert.Equal("Bob", game.Players[0].Username)
	assert.False(game.Players[0].Ready)
}

// TestLeaveGame_GameAlreadyStarted verifies status check.
func TestLeaveGame_GameAlreadyStarted(t *testing.T) {
	assert := assert.New(t)
	gm := NewGameManager()

	game, aliceToken, _ := gm.CreateGame("Alice", true)

	// Manually start game
	game.Status = StatusPlaying

	// Try to leave
	_, err := gm.LeaveGame(game.RoomCode, aliceToken)

	assert.Error(err)
}

// TestLeaveGame_InvalidToken verifies token validation.
func TestLeaveGame_InvalidToken(t *testing.T) {
	assert := assert.New(t)
	gm := NewGameManager()

	game, _, _ := gm.CreateGame("Alice", true)

	_, err := gm.LeaveGame(game.RoomCode, "invalid-token")

	assert.Error(err)
}

// TestLeaveGame_RoomNotFound verifies error handling.
func TestLeaveGame_RoomNotFound(t *testing.T) {
	assert := assert.New(t)
	gm := NewGameManager()

	_, err := gm.LeaveGame("ZZZZ", "token")

	assert.Error(err)
}

// TestLeaveGame_UpdatesTimestamp verifies timestamp tracking.
func TestLeaveGame_UpdatesTimestamp(t *testing.T) {
	assert := assert.New(t)
	gm := NewGameManager()

	game, _, _ := gm.CreateGame("Alice", true)
	_, bobToken, _, _ := gm.JoinGame(game.RoomCode, "Bob")

	originalTime := game.UpdatedAt
	time.Sleep(10 * time.Millisecond)

	gm.LeaveGame(game.RoomCode, bobToken)

	assert.True(game.UpdatedAt.After(originalTime))
}

// TestGetGame_Success verifies basic lookup.
//
// Why this test:
// - Happy path - game exists
func TestGetGame_Success(t *testing.T) {
	assert := assert.New(t)
	gm := NewGameManager()

	createdGame, _, _ := gm.CreateGame("Alice", true)
	roomCode := createdGame.RoomCode

	// Look up game
	foundGame, err := gm.GetGame(roomCode)

	assert.NoError(err)
	assert.NotNil(foundGame)
	assert.Equal(createdGame, foundGame)
	assert.Equal(roomCode, foundGame.RoomCode)
}

// TestGetGame_NotFound verifies error handling.
func TestGetGame_NotFound(t *testing.T) {
	assert := assert.New(t)
	gm := NewGameManager()

	game, err := gm.GetGame("ZZZZ")

	assert.Error(err)
	assert.Nil(game)
}

// TestGetGame_MultipleGames verifies correct game returned.
func TestGetGame_MultipleGames(t *testing.T) {
	assert := assert.New(t)
	gm := NewGameManager()

	// Create 3 games
	_, _, _ = gm.CreateGame("Alice", true)
	game2, _, _ := gm.CreateGame("Bob", true)
	_, _, _ = gm.CreateGame("Charlie", true)

	// Look up middle one
	found, err := gm.GetGame(game2.RoomCode)

	assert.NoError(err)
	assert.Equal(game2, found)
	assert.Equal("Bob", found.Players[0].Username)
}

// TestGetGameByToken_Success verifies token lookup.
func TestGetGameByToken_Success(t *testing.T) {
	assert := assert.New(t)
	gm := NewGameManager()

	game, token, _ := gm.CreateGame("Alice", true)

	// Look up by token
	foundGame, slotID, err := gm.GetGameByToken(token)

	assert.NoError(err)
	assert.NotNil(foundGame)
	assert.Equal(game, foundGame)
	assert.Equal(0, slotID) // Creator is slot 0
}

// TestGetGameByToken_NotFound verifies error handling.
func TestGetGameByToken_NotFound(t *testing.T) {
	assert := assert.New(t)
	gm := NewGameManager()

	gm.CreateGame("Alice", true)

	// Try with fake token
	game, slotID, err := gm.GetGameByToken("invalid-token-12345")

	assert.Error(err)
	assert.Nil(game)
	assert.Equal(-1, slotID)
}

// TestGetGameByToken_MultipleGamesMultiplePlayers verifies search.
func TestGetGameByToken_MultipleGamesMultiplePlayers(t *testing.T) {
	assert := assert.New(t)
	gm := NewGameManager()

	// Game 1: Alice, Bob
	game1, _, _ := gm.CreateGame("Alice", true)
	_, bobToken, _, _ := gm.JoinGame(game1.RoomCode, "Bob")

	// Game 2: Charlie, Diana
	game2, _, _ := gm.CreateGame("Charlie", true)
	_, dianaToken, _, _ := gm.JoinGame(game2.RoomCode, "Diana")

	// Look up Bob (game 1, slot 1)
	foundGame, slotID, err := gm.GetGameByToken(bobToken)
	assert.NoError(err)
	assert.Equal(game1, foundGame)
	assert.Equal(1, slotID)

	// Look up Diana (game 2, slot 1)
	foundGame, slotID, err = gm.GetGameByToken(dianaToken)
	assert.NoError(err)
	assert.Equal(game2, foundGame)
	assert.Equal(1, slotID)
}

// TestGetGameByToken_AllSlots verifies all 4 slots searchable.
//
// Why this test:
// - Ensures search covers all slots (0-3)
func TestGetGameByToken_AllSlots(t *testing.T) {
	assert := assert.New(t)
	gm := NewGameManager()

	game, token0, _ := gm.CreateGame("Alice", true)
	_, token1, _, _ := gm.JoinGame(game.RoomCode, "Bob")
	_, token2, _, _ := gm.JoinGame(game.RoomCode, "Charlie")
	_, token3, _, _ := gm.JoinGame(game.RoomCode, "Diana")

	tokens := []string{token0, token1, token2, token3}

	// Verify each token found with correct slot
	for expectedSlot, token := range tokens {
		foundGame, actualSlot, err := gm.GetGameByToken(token)
		assert.NoError(err)
		assert.Equal(game, foundGame)
		assert.Equal(expectedSlot, actualSlot)
	}
}

// TestGetGame_ThreadSafe verifies concurrent access.
//
// Why this test:
// - Validates read lock works correctly
// - Multiple goroutines can lookup simultaneously
func TestGetGame_ThreadSafe(t *testing.T) {
	assert := assert.New(t)
	gm := NewGameManager()

	game, _, _ := gm.CreateGame("Alice", true)
	roomCode := game.RoomCode

	// Launch 10 concurrent lookups
	results := make(chan error, 10)
	for i := 0; i < 10; i++ {
		go func() {
			_, err := gm.GetGame(roomCode)
			results <- err
		}()
	}

	// All should succeed
	for i := 0; i < 10; i++ {
		err := <-results
		assert.NoError(err)
	}
}

// TestGetGameByToken_ThreadSafe verifies concurrent token lookup.
//
// Why this test:
// - Validates read lock for token search
func TestGetGameByToken_ThreadSafe(t *testing.T) {
	assert := assert.New(t)
	gm := NewGameManager()

	_, token, _ := gm.CreateGame("Alice", true)

	// Launch 10 concurrent token lookups
	results := make(chan error, 10)
	for i := 0; i < 10; i++ {
		go func() {
			_, _, err := gm.GetGameByToken(token)
			results <- err
		}()
	}

	// All should succeed
	for i := 0; i < 10; i++ {
		err := <-results
		assert.NoError(err)
	}
}

// Test: Reconnect player successfully
// Why: Core reconnection functionality - player reconnects after disconnect
func TestGameManager_ReconnectPlayer_Success(t *testing.T) {
	assert := assert.New(t)
	gm := NewGameManager()

	// Create game with 4 players and start it
	game, token1, _ := gm.CreateGame("Alice", false)
	roomCode := game.RoomCode

	_, token2, _, _ := gm.JoinGame(roomCode, "Bob")
	_, token3, _, _ := gm.JoinGame(roomCode, "Charlie")
	_, token4, _, _ := gm.JoinGame(roomCode, "Diana")

	// Ready all players
	gm.SetReady(roomCode, token1, true)
	gm.SetReady(roomCode, token2, true)
	gm.SetReady(roomCode, token3, true)
	gm.SetReady(roomCode, token4, true)

	// Start game
	err := gm.StartGame(roomCode)
	assert.NoError(err)

	// Verify game is playing
	game, _ = gm.GetGame(roomCode)
	assert.Equal(StatusPlaying, game.Status)

	// Player 0 disconnects
	shouldPause, game, playerID, err := gm.MarkPlayerDisconnected(token1)
	assert.NoError(err)
	assert.True(shouldPause, "Game should pause when player disconnects")
	assert.Equal(0, playerID)
	assert.False(game.Players[0].Connected)
	assert.Equal(StatusPaused, game.Status)

	// Player 0 reconnects
	game, err = gm.ReconnectPlayer(token1, roomCode, 0)
	assert.NoError(err)
	assert.True(game.Players[0].Connected)
	assert.Equal(StatusPlaying, game.Status, "Game should resume when all players reconnected")
}

// Test: Reconnect to lobby (Status=lobby)
// Why: Players should be able to reconnect to lobby, not just active games
func TestGameManager_ReconnectPlayer_ToLobby(t *testing.T) {
	assert := assert.New(t)
	gm := NewGameManager()

	game, token, _ := gm.CreateGame("Alice", false)
	roomCode := game.RoomCode

	// Disconnect from lobby
	game.Players[0].Connected = false

	// Reconnect to lobby
	game, err := gm.ReconnectPlayer(token, roomCode, 0)
	assert.NoError(err)
	assert.True(game.Players[0].Connected)
	assert.Equal(StatusLobby, game.Status, "Game should still be in lobby")
}

// Test: Reconnect with invalid token
// Why: Security - can't reconnect as someone else
func TestGameManager_ReconnectPlayer_InvalidToken(t *testing.T) {
	assert := assert.New(t)
	gm := NewGameManager()

	game, _, _ := gm.CreateGame("Alice", false)
	roomCode := game.RoomCode

	// Try to reconnect with wrong token
	_, err := gm.ReconnectPlayer("wrong-token", roomCode, 0)
	assert.Error(err)
	assert.Contains(err.Error(), "TOKEN_MISMATCH")
}

// Test: Reconnect to non-existent game
// Why: Handle case where game was deleted
func TestGameManager_ReconnectPlayer_GameNotFound(t *testing.T) {
	assert := assert.New(t)
	gm := NewGameManager()

	_, err := gm.ReconnectPlayer("any-token", "FAKE", 0)
	assert.Error(err)
	assert.Contains(err.Error(), "ROOM_NOT_FOUND")
}

// Test: Reconnect with invalid player ID
// Why: Validate player ID is in range [0,3]
func TestGameManager_ReconnectPlayer_InvalidPlayerID(t *testing.T) {
	assert := assert.New(t)
	gm := NewGameManager()

	game, token, _ := gm.CreateGame("Alice", false)
	roomCode := game.RoomCode

	// Try to reconnect with invalid player IDs
	_, err := gm.ReconnectPlayer(token, roomCode, -1)
	assert.Error(err)
	assert.Contains(err.Error(), "INVALID_PLAYER_ID")

	_, err = gm.ReconnectPlayer(token, roomCode, 4)
	assert.Error(err)
	assert.Contains(err.Error(), "INVALID_PLAYER_ID")
}

// Test: Reconnect to empty slot
// Why: Can't reconnect to slot with no player
func TestGameManager_ReconnectPlayer_EmptySlot(t *testing.T) {
	assert := assert.New(t)
	gm := NewGameManager()

	game, _, _ := gm.CreateGame("Alice", false)
	roomCode := game.RoomCode

	// Try to reconnect to empty slot 1
	_, err := gm.ReconnectPlayer("some-token", roomCode, 1)
	assert.Error(err)
	// Token mismatch happens before empty slot check (security first)
	assert.Contains(err.Error(), "TOKEN_MISMATCH")
}

// Test: Multiple players disconnect and reconnect
// Why: Game should only resume when ALL players reconnected
func TestGameManager_ReconnectPlayer_MultipleDisconnects(t *testing.T) {
	assert := assert.New(t)
	gm := NewGameManager()

	// Create game with 4 players and start it
	game, token1, _ := gm.CreateGame("Alice", false)
	roomCode := game.RoomCode

	_, token2, _, _ := gm.JoinGame(roomCode, "Bob")
	_, token3, _, _ := gm.JoinGame(roomCode, "Charlie")
	_, token4, _, _ := gm.JoinGame(roomCode, "Diana")

	// Ready all and start
	gm.SetReady(roomCode, token1, true)
	gm.SetReady(roomCode, token2, true)
	gm.SetReady(roomCode, token3, true)
	gm.SetReady(roomCode, token4, true)
	gm.StartGame(roomCode)

	// Players 0 and 2 disconnect
	shouldPause, game, _, _ := gm.MarkPlayerDisconnected(token1)
	assert.True(shouldPause)
	assert.Equal(StatusPaused, game.Status)

	shouldPause, game, _, _ = gm.MarkPlayerDisconnected(token3)
	assert.False(shouldPause, "Already paused, should not pause again")
	assert.Equal(StatusPaused, game.Status)

	// Player 0 reconnects - game still paused (Player 2 still disconnected)
	game, err := gm.ReconnectPlayer(token1, roomCode, 0)
	assert.NoError(err)
	assert.Equal(StatusPaused, game.Status, "Game should stay paused until all reconnect")

	// Player 2 reconnects - now game resumes
	game, err = gm.ReconnectPlayer(token3, roomCode, 2)
	assert.NoError(err)
	assert.Equal(StatusPlaying, game.Status, "Game should resume when all players back")
}

// Test: Mark player disconnected
// Why: Core disconnect functionality - mark player as disconnected
func TestGameManager_MarkPlayerDisconnected_Success(t *testing.T) {
	assert := assert.New(t)
	gm := NewGameManager()

	// Create and start game
	game, token1, _ := gm.CreateGame("Alice", false)
	roomCode := game.RoomCode

	_, token2, _, _ := gm.JoinGame(roomCode, "Bob")
	_, token3, _, _ := gm.JoinGame(roomCode, "Charlie")
	_, token4, _, _ := gm.JoinGame(roomCode, "Diana")

	gm.SetReady(roomCode, token1, true)
	gm.SetReady(roomCode, token2, true)
	gm.SetReady(roomCode, token3, true)
	gm.SetReady(roomCode, token4, true)
	gm.StartGame(roomCode)

	// Disconnect player 1
	shouldPause, game, playerID, err := gm.MarkPlayerDisconnected(token2)
	assert.NoError(err)
	assert.True(shouldPause)
	assert.Equal(1, playerID)
	assert.False(game.Players[1].Connected)
	assert.Equal(StatusPaused, game.Status)
}

// Test: Mark player disconnected from lobby (should NOT pause)
// Why: Lobby doesn't pause, only playing games do
func TestGameManager_MarkPlayerDisconnected_FromLobby(t *testing.T) {
	assert := assert.New(t)
	gm := NewGameManager()

	game, token, _ := gm.CreateGame("Alice", false)

	// Disconnect from lobby
	shouldPause, game, playerID, err := gm.MarkPlayerDisconnected(token)
	assert.NoError(err)
	assert.False(shouldPause, "Lobby should not pause")
	assert.Equal(0, playerID)
	assert.False(game.Players[0].Connected)
	assert.Equal(StatusLobby, game.Status)
}

// Test: Mark player disconnected with invalid token
// Why: Error handling for invalid token
func TestGameManager_MarkPlayerDisconnected_InvalidToken(t *testing.T) {
	assert := assert.New(t)
	gm := NewGameManager()

	_, _, _, err := gm.MarkPlayerDisconnected("invalid-token")
	assert.Error(err)
	assert.Contains(err.Error(), "TOKEN_NOT_FOUND")
}

// Test: Pause game
// Why: Explicit pause method for game state transition
func TestGameManager_PauseGame_Success(t *testing.T) {
	assert := assert.New(t)
	gm := NewGameManager()

	// Create and start game
	game, token1, _ := gm.CreateGame("Alice", false)
	roomCode := game.RoomCode

	_, token2, _, _ := gm.JoinGame(roomCode, "Bob")
	_, token3, _, _ := gm.JoinGame(roomCode, "Charlie")
	_, token4, _, _ := gm.JoinGame(roomCode, "Diana")

	gm.SetReady(roomCode, token1, true)
	gm.SetReady(roomCode, token2, true)
	gm.SetReady(roomCode, token3, true)
	gm.SetReady(roomCode, token4, true)
	gm.StartGame(roomCode)

	// Pause game
	didPause, err := gm.PauseGame(roomCode)
	assert.NoError(err)
	assert.True(didPause)

	game, _ = gm.GetGame(roomCode)
	assert.Equal(StatusPaused, game.Status)
}

// Test: Pause game that's already paused (no-op)
// Why: Should handle gracefully without error
func TestGameManager_PauseGame_AlreadyPaused(t *testing.T) {
	assert := assert.New(t)
	gm := NewGameManager()

	// Create and start game
	game, token1, _ := gm.CreateGame("Alice", false)
	roomCode := game.RoomCode

	_, token2, _, _ := gm.JoinGame(roomCode, "Bob")
	_, token3, _, _ := gm.JoinGame(roomCode, "Charlie")
	_, token4, _, _ := gm.JoinGame(roomCode, "Diana")

	gm.SetReady(roomCode, token1, true)
	gm.SetReady(roomCode, token2, true)
	gm.SetReady(roomCode, token3, true)
	gm.SetReady(roomCode, token4, true)
	gm.StartGame(roomCode)

	// Pause once
	didPause, err := gm.PauseGame(roomCode)
	assert.NoError(err)
	assert.True(didPause)

	// Pause again - should be no-op
	didPause, err = gm.PauseGame(roomCode)
	assert.NoError(err)
	assert.False(didPause, "Should not pause again if already paused")

	game, _ = gm.GetGame(roomCode)
	assert.Equal(StatusPaused, game.Status)
}

// Test: Pause lobby (should not pause)
// Why: Only playing games can be paused
func TestGameManager_PauseGame_Lobby(t *testing.T) {
	assert := assert.New(t)
	gm := NewGameManager()

	game, _, _ := gm.CreateGame("Alice", false)
	roomCode := game.RoomCode

	// Try to pause lobby
	didPause, err := gm.PauseGame(roomCode)
	assert.NoError(err)
	assert.False(didPause, "Lobby should not pause")

	game, _ = gm.GetGame(roomCode)
	assert.Equal(StatusLobby, game.Status)
}

// Test: Pause non-existent game
// Why: Error handling
func TestGameManager_PauseGame_NotFound(t *testing.T) {
	assert := assert.New(t)
	gm := NewGameManager()

	_, err := gm.PauseGame("FAKE")
	assert.Error(err)
	assert.Contains(err.Error(), "ROOM_NOT_FOUND")
}

// Test: All players disconnect then all reconnect
// Why: Edge case - complete disconnect/reconnect cycle
func TestGameManager_Reconnection_AllDisconnectAllReconnect(t *testing.T) {
	assert := assert.New(t)
	gm := NewGameManager()

	// Create game with 4 players and start it
	game, token1, _ := gm.CreateGame("Alice", false)
	roomCode := game.RoomCode

	_, token2, _, _ := gm.JoinGame(roomCode, "Bob")
	_, token3, _, _ := gm.JoinGame(roomCode, "Charlie")
	_, token4, _, _ := gm.JoinGame(roomCode, "Diana")

	gm.SetReady(roomCode, token1, true)
	gm.SetReady(roomCode, token2, true)
	gm.SetReady(roomCode, token3, true)
	gm.SetReady(roomCode, token4, true)
	gm.StartGame(roomCode)

	// All players disconnect
	gm.MarkPlayerDisconnected(token1)
	gm.MarkPlayerDisconnected(token2)
	gm.MarkPlayerDisconnected(token3)
	gm.MarkPlayerDisconnected(token4)

	game, _ = gm.GetGame(roomCode)
	assert.Equal(StatusPaused, game.Status)
	assert.False(game.Players[0].Connected)
	assert.False(game.Players[1].Connected)
	assert.False(game.Players[2].Connected)
	assert.False(game.Players[3].Connected)

	// All players reconnect
	gm.ReconnectPlayer(token1, roomCode, 0)
	game, _ = gm.GetGame(roomCode)
	assert.Equal(StatusPaused, game.Status, "Still paused with only 1 player")

	gm.ReconnectPlayer(token2, roomCode, 1)
	game, _ = gm.GetGame(roomCode)
	assert.Equal(StatusPaused, game.Status, "Still paused with only 2 players")

	gm.ReconnectPlayer(token3, roomCode, 2)
	game, _ = gm.GetGame(roomCode)
	assert.Equal(StatusPaused, game.Status, "Still paused with only 3 players")

	gm.ReconnectPlayer(token4, roomCode, 3)
	game, _ = gm.GetGame(roomCode)
	assert.Equal(StatusPlaying, game.Status, "Should resume with all 4 players")
}
