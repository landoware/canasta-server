package game_test

import (
	"canasta-server/internal/game"
	"slices"
	"testing"
)

func TestDeal(t *testing.T) {
	names := []string{"One", "Two", "Three", "Four"}
	gameObj := game.NewGame(names)

	gameObj.Deal()

	for _, player := range gameObj.Players {
		if len(player.Hand) != 15 {
			t.Errorf("Player %s has %d cards in their hand, 15 expected", player.Name, len(player.Hand))
		}
		if len(player.Foot) != 11 {
			t.Errorf("Player %s has %d cards in their foot, 11 expected", player.Name, len(player.Foot))
		}
	}

	if len(gameObj.Hand.DiscardPile) != 1 {
		t.Errorf("Only one card should be discarded. %d given.", len(gameObj.State.DiscardPile))
	}

	if gameObj.Hand.Deck.Count() != 111 {
		t.Errorf("Too many cards delt. Have %d in deck, 111 expected.", gameObj.State.Deck.Count())
	}

}

func getMeldTestScenarios() []struct {
	name  string
	hand  []game.Card
	valid bool
} {
	return []struct {
		name  string
		hand  []game.Card
		valid bool
	}{
		{
			name:  "natural",
			hand:  []game.Card{game.Card{game.Clubs, game.Five}, game.Card{game.Clubs, game.Five}, game.Card{game.Clubs, game.Five}},
			valid: true,
		},
		{
			hand:  []game.Card{game.Card{game.Clubs, game.Five}, game.Card{game.Clubs, game.Six}, game.Card{game.Clubs, game.Seven}},
			valid: false,
			name:  "mixed rank",
		},
		{
			name:  "mixed rank with wildcard",
			hand:  []game.Card{game.Card{game.Clubs, game.Joker}, game.Card{game.Clubs, game.Six}, game.Card{game.Clubs, game.Seven}},
			valid: false,
		},
		{
			name:  "unnatural",
			hand:  []game.Card{game.Card{game.Clubs, game.Two}, game.Card{game.Clubs, game.Joker}, game.Card{game.Clubs, game.Four}},
			valid: true,
		},
		{
			name:  "unnatural",
			hand:  []game.Card{game.Card{game.Clubs, game.Two}, game.Card{game.Clubs, game.Four}, game.Card{game.Clubs, game.Four}},
			valid: true,
		},
		{
			name:  "unnatural with sevens",
			hand:  []game.Card{game.Card{game.Clubs, game.Two}, game.Card{game.Clubs, game.Joker}, game.Card{game.Clubs, game.Seven}},
			valid: false,
		},
		{
			name:  "contains a three",
			hand:  []game.Card{game.Card{game.Clubs, game.Five}, game.Card{game.Clubs, game.Five}, game.Card{game.Clubs, game.Three}},
			valid: false,
		},
		{
			name:  "max wildcards",
			hand:  []game.Card{game.Card{game.Clubs, game.Two}, game.Card{game.Clubs, game.Joker}, game.Card{game.Clubs, game.Joker}, game.Card{game.Clubs, game.Five}},
			valid: true,
		},
		{
			name:  "wildcards meld",
			hand:  []game.Card{game.Card{game.Clubs, game.Two}, game.Card{game.Clubs, game.Joker}, game.Card{game.Clubs, game.Joker}},
			valid: true,
		},
		{
			name:  "too many wildcards",
			hand:  []game.Card{game.Card{game.Clubs, game.Two}, game.Card{game.Clubs, game.Joker}, game.Card{game.Clubs, game.Joker}, game.Card{game.Clubs, game.Joker}, game.Card{game.Clubs, game.Four}},
			valid: false,
		},
		{
			name:  "no cards",
			hand:  []game.Card{},
			valid: false,
		},
	}
}

func TestValidateMeld(t *testing.T) {
	tests := getMeldTestScenarios()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			player := game.Player{
				Name: tt.name,
				Hand: tt.hand,
			}
			var cardsToPlay []int
			for i := range player.Hand {
				cardsToPlay = append(cardsToPlay, i)
			}

			meld, err := player.ValidateMeld(cardsToPlay)
			meldLength := len(meld.Cards)
			cardsPlayed := len(tt.hand)

			if tt.valid && cardsPlayed != meldLength {
				t.Errorf("Meld does not match the number of cards played. %d played, %d found in meld.", cardsPlayed, meldLength)
			}

			if err != nil && tt.valid {
				t.Error(err)
			}

			if err == nil && !tt.valid {
				t.Error("Expected error")
			}

			if tt.valid && !slices.Equal(tt.hand, meld.Cards) {
				t.Errorf("Expected meld matching %s got %s", tt.hand, meld.Cards)
			}
		})
	}
}

func TestNewMeld(t *testing.T) {

}
