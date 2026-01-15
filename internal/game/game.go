package game

import (
	"errors"
	"sort"
)

type Game struct {
	Id    string `json:"id"`
	State State  `json:"state"`
}

type State struct {
	Deck        *Deck     `json:"deck"`
	DiscardPile []Card    `json:"discardPile"`
	Players     []*Player `json:"players"`
	TeamA       *Team     `json:"teamA"`
	TeamB       *Team     `json:"teamB"`
}

type Player struct {
	Name         string `json:"name"`
	Team         *Team  `json:"team"`
	Hand         []Card `json:"hand"`
	Foot         []Card `json:"foot"`
	StagingMelds []Meld `json:"stagingMelds"`
	MadeCanasta  bool   `json:"madeCanasta"`
}

type Meld struct {
	Rank      Rank   `json:"rank"`
	Cards     []Card `json:"cards"`
	Unnatural bool   `json:"unnatural"`
}

type Canasta struct {
	Rank  Rank   `json:"rank"`
	Cards []Card `json:"cards"`
	Count int    `json:"count"`
}

type Team struct {
	Melds    []Meld    `json:"melds"`
	Canastas []Canasta `json:"canastas"`
	GoneDown bool      `json:"goneDown"`
}

func NewGame(playerNames []string) Game {
	teamA := Team{
		make([]Meld, 0),
		make([]Canasta, 0),
		false,
	}
	teamB := Team{
		make([]Meld, 0),
		make([]Canasta, 0),
		false,
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

func (p *Player) NewMeld(cardIndexes []int) error {
	meld, err := p.ValidateMeld(cardIndexes)
	if err != nil {
		return err
	}

	// Cool let's do it then
	if p.Team.GoneDown {
		p.Team.Melds = append(p.Team.Melds, meld)

		// Push it over there Patrick
		p.Hand = removeCards(p.Hand, cardIndexes)

		return nil
	} else {
		// Add it to the player's "staging" melds.
		p.StagingMelds = append(p.StagingMelds, meld)
		return nil
	}
}

func (p *Player) ValidateMeld(cardIndexes []int) (meld Meld, err error) {
	if len(cardIndexes) < 3 {
		return meld, errors.New("Melds require at least three cards.")
	}

	// Get the cards themselves without affecting the player's hand yet.
	// We'll check that they aren't trying to pull a fast one first.
	var cards []Card
	allWilds := true
	var rank Rank

	for i, handIndex := range cardIndexes {
		if i > 0 && !cards[i-1].IsWild() && !p.Hand[handIndex].IsWild() && p.Hand[handIndex].Rank != rank {
			return meld, errors.New("Cannot mix rank in a meld")
		}

		// Keep track of Rank
		if !p.Hand[handIndex].IsWild() {
			rank = p.Hand[handIndex].Rank
			allWilds = false
		}

		// Can't use a three for a canasta
		if p.Hand[i].Rank == Three {
			return meld, errors.New("Cannot use threes in melds")
		}

		cards = append(cards, p.Hand[handIndex])
	}

	wildCount := WildCount(cards)
	// Can't mix wilds with sevens
	if rank == Seven && wildCount > 0 {
		return meld, errors.New("Cannot use wildcards for a sevens meld")
	}

	// Can't have majority wildcards
	if !allWilds && wildCount > 3 {
		return meld, errors.New("Cannot use more than three wildcards in an unnatural Canasta")
	}

	meld = Meld{
		Rank:      rank,
		Cards:     cards,
		Unnatural: wildCount > 0,
	}

	return meld, nil
}

func removeCards(hand []Card, indices []int) []Card {
	sort.Sort(sort.Reverse(sort.IntSlice(indices)))

	for _, i := range indices {
		if i >= 0 && i < len(hand) {
			hand = append(hand[:i], hand[i+1:]...)
		}
	}
	return hand
}

func (p *Player) GoDown() {

}

func (p *Player) NewCanasta() {

}

func (p *Player) BurnCard(canastaId int) {

}

func (p *Player) Discard(cardId int) {

}
