package canasta_test

import (
	"canasta-server/internal/canasta"
	"slices"
	"testing"
)

func TestDeal(t *testing.T) {
	names := []string{"One", "Two", "Three", "Four"}
	game := canasta.NewGame(names)

	game.Deal()

	for _, player := range game.Players {
		if len(player.Hand) != 15 {
			t.Errorf("Player %s has %d cards in their hand, 15 expected", player.Name, len(player.Hand))
		}
		if len(player.Foot) != 11 {
			t.Errorf("Player %s has %d cards in their foot, 11 expected", player.Name, len(player.Foot))
		}
	}

	if len(game.Hand.DiscardPile) != 1 {
		t.Errorf("Only one card should be discarded. %d given.", len(game.Hand.DiscardPile))
	}

	if game.Hand.Deck.Count() != 111 {
		t.Errorf("Too many cards delt. Have %d in deck, 111 expected.", game.Hand.Deck.Count())
	}

}

func TestValidateMeld(t *testing.T) {
	tests := []struct {
		name  string
		rank  canasta.Rank
		hand  []canasta.Card
		valid bool
	}{
		{
			name:  "natural",
			rank:  canasta.Five,
			hand:  []canasta.Card{{0, canasta.Clubs, canasta.Five}, {1, canasta.Clubs, canasta.Five}, {2, canasta.Clubs, canasta.Five}},
			valid: true,
		},
		{
			name:  "mixed rank",
			hand:  []canasta.Card{{0, canasta.Clubs, canasta.Five}, {1, canasta.Clubs, canasta.Six}, {2, canasta.Clubs, canasta.Seven}},
			valid: false,
		},
		{
			name:  "mixed rank with wildcard",
			hand:  []canasta.Card{{0, canasta.Wild, canasta.Joker}, {1, canasta.Clubs, canasta.Six}, {2, canasta.Clubs, canasta.Seven}},
			valid: false,
		},
		{
			name:  "unnatural",
			rank:  canasta.Four,
			hand:  []canasta.Card{{0, canasta.Clubs, canasta.Two}, {1, canasta.Wild, canasta.Joker}, {2, canasta.Clubs, canasta.Four}},
			valid: true,
		},
		{
			name:  "unnatural mixed order",
			rank:  canasta.Four,
			hand:  []canasta.Card{{0, canasta.Clubs, canasta.Two}, {1, canasta.Clubs, canasta.Four}, {2, canasta.Clubs, canasta.Four}},
			valid: true,
		},
		{
			name:  "unnatural with sevens",
			hand:  []canasta.Card{{0, canasta.Clubs, canasta.Two}, {1, canasta.Wild, canasta.Joker}, {2, canasta.Clubs, canasta.Seven}},
			valid: false,
		},
		{
			name:  "contains a three",
			hand:  []canasta.Card{{0, canasta.Clubs, canasta.Five}, {1, canasta.Clubs, canasta.Five}, {2, canasta.Clubs, canasta.Three}},
			valid: false,
		},
		{
			name:  "max wildcards",
			rank:  canasta.Five,
			hand:  []canasta.Card{{0, canasta.Clubs, canasta.Two}, {1, canasta.Wild, canasta.Joker}, {2, canasta.Wild, canasta.Joker}, {3, canasta.Clubs, canasta.Five}},
			valid: true,
		},
		{
			name:  "wildcards meld",
			rank:  canasta.Wild,
			hand:  []canasta.Card{{0, canasta.Clubs, canasta.Two}, {1, canasta.Wild, canasta.Joker}, {2, canasta.Wild, canasta.Joker}},
			valid: true,
		},
		{
			name:  "too many wildcards",
			hand:  []canasta.Card{{0, canasta.Clubs, canasta.Two}, {1, canasta.Wild, canasta.Joker}, {2, canasta.Wild, canasta.Joker}, {3, canasta.Wild, canasta.Joker}, {4, canasta.Clubs, canasta.Four}},
			valid: false,
		},
		{
			name:  "no cards",
			hand:  []canasta.Card{},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hand := make(canasta.PlayerHand)
			var cardsToPlay []int
			for _, card := range tt.hand {
				hand[card.GetId()] = card
				cardsToPlay = append(cardsToPlay, card.GetId())
			}

			player := canasta.Player{
				Name: tt.name,
				Hand: hand,
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

			if tt.valid && len(tt.hand) != len(meld.Cards) {
				for _, card := range tt.hand {
					if !slices.Contains(meld.Cards, card) {
						t.Logf("meld: %v", meld)
						t.Errorf("Expected meld to contain %s got %s", card, meld.Cards)

					}
				}
			}
		})
	}
}

func TestNewMeld(t *testing.T) {
	tests := []struct {
		name  string
		rank  canasta.Rank
		hand  []canasta.Card
		valid bool
	}{
		{
			name:  "natural",
			rank:  canasta.Five,
			hand:  []canasta.Card{{0, canasta.Clubs, canasta.Five}, {1, canasta.Clubs, canasta.Five}, {2, canasta.Clubs, canasta.Five}},
			valid: true,
		},
		{
			name:  "mixed rank",
			hand:  []canasta.Card{{0, canasta.Clubs, canasta.Five}, {1, canasta.Clubs, canasta.Six}, {2, canasta.Clubs, canasta.Seven}},
			valid: false,
		},
		{
			name:  "mixed rank with wildcard",
			hand:  []canasta.Card{{0, canasta.Wild, canasta.Joker}, {1, canasta.Clubs, canasta.Six}, {2, canasta.Clubs, canasta.Seven}},
			valid: false,
		},
		{
			name:  "unnatural",
			rank:  canasta.Four,
			hand:  []canasta.Card{{0, canasta.Clubs, canasta.Two}, {1, canasta.Wild, canasta.Joker}, {2, canasta.Clubs, canasta.Four}},
			valid: true,
		},
		{
			name:  "unnatural mixed order",
			rank:  canasta.Four,
			hand:  []canasta.Card{{0, canasta.Clubs, canasta.Two}, {1, canasta.Clubs, canasta.Four}, {2, canasta.Clubs, canasta.Four}},
			valid: true,
		},
		{
			name:  "unnatural with sevens",
			hand:  []canasta.Card{{0, canasta.Clubs, canasta.Two}, {1, canasta.Wild, canasta.Joker}, {2, canasta.Clubs, canasta.Seven}},
			valid: false,
		},
		{
			name:  "contains a three",
			hand:  []canasta.Card{{0, canasta.Clubs, canasta.Five}, {1, canasta.Clubs, canasta.Five}, {2, canasta.Clubs, canasta.Three}},
			valid: false,
		},
		{
			name:  "max wildcards",
			rank:  canasta.Five,
			hand:  []canasta.Card{{0, canasta.Clubs, canasta.Two}, {1, canasta.Wild, canasta.Joker}, {2, canasta.Wild, canasta.Joker}, {3, canasta.Clubs, canasta.Five}},
			valid: true,
		},
		{
			name:  "wildcards meld",
			rank:  canasta.Wild,
			hand:  []canasta.Card{{0, canasta.Clubs, canasta.Two}, {1, canasta.Wild, canasta.Joker}, {2, canasta.Wild, canasta.Joker}},
			valid: true,
		},
		{
			name:  "too many wildcards",
			hand:  []canasta.Card{{0, canasta.Clubs, canasta.Two}, {1, canasta.Wild, canasta.Joker}, {2, canasta.Wild, canasta.Joker}, {3, canasta.Wild, canasta.Joker}, {4, canasta.Clubs, canasta.Four}},
			valid: false,
		},
		{
			name:  "no cards",
			hand:  []canasta.Card{},
			valid: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hand := make(canasta.PlayerHand, 0)
			var cardsToPlay []int
			for _, card := range tt.hand {
				hand[card.GetId()] = card
				cardsToPlay = append(cardsToPlay, card.GetId())
			}

			player := canasta.Player{
				Name: tt.name,
				Hand: hand,
				Team: &canasta.Team{
					Melds:    make([]canasta.Meld, 0),
					GoneDown: true,
				},
			}

			err := player.NewMeld(cardsToPlay)

			if tt.valid && err != nil {
				t.Error(err)
			}
			if !tt.valid && err == nil {
				t.Log(player.Team.Melds)
				t.Error("Expected error")
			}

			if tt.valid && len(player.Hand) != 0 {
				t.Log(player.Hand)
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

			if tt.name == "wildcards meld" && player.Team.Melds[0].Rank != canasta.Wild {
				t.Errorf("Expected meld to be of rank %s, %s given", tt.rank.String(), player.Team.Melds[0].Rank.String())
			}
		})
	}
}

func TestAddToMeld(t *testing.T) {
	tests := []struct {
		name  string
		hand  []canasta.Card
		add   []int
		meld  canasta.Meld
		valid bool
	}{
		{
			name: "natural",
			hand: []canasta.Card{
				{3, canasta.Hearts, canasta.Queen},
			},
			add: []int{3},
			meld: canasta.Meld{
				Id:   0,
				Rank: canasta.Queen,
				Cards: []canasta.Card{
					{0, canasta.Hearts, canasta.Queen},
					{1, canasta.Spades, canasta.Queen},
					{2, canasta.Diamonds, canasta.Queen},
				},
			},
			valid: true,
		},
		{
			name: "making it unnatural",
			hand: []canasta.Card{
				{3, canasta.Hearts, canasta.Joker},
			},
			add: []int{3},
			meld: canasta.Meld{
				Id:   0,
				Rank: canasta.Queen,
				Cards: []canasta.Card{
					{0, canasta.Hearts, canasta.Queen},
					{1, canasta.Spades, canasta.Queen},
					{2, canasta.Diamonds, canasta.Queen},
				},
			},
			valid: true,
		},
		{
			name: "wrong card on a meld",
			hand: []canasta.Card{
				{3, canasta.Hearts, canasta.King},
			},
			add: []int{3},
			meld: canasta.Meld{
				Id:   0,
				Rank: canasta.Ten,
				Cards: []canasta.Card{
					{0, canasta.Hearts, canasta.Ten},
					{1, canasta.Spades, canasta.Ten},
					{2, canasta.Diamonds, canasta.Ten},
				},
			},
			valid: false,
		},
		{
			name: "unnatural on sevens",
			hand: []canasta.Card{
				{3, canasta.Wild, canasta.Joker},
			},
			add: []int{3},
			meld: canasta.Meld{
				Id:   0,
				Rank: canasta.Seven,
				Cards: []canasta.Card{
					{0, canasta.Hearts, canasta.Seven},
					{1, canasta.Spades, canasta.Seven},
					{3, canasta.Diamonds, canasta.Seven},
				},
			},
			valid: false,
		},
		{
			name: "playing a three",
			hand: []canasta.Card{
				{3, canasta.Spades, canasta.Three},
			},
			add: []int{3},
			meld: canasta.Meld{
				Id:   0,
				Rank: canasta.Seven,
				Cards: []canasta.Card{
					{0, canasta.Hearts, canasta.Queen},
					{1, canasta.Spades, canasta.Queen},
					{2, canasta.Diamonds, canasta.Queen},
				},
			},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hand := make(canasta.PlayerHand)

			game := canasta.NewGame([]string{"A"})

			for _, card := range tt.hand {
				hand[card.GetId()] = card
			}

			player := game.Players[0]
			player.Hand = hand
			player.Team.Melds = append(player.Team.Melds, tt.meld)

			err := player.AddToMeld(tt.add, game.TeamA.Melds[0].Id)

			if tt.valid && err != nil {
				t.Log(game.TeamA.Melds)
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
					break
				}
			}

			if tt.valid && isPlayingAWildCard && player.Team.Melds[0].WildCount < 1 {
				t.Log(player.Team.Melds)
				t.Error("Expected meld to become unnatural")
			}

		})
	}
}

func TestAddToMeldCreatesACanasta(t *testing.T) {
	tests := []struct {
		name  string
		hand  []canasta.Card
		add   []int
		meld  canasta.Meld
		valid bool
	}{
		{
			name: "natural",
			hand: []canasta.Card{
				{0, canasta.Hearts, canasta.Queen},
			},
			add: []int{0},
			meld: canasta.Meld{
				Id:   0,
				Rank: canasta.Queen,
				Cards: []canasta.Card{
					{1, canasta.Hearts, canasta.Queen},
					{2, canasta.Spades, canasta.Queen},
					{3, canasta.Spades, canasta.Queen},
					{4, canasta.Diamonds, canasta.Queen},
					{5, canasta.Diamonds, canasta.Queen},
					{6, canasta.Diamonds, canasta.Queen},
				},
			},
			valid: true,
		},
		{
			name: "unnatural",
			hand: []canasta.Card{
				{0, canasta.Hearts, canasta.Queen},
			},
			add: []int{0},
			meld: canasta.Meld{
				Id:   0,
				Rank: canasta.Queen,
				Cards: []canasta.Card{
					{1, canasta.Hearts, canasta.Queen},
					{2, canasta.Hearts, canasta.Queen},
					{3, canasta.Spades, canasta.Two},
					{4, canasta.Spades, canasta.Two},
					{5, canasta.Diamonds, canasta.Queen},
					{6, canasta.Diamonds, canasta.Queen},
				},
			},
			valid: true,
		},
		{
			name: "finish with wildcard",
			hand: []canasta.Card{
				{0, canasta.Hearts, canasta.Two},
			},
			add: []int{0},
			meld: canasta.Meld{
				Id:   0,
				Rank: canasta.Queen,
				Cards: []canasta.Card{
					{1, canasta.Hearts, canasta.Queen},
					{2, canasta.Hearts, canasta.Queen},
					{3, canasta.Diamonds, canasta.Queen},
					{4, canasta.Diamonds, canasta.Queen},
					{5, canasta.Spades, canasta.Queen},
					{6, canasta.Spades, canasta.Queen},
				},
			},
			valid: true,
		},
		{
			name: "try to finish with too many wilds",
			hand: []canasta.Card{
				{0, canasta.Hearts, canasta.Two},
			},
			add: []int{0},
			meld: canasta.Meld{
				Id:        0,
				Rank:      canasta.Queen,
				WildCount: 3,
				Cards: []canasta.Card{
					{1, canasta.Hearts, canasta.Two},
					{2, canasta.Hearts, canasta.Two},
					{3, canasta.Hearts, canasta.Two},
					{4, canasta.Diamonds, canasta.Queen},
					{5, canasta.Spades, canasta.Queen},
					{6, canasta.Spades, canasta.Queen},
				},
			},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			game := canasta.NewGame([]string{"A"})

			hand := make(canasta.PlayerHand)
			for _, card := range tt.hand {
				hand[card.GetId()] = card
			}

			player := game.Players[0]
			player.Hand = hand
			player.Team.Melds = append(player.Team.Melds, tt.meld)

			err := player.AddToMeld(tt.add, 0)

			if tt.valid && err != nil {
				t.Log(err)
				t.FailNow()
			}
			if !tt.valid && err == nil {
				t.Log(player.Team.Melds)
				t.Log(player.Team.Canastas)
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
	tests := []struct {
		name  string
		hand  []canasta.Card
		add   []int
		valid bool
	}{
		{
			name: "all seven at once",
			hand: []canasta.Card{
				{0, canasta.Hearts, canasta.Five},
				{1, canasta.Hearts, canasta.Five},
				{2, canasta.Diamonds, canasta.Five},
				{3, canasta.Diamonds, canasta.Five},
				{4, canasta.Diamonds, canasta.Five},
				{5, canasta.Spades, canasta.Five},
				{6, canasta.Spades, canasta.Five},
			},
			add:   []int{0, 1, 2, 3, 4, 5, 6},
			valid: true,
		},
		{
			name: "all seven wildcards at once",
			hand: []canasta.Card{
				{0, canasta.Wild, canasta.Joker},
				{1, canasta.Wild, canasta.Two},
				{2, canasta.Wild, canasta.Joker},
				{3, canasta.Wild, canasta.Two},
				{4, canasta.Wild, canasta.Joker},
				{5, canasta.Wild, canasta.Joker},
				{6, canasta.Wild, canasta.Joker},
			},
			add:   []int{0, 1, 2, 3, 4, 5, 6},
			valid: true,
		},
		{
			name: "mixed cards",
			hand: []canasta.Card{
				{0, canasta.Hearts, canasta.Seven},
				{1, canasta.Hearts, canasta.Five},
				{2, canasta.Diamonds, canasta.Six},
				{3, canasta.Diamonds, canasta.Five},
				{4, canasta.Diamonds, canasta.Five},
				{5, canasta.Spades, canasta.Five},
				{6, canasta.Spades, canasta.Five},
			},
			add:   []int{0, 1, 2, 3, 4, 5, 6},
			valid: false,
		},
		{
			name: "less than 7 cards",
			hand: []canasta.Card{
				{0, canasta.Hearts, canasta.Seven},
				{1, canasta.Hearts, canasta.Five},
				{2, canasta.Diamonds, canasta.Six},
				{3, canasta.Diamonds, canasta.Five},
				{4, canasta.Diamonds, canasta.Five},
				{5, canasta.Spades, canasta.Five},
			},
			add:   []int{0, 1, 2, 3, 4, 5},
			valid: false,
		},
		{
			name: "contains a three",
			hand: []canasta.Card{
				{0, canasta.Hearts, canasta.Seven},
				{1, canasta.Hearts, canasta.Five},
				{2, canasta.Diamonds, canasta.Six},
				{3, canasta.Diamonds, canasta.Five},
				{4, canasta.Diamonds, canasta.Five},
				{5, canasta.Spades, canasta.Five},
				{6, canasta.Spades, canasta.Three},
			},
			add:   []int{0, 1, 2, 3, 4, 5, 6},
			valid: false,
		},
		{
			name: "ten cards at once",
			hand: []canasta.Card{
				{0, canasta.Hearts, canasta.King},
				{1, canasta.Hearts, canasta.King},
				{2, canasta.Diamonds, canasta.King},
				{3, canasta.Diamonds, canasta.King},
				{4, canasta.Diamonds, canasta.Five},
				{5, canasta.Spades, canasta.King},
				{6, canasta.Spades, canasta.King},
				{7, canasta.Spades, canasta.King},
				{8, canasta.Spades, canasta.King},
				{9, canasta.Spades, canasta.King},
			},
			add:   []int{0, 1, 2, 3, 4, 5, 6},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			game := canasta.NewGame([]string{"A"})

			hand := make(canasta.PlayerHand)
			for _, card := range tt.hand {
				hand[card.GetId()] = card
			}

			player := game.Players[0]
			player.Hand = hand
			player.Team.GoneDown = true

			err := player.NewMeld(tt.add)

			if tt.valid && err != nil {
				t.Log(err)
				t.FailNow()
			}
			if !tt.valid && err == nil {
				t.Log(player.Team.Melds)
				t.Log(player.Team.Canastas)
				t.Log("Expected error")
				t.FailNow()
			}

			if tt.valid && len(player.Hand) != len(tt.hand)-len(tt.add) {
				t.Logf("Hand: %v\n", player.Hand)
				t.Errorf("Expected %d cards removed from hand. %d remaining", len(tt.add), len(player.Hand))
			}

			if tt.valid && len(player.Team.Canastas) == 0 {
				t.Logf("Melds: %v\n", player.Team.Melds)
				t.Logf("Hand: %v\n", player.Hand)
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
	tests := []struct {
		name          string
		hand          []canasta.Card
		discardedCard int
		canGoOut      bool
		valid         bool
	}{
		{
			name: "normal discard",
			hand: []canasta.Card{
				{0, canasta.Clubs, canasta.Ace},
				{1, canasta.Clubs, canasta.Three},
			},
			discardedCard: 1,
			canGoOut:      false,
			valid:         true,
		},
		{
			name: "going out",
			hand: []canasta.Card{
				{1, canasta.Clubs, canasta.Three},
			},
			discardedCard: 1,
			canGoOut:      true,
			valid:         true,
		},
		{
			name: "going out too early",
			hand: []canasta.Card{
				{1, canasta.Clubs, canasta.Three},
			},
			discardedCard: 1,
			canGoOut:      false,
			valid:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			game := canasta.NewGame([]string{"A", "B", "C", "D"})

			hand := make(canasta.PlayerHand)
			for _, card := range tt.hand {
				hand[card.GetId()] = card
			}

			player := game.Players[0]
			player.Hand = hand
			player.Team.CanGoOut = tt.canGoOut

			err := game.Discard(player, tt.discardedCard)

			if tt.valid && err != nil {
				t.Log(err)
				t.FailNow()
			}
			if !tt.valid && err == nil {
				t.Log("expected error")
				t.FailNow()
			}

			if tt.valid && len(player.Hand) != len(tt.hand)-1 && game.HandNumber == 1 {
				t.Log(game.HandNumber)
				t.Log(player.Hand)
				t.Error("Expected card to be discarded from hand")
			}

			if tt.valid && len(game.Hand.DiscardPile) < 1 {
				t.Log(game.Hand.DiscardPile)
				t.Error("Expected additional card in the discard pile")
			}

			if !tt.valid && len(game.Hand.DiscardPile) > 1 {
				t.Log(game.Hand.DiscardPile)
				t.Error("Unexpected discard")
			}
		})
	}
}

func TestAddToStagingMeld(t *testing.T) {

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
