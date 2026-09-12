package protocol

import "canasta-server/internal/canasta"

// StateMessage layers session/turn information on top of the existing
// per-player redacted view from internal/canasta, which has no notion of
// seats, turns, or connections. Embedding (rather than extending
// canasta.ClientState itself) keeps internal/canasta free of transport
// concerns while ClientState stays reusable on its own.
type StateMessage struct {
	canasta.ClientState `tstype:",extends"`
	SeatIndex           int               `json:"seatIndex"`
	CurrentPlayer       int               `json:"currentPlayer"`
	IsYourTurn          bool              `json:"isYourTurn"`
	Phase               canasta.TurnPhase `json:"phase"`
	HandNumber          int               `json:"handNumber"`
	GameOver            bool              `json:"gameOver"`
	Winner              string            `json:"winner,omitempty"`
	CanGoOut            bool              `json:"canGoOut"`
}

// NewStateMessage builds the full broadcast payload for one seat.
func NewStateMessage(g *canasta.Game, seatIndex int) StateMessage {
	clientState := g.GetClientState(seatIndex)

	return StateMessage{
		ClientState:   *clientState,
		SeatIndex:     seatIndex,
		CurrentPlayer: g.CurrentPlayer,
		IsYourTurn:    g.CurrentPlayer == seatIndex,
		Phase:         g.Phase,
		HandNumber:    g.HandNumber,
		GameOver:      g.GameOver,
		Winner:        g.Winner,
		CanGoOut:      g.Players[seatIndex].Team.CanGoOut,
	}
}
