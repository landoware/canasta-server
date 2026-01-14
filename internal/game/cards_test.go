package game_test

import (
	"canasta-server/internal/game"
	"fmt"
	"slices"
	"testing"
)

func TestPointValues(t *testing.T) {
	var tests = []struct {
		card game.Card
		want int
	}{
		{game.Card{game.Hearts, game.Three}, 100},
		{game.Card{game.Diamonds, game.Three}, 100},
		{game.Card{game.Clubs, game.Three}, -100},
		{game.Card{game.Spades, game.Three}, -100},
		{game.Card{game.Spades, game.Ace}, 20},
		{game.Card{game.Spades, game.Joker}, 50},
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
	deck := game.NewDeck()

	if deck.Count() != 4*54 {
		t.Errorf("Deck should be %d cards, %d given.", 54*4, deck.Count())
	}
}

func TestDraw(t *testing.T) {
	deck := game.NewDeck()
	drawnCards := deck.Draw(3)

	expected := []game.Card{
		{game.Clubs, game.Joker},
		{game.Spades, game.Joker},
		{game.Spades, game.Ace},
	}

	if deck.Count() != 213 {
		t.Errorf("Deck should have %d cards, %d given", 213, deck.Count())
	}

	for i, expectedCard := range expected {
		if expectedCard.Rank != drawnCards[i].Rank || expectedCard.Suit != drawnCards[i].Suit {
			t.Errorf("Expected to draw %s, got %s", expectedCard, drawnCards[i])
		}
	}
}

func TestShuffle(t *testing.T) {
	deckA := game.NewDeck()
	deckB := game.NewDeck()

	if !slices.Equal(deckA.Cards, deckB.Cards) {
		t.Error("Your decks aren't equal to start")
	}

	deckB.Shuffle()

	if slices.Equal(deckA.Cards, deckB.Cards) {
		t.Error("Shuffling didn't work")
	}
}
