package game

type Game struct {
	id    string
	state State
}

type State struct {
	deck        []Deck
	discardPile []Card
}

type Player struct {
	Name string
	Team Team
	Hand []Card
	Foot []Card
}

type Deck struct {
	Cards []Card
}

type meld struct {
	Rank Rank
}

type Canasta struct {
	Rank  Rank
	Count int
}

type Team struct {
	Players []Player
}

func NewGame(players []string) {

}
