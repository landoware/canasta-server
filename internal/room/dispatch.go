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
}

// turnAndPhaseExempt lists commands that skip the generic turn-order and
// phase checks in applyCommand below because they have their own
// tailored logic instead — see PickUpFoot in moves.go, which allows
// picking up your foot on any other player's turn, but only before
// you've drawn (or after you've discarded) on your own.
var turnAndPhaseExempt = map[protocol.MessageType]bool{
	protocol.TypePickUpFoot: true,
}

// meldAllowedInDrawPhase lists commands that may also run during
// PhaseDrawing (in addition to their normal PhasePlaying requirement in
// phaseForType), but only for a player whose team hasn't gone down yet —
// see NewMeld/AddToMeld in moves.go, which route to Player.StagingMelds
// pre-go-down. This lets a player stage a meld before picking up the
// discard pile, without loosening the phase rule for a player extending
// their team's real, already-gone-down melds.
var meldAllowedInDrawPhase = map[protocol.MessageType]bool{
	protocol.TypeNewMeld:   true,
	protocol.TypeAddToMeld: true,
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
	pos := r.posForConnSlot(seatIdx)

	if msg.Type == protocol.TypeGrantPermissionToGoOut {
		return r.applyGrantPermission(pos)
	}
	if msg.Type == protocol.TypeAskToGoOut {
		return r.applyAskToGoOut(pos)
	}

	fn, ok := mutators[msg.Type]
	if !ok {
		return &protocol.ErrorPayload{Code: string(protocol.ErrUnknownType), Message: "unknown command: " + string(msg.Type)}
	}

	if !turnAndPhaseExempt[msg.Type] {
		if pos != r.game.CurrentPlayer {
			return &protocol.ErrorPayload{Code: string(protocol.ErrNotYourTurn), Message: "it is not your turn"}
		}

		if requiredPhase, ok := phaseForType[msg.Type]; ok && r.game.Phase != requiredPhase {
			stagingExempt := meldAllowedInDrawPhase[msg.Type] &&
				r.game.Phase == canasta.PhaseDrawing &&
				!r.game.Players[pos].Team.GoneDown
			if !stagingExempt {
				return &protocol.ErrorPayload{Code: string(protocol.ErrWrongPhase), Message: "wrong phase for this action"}
			}
		}
	}

	player := r.game.Players[pos]
	if err := fn(r.game, player, msg.Data); err != nil {
		code, message := protocol.ClassifyGameError(err)
		return &protocol.ErrorPayload{Code: string(code), Message: message}
	}

	return nil
}

// applyGrantPermission requires the sender to be the current player's
// partner (via the fixed 0/2-1/3 table-position seating invariant) rather
// than requiring it to be their own turn. pos is the sender's table
// position (see posForConnSlot), not their raw connection slot.
func (r *Room) applyGrantPermission(pos int) *protocol.ErrorPayload {
	partnerOf := (r.game.CurrentPlayer + 2) % 4
	if pos != partnerOf {
		return &protocol.ErrorPayload{Code: string(protocol.ErrNotPartner), Message: "only the current player's partner may grant permission to go out"}
	}

	if err := r.game.GrantPermissionToGoOut(r.game.Players[pos]); err != nil {
		code, message := protocol.ClassifyGameError(err)
		return &protocol.ErrorPayload{Code: string(code), Message: message}
	}

	return nil
}

// applyAskToGoOut lets any player on an eligible, not-yet-granted team
// notify their partner that they'd like to go out — see
// GrantPermissionToGoOut in moves.go for the authoritative check this
// mirrors (the actual grant re-validates independently; this is just
// the notification trigger, not a state mutation). Unlike every other
// command, this isn't turn-scoped at all: Team.CanGoOut never resets
// once granted, so there's no reason to require it be the asker's turn.
// pos is the sender's table position (see posForConnSlot).
func (r *Room) applyAskToGoOut(pos int) *protocol.ErrorPayload {
	team := r.game.Players[pos].Team
	if !team.GoneDown {
		return &protocol.ErrorPayload{Code: "CANNOT_GO_OUT", Message: "team must go down before asking to go out"}
	}
	if !team.MeetsGoOutRequirements() {
		return &protocol.ErrorPayload{Code: "CANASTA_REQUIREMENTS_NOT_MET", Message: "team needs a natural, unnatural, sevens, and wildcards canasta before going out"}
	}
	if team.CanGoOut {
		return &protocol.ErrorPayload{Code: "ALREADY_GRANTED", Message: "permission to go out has already been granted"}
	}

	partnerPos := (pos + 2) % 4
	partnerConnSlot := r.tableOrder[partnerPos]
	askerName := r.game.Players[pos].Name
	r.sendTo(partnerConnSlot, protocol.NewServerMessage(protocol.TypeGoOutRequested, protocol.GoOutRequestedPayload{AskerName: askerName}))
	return nil
}
