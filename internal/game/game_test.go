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
			hand:  []game.Card{{0, game.Clubs, game.Five}, {1, game.Clubs, game.Five}, {2, game.Clubs, game.Five}},
			valid: true,
		},
		{
			name:  "mixed rank",
			hand:  []game.Card{{0, game.Clubs, game.Five}, {1, game.Clubs, game.Six}, {2, game.Clubs, game.Seven}},
			valid: false,
		},
		{
			name:  "mixed rank with wildcard",
			hand:  []game.Card{{0, game.Clubs, game.Joker}, {1, game.Clubs, game.Six}, {2, game.Clubs, game.Seven}},
			valid: false,
		},
		{
			name:  "unnatural",
			rank:  game.Four,
			hand:  []game.Card{{0, game.Clubs, game.Two}, {1, game.Clubs, game.Joker}, {2, game.Clubs, game.Four}},
			valid: true,
		},
		{
			name:  "unnatural mixed order",
			rank:  game.Four,
			hand:  []game.Card{{0, game.Clubs, game.Two}, {1, game.Clubs, game.Four}, {2, game.Clubs, game.Four}},
			valid: true,
		},
		{
			name:  "unnatural with sevens",
			hand:  []game.Card{{0, game.Clubs, game.Two}, {1, game.Clubs, game.Joker}, {2, game.Clubs, game.Seven}},
			valid: false,
		},
		{
			name:  "contains a three",
			hand:  []game.Card{{0, game.Clubs, game.Five}, {1, game.Clubs, game.Five}, {2, game.Clubs, game.Three}},
			valid: false,
		},
		{
			name:  "max wildcards",
			rank:  game.Five,
			hand:  []game.Card{{0, game.Clubs, game.Two}, {1, game.Clubs, game.Joker}, {2, game.Clubs, game.Joker}, {3, game.Clubs, game.Five}},
			valid: true,
		},
		{
			name:  "wildcards meld",
			rank:  game.Wild,
			hand:  []game.Card{{0, game.Clubs, game.Two}, {1, game.Clubs, game.Joker}, {2, game.Clubs, game.Joker}},
			valid: true,
		},
		{
			name:  "too many wildcards",
			hand:  []game.Card{{0, game.Clubs, game.Two}, {1, game.Clubs, game.Joker}, {2, game.Clubs, game.Joker}, {3, game.Clubs, game.Joker}, {4, game.Clubs, game.Four}},
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
				{3, game.Hearts, game.Queen},
			},
			add: []int{0},
			meld: game.Meld{
				Id:   0,
				Rank: game.Queen,
				Cards: []game.Card{
					{0, game.Hearts, game.Queen},
					{1, game.Spades, game.Queen},
					{2, game.Diamonds, game.Queen},
				},
			},
			valid: true,
		},
		{
			name: "making it unnatural",
			hand: []game.Card{
				{3, game.Hearts, game.Joker},
			},
			add: []int{0},
			meld: game.Meld{
				Id:   0,
				Rank: game.Queen,
				Cards: []game.Card{
					{0, game.Hearts, game.Queen},
					{1, game.Spades, game.Queen},
					{2, game.Diamonds, game.Queen},
				},
			},
			valid: true,
		},
		{
			name: "wrong card on a meld",
			hand: []game.Card{
				{3, game.Hearts, game.King},
			},
			add: []int{0},
			meld: game.Meld{
				Id:   0,
				Rank: game.Ten,
				Cards: []game.Card{
					{0, game.Hearts, game.Ten},
					{1, game.Spades, game.Ten},
					{2, game.Diamonds, game.Ten},
				},
			},
			valid: false,
		},
		{
			name: "unnatural on sevens",
			hand: []game.Card{
				{3, game.Wild, game.Joker},
			},
			add: []int{0},
			meld: game.Meld{
				Id:   0,
				Rank: game.Seven,
				Cards: []game.Card{
					{0, game.Hearts, game.Seven},
					{1, game.Spades, game.Seven},
					{3, game.Diamonds, game.Seven},
				},
			},
			valid: false,
		},
		{
			name: "playing a three",
			hand: []game.Card{
				{3, game.Spades, game.Three},
			},
			add: []int{0},
			meld: game.Meld{
				Id:   0,
				Rank: game.Seven,
				Cards: []game.Card{
					{0, game.Hearts, game.Queen},
					{1, game.Spades, game.Queen},
					{2, game.Diamonds, game.Queen},
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

			err := player.AddToMeld(tt.add, game.TeamA.Melds[0].Id)

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
				{0, game.Hearts, game.Queen},
			},
			add: []int{0},
			meld: game.Meld{
				Id:   0,
				Rank: game.Queen,
				Cards: []game.Card{
					{1, game.Hearts, game.Queen},
					{2, game.Spades, game.Queen},
					{3, game.Spades, game.Queen},
					{4, game.Diamonds, game.Queen},
					{5, game.Diamonds, game.Queen},
					{6, game.Diamonds, game.Queen},
				},
			},
			valid: true,
		},
		{
			name: "unnatural",
			hand: []game.Card{
				{0, game.Hearts, game.Queen},
			},
			add: []int{0},
			meld: game.Meld{
				Id:   0,
				Rank: game.Queen,
				Cards: []game.Card{
					{1, game.Hearts, game.Queen},
					{2, game.Hearts, game.Queen},
					{3, game.Spades, game.Two},
					{4, game.Spades, game.Two},
					{5, game.Diamonds, game.Queen},
					{6, game.Diamonds, game.Queen},
				},
			},
			valid: true,
		},
		{
			name: "finish with wildcard",
			hand: []game.Card{
				{0, game.Hearts, game.Two},
			},
			add: []int{0},
			meld: game.Meld{
				Id:   0,
				Rank: game.Queen,
				Cards: []game.Card{
					{1, game.Hearts, game.Queen},
					{2, game.Hearts, game.Queen},
					{3, game.Diamonds, game.Queen},
					{4, game.Diamonds, game.Queen},
					{5, game.Spades, game.Queen},
					{6, game.Spades, game.Queen},
				},
			},
			valid: true,
		},
		{
			name: "try to finish with too many wilds",
			hand: []game.Card{
				{0, game.Hearts, game.Two},
			},
			add: []int{0},
			meld: game.Meld{
				Id:   0,
				Rank: game.Queen,
				Cards: []game.Card{
					{1, game.Hearts, game.Two},
					{2, game.Hearts, game.Two},
					{3, game.Hearts, game.Two},
					{4, game.Diamonds, game.Queen},
					{5, game.Spades, game.Queen},
					{6, game.Spades, game.Queen},
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

			err := player.AddToMeld(tt.add, game.TeamA.Melds[0].Id)

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

func TestAddSevenCardsToMeld(t *testing.T) {
	// {
	// 	name: "all seven at once",
	// 	hand: []game.Card{
	// 		{game.Hearts, game.Five},
	// 		{game.Hearts, game.Five},
	// 		{game.Diamonds, game.Five},
	// 		{game.Diamonds, game.Five},
	// 		{game.Diamonds, game.Five},
	// 		{game.Spades, game.Five},
	// 		{game.Spades, game.Five},
	// 	},
	// 	add: []int{0},
	// 	meld: game.Meld{
	// 		Rank: game.Queen,
	// 		Cards: []game.Card{
	// 			{game.Hearts, game.Queen},
	// 			{game.Spades, game.Two},
	// 			{game.Diamonds, game.Queen},
	// 		},
	// 	},
	// 	valid: true,
	// },
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
