package canasta

import (
	"errors"
	"math/rand"
	"slices"
)

type Game struct {
	Id                  string    `json:"id"`
	Players             []*Player `json:"players"`
	TeamA               *Team     `json:"teamA"`
	TeamB               *Team     `json:"teamB"`
	Hand                *Hand     `json:"hand"`
	HandNumber          int       `json:"handNumber"`
	CurrentPlayer       int       `json:"currentPlayer"`
	Phase               TurnPhase `json:"phase"`
	GoOutRequestPending bool      `json:"goOutRequestPending"` // True when permission request is active
	GoOutRequester      int       `json:"goOutRequester"`      // Player ID who asked permission (-1 if none)
	GoOutPartner        int       `json:"goOutPartner"`        // Partner ID who needs to respond (-1 if none)
}

type TurnPhase string

const (
	PhaseDrawing TurnPhase = "drawing"
	PhasePlaying TurnPhase = "playing"
)

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
	Score     int       `json:"score"`
	Melds     []Meld    `json:"melds"`
	Canastas  []Canasta `json:"canastas"`
	GoneDown  bool      `json:"goneDown"`
	CanGoOut  bool      `json:"canGoOut"`
	RedThrees []Card    `json:"RedThrees"`
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

type GameConfig struct {
	RandomTeamOrder bool
}

type GameOption func(*GameConfig)

func WithFixedTeamOrder() GameOption {
	return func(c *GameConfig) {
		c.RandomTeamOrder = false
	}
}

func NewGame(id string, playerNames []string, options ...GameOption) Game {
	config := &GameConfig{RandomTeamOrder: true}
	for _, option := range options {
		option(config)
	}

	teamA := Team{
		Score:     0,
		Melds:     make([]Meld, 0),
		Canastas:  make([]Canasta, 0),
		GoneDown:  false,
		CanGoOut:  false,
		RedThrees: make([]Card, 0),
	}
	teamB := Team{
		Score:     0,
		Melds:     make([]Meld, 0),
		Canastas:  make([]Canasta, 0),
		GoneDown:  false,
		CanGoOut:  false,
		RedThrees: make([]Card, 0),
	}

	if config.RandomTeamOrder {
		// Randomize playing order, preserving partner position
		a := []string{playerNames[0], playerNames[2]}
		b := []string{playerNames[1], playerNames[3]}
		rand.Shuffle(len(a), func(i, j int) {
			a[i], a[j] = a[j], a[i]
		})
		rand.Shuffle(len(b), func(i, j int) {
			b[i], b[j] = b[j], b[i]
		})

		firstTeam := rand.Int() % 2
		if firstTeam == 0 {
			playerNames[0] = a[0]
			playerNames[1] = b[0]
			playerNames[2] = a[1]
			playerNames[3] = b[1]
		} else {
			playerNames[0] = b[0]
			playerNames[1] = a[0]
			playerNames[2] = b[1]
			playerNames[3] = a[1]
		}
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

	partnerMap := map[int]int{
		0: 2,
		1: 3,
		2: 0,
		3: 1,
	}

	for i, player := range players {
		player.partner = players[partnerMap[i]]
	}

	hand := &Hand{
		Deck:        NewDeck(),
		DiscardPile: make([]Card, 0),
	}

	hand.Deck.Shuffle()

	return Game{
		Id:                  id,
		TeamA:               &teamA,
		TeamB:               &teamB,
		Players:             players,
		Hand:                hand,
		HandNumber:          1,
		GoOutRequestPending: false,
		GoOutRequester:      -1,
		GoOutPartner:        -1,
	}
}

func (g *Game) EndHand() {
	g.HandNumber++

	g.Score()

	if g.HandNumber >= 4 {
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
	g.TeamA.RedThrees = make([]Card, 0)

	g.TeamB.Melds = make([]Meld, 0)
	g.TeamB.Canastas = make([]Canasta, 0)
	g.TeamB.GoneDown = false
	g.TeamB.RedThrees = make([]Card, 0)

	// Reset go-out permission state
	g.GoOutRequestPending = false
	g.GoOutRequester = -1
	g.GoOutPartner = -1

	hand := &Hand{
		Deck:        NewDeck(),
		DiscardPile: make([]Card, 0),
	}

	hand.Deck.Shuffle()

	g.Deal()
}

func (g *Game) Score() {
	for _, team := range []*Team{g.TeamA, g.TeamB} {
		score := team.Score

		// Score melds and canastas
		for _, meld := range team.Melds {
			score += meld.Score()
		}
		for _, c := range team.Canastas {
			score += c.Score()
		}

		team.Score = score
	}

	for _, p := range g.Players {
		for _, card := range p.Hand {
			// Subtract card values from score (cards left in hand count against you)
			// Black threes have negative value, but we still want to subtract them
			if card.Rank == Three && card.Suit.isBlack() {
				p.Team.Score -= 100 // Black threes cost 100 points
			} else {
				p.Team.Score -= card.Value()
			}
		}
	}
}

func (g Game) EndGame() {

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

	// Initialize the turn
	g.CurrentPlayer = (-1 + g.HandNumber) % 4
	g.Phase = PhaseDrawing
}
