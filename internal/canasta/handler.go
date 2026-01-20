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
)

type Move struct {
	PlayerId int
	Type     MoveType
	Id       int
	Ids      []int
}

type MoveResponse struct {
	Success bool   `json:"successs"`
	Message string `json:"message"`
}

func (g *Game) ExecuteMove(move Move) MoveResponse {
	// Validate it's the player's turn
	if g.CurrentPlayer != move.PlayerId {
		return MoveResponse{
			Success: false,
			Message: "Not your turn",
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
