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

	for _, player := range gameObj.State.Players {
		if len(player.Hand) != 15 {
			t.Errorf("Player %s has %d cards in their hand, 15 expected", player.Name, len(player.Hand))
		}
		if len(player.Foot) != 11 {
			t.Errorf("Player %s has %d cards in their foot, 11 expected", player.Name, len(player.Foot))
		}
	}

	if len(gameObj.State.DiscardPile) != 1 {
		t.Errorf("Only one card should be discarded. %d given.", len(gameObj.State.DiscardPile))
	}

	if gameObj.State.Deck.Count() != 111 {
		t.Errorf("Too many cards delt. Have %d in deck, 111 expected.", gameObj.State.Deck.Count())
	}

}

func TestMakeMeld(t *testing.T) {
	var tests = []struct {
		hand  []game.Card
		valid bool
		name  string
	}{
		{[]game.Card{game.Card{game.Clubs, game.Five}, game.Card{game.Clubs, game.Five}, game.Card{game.Clubs, game.Five}}, true, "natural"},
		{[]game.Card{game.Card{game.Clubs, game.Five}, game.Card{game.Clubs, game.Six}, game.Card{game.Clubs, game.Seven}}, false, "mixed rank"},
		{[]game.Card{game.Card{game.Clubs, game.Joker}, game.Card{game.Clubs, game.Six}, game.Card{game.Clubs, game.Seven}}, false, "mixed rank with wildcard"},
		{[]game.Card{game.Card{game.Clubs, game.Two}, game.Card{game.Clubs, game.Joker}, game.Card{game.Clubs, game.Four}}, true, "unnatural"},
		{[]game.Card{game.Card{game.Clubs, game.Two}, game.Card{game.Clubs, game.Four}, game.Card{game.Clubs, game.Four}}, true, "unnatural"},
		{[]game.Card{game.Card{game.Clubs, game.Two}, game.Card{game.Clubs, game.Joker}, game.Card{game.Clubs, game.Seven}}, false, "unnatural with sevens"},
		{[]game.Card{game.Card{game.Clubs, game.Five}, game.Card{game.Clubs, game.Five}, game.Card{game.Clubs, game.Three}}, false, "contains a three"},
		{[]game.Card{game.Card{game.Clubs, game.Two}, game.Card{game.Clubs, game.Joker}, game.Card{game.Clubs, game.Joker}, game.Card{game.Clubs, game.Five}}, true, "max wildcards"},
		{[]game.Card{game.Card{game.Clubs, game.Two}, game.Card{game.Clubs, game.Joker}, game.Card{game.Clubs, game.Joker}}, true, "wildcards meld"},
		{[]game.Card{game.Card{game.Clubs, game.Two}, game.Card{game.Clubs, game.Joker}, game.Card{game.Clubs, game.Joker}, game.Card{game.Clubs, game.Joker}, game.Card{game.Clubs, game.Four}}, false, "too many wildcards"},
		{[]game.Card{}, false, "no cards"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			team := game.Team{
				make([]game.Meld, 0),
				make([]game.Canasta, 0),
			}
			player := game.Player{
				Name: tt.name,
				Hand: tt.hand,
				Team: &team,
			}
			var cardsToPlay []int
			for i := range player.Hand {
				cardsToPlay = append(cardsToPlay, i)
			}
			startingHandLength := len(player.Hand)

			err := player.NewMeld(cardsToPlay)

			endingHandLength := len(player.Hand)
			if tt.valid && endingHandLength != 0 {
				t.Errorf("Hand should be reduced by three cards, started with %d and ended with %d", startingHandLength, endingHandLength)
			}

			if err != nil && tt.valid {
				t.Error(err)
			}

			if err == nil && !tt.valid {
				t.Error("Expected error")
			}

			if tt.valid && len(team.Melds) != 1 {
				t.Error("Should have one meld")
				t.FailNow()
			}

			if tt.valid && !slices.Equal(tt.hand, team.Melds[0].Cards) {
				t.Errorf("Expected Canasta matching %s got %s", tt.hand, team.Melds[0].Cards)
			}
		})
	}

}
