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
		t.Errorf("Only one card should be discarded. %d given.", len(gameObj.Hand.DiscardPile))
	}

	if gameObj.Hand.Deck.Count() != 111 {
		t.Errorf("Too many cards delt. Have %d in deck, 111 expected.", gameObj.Hand.Deck.Count())
	}

}

func getMeldTestScenarios() []struct {
	name  string
	rank  game.Rank
	hand  []game.Card
	valid bool
} {
	return []struct {
		name  string
		rank  game.Rank
		hand  []game.Card
		valid bool
	}{
		{
			name:  "natural",
			rank:  game.Five,
			hand:  []game.Card{{game.Clubs, game.Five}, {game.Clubs, game.Five}, {game.Clubs, game.Five}},
			valid: true,
		},
		{
			name:  "mixed rank",
			hand:  []game.Card{{game.Clubs, game.Five}, {game.Clubs, game.Six}, {game.Clubs, game.Seven}},
			valid: false,
		},
		{
			name:  "mixed rank with wildcard",
			hand:  []game.Card{{game.Clubs, game.Joker}, {game.Clubs, game.Six}, {game.Clubs, game.Seven}},
			valid: false,
		},
		{
			name:  "unnatural",
			rank:  game.Four,
			hand:  []game.Card{{game.Clubs, game.Two}, {game.Clubs, game.Joker}, {game.Clubs, game.Four}},
			valid: true,
		},
		{
			name:  "unnatural mixed order",
			rank:  game.Four,
			hand:  []game.Card{{game.Clubs, game.Two}, {game.Clubs, game.Four}, {game.Clubs, game.Four}},
			valid: true,
		},
		{
			name:  "unnatural with sevens",
			hand:  []game.Card{{game.Clubs, game.Two}, {game.Clubs, game.Joker}, {game.Clubs, game.Seven}},
			valid: false,
		},
		{
			name:  "contains a three",
			hand:  []game.Card{{game.Clubs, game.Five}, {game.Clubs, game.Five}, {game.Clubs, game.Three}},
			valid: false,
		},
		{
			name:  "max wildcards",
			rank:  game.Five,
			hand:  []game.Card{{game.Clubs, game.Two}, {game.Clubs, game.Joker}, {game.Clubs, game.Joker}, {game.Clubs, game.Five}},
			valid: true,
		},
		{
			name:  "wildcards meld",
			rank:  game.Wild,
			hand:  []game.Card{{game.Clubs, game.Two}, {game.Clubs, game.Joker}, {game.Clubs, game.Joker}},
			valid: true,
		},
		{
			name:  "too many wildcards",
			hand:  []game.Card{{game.Clubs, game.Two}, {game.Clubs, game.Joker}, {game.Clubs, game.Joker}, {game.Clubs, game.Joker}, {game.Clubs, game.Four}},
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
	tests := getMeldTestScenarios()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			player := game.Player{
				Name: tt.name,
				Hand: tt.hand,
				Team: &game.Team{
					Melds:    make([]game.Meld, 0),
					GoneDown: true,
				},
			}
			var cardsToPlay []int
			for i := range player.Hand {
				cardsToPlay = append(cardsToPlay, i)
			}

			err := player.NewMeld(cardsToPlay)

			if tt.valid && err != nil {
				t.Error(err)
			}
			if !tt.valid && err == nil {
				t.Error("Expected error")
			}

			if tt.valid && len(player.Hand) != 0 {
				t.Error("Cards remained in player's hand")
			}
			if !tt.valid && len(player.Hand) == 0 && len(tt.hand) != 0 {
				t.Error("Took cards from hand for an invalid meld")
			}

			if tt.valid && len(player.Team.Melds) == 0 {
				t.Error("Cards not found in meld")
			}

			if !tt.valid && len(player.Team.Melds) != 0 {
				t.Error("Cards found in meld for invalid move")
			}

			if tt.name == "wildcards meld" && player.Team.Melds[0].Rank != game.Wild {
				t.Errorf("Expected meld to be of rank %s, %s given", tt.rank.String(), player.Team.Melds[0].Rank.String())
			}
		})
	}
}

func TestAddToMeld(t *testing.T) {
	tests := []struct {
		name  string
		hand  []game.Card
		add   []int
		meld  game.Meld
		valid bool
	}{
		{
			name: "natural",
			hand: []game.Card{
				{game.Hearts, game.Queen},
			},
			add: []int{0},
			meld: game.Meld{
				Rank: game.Queen,
				Cards: []game.Card{
					{game.Hearts, game.Queen},
					{game.Spades, game.Queen},
					{game.Diamonds, game.Queen},
				},
			},
			valid: true,
		},
		{
			name: "making it unnatural",
			hand: []game.Card{
				{game.Hearts, game.Joker},
			},
			add: []int{0},
			meld: game.Meld{
				Rank: game.Queen,
				Cards: []game.Card{
					{game.Hearts, game.Queen},
					{game.Spades, game.Queen},
					{game.Diamonds, game.Queen},
				},
			},
			valid: true,
		},
		{
			name: "wrong card on a meld",
			hand: []game.Card{
				{game.Hearts, game.King},
			},
			add: []int{0},
			meld: game.Meld{
				Rank: game.Ten,
				Cards: []game.Card{
					{game.Hearts, game.Ten},
					{game.Spades, game.Ten},
					{game.Diamonds, game.Ten},
				},
			},
			valid: false,
		},
		{
			name: "unnatural on sevens",
			hand: []game.Card{
				{game.Hearts, game.Joker},
			},
			add: []int{0},
			meld: game.Meld{
				Rank: game.Seven,
				Cards: []game.Card{
					{game.Hearts, game.Seven},
					{game.Spades, game.Seven},
					{game.Diamonds, game.Seven},
				},
			},
			valid: false,
		},
		{
			name: "playing a three",
			hand: []game.Card{
				{game.Spades, game.Three},
			},
			add: []int{0},
			meld: game.Meld{
				Rank: game.Seven,
				Cards: []game.Card{
					{game.Hearts, game.Queen},
					{game.Spades, game.Queen},
					{game.Diamonds, game.Queen},
				},
			},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			game := game.NewGame([]string{"A"})

			player := game.Players[0]
			player.Hand = append(player.Hand, tt.hand...)
			player.Team.Melds = append(player.Team.Melds, tt.meld)

			err := player.AddToMeld(tt.add, &game.TeamA.Melds[0])

			if tt.valid && err != nil {
				t.Log(err)
				t.FailNow()
			}
			if !tt.valid && err == nil {
				t.Log("Expected error")
				t.FailNow()
			}

			if tt.valid && len(player.Team.Melds[0].Cards) != len(tt.add)+len(tt.meld.Cards) {
				t.Error("Meld should have the new card(s)")
			}

			if tt.valid && len(player.Hand) != len(tt.hand)-len(tt.add) {
				t.Log(player.Hand)
				t.Error("Player's hand did not have cards removed")
			}

			isPlayingAWildCard := false
			for _, card := range tt.hand {
				if card.IsWild() {
					isPlayingAWildCard = true
				}
			}

			if tt.valid && isPlayingAWildCard && !player.Team.Melds[0].Unnatural {
				t.Error("Expected meld to become unnatural")
			}

		})
	}
}

func TestAddToMeldCreatesACanasta(t *testing.T) {
	tests := []struct {
		name  string
		hand  []game.Card
		add   []int
		meld  game.Meld
		valid bool
	}{
		{
			name: "natural",
			hand: []game.Card{
				{game.Hearts, game.Queen},
			},
			add: []int{0},
			meld: game.Meld{
				Rank: game.Queen,
				Cards: []game.Card{
					{game.Hearts, game.Queen},
					{game.Spades, game.Queen},
					{game.Diamonds, game.Queen},
				},
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			game := game.NewGame([]string{"A"})

			player := game.Players[0]
			player.Hand = append(player.Hand, tt.hand...)
			player.Team.Melds = append(player.Team.Melds, tt.meld)

			err := player.AddToMeld(tt.add, &game.TeamA.Melds[0])

			if tt.valid && err != nil {
				t.Log(err)
				t.FailNow()
			}
			if !tt.valid && err == nil {
				t.Log("Expected error")
				t.FailNow()
			}

			if tt.valid && len(player.Team.Canastas) == 0 {
				t.Error("Expected a canaasta to be made")
			}

			if tt.valid && len(player.Team.Melds) != 0 {
				t.Error("Meld should have been removed")
			}

		})
	}
}

func TestCanDiscard(t *testing.T) {

}

func TestDiscard(t *testing.T) {

}

func TestCanGoDown(t *testing.T) {

}

func TestGoDown(t *testing.T) {

}

func TestNewCanasta(t *testing.T) {

}

func TestBurnCard(t *testing.T) {

}

func TestCanPickupDiscardPile(t *testing.T) {

}

func TestPickupDiscardPile(t *testing.T) {

}
