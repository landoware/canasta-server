package server

// ============================================================================
// ERROR RESPONSES
// ============================================================================
// tygo:generate
type ErrorMessage struct {
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

// ============================================================================
// CREATE GAME (create_game)
// ============================================================================
// tygo:generate
type CreateGameRequest struct {
	Username        string `json:"username"`
	RandomTeamOrder bool   `json:"randomTeamOrder"`
}

// tygo:generate
type CreateGameResponse struct {
	RoomCode string `json:"roomCode"`
	Token    string `json:"token"`
	PlayerID int    `json:"playerId"`
}

// ============================================================================
// JOIN GAME (join_game)
// ============================================================================
// tygo:generate
type JoinGameRequest struct {
	RoomCode string `json:"roomCode"`
	Username string `json:"username"`
}

// tygo:generate
type JoinGameResponse struct {
	Success  bool   `json:"success"`
	Token    string `json:"token"`
	PlayerID int    `json:"playerId"`
	Message  string `json:"message,omitempty"`
}

// ============================================================================
// RECONNECT (reconnect) - Phase 3
// ============================================================================
// tygo:generate
type ReconnectRequest struct {
	Token string `json:"token"`
}

// tygo:generate
type ReconnectResponse struct {
	Success  bool   `json:"success"`
	Message  string `json:"message,omitempty"`
	RoomCode string `json:"roomCode,omitempty"`
	PlayerID int    `json:"playerId,omitempty"`
}

// tygo:generate
type PlayerStatusNotification struct {
	PlayerID  int    `json:"playerId"`
	Username  string `json:"username"`
	Connected bool   `json:"connected"`
}

// tygo:generate
type GameResumedNotification struct {
	Message string `json:"message"`
}

// tygo:generate
type GamePausedNotification struct {
	Message string `json:"message"`
}

// tygo:generate
type DisconnectedElsewhereNotification struct {
	Message string `json:"message"`
}

// ============================================================================
// SET READY (set_ready)
// ============================================================================
// tygo:generate
type SetReadyRequest struct {
	Ready bool `json:"ready"`
}

// ============================================================================
// UPDATE TEAM ORDER (update_team_order)
// ============================================================================
// tygo:generate
type UpdateTeamOrderRequest struct {
	PlayerOrder [4]string `json:"playerOrder"`
}

// ============================================================================
// LEAVE GAME (leave_game)
// ============================================================================
// tygo:generate
type LeaveGameRequest struct {
	// No fields - token identifies player
}

// ============================================================================
// LOBBY STATE (lobby_update broadcast)
// ============================================================================
// tygo:generate
type LobbyState struct {
	RoomCode        string         `json:"roomCode"`
	Players         [4]LobbyPlayer `json:"players"`
	PlayerCount     int            `json:"playerCount"`
	RandomTeamOrder bool           `json:"randomTeamOrder"`
	Status          string         `json:"status"`
	AllReady        bool           `json:"allReady"`
}

// tygo:generate
type LobbyPlayer struct {
	Username  string `json:"username"`
	Ready     bool   `json:"ready"`
	Connected bool   `json:"connected"`
	IsYou     bool   `json:"isYou"` // Personalized for each client
}

// ============================================================================
// GAME STARTED (game_started broadcast)
// ============================================================================
// tygo:generate
type GameStartedNotification struct {
	Message string `json:"message"`
}

// ============================================================================
// GAME STATE (game_state broadcast) - Phase 4
// ============================================================================
// tygo:generate
type GameStateMessage struct {
	State         interface{} `json:"state"` // *canasta.ClientState - using interface{} to avoid circular import
	CurrentPlayer int         `json:"currentPlayer"`
	Phase         string      `json:"phase"`
	Status        string      `json:"status"`
}
