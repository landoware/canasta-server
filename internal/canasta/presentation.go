package canasta

// Move represents a single player action
type Move struct {
	Type      MoveType `json:"type" ts_type:"MoveType"`
	PlayerID  string   `json:"playerId"`
	CardIds   []int    `json:"cardIds,omitempty"`
	MeldId    *int     `json:"meldId,omitempty"`
	CanastaId *int     `json:"canastaId,omitempty"`
	Response  *bool    `json:"response,omitempty"` // For yes/no responses (e.g., partner asking to go out)
}

// MoveRequest is what the client sends to the server via WebSocket
type MoveRequest struct {
	GameID string `json:"gameId"`
	Move   Move   `json:"move"`
}

// MoveResponse is what the server sends back to the client
type MoveResponse struct {
	Success   bool       `json:"success"`
	Error     *MoveError `json:"error,omitempty"`
	GameState *GameState `json:"gameState,omitempty"` // Filtered per player
	Message   *string    `json:"message,omitempty"`
}

// MoveError represents specific error types for move validation
type MoveError struct {
	Code    MoveErrorCode `json:"code"`
	Message string        `json:"message"`
}

type MoveErrorCode string

const (
	ErrNotYourTurn        MoveErrorCode = "not_your_turn"
	ErrInvalidMove        MoveErrorCode = "invalid_move"
	ErrInsufficientCards  MoveErrorCode = "insufficient_cards"
	ErrInvalidMeld        MoveErrorCode = "invalid_meld"
	ErrNotGoneDown        MoveErrorCode = "not_gone_down"
	ErrInsufficientPoints MoveErrorCode = "insufficient_points"
	ErrCannotPickupPile   MoveErrorCode = "cannot_pickup_pile"
	ErrGameNotStarted     MoveErrorCode = "game_not_started"
	ErrGameEnded          MoveErrorCode = "game_ended"
	ErrInvalidPhase       MoveErrorCode = "invalid_phase"
)

// GameState represents the filtered game state sent to a specific player
// (hides other players' hands)
type GameState struct {
	GameID           string       `json:"gameId"`
	CurrentPlayer    int          `json:"currentPlayer"`
	CurrentPhase     TurnPhase    `json:"currentPhase"`
	HandNumber       int          `json:"handNumber"`
	YourHand         []Card       `json:"yourHand"`
	YourFoot         []Card       `json:"yourFoot,omitempty"`
	YourStagingMelds []Meld       `json:"yourStagingMelds"`
	Team1Score       int          `json:"team1Score"`
	Team2Score       int          `json:"team2Score"`
	Team1Melds       []Meld       `json:"team1Melds"`
	Team2Melds       []Meld       `json:"team2Melds"`
	Team1Canastas    []Canasta    `json:"team1Canastas"`
	Team2Canastas    []Canasta    `json:"team2Canastas"`
	DiscardPile      []Card       `json:"discardPile"`
	DeckCount        int          `json:"deckCount"`
	OtherPlayers     []PlayerInfo `json:"otherPlayers"`
	ValidMoves       []MoveType   `json:"validMoves"` // Optional: list of valid moves for current player
}

// PlayerInfo represents minimal info about other players (no hand visibility)
type PlayerInfo struct {
	PlayerID      int    `json:"playerId"`
	Name          string `json:"name"`
	HandCount     int    `json:"handCount"`
	FootCount     int    `json:"footCount"`
	HasPickedFoot bool   `json:"hasPickedFoot"`
	Team          int    `json:"team"`
}
