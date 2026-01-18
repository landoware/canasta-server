package canasta

type MoveType string

const (
	// Draw phase
	MoveDrawFromDeck      MoveType = "draw_from_deck"
	MovePickupDiscardPile MoveType = "pickup_discard_pile"

	// Play phase
	MoveCreateMeld MoveType = "create_meld"
	MoveAddToMeld  MoveType = "add_to_meld"
	MoveBurnCard   MoveType = "burn_card"
	MoveGoDown     MoveType = "go_down"

	// End phase moves
	MoveDiscard MoveType = "discard"

	// Special moves
	MoveAskToGoOut   MoveType = "ask_to_go_out"
	MoveRespondGoOut MoveType = "respond_go_out"
	MovePickupFoot   MoveType = "pickup_foot"
)
