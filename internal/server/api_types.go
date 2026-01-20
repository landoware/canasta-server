package server

// tygo:generate
type ErrorMessage struct {
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

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

// tygo:generate
type SetReadyRequest struct {
	Ready bool `json:"ready"`
}

// tygo:generate
type UpdateTeamOrderRequest struct {
	PlayerOrder [4]string `json:"playerOrder"`
}

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
	IsYou     bool   `json:"isYou"`
}
