package canasta

type MoveType string

const (
	// Draw phase
	MoveDrawFromDeck      MoveType = "draw_from_deck"
	MovePickupDiscardPile MoveType = "pickup_discard_pile"

	// Play phase
	MoveCreateMeld MoveType = "create_meld"
	MoveAddToMeld  MoveType = "add_to_meld"
	MoveBurnCards  MoveType = "burn_card"
	MoveGoDown     MoveType = "go_down"

	// End phase moves
	MoveDiscard MoveType = "discard"

	// Special moves
	MoveAskToGoOut   MoveType = "ask_to_go_out"
	MoveRespondGoOut MoveType = "respond_go_out"
	MovePickupFoot   MoveType = "pickup_foot"
	MovePlayRedThree MoveType = "play_red_three"
)

type Move struct {
	PlayerId int
	Type     MoveType
	Id       int
	Ids      []int
	FromFoot bool // True if red threes came from foot (no replacement draw)
}

type MoveResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func (g *Game) ExecuteMove(move Move) MoveResponse {
	// Special moves that don't require it to be player's turn
	// Why: Partner responses happen on partner's turn, not requester's
	if move.Type == MoveRespondGoOut {
		return g.handleRespondGoOut(move.PlayerId, move.Id)
	}

	// Validate it's the player's turn for all other moves
	if g.CurrentPlayer != move.PlayerId {
		return MoveResponse{
			Success: false,
			Message: "NOT_YOUR_TURN: Wait for your turn",
		}
	}

	// Execute move based on type
	switch move.Type {
	case MoveDrawFromDeck:
		return g.handleDrawFromDeck(move.PlayerId)
	case MovePickupDiscardPile:
		return g.handlePickupDiscardPile(move.PlayerId, move.Ids)
	case MoveCreateMeld:
		return g.handleCreateMeld(move.PlayerId, move.Ids)
	case MoveAddToMeld:
		return g.handleAddToMeld(move.PlayerId, move.Ids, move.Id)
	case MoveBurnCards:
		return g.handleBurnCards(move.PlayerId, move.Ids, move.Id)
	case MoveGoDown:
		return g.handleGoDown(move.PlayerId)
	case MoveDiscard:
		return g.handleDiscard(move.PlayerId, move.Id)
	case MovePickupFoot:
		return g.handlePickupFoot(move.PlayerId)
	case MoveAskToGoOut:
		return g.handleAskToGoOut(move.PlayerId)
	case MoveRespondGoOut:
		return g.handleRespondGoOut(move.PlayerId, move.Id)
	case MovePlayRedThree:
		return g.handlePlayRedThree(move.PlayerId, move.Ids, move.FromFoot)
	default:
		return MoveResponse{
			Success: false,
			Message: "Unknown move type",
		}
	}
}

func (g *Game) handleDrawFromDeck(playerID int) MoveResponse {
	g.DrawFromDeck(g.Players[playerID])
	return MoveResponse{Success: true}
}

func (g *Game) handlePickupDiscardPile(playerID int, ids []int) MoveResponse {
	err := g.PickUpDiscardPile(g.Players[playerID], ids)
	if err != nil {
		return MoveResponse{Success: false, Message: err.Error()}
	}
	return MoveResponse{Success: true}
}

func (g *Game) handleCreateMeld(playerID int, ids []int) MoveResponse {
	err := g.NewMeld(g.Players[playerID], ids)
	if err != nil {
		return MoveResponse{Success: false, Message: err.Error()}
	}
	return MoveResponse{Success: true}
}

func (g *Game) handleAddToMeld(playerID int, ids []int, id int) MoveResponse {
	err := g.AddToMeld(g.Players[playerID], ids, id)
	if err != nil {
		return MoveResponse{Success: false, Message: err.Error()}
	}
	return MoveResponse{Success: true}
}

func (g *Game) handleBurnCards(playerID int, ids []int, id int) MoveResponse {
	err := g.BurnCards(g.Players[playerID], ids, id)
	if err != nil {
		return MoveResponse{Success: false, Message: err.Error()}
	}
	return MoveResponse{Success: true}
}

func (g *Game) handleGoDown(playerID int) MoveResponse {
	err := g.GoDown(g.Players[playerID])
	if err != nil {
		return MoveResponse{Success: false, Message: err.Error()}
	}
	return MoveResponse{Success: true}
}

func (g *Game) handleDiscard(playerID int, id int) MoveResponse {
	err := g.Discard(g.Players[playerID], id)
	if err != nil {
		return MoveResponse{Success: false, Message: err.Error()}
	}
	return MoveResponse{Success: true}
}

func (g *Game) handlePickupFoot(playerID int) MoveResponse {
	err := g.PickUpFoot(g.Players[playerID])
	if err != nil {
		return MoveResponse{Success: false, Message: err.Error()}
	}
	return MoveResponse{Success: true}
}

func (g *Game) handleAskToGoOut(playerID int) MoveResponse {
	err := g.MoveAskToGoOut(g.Players[playerID])
	if err != nil {
		return MoveResponse{Success: false, Message: err.Error()}
	}
	// Success - permission request sent to partner
	// Server will broadcast this to the partner
	return MoveResponse{Success: true, Message: "Permission requested from partner"}
}

func (g *Game) handleRespondGoOut(playerID int, approved int) MoveResponse {
	// Convert int to bool: 1 = approved, 0 = denied
	// Why use int: Move struct uses Id field which is an int
	approvedBool := approved == 1

	err := g.RespondToGoOut(g.Players[playerID], approvedBool)
	if err != nil {
		return MoveResponse{Success: false, Message: err.Error()}
	}

	// Success - response recorded
	// Server will broadcast this to the requester
	if approvedBool {
		return MoveResponse{Success: true, Message: "Permission granted"}
	} else {
		return MoveResponse{Success: true, Message: "Permission denied"}
	}
}

func (g *Game) handlePlayRedThree(playerID int, cardIds []int, fromFoot bool) MoveResponse {
	err := g.PlayRedThree(g.Players[playerID], cardIds, fromFoot)
	if err != nil {
		return MoveResponse{Success: false, Message: err.Error()}
	}
	return MoveResponse{Success: true}
}
