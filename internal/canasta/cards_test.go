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

func TestDrawDeckMethod(t *testing.T) {
	deck := canasta.NewDeck()
	drawnCards, exhausted := deck.Draw(3)

	expected := []canasta.Card{
		{215, canasta.Wild, canasta.Joker},
		{214, canasta.Wild, canasta.Joker},
		{213, canasta.Spades, canasta.Ace},
	}

	if exhausted {
		t.Errorf("Deck should not be exhausted after drawing 3 of 216 cards")
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

func TestDrawDeckExhaustion(t *testing.T) {
	deck := canasta.NewDeck()

	cards, exhausted := deck.Draw(300)

	if len(cards) != 216 {
		t.Errorf("Expected to draw all %d remaining cards, got %d", 216, len(cards))
	}

	if !exhausted {
		t.Error("Deck should report exhausted after draining it completely")
	}

	if deck.Count() != 0 {
		t.Errorf("Deck should have 0 cards left, %d given", deck.Count())
	}

	moreCards, stillExhausted := deck.Draw(1)
	if len(moreCards) != 0 {
		t.Errorf("Drawing from an empty deck should return no cards, got %d", len(moreCards))
	}
	if !stillExhausted {
		t.Error("Drawing from an already-empty deck should still report exhausted")
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
