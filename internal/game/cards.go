package game

import (
	"fmt"
	"math/rand"
)

type Suit int

const (
	Hearts Suit = iota
	Diamonds
	Clubs
	Spades
)

var suitString = map[Suit]string{
	Hearts:   "Hearts",
	Diamonds: "Diamonds",
	Clubs:    "Clubs",
	Spades:   "Spades",
}

func (suit Suit) isBlack() bool {
	return suit == Clubs || suit == Spades
}

type Rank int

const (
	Four = iota
	Five
	Six
	Seven
	Eight
	Nine
	Ten
	Jack
	Queen
	King
	Ace
	Two
	Joker
	Three
)

var rankString = map[Rank]string{
	Four:  "Four",
	Five:  "Five",
	Six:   "Six",
	Seven: "Seven",
	Eight: "Eight",
	Nine:  "Nine",
	Ten:   "Ten",
	Jack:  "Jack",
	Queen: "Queen",
	King:  "King",
	Ace:   "Ace",
	Two:   "Two",
	Joker: "Joker",
	Three: "Three",
}

var pointValues = map[Rank]int{
	Four:  5,
	Five:  5,
	Six:   5,
	Seven: 5,
	Eight: 10,
	Nine:  10,
	Ten:   10,
	Jack:  10,
	Queen: 10,
	King:  10,
	Ace:   20,
	Two:   20,
	Joker: 50,
	Three: 100,
}

type Card struct {
	Suit Suit
	Rank Rank
}

func (card Card) Value() int {
	if card.Rank == Three && card.Suit.isBlack() {
		return pointValues[card.Rank] * -1
	} else {
		return pointValues[card.Rank]
	}
}

func (card Card) String() string {
	if card.Rank == Joker {
		return "Joker"
	}
	return fmt.Sprintf("%s of %s", rankString[card.Rank], suitString[card.Suit])
}

func NewDeck() Deck {
	deck := make([]Card, 0)
	ranks := []Rank{Two, Three, Four, Five, Six, Seven, Eight, Nine, Ten, Jack, Queen, King, Ace}
	suits := []Suit{Hearts, Diamonds, Clubs, Spades}

	for range 4 {
		for _, suit := range suits {
			for _, rank := range ranks {
				deck = append(deck, Card{suit, rank})
			}
		}
		deck = append(deck, Card{Spades, Joker})
		deck = append(deck, Card{Clubs, Joker})
	}

	return Deck{deck}
}

func (deck Deck) Count() int {
	return len(deck.Cards)
}

func (deck *Deck) Draw(i int) (Cards []Card) {
	for range i {
		card := deck.Cards[len(deck.Cards)-1]
		Cards = append(Cards, card)
		deck.Cards = deck.Cards[:len(deck.Cards)-1]
	}
	return
}

func (d *Deck) Shuffle() {
	rand.Shuffle(d.Count(), func(i, j int) {
		d.Cards[i], d.Cards[j] = d.Cards[j], d.Cards[i]
	})
}
