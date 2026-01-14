package game

type Game struct {
	Id    string
	State State
}

type State struct {
	Deck        *Deck     `json:"deck"`
	DiscardPile []Card    `json:"discardPile"`
	Players     []*Player `json:"players"`
	TeamA       *Team     `json:"teamA"`
	TeamB       *Team     `json:"teamB"`
}

type Player struct {
	Name string `json:"name"`
	Team *Team  `json:"team"`
	Hand []Card `json:"hand"`
	Foot []Card `json:"foot"`
}

type Meld struct {
	Rank  Rank
	Cards []Card
}

type Canasta struct {
	Rank  Rank
	Cards []Card
	Count int
}

type Team struct {
	Melds    []Meld    `json:"melds"`
	Canastas []Canasta `json:"canastas"`
}

func NewGame(playerNames []string) Game {
	teamA := Team{
		make([]Meld, 0),
		make([]Canasta, 0),
	}
	teamB := Team{
		make([]Meld, 0),
		make([]Canasta, 0),
	}

	players := make([]*Player, 0)
	for i, playerName := range playerNames {
		if i%2 == 0 {
			players = append(players, &Player{
				Name: playerName,
				Team: &teamA,
				Hand: make([]Card, 0),
				Foot: make([]Card, 0),
			})
		} else {
			players = append(players, &Player{
				Name: playerName,
				Team: &teamB,
				Hand: make([]Card, 0),
				Foot: make([]Card, 0),
			})
		}
	}

	state := State{
		Deck:    NewDeck(),
		TeamA:   &teamA,
		TeamB:   &teamB,
		Players: players,
	}

	state.Deck.Shuffle()

	return Game{
		Id:    getID(),
		State: state,
	}
}

func getID() string {
	return "ABCD"
}

func (g *Game) Deal() {
	// Deal the Hand
	for range 15 {
		for _, player := range g.State.Players {
			card := g.State.Deck.Draw(1)[0]
			player.Hand = append(player.Hand, card)
		}
	}

	// Deal the Feet
	for range 11 {
		for _, player := range g.State.Players {
			card := g.State.Deck.Draw(1)[0]
			player.Foot = append(player.Foot, card)
		}
	}
	// Discard the top card
	discard := g.State.Deck.Draw(1)[0]
	g.State.DiscardPile = append(g.State.DiscardPile, discard)

}
