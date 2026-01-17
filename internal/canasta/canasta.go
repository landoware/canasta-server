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

type Canasta struct {
	Id    int    `json:"id"`
	Rank  Rank   `json:"rank"`
	Cards []Card `json:"cards"`
	Count int    `json:"count"`
}

func (c Canasta) GetId() int { return c.Id }

type Team struct {
	Score    int       `json:"score"`
	Melds    []Meld    `json:"melds"`
	Canastas []Canasta `json:"canastas"`
	GoneDown bool      `json:"goneDown"`
	CanGoOut bool      `json:"canGoOut"`
}

var meldRequirements = []int{
	50,
	90,
	120,
	150,
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

func (p *Player) NewMeld(cardIds []int) error {
	meld, err := p.ValidateMeld(cardIds)
	if err != nil {
		return err
	}

	// Cool let's do it then
	if p.Team.GoneDown {
		p.Hand.removeCards(cardIds)
		p.Team.Melds = append(p.Team.Melds, meld)

		if len(meld.Cards) >= 7 {
			p.NewCanasta(len(p.Team.Melds) - 1)
		}

		return nil
	} else {
		// Add it to the player's "staging" melds.
		p.StagingMelds = append(p.StagingMelds, meld)
		return nil
	}
}

func (p *Player) ValidateMeld(cardIds []int) (meld Meld, err error) {
	if len(cardIds) < 3 {
		return meld, errors.New("Melds require at least three cards.")
	}

	// Get the cards themselves without affecting the player's hand yet.
	// We'll check that they aren't trying to pull a fast one first.
	var cards []Card
	allWilds := true
	var rank Rank

	for _, handIndex := range cardIds {
		card := p.Hand[handIndex]

		// Can't use a three for a canasta
		if card.Rank == Three {
			return meld, errors.New("Cannot use threes in melds")
		}

		// Set the rank based on the first non-wild
		if !card.IsWild() {
			if allWilds {
				rank = card.Rank
				allWilds = false
			} else {
				if card.Rank != rank {
					return meld, errors.New("Cannot mix rank in a meld")
				}
			}
		}

		cards = append(cards, card)
	}

	wildCount := WildCount(cards)
	// Can't mix wilds with sevens
	if rank == Seven && wildCount > 0 {
		return meld, errors.New("Cannot use wildcards for a sevens meld")
	}

	// Can't have majority wildcards
	if !allWilds && wildCount > 3 {
		return meld, errors.New("Cannot use more than three wildcards in an unnatural meld")
	}

	if allWilds {
		rank = Wild
	}
	meld = Meld{
		Id:        cardIds[0],
		Rank:      rank,
		Cards:     cards,
		WildCount: wildCount,
	}

	return meld, nil
}

func (p *Player) AddToMeld(cardIds []int, meldId int) error {
	var cards []Card

	meldIndex, err := findIndex(meldId, p.Team.Melds)
	if err != nil {
		return err
	}

	meld := &p.Team.Melds[meldIndex]

	for _, handIndex := range cardIds {
		card := p.Hand[handIndex]
		if card.Rank != meld.Rank && !card.IsWild() {
			return errors.New("Card does not match this meld")
		}
		if card.Rank == Three {
			return errors.New("Cannot use threes in melds")
		}
		if meld.Rank == Seven && card.IsWild() {
			return errors.New("Cannot use wildcards in a Sevens meld")
		}

		if card.IsWild() {
			meld.WildCount++
			if meld.WildCount > 3 {
				return errors.New("Cannot add more wildcards to this Meld")
			}
		}
		cards = append(cards, p.Hand[handIndex])
	}

	meld.Cards = append(meld.Cards, cards...)
	p.Hand.removeCards(cardIds)

	if len(meld.Cards) >= 7 {
		p.NewCanasta(meldIndex)
	}

	return nil
}

func (h *PlayerHand) removeCards(ids []int) {
	for _, id := range ids {
		delete(*h, id)
	}
}

func (g *Game) GoDown(p *Player) {
	// Handle a player having 7+ cards in a staging meld

}

func (p *Player) NewCanasta(meldIndex int) {

	meld := p.Team.Melds[meldIndex]

	p.Team.Canastas = append(p.Team.Canastas, Canasta{
		Rank:  meld.Rank,
		Cards: meld.Cards,
		Count: len(meld.Cards),
	})

	remainingMelds := slices.Delete(p.Team.Melds, meldIndex, meldIndex+1)

	p.Team.Melds = remainingMelds
}

func (p *Player) BurnCard(canastaId int) {

}

func (g *Game) Discard(p *Player, cardId int) error {
	// Are they allowed to go out?
	// If not they need at least two cards in their hand PRIOR to discarding.
	if !p.Team.CanGoOut {
		if len(p.Hand) < 2 {
			return errors.New("Can't go out yet!")
		}
	}

	card := p.Hand[cardId]
	p.Hand.removeCards([]int{cardId})
	g.Hand.DiscardPile = append(g.Hand.DiscardPile, card)

	if p.Team.CanGoOut && len(p.Hand) == 0 {
		g.EndHand()
	}

	return nil
}

func (p Player) CanPickUpDiscardPile(topCard Card) bool {
	return true
}

func (g *Game) PickUpDiscardPile(p *Player, cardIds []int) {

}
