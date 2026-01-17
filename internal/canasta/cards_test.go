package canasta_test

import (
	"canasta-server/internal/canasta"
	"fmt"
	"slices"
	"testing"
)

func TestPointValues(t *testing.T) {
	var tests = []struct {
		card canasta.Card
		want int
	}{
		{canasta.Card{0, canasta.Hearts, canasta.Three}, 100},
		{canasta.Card{0, canasta.Diamonds, canasta.Three}, 100},
		{canasta.Card{0, canasta.Clubs, canasta.Three}, -100},
		{canasta.Card{0, canasta.Spades, canasta.Three}, -100},
		{canasta.Card{0, canasta.Spades, canasta.Ace}, 20},
		{canasta.Card{0, canasta.Spades, canasta.Joker}, 50},
	}

	for _, tt := range tests {
		testName := fmt.Sprintf("%s", tt.card.String())
		t.Run(testName, func(t *testing.T) {
			value := tt.card.Value()
			if value != tt.want {
				t.Errorf("Card valued at %d, %d expected.", value, tt.want)
			}
		})
	}
}

func TestBuildDeck(t *testing.T) {
	deck := canasta.NewDeck()

	if deck.Count() != 4*54 {
		t.Errorf("Deck should be %d cards, %d given.", 54*4, deck.Count())
	}
}

func TestDraw(t *testing.T) {
	deck := canasta.NewDeck()
	drawnCards := deck.Draw(3)

	expected := []canasta.Card{
		{215, canasta.Wild, canasta.Joker},
		{214, canasta.Wild, canasta.Joker},
		{213, canasta.Spades, canasta.Ace},
	}

	if deck.Count() != 213 {
		t.Errorf("Deck should have %d cards, %d given", 213, deck.Count())
	}

	for i, expectedCard := range expected {
		if expectedCard.Id != drawnCards[i].Id || expectedCard.Rank != drawnCards[i].Rank || expectedCard.Suit != drawnCards[i].Suit {
			t.Log(drawnCards)
			t.Errorf("Expected to draw %d: %s, got %d: %s", expectedCard.Id, expectedCard, drawnCards[i].Id, drawnCards[i])
		}
	}
}

func TestShuffle(t *testing.T) {
	deckA := canasta.NewDeck()
	deckB := canasta.NewDeck()

	if !slices.Equal(deckA.Cards, deckB.Cards) {
		t.Error("Your decks aren't equal to start")
	}

	deckB.Shuffle()

	if slices.Equal(deckA.Cards, deckB.Cards) {
		t.Error("Shuffling didn't work")
	}
}
