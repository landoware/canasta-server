package canasta_test

import (
	"canasta-server/internal/canasta"
	"testing"
)

func TestDraw(t *testing.T) {
	g := canasta.NewGame([]string{"A", "B", "C", "D"})
	p := g.Players[0]

	startingHandLength := len(p.Hand)
	startingDeckLength := g.Hand.Deck.Count()

	g.DrawFromDeck(p)

	if len(p.Hand) != startingHandLength+2 {
		t.Error("Did not increase player's card count")
	}
	if g.Hand.Deck.Count() != startingDeckLength-2 {
		t.Error("Cards remaining in deck")
	}
	if g.Phase != canasta.PhasePlaying {
		t.Error("Phase did not advance")
	}
}

func TestNewMeld(t *testing.T) {
	tests := []struct {
		name        string
		rank        canasta.Rank
		hand        []canasta.Card
		valid       bool
		hasGoneDown bool
	}{
		{
			name:        "natural",
			rank:        canasta.Five,
			hand:        []canasta.Card{{0, canasta.Clubs, canasta.Five}, {1, canasta.Clubs, canasta.Five}, {2, canasta.Clubs, canasta.Five}},
			valid:       true,
			hasGoneDown: true,
		},
		{
			name:        "mixed rank",
			hand:        []canasta.Card{{0, canasta.Clubs, canasta.Five}, {1, canasta.Clubs, canasta.Six}, {2, canasta.Clubs, canasta.Seven}},
			valid:       false,
			hasGoneDown: true,
		},
		{
			name:        "mixed rank with wildcard",
			hand:        []canasta.Card{{0, canasta.Wild, canasta.Joker}, {1, canasta.Clubs, canasta.Six}, {2, canasta.Clubs, canasta.Seven}},
			valid:       false,
			hasGoneDown: true,
		},
		{
			name:        "unnatural",
			rank:        canasta.Four,
			hand:        []canasta.Card{{0, canasta.Clubs, canasta.Two}, {1, canasta.Wild, canasta.Joker}, {2, canasta.Clubs, canasta.Four}},
			valid:       true,
			hasGoneDown: true,
		},
		{
			name:        "unnatural mixed order",
			rank:        canasta.Four,
			hand:        []canasta.Card{{0, canasta.Clubs, canasta.Two}, {1, canasta.Clubs, canasta.Four}, {2, canasta.Clubs, canasta.Four}},
			valid:       true,
			hasGoneDown: true,
		},
		{
			name:        "unnatural with sevens",
			hand:        []canasta.Card{{0, canasta.Clubs, canasta.Two}, {1, canasta.Wild, canasta.Joker}, {2, canasta.Clubs, canasta.Seven}},
			valid:       false,
			hasGoneDown: true,
		},
		{
			name:        "contains a three",
			hand:        []canasta.Card{{0, canasta.Clubs, canasta.Five}, {1, canasta.Clubs, canasta.Five}, {2, canasta.Clubs, canasta.Three}},
			valid:       false,
			hasGoneDown: true,
		},
		{
			name:        "max wildcards",
			rank:        canasta.Five,
			hand:        []canasta.Card{{0, canasta.Clubs, canasta.Two}, {1, canasta.Wild, canasta.Joker}, {2, canasta.Wild, canasta.Joker}, {3, canasta.Clubs, canasta.Five}},
			valid:       true,
			hasGoneDown: true,
		},
		{
			name:        "wildcards meld",
			rank:        canasta.Wild,
			hand:        []canasta.Card{{0, canasta.Clubs, canasta.Two}, {1, canasta.Wild, canasta.Joker}, {2, canasta.Wild, canasta.Joker}},
			valid:       true,
			hasGoneDown: true,
		},
		{
			name:        "too many wildcards",
			hand:        []canasta.Card{{0, canasta.Clubs, canasta.Two}, {1, canasta.Wild, canasta.Joker}, {2, canasta.Wild, canasta.Joker}, {3, canasta.Wild, canasta.Joker}, {4, canasta.Clubs, canasta.Four}},
			valid:       false,
			hasGoneDown: true,
		},
		{
			name:        "no cards",
			hand:        []canasta.Card{},
			valid:       false,
			hasGoneDown: true,
		},
		{
			name:        "natural hasn't gone down",
			rank:        canasta.Five,
			hand:        []canasta.Card{{0, canasta.Clubs, canasta.Five}, {1, canasta.Clubs, canasta.Five}, {2, canasta.Clubs, canasta.Five}},
			valid:       true,
			hasGoneDown: false,
		},
		{
			name:        "max wildcards hasn't gone down",
			rank:        canasta.Five,
			hand:        []canasta.Card{{0, canasta.Clubs, canasta.Two}, {1, canasta.Wild, canasta.Joker}, {2, canasta.Wild, canasta.Joker}, {3, canasta.Clubs, canasta.Five}},
			valid:       true,
			hasGoneDown: false,
		},
		{
			name:        "wildcards meld hasn't gone down",
			rank:        canasta.Wild,
			hand:        []canasta.Card{{0, canasta.Clubs, canasta.Two}, {1, canasta.Wild, canasta.Joker}, {2, canasta.Wild, canasta.Joker}},
			valid:       true,
			hasGoneDown: false,
		},
		{
			name:        "unnatural hasn't gone down",
			rank:        canasta.Four,
			hand:        []canasta.Card{{0, canasta.Clubs, canasta.Two}, {1, canasta.Wild, canasta.Joker}, {2, canasta.Clubs, canasta.Four}},
			valid:       true,
			hasGoneDown: false,
		},
		{
			name:        "unnatural mixed order hasn't gone down",
			rank:        canasta.Four,
			hand:        []canasta.Card{{0, canasta.Clubs, canasta.Two}, {1, canasta.Clubs, canasta.Four}, {2, canasta.Clubs, canasta.Four}},
			valid:       true,
			hasGoneDown: false,
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

			g := canasta.NewGame([]string{"A", "B", "C", "D"})

			g.Players[0] = &canasta.Player{
				Name: tt.name,
				Hand: hand,
				Team: &canasta.Team{
					Melds:    make([]canasta.Meld, 0),
					GoneDown: tt.hasGoneDown,
				},
			}

			p := g.Players[0]

			err := g.NewMeld(p, cardsToPlay)

			if tt.valid && err != nil {
				t.Error(err)
			}

			if !tt.valid && err == nil {
				t.Log(p.Team.Melds)
				t.Error("Expected error")
			}

			if tt.valid && len(p.Hand) != 0 {
				t.Log(p.Hand)
				t.Error("Cards remained in player's hand")
			}

			if !tt.valid && len(p.Hand) == 0 && len(tt.hand) != 0 {
				t.Error("Took cards from hand for an invalid meld")
			}

			if tt.valid && tt.hasGoneDown && len(p.Team.Melds) == 0 {
				t.Error("Cards not found in meld")
			}

			if tt.valid && !tt.hasGoneDown && len(p.StagingMelds) == 0 {
				t.Error("Cards not found in staging meld")
			}

			if !tt.valid && len(p.Team.Melds) != 0 {
				t.Error("Cards found in meld for invalid move")
			}

			if tt.name == "wildcards meld" && p.Team.Melds[0].Rank != canasta.Wild {
				t.Errorf("Expected meld to be of rank %s, %s given", tt.rank.String(), p.Team.Melds[0].Rank.String())
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

			game := canasta.NewGame([]string{"A", "B", "C", "D"})

			for _, card := range tt.hand {
				hand[card.GetId()] = card
			}

			player := game.Players[0]
			player.Hand = hand
			player.Team.Melds = append(player.Team.Melds, tt.meld)

			err := game.AddToMeld(player, tt.add, game.TeamA.Melds[0].Id)

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
			game := canasta.NewGame([]string{"A", "B", "C", "D"})

			hand := make(canasta.PlayerHand)
			for _, card := range tt.hand {
				hand[card.GetId()] = card
			}

			player := game.Players[0]
			player.Hand = hand
			player.Team.Melds = append(player.Team.Melds, tt.meld)

			err := game.AddToMeld(player, tt.add, 0)

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
			game := canasta.NewGame([]string{"A", "B", "C", "D"})

			hand := make(canasta.PlayerHand)
			for _, card := range tt.hand {
				hand[card.GetId()] = card
			}

			player := game.Players[0]
			player.Hand = hand
			player.Team.GoneDown = true

			err := game.NewMeld(player, tt.add)

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

func TestValidBurnCards(t *testing.T) {
	tests := []struct {
		name        string
		playerHand  []canasta.Card
		cardsToBurn []int
		teamCanasta canasta.Canasta
	}{
		{
			name: "burn a card",
			playerHand: []canasta.Card{
				{7, canasta.Clubs, canasta.Seven},
			},
			cardsToBurn: []int{7},
			teamCanasta: canasta.Canasta{
				Id:   0,
				Rank: canasta.Seven,
				Cards: []canasta.Card{
					{0, canasta.Clubs, canasta.Seven},
					{1, canasta.Diamonds, canasta.Seven},
					{2, canasta.Hearts, canasta.Seven},
					{3, canasta.Spades, canasta.Seven},
					{4, canasta.Diamonds, canasta.Seven},
					{5, canasta.Clubs, canasta.Seven},
					{6, canasta.Hearts, canasta.Seven},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hand := make(canasta.PlayerHand)
			for _, card := range tt.playerHand {
				hand[card.GetId()] = card
			}
			g := canasta.NewGame([]string{"A", "B", "C", "D"})
			p := g.Players[0]
			p.Hand = hand
			p.Team.Canastas = append(p.Team.Canastas, tt.teamCanasta)

			err := g.BurnCards(p, tt.cardsToBurn, 0)

			if err != nil {
				t.Log(err)
				t.FailNow()
			}

			if len(p.Team.Canastas) != 1 {
				t.Error("No cannasta!")
				t.FailNow()
			}

			if len(p.Team.Canastas[0].Cards) != len(tt.cardsToBurn)+len(tt.teamCanasta.Cards) {
				t.Error("Did not burn the right number of cards")
			}

		})

	}
}

func TestInvalidBurnCards(t *testing.T) {
	tests := []struct {
		name        string
		playerHand  []canasta.Card
		cardsToBurn []int
		teamCanasta canasta.Canasta
	}{
		{
			name: "burn a wild on sevens",
			playerHand: []canasta.Card{
				{7, canasta.Wild, canasta.Joker},
			},
			cardsToBurn: []int{7},
			teamCanasta: canasta.Canasta{
				Id:   0,
				Rank: canasta.Seven,
				Cards: []canasta.Card{
					{0, canasta.Clubs, canasta.Seven},
					{1, canasta.Diamonds, canasta.Seven},
					{2, canasta.Hearts, canasta.Seven},
					{3, canasta.Spades, canasta.Seven},
					{4, canasta.Diamonds, canasta.Seven},
					{5, canasta.Clubs, canasta.Seven},
					{6, canasta.Hearts, canasta.Seven},
				},
			},
		},
		{
			name: "burn the wrong rank",
			playerHand: []canasta.Card{
				{7, canasta.Diamonds, canasta.Seven},
			},
			cardsToBurn: []int{7},
			teamCanasta: canasta.Canasta{
				Id:   0,
				Rank: canasta.Six,
				Cards: []canasta.Card{
					{0, canasta.Clubs, canasta.Six},
					{1, canasta.Diamonds, canasta.Six},
					{2, canasta.Hearts, canasta.Six},
					{3, canasta.Spades, canasta.Six},
					{4, canasta.Diamonds, canasta.Six},
					{5, canasta.Clubs, canasta.Six},
					{6, canasta.Hearts, canasta.Six},
				},
			},
		},
		{
			name: "burn too many wilds",
			playerHand: []canasta.Card{
				{7, canasta.Wild, canasta.Joker},
			},
			cardsToBurn: []int{7},
			teamCanasta: canasta.Canasta{
				Id:   0,
				Rank: canasta.King,
				Cards: []canasta.Card{
					{0, canasta.Clubs, canasta.King},
					{1, canasta.Diamonds, canasta.King},
					{2, canasta.Hearts, canasta.King},
					{3, canasta.Spades, canasta.King},
					{4, canasta.Diamonds, canasta.Two},
					{5, canasta.Clubs, canasta.Two},
					{6, canasta.Hearts, canasta.Two},
				},
			},
		},
		{
			name: "burn wild on a natural",
			playerHand: []canasta.Card{
				{7, canasta.Wild, canasta.Joker},
			},
			cardsToBurn: []int{7},
			teamCanasta: canasta.Canasta{
				Id:      0,
				Rank:    canasta.King,
				Natural: true,
				Cards: []canasta.Card{
					{0, canasta.Clubs, canasta.King},
					{1, canasta.Diamonds, canasta.King},
					{2, canasta.Hearts, canasta.King},
					{3, canasta.Spades, canasta.King},
					{4, canasta.Diamonds, canasta.King},
					{5, canasta.Clubs, canasta.King},
					{6, canasta.Hearts, canasta.King},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hand := make(canasta.PlayerHand)
			for _, card := range tt.playerHand {
				hand[card.GetId()] = card
			}
			g := canasta.NewGame([]string{"A", "B", "C", "D"})
			p := g.Players[0]
			p.Hand = hand
			p.Team.Canastas = append(p.Team.Canastas, tt.teamCanasta)

			err := g.BurnCards(p, tt.cardsToBurn, 0)

			if err == nil {
				t.Error("Expected error")
			}

			if len(p.Team.Canastas) != 1 {
				t.Error("No cannasta!")
				t.FailNow()
			}

			if len(p.Team.Canastas[0].Cards) == len(tt.cardsToBurn)+len(tt.teamCanasta.Cards) {
				t.Error("Should not burn any cards")
			}
		})
	}
}

func TestValidPickupDiscardPile(t *testing.T) {
	tests := []struct {
		name       string
		topCard    canasta.Card
		playedIds  []int
		playerHand []canasta.Card
	}{
		{
			name:      "regular card",
			topCard:   canasta.Card{0, canasta.Spades, canasta.Six},
			playedIds: []int{1, 2},
			playerHand: []canasta.Card{
				{1, canasta.Clubs, canasta.Six},
				{2, canasta.Hearts, canasta.Six},
			},
		},
		{
			name:      "pickup to start a wild meld",
			topCard:   canasta.Card{0, canasta.Wild, canasta.Joker},
			playedIds: []int{1, 2},
			playerHand: []canasta.Card{
				{1, canasta.Clubs, canasta.Two},
				{2, canasta.Hearts, canasta.Two},
			},
		},
		{
			name:      "start an unnatural meld with two wildcards",
			topCard:   canasta.Card{0, canasta.Wild, canasta.Eight},
			playedIds: []int{1, 2},
			playerHand: []canasta.Card{
				{1, canasta.Clubs, canasta.Two},
				{2, canasta.Hearts, canasta.Two},
			},
		},
		{
			name:      "start an unnatural meld with one wildcards",
			topCard:   canasta.Card{0, canasta.Wild, canasta.King},
			playedIds: []int{1, 2},
			playerHand: []canasta.Card{
				{1, canasta.Clubs, canasta.King},
				{2, canasta.Hearts, canasta.Two},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hand := make(canasta.PlayerHand)
			for _, card := range tt.playerHand {
				hand[card.GetId()] = card
			}
			g := canasta.NewGame([]string{"A", "B", "C", "D"})
			p := g.Players[0]
			p.Team.GoneDown = true
			for i := range 3 {
				g.Hand.DiscardPile = append(g.Hand.DiscardPile, canasta.Card{i + 10, canasta.Spades, canasta.Three})
			}
			g.Hand.DiscardPile = append(g.Hand.DiscardPile, tt.topCard)
			p.Hand = hand

			err := g.PickUpDiscardPile(p, tt.playedIds)

			if err != nil {
				t.Error(err)
			}

			if len(p.Team.Melds) == 0 {
				t.Error("Expected meld to be created")
			}

			if len(g.Hand.DiscardPile) != 0 {
				t.Error("Expected empty discard pile")
			}

			if len(p.Hand) != 3 {
				t.Log(p.Hand)
				t.Error("Expected 3 cards to be in player's hand")
			}

			for _, card := range p.Hand {
				if card.Rank != canasta.Three {
					t.Error("Expected all threes in player's hand")
				}
			}

			if g.Phase != canasta.PhasePlaying {
				t.Error("Phase did not advance")
			}
		})
	}
}

func TestInvalidPickupDiscardPile(t *testing.T) {
	tests := []struct {
		name       string
		topCard    canasta.Card
		playedIds  []int
		playerHand []canasta.Card
	}{
		{
			name:      "regular incorrect card",
			topCard:   canasta.Card{0, canasta.Spades, canasta.Ace},
			playedIds: []int{1, 2},
			playerHand: []canasta.Card{
				{1, canasta.Clubs, canasta.Six},
				{2, canasta.Hearts, canasta.Six},
			},
		},
		{
			name:      "pickup on black three",
			topCard:   canasta.Card{0, canasta.Spades, canasta.Three},
			playedIds: []int{1, 2},
			playerHand: []canasta.Card{
				{1, canasta.Clubs, canasta.Three},
				{2, canasta.Hearts, canasta.Three},
			},
		},
		{
			name:      "try to start an unnatural sevens meld",
			topCard:   canasta.Card{0, canasta.Wild, canasta.Seven},
			playedIds: []int{1, 2},
			playerHand: []canasta.Card{
				{1, canasta.Clubs, canasta.Two},
				{2, canasta.Hearts, canasta.Two},
			},
		},
		{
			name:      "too few cards",
			topCard:   canasta.Card{0, canasta.Wild, canasta.Six},
			playedIds: []int{1, 2},
			playerHand: []canasta.Card{
				{1, canasta.Clubs, canasta.Six},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hand := make(canasta.PlayerHand)
			for _, card := range tt.playerHand {
				hand[card.GetId()] = card
			}
			g := canasta.NewGame([]string{"A", "B", "C", "D"})
			p := g.Players[0]
			p.Team.GoneDown = true
			for i := range 3 {
				g.Hand.DiscardPile = append(g.Hand.DiscardPile, canasta.Card{i + 10, canasta.Spades, canasta.Three})
			}
			g.Hand.DiscardPile = append(g.Hand.DiscardPile, tt.topCard)
			p.Hand = hand

			err := g.PickUpDiscardPile(p, tt.playedIds)

			if err == nil {
				t.Error("Expected error")
			}

			if len(p.Team.Melds) != 0 {
				t.Error("Should not create a meld")
			}

			if len(g.Hand.DiscardPile) != 4 {
				t.Log(g.Hand.DiscardPile)
				t.Error("Expected 4 cards in discard pile")
			}

			if len(p.Hand) != len(tt.playerHand) {
				t.Log(p.Hand)
				t.Error("Expected 2 cards to be in player's hand")
			}

			for _, card := range p.Hand {
				if card.Rank == canasta.Three && tt.name != "pickup on black three" {
					t.Error("Expected no threes in player's hand")
				}
			}

			if g.Phase == canasta.PhasePlaying {
				t.Error("Phase should not advance")
			}
		})
	}
}

func TestGoingDownByPickingUpThePile(t *testing.T) {
	tests := []struct {
		name         string
		topCard      canasta.Card
		playedIds    []int
		playerHand   []canasta.Card
		stagingMelds []canasta.Meld
	}{
		{
			name:      "regular cards no staging meld",
			topCard:   canasta.Card{0, canasta.Spades, canasta.Ace},
			playedIds: []int{1, 2},
			playerHand: []canasta.Card{
				{1, canasta.Clubs, canasta.Ace},
				{2, canasta.Hearts, canasta.Ace},
			},
		},
		{
			name:      "regular cards with staging meld",
			topCard:   canasta.Card{0, canasta.Spades, canasta.Four},
			playedIds: []int{1, 2},
			playerHand: []canasta.Card{
				{1, canasta.Clubs, canasta.Four},
				{2, canasta.Hearts, canasta.Four},
			},
			stagingMelds: []canasta.Meld{
				{
					Id:   3,
					Rank: canasta.King,
					Cards: []canasta.Card{
						{3, canasta.Clubs, canasta.King},
						{4, canasta.Hearts, canasta.King},
						{5, canasta.Hearts, canasta.King},
						{6, canasta.Hearts, canasta.King},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hand := make(canasta.PlayerHand)
			for _, card := range tt.playerHand {
				hand[card.GetId()] = card
			}
			g := canasta.NewGame([]string{"A", "B", "C", "D"})
			p := g.Players[0]
			p.Team.GoneDown = false
			for i := range 3 {
				g.Hand.DiscardPile = append(g.Hand.DiscardPile, canasta.Card{i + 10, canasta.Spades, canasta.Three})
			}
			g.Hand.DiscardPile = append(g.Hand.DiscardPile, tt.topCard)
			p.Hand = hand
			p.StagingMelds = tt.stagingMelds

			err := g.PickUpDiscardPile(p, tt.playedIds)

			if err != nil {
				t.Error(err)
			}

			if len(p.Team.Melds) == 0 {
				t.Error("Expected meld to be created")
			}

			if len(g.Hand.DiscardPile) != 0 {
				t.Error("Expected empty discard pile")
			}

			if len(p.Hand) != 3 {
				t.Log(p.Hand)
				t.Error("Expected 3 cards to be in player's hand")
			}

			for _, card := range p.Hand {
				if card.Rank != canasta.Three {
					t.Error("Expected all threes in player's hand")
				}
			}
		})
	}
}

func TestPickupFoot(t *testing.T) {
	tests := []struct {
		name        string
		madeCanasta bool
	}{
		{name: "valid", madeCanasta: true},
		{name: "invalid", madeCanasta: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := canasta.NewGame([]string{"A", "B", "C", "D"})
			g.Deal()

			p := g.Players[0]
			p.MadeCanasta = tt.madeCanasta
			startingLength := len(p.Hand)

			err := g.PickUpFoot(p)

			if tt.madeCanasta {
				if err != nil {
					t.Error(err)
				}

				if len(p.Hand) != startingLength+11 {
					t.Log(p.Hand)
					t.Error("Missing cards from player's hand")
				}
			} else {
				if err == nil {
					t.Error("Expected error")
				}

				if len(p.Hand) != startingLength {
					t.Error("Should not have added to player's hand")
				}

			}
		})
	}
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

			game.CurrentPlayer = 3
			player := game.Players[3]
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

			if tt.valid && game.Phase != canasta.PhaseDrawing {
				t.Error("Expected phase to reset")
			}

			if !tt.valid && game.Phase == canasta.PhaseDrawing {
				t.Error("Phase should not have reset")
			}

			if tt.valid && game.CurrentPlayer == 3 {
				t.Log(game.CurrentPlayer)
				t.Error("Turn should advance")
			}

			if !tt.valid && game.CurrentPlayer != 3 {
				t.Log(game.CurrentPlayer)
				t.Error("Turn should not advance")
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
