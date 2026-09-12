package room

import (
	"encoding/json"
	"errors"

	"canasta-server/internal/canasta"
	"canasta-server/internal/protocol"
)

// mutator decodes a command's payload and invokes the matching
// internal/canasta method against the game and acting player.
type mutator func(g *canasta.Game, p *canasta.Player, data json.RawMessage) error

var errInvalidPayload = errors.New("INVALID_PAYLOAD: could not parse command payload")

// phaseForType lists the canasta.TurnPhase each command requires. Commands
// absent from this map (currently none besides join and
// grant_permission_to_go_out, which are handled separately) run in either
// phase.
var phaseForType = map[protocol.MessageType]canasta.TurnPhase{
	protocol.TypeDrawFromDeck:      canasta.PhaseDrawing,
	protocol.TypePickUpDiscardPile: canasta.PhaseDrawing,
	protocol.TypePlayRedThree:      canasta.PhaseDrawing,
	protocol.TypeNewMeld:           canasta.PhasePlaying,
	protocol.TypeAddToMeld:         canasta.PhasePlaying,
	protocol.TypeBurnCards:         canasta.PhasePlaying,
	protocol.TypeGoDown:            canasta.PhasePlaying,
	protocol.TypeDiscard:           canasta.PhasePlaying,
	protocol.TypePickUpFoot:        canasta.PhasePlaying,
}

var mutators = map[protocol.MessageType]mutator{
	protocol.TypeDrawFromDeck: func(g *canasta.Game, p *canasta.Player, data json.RawMessage) error {
		if exhausted := g.DrawFromDeck(p); exhausted {
			g.EndHand()
		}
		return nil
	},
	protocol.TypePickUpDiscardPile: func(g *canasta.Game, p *canasta.Player, data json.RawMessage) error {
		var payload protocol.PickUpDiscardPilePayload
		if err := json.Unmarshal(data, &payload); err != nil {
			return errInvalidPayload
		}
		return g.PickUpDiscardPile(p, payload.CardIds)
	},
	protocol.TypeNewMeld: func(g *canasta.Game, p *canasta.Player, data json.RawMessage) error {
		var payload protocol.NewMeldPayload
		if err := json.Unmarshal(data, &payload); err != nil {
			return errInvalidPayload
		}
		return g.NewMeld(p, payload.CardIds)
	},
	protocol.TypeAddToMeld: func(g *canasta.Game, p *canasta.Player, data json.RawMessage) error {
		var payload protocol.AddToMeldPayload
		if err := json.Unmarshal(data, &payload); err != nil {
			return errInvalidPayload
		}
		return g.AddToMeld(p, payload.CardIds, payload.MeldId)
	},
	protocol.TypeBurnCards: func(g *canasta.Game, p *canasta.Player, data json.RawMessage) error {
		var payload protocol.BurnCardsPayload
		if err := json.Unmarshal(data, &payload); err != nil {
			return errInvalidPayload
		}
		return g.BurnCards(p, payload.CardIds, payload.CanastaId)
	},
	protocol.TypeGoDown: func(g *canasta.Game, p *canasta.Player, data json.RawMessage) error {
		return g.GoDown(p)
	},
	protocol.TypeDiscard: func(g *canasta.Game, p *canasta.Player, data json.RawMessage) error {
		var payload protocol.DiscardPayload
		if err := json.Unmarshal(data, &payload); err != nil {
			return errInvalidPayload
		}
		return g.Discard(p, payload.CardId)
	},
	protocol.TypePickUpFoot: func(g *canasta.Game, p *canasta.Player, data json.RawMessage) error {
		return g.PickUpFoot(p)
	},
	protocol.TypePlayRedThree: func(g *canasta.Game, p *canasta.Player, data json.RawMessage) error {
		var payload protocol.PlayRedThreePayload
		if err := json.Unmarshal(data, &payload); err != nil {
			return errInvalidPayload
		}
		return g.PlayRedThree(p, payload.CardIds, payload.FromFoot)
	},
}

// applyCommand enforces turn order and phase (and, for
// grant_permission_to_go_out, sender identity) before invoking the
// matching internal/canasta mutator. On success it returns nil and the
// mutation has already been applied to r.game; on rejection it returns a
// non-nil payload and r.game is untouched. Only ever called from the room
// actor goroutine.
func (r *Room) applyCommand(seatIdx int, msg protocol.ClientMessage) *protocol.ErrorPayload {
	if msg.Type == protocol.TypeGrantPermissionToGoOut {
		return r.applyGrantPermission(seatIdx)
	}

	fn, ok := mutators[msg.Type]
	if !ok {
		return &protocol.ErrorPayload{Code: string(protocol.ErrUnknownType), Message: "unknown command: " + string(msg.Type)}
	}

	if seatIdx != r.game.CurrentPlayer {
		return &protocol.ErrorPayload{Code: string(protocol.ErrNotYourTurn), Message: "it is not your turn"}
	}

	if requiredPhase, ok := phaseForType[msg.Type]; ok && r.game.Phase != requiredPhase {
		return &protocol.ErrorPayload{Code: string(protocol.ErrWrongPhase), Message: "wrong phase for this action"}
	}

	player := r.game.Players[seatIdx]
	if err := fn(r.game, player, msg.Data); err != nil {
		code, message := protocol.ClassifyGameError(err)
		return &protocol.ErrorPayload{Code: string(code), Message: message}
	}

	return nil
}

// applyGrantPermission requires the sender to be the current player's
// partner (via the fixed 0/2-1/3 seating invariant) rather than requiring
// it to be their own turn.
func (r *Room) applyGrantPermission(seatIdx int) *protocol.ErrorPayload {
	partnerOf := (r.game.CurrentPlayer + 2) % 4
	if seatIdx != partnerOf {
		return &protocol.ErrorPayload{Code: string(protocol.ErrNotPartner), Message: "only the current player's partner may grant permission to go out"}
	}

	if err := r.game.GrantPermissionToGoOut(r.game.Players[seatIdx]); err != nil {
		code, message := protocol.ClassifyGameError(err)
		return &protocol.ErrorPayload{Code: string(code), Message: message}
	}

	return nil
}
