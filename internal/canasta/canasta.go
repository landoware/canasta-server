package canasta

import (
	"errors"
	"slices"
)

type Game struct {
	Id         string    `json:"id"`
	Players    []*Player `json:"players"`
	TeamA      *Team     `json:"teamA"`
	TeamB      *Team     `json:"teamB"`
	Hand       *Hand     `json:"hand"`
	HandNumber int       `json:"handNumber"`
}

type Hand struct {
	Deck        *Deck  `json:"deck"`
	DiscardPile []Card `json:"discardPile"`
}

type Player struct {
	Name         string     `json:"name"`
	Team         *Team      `json:"team"`
	Hand         PlayerHand `json:"hand"`
	Foot         []Card     `json:"foot"`
	StagingMelds []Meld     `json:"stagingMelds"`
	MadeCanasta  bool       `json:"madeCanasta"`
	partner      *Player
}

type PlayerHand map[int]Card

type HasId interface {
	GetId() int
}

type Meld struct {
	Id        int    `json:"id"`
	Rank      Rank   `json:"rank"`
	Cards     []Card `json:"cards"`
	WildCount int    `json:"wildCount"`
}

func (m Meld) GetId() int { return m.Id }

func (m Meld) Score() (score int) {
	for _, card := range m.Cards {
		score += card.Value()
	}
	return
}

type Canasta struct {
	Id      int    `json:"id"`
	Rank    Rank   `json:"rank"`
	Cards   []Card `json:"cards"`
	Count   int    `json:"count"`
	Natural bool   `json:"natural"`
}

func (c Canasta) GetId() int { return c.Id }

func (c Canasta) Score() (score int) {
	for _, card := range c.Cards {
		score += card.Value()
	}
	if c.Rank == Wild {
		return score + 2500
	}
	if c.Rank == Seven {
		return score + 1500
	}
	if slices.ContainsFunc(c.Cards, func(card Card) bool { return card.IsWild() }) {
		return score + 300
	} else {
		return score + 500
	}
}

type Team struct {
	Score    int       `json:"score"`
	Melds    []Meld    `json:"melds"`
	Canastas []Canasta `json:"canastas"`
	GoneDown bool      `json:"goneDown"`
	CanGoOut bool      `json:"canGoOut"`
}

var meldRequirements = map[int]int{
	1: 50,
	2: 90,
	3: 120,
	4: 150,
}

func findIndex[T HasId](id int, slice []T) (index int, err error) {
	for i, item := range slice {
		if item.GetId() == id {
			return i, nil
		}
	}
	return -1, errors.New("Not found")
}

func NewGame(playerNames []string) Game {
	teamA := Team{
		Score:    0,
		Melds:    make([]Meld, 0),
		Canastas: make([]Canasta, 0),
		GoneDown: false,
		CanGoOut: false,
	}
	teamB := Team{
		Score:    0,
		Melds:    make([]Meld, 0),
		Canastas: make([]Canasta, 0),
		GoneDown: false,
		CanGoOut: false,
	}

	players := make([]*Player, 0)
	for i, playerName := range playerNames {
		if i%2 == 0 {
			players = append(players, &Player{
				Name: playerName,
				Team: &teamA,
				Hand: make(map[int]Card, 0),
				Foot: make([]Card, 0),
			})
		} else {
			players = append(players, &Player{
				Name: playerName,
				Team: &teamB,
				Hand: make(map[int]Card, 0),
				Foot: make([]Card, 0),
			})
		}
	}

	teamMap := map[int]int{
		0: 2,
		1: 3,
		2: 0,
		3: 1,
	}
	for i, player := range players {
		player.partner = players[teamMap[i]]
	}

	hand := &Hand{
		Deck:        NewDeck(),
		DiscardPile: make([]Card, 0),
	}

	hand.Deck.Shuffle()

	return Game{
		Id:         getID(),
		TeamA:      &teamA,
		TeamB:      &teamB,
		Players:    players,
		Hand:       hand,
		HandNumber: 1,
	}
}

func (g *Game) EndHand() {
	g.HandNumber++
	// Score here?
	g.Score()

	if g.HandNumber == 4 {
		g.EndGame()
	}

	g.NewHand()

}

func (g *Game) NewHand() {
	// Reset the player states
	for _, player := range g.Players {
		clear(player.Hand)
		player.Foot = make([]Card, 0)
		player.StagingMelds = make([]Meld, 0)
		player.MadeCanasta = false
	}
	// Clear out team melds and canastas
	g.TeamA.Melds = make([]Meld, 0)
	g.TeamA.Canastas = make([]Canasta, 0)
	g.TeamA.GoneDown = false
	g.TeamB.Melds = make([]Meld, 0)
	g.TeamB.Canastas = make([]Canasta, 0)
	g.TeamB.GoneDown = false

	hand := &Hand{
		Deck:        NewDeck(),
		DiscardPile: make([]Card, 0),
	}

	hand.Deck.Shuffle()

	g.Deal()
}

func (g Game) Score() {

}

func (g Game) EndGame() {

}

func getID() string {
	return "ABCD"
}

func (g *Game) Deal() {
	// Deal the Hand
	for range 15 {
		for _, player := range g.Players {
			card := g.Hand.Deck.Draw(1)[0]
			player.Hand[card.GetId()] = card
		}
	}

	// Deal the Feet
	for range 11 {
		for _, player := range g.Players {
			card := g.Hand.Deck.Draw(1)[0]
			player.Foot = append(player.Foot, card)
		}
	}
	// Discard the top card
	discard := g.Hand.Deck.Draw(1)[0]
	g.Hand.DiscardPile = append(g.Hand.DiscardPile, discard)
}
