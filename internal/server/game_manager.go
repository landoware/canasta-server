package server

import (
	"canasta-server/internal/canasta"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type GameManager struct {
	games     map[string]*ActiveGame
	usedCodes map[string]bool
	mu        sync.RWMutex
}

type ActiveGame struct {
	Game        *canasta.Game
	RoomCode    string
	Config      LobbyConfig
	Status      GameStatus
	Players     [4]PlayerSlot
	CreatedAt   time.Time
	UpdatedAt   time.Time
	LobbyExpiry time.Time
}

type LobbyConfig struct {
	PlayerOrder     [4]string
	RandomTeamOrder bool
}

type PlayerSlot struct {
	Username  string
	Token     string
	Connected bool
	Ready     bool
	JoinedAt  time.Time
}

type GameStatus string

const (
	StatusLobby     GameStatus = "lobby"
	StatusPlaying   GameStatus = "playing"
	StatusPaused    GameStatus = "paused"
	StatusCompleted GameStatus = "completed"
)

func NewGameManager() *GameManager {
	return &GameManager{
		games:     make(map[string]*ActiveGame),
		usedCodes: make(map[string]bool),
	}
}

func (gm *GameManager) CreateGame(username string, randomTeamOrder bool) (*ActiveGame, string, error) {
	if err := gm.validateUsernameFormat(username); err != nil {
		return nil, "", err
	}

	// Generate a Room Code
	gm.mu.Lock()
	roomCode := GenerateRoomCode(gm.usedCodes)
	gm.usedCodes[roomCode] = true
	gm.mu.Unlock()

	// Get a token for the player
	token := uuid.New().String()

	now := time.Now()
	game := &ActiveGame{
		Game:     nil, // Initalize it later, after everyone joins.
		RoomCode: roomCode,
		Status:   StatusLobby,
		Config: LobbyConfig{
			PlayerOrder:     [4]string{},
			RandomTeamOrder: randomTeamOrder,
		},
		Players:     [4]PlayerSlot{},
		CreatedAt:   now,
		UpdatedAt:   now,
		LobbyExpiry: now.Add(10 * time.Minute),
	}

	game.Players[0] = PlayerSlot{
		Username:  username,
		Token:     token,
		Connected: true,
		Ready:     false,
		JoinedAt:  now,
	}

	game.Config.PlayerOrder[0] = username

	gm.mu.Lock()
	gm.games[roomCode] = game
	gm.mu.Unlock()

	return game, token, nil
}

func (gm *GameManager) JoinGame(roomCode, username string) (*ActiveGame, string, int, error) {
	roomCode = strings.ToUpper(roomCode)
	if err := ValidateRoomCode(roomCode); err != nil {
		return nil, "", -1, err
	}

	gm.mu.RLock()
	game, exists := gm.games[roomCode]
	gm.mu.RUnlock()

	if !exists {
		return nil, "", -1, errors.New("ROOM_NOT_FOUND: Game not found")
	}

	if game.Status != StatusLobby {
		return nil, "", -1, errors.New("GAME_ALREADY_STARTED: Cannot join game in progress")
	}

	if err := gm.validateUsername(game, username, -1); err != nil {
		return nil, "", -1, err
	}

	slotId := -1
	for i, slot := range game.Players {
		if slot.Username == "" {
			slotId = i
			break
		}
	}

	if slotId == -1 {
		return nil, "", -1, errors.New("ROOM_FULL: Lobby is full (4/4 players)")
	}

	token := uuid.New().String()

	now := time.Now()
	game.Players[slotId] = PlayerSlot{
		Username:  username,
		Token:     token,
		Connected: true,
		Ready:     false,
		JoinedAt:  now,
	}
	game.Config.PlayerOrder[slotId] = username
	game.UpdatedAt = now

	return game, token, slotId, nil
}

func (gm *GameManager) SetReady(roomCode, token string, ready bool) (*ActiveGame, bool, error) {
	// 1. Look up game
	gm.mu.RLock()
	game, exists := gm.games[roomCode]
	gm.mu.RUnlock()

	if !exists {
		return nil, false, errors.New("ROOM_NOT_FOUND: Game not found")
	}

	if game.Status != StatusLobby {
		return nil, false, errors.New("GAME_ALREADY_STARTED: Cannot change ready state after game starts")
	}

	// 2. Find player by token
	slotID := -1
	for i, slot := range game.Players {
		if slot.Token == token {
			slotID = i
			break
		}
	}

	if slotID == -1 {
		return nil, false, errors.New("NOT_IN_GAME: Invalid token")
	}

	// 4. Update ready state
	game.Players[slotID].Ready = ready
	game.UpdatedAt = time.Now()

	// 5. Check if all players ready
	allReady := gm.checkAllReady(game)

	return game, allReady, nil
}

func (gm *GameManager) StartGame(roomCode string) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	game, exists := gm.games[roomCode]

	if !exists {
		return errors.New("ROOM_NOT_FOUND: Game not found")
	}

	if game.Status != StatusLobby {
		return errors.New("INVALID_STATUS: Game already started")
	}

	if !gm.checkAllReady(game) {
		return errors.New("NOT_ALL_READY: Cannot start game, not all players ready")
	}

	playerNames := game.Config.PlayerOrder

	var canastaGame canasta.Game
	if game.Config.RandomTeamOrder {
		canastaGame = canasta.NewGame(roomCode, playerNames[:])
	} else {
		canastaGame = canasta.NewGame(roomCode, playerNames[:], canasta.WithFixedTeamOrder())
	}

	canastaGame.Deal()

	game.Game = &canastaGame
	game.Status = StatusPlaying
	game.UpdatedAt = time.Now()

	return nil
}

func (gm *GameManager) UpdateTeamOrder(roomCode, creatorToken string, newOrder [4]string) (*ActiveGame, error) {
	gm.mu.RLock()
	game, exists := gm.games[roomCode]
	gm.mu.RUnlock()

	if !exists {
		return nil, errors.New("ROOM_NOT_FOUND: Game not found")
	}

	if game.Status != StatusLobby {
		return nil, errors.New("GAME_ALREADY_STARTED: Cannot change team order after game starts")
	}

	if game.Players[0].Token != creatorToken {
		return nil, errors.New("NOT_CREATOR: Only room creator can update team order")
	}

	if err := gm.validateTeamOrder(game, newOrder); err != nil {
		return nil, err
	}

	game.Config.PlayerOrder = newOrder
	game.UpdatedAt = time.Now()

	return game, nil
}

func (gm *GameManager) LeaveGame(roomCode, token string) (*ActiveGame, error) {
	gm.mu.RLock()
	game, exists := gm.games[roomCode]
	gm.mu.RUnlock()

	if !exists {
		return nil, errors.New("ROOM_NOT_FOUND: Game not found")
	}

	if game.Status != StatusLobby {
		return nil, errors.New("GAME_STARTED: Use disconnect for active games")
	}

	// Find player
	slotID := -1
	for i, slot := range game.Players {
		if slot.Token == token {
			slotID = i
			break
		}
	}

	if slotID == -1 {
		return nil, errors.New("NOT_IN_GAME: Invalid token")
	}

	game.Players[slotID].Connected = false
	game.Players[slotID].Ready = false
	game.UpdatedAt = time.Now()

	if slotID == 0 {
		gm.promoteNewCreator(game)
	}

	return game, nil
}

func (gm *GameManager) GetGame(roomCode string) (*ActiveGame, error) {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	game, exists := gm.games[roomCode]
	if !exists {
		return nil, errors.New("ROOM_NOT_FOUND: Game not found")
	}

	return game, nil
}

func (gm *GameManager) GetGameByToken(token string) (*ActiveGame, int, error) {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	for _, game := range gm.games {
		for i, slot := range game.Players {
			if slot.Token == token {
				return game, i, nil
			}
		}
	}

	return nil, -1, errors.New("TOKEN_NOT_FOUND: Invalid session token")
}

func (gm *GameManager) ReconnectPlayer(token, roomCode string, playerID int) (*ActiveGame, error) {
	gm.mu.RLock()
	game, exists := gm.games[roomCode]
	gm.mu.RUnlock()

	if !exists {
		return nil, errors.New("ROOM_NOT_FOUND: Game not found")
	}

	if playerID < 0 || playerID >= 4 {
		return nil, errors.New("INVALID_PLAYER_ID: Player ID out of range")
	}

	slot := &game.Players[playerID]

	if slot.Token != token {
		return nil, errors.New("TOKEN_MISMATCH: Token does not match player slot")
	}

	if slot.Username == "" {
		return nil, errors.New("INVALID_SLOT: Slot is empty")
	}

	slot.Connected = true
	game.UpdatedAt = time.Now()

	// Resume if we have everyone
	// Why check: If all players reconnected and game was paused, resume
	if game.Status == StatusPaused {
		allConnected := true
		for _, s := range game.Players {
			// Only check slots with players (Username not empty)
			if s.Username != "" && !s.Connected {
				allConnected = false
				break
			}
		}
		if allConnected {
			game.Status = StatusPlaying
		}
	}

	return game, nil
}

func (gm *GameManager) PauseGame(roomCode string) (didPause bool, err error) {
	gm.mu.RLock()
	game, exists := gm.games[roomCode]
	gm.mu.RUnlock()

	if !exists {
		return false, errors.New("ROOM_NOT_FOUND: Game not found")
	}

	if game.Status == StatusPlaying {
		game.Status = StatusPaused
		game.UpdatedAt = time.Now()
		return true, nil
	}
	return false, nil
}

func (gm *GameManager) MarkPlayerDisconnected(token string) (bool, *ActiveGame, int, error) {
	game, playerID, err := gm.GetGameByToken(token)
	if err != nil {
		return false, nil, -1, err
	}

	game.Players[playerID].Connected = false
	game.UpdatedAt = time.Now()

	gamePaused, err := gm.PauseGame(game.RoomCode)
	if err != nil {
		return false, nil, -1, err
	}

	return gamePaused, game, playerID, nil

}

func (gm *GameManager) promoteNewCreator(game *ActiveGame) {
	// Find first connected player in slots 1-3
	newCreatorSlot := -1
	for i := 1; i < 4; i++ {
		if game.Players[i].Username != "" && game.Players[i].Connected {
			newCreatorSlot = i
			break
		}
	}

	// If no one left, mark lobby for expiry
	if newCreatorSlot == -1 {
		game.LobbyExpiry = time.Now() // Expire immediately
		return
	}

	// Swap new creator into slot 0
	game.Players[0] = game.Players[newCreatorSlot]

	// Mark old slot as empty
	game.Players[newCreatorSlot] = PlayerSlot{}

	// Update PlayerOrder to reflect new arrangement
	game.Config.PlayerOrder[0] = game.Players[0].Username
	game.Config.PlayerOrder[newCreatorSlot] = ""

	// Unready the promoted player
	game.Players[0].Ready = false
}

func (gm *GameManager) checkAllReady(game *ActiveGame) bool {
	playerCount := 0
	readyCount := 0

	for _, slot := range game.Players {
		if slot.Username != "" {
			playerCount++
			if slot.Ready {
				readyCount++
			}
		}
	}

	return playerCount == 4 && readyCount == 4
}

func (gm *GameManager) validateUsernameFormat(username string) error {
	username = strings.TrimSpace(username)
	if username == "" {
		return errors.New("USERNAME_INVALID: Username cannot be empty")
	}
	if len(username) > 20 {
		return errors.New("USERNAME_INVALID: Username too long (max 20 characters)")
	}
	return nil
}

func (gm *GameManager) validateUsername(game *ActiveGame, username string, skipSlot int) error {
	err := gm.validateUsernameFormat(username)
	if err != nil {
		return err
	}

	for i, slot := range game.Players {
		if i == skipSlot {
			continue
		}
		if slot.Username == username {
			return errors.New("USERNAME_TAKEN: Username already taken")
		}
	}

	return nil
}

func (gm *GameManager) validateTeamOrder(game *ActiveGame, order [4]string) error {
	// Build set of valid player names (including empty for unfilled slots)
	playerNames := make(map[string]bool)
	playerNames[""] = true // Allow empty strings for unfilled slots
	for _, slot := range game.Players {
		if slot.Username != "" {
			playerNames[slot.Username] = true
		}
	}

	// Check all names in order are valid
	for _, name := range order {
		if !playerNames[name] {
			return errors.New("INVALID_PLAYER: Invalid player name in team order")
		}
	}

	// Check for duplicates (excluding empty strings)
	// Why: Can't have same player in multiple positions
	// Empty slots can repeat (multiple unfilled positions)
	seenNames := make(map[string]bool)
	for _, name := range order {
		if name != "" { // Skip empty slots
			if seenNames[name] {
				return errors.New("DUPLICATE_NAME: Player cannot appear in multiple positions")
			}
			seenNames[name] = true
		}
	}

	return nil
}
