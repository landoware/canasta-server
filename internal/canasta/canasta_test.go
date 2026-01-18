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

func TestValidGoDown(t *testing.T) {
	tests := []struct {
		name           string
		playerAStaging []canasta.Meld
		playerAHand    []canasta.Card
		playerCStaging []canasta.Meld
		playerCHand    []canasta.Card
	}{
		{
			name:        "nothing in either staging meld",
			playerAHand: make([]canasta.Card, 0),
			playerAStaging: []canasta.Meld{{
				Id: 0,
				Cards: []canasta.Card{
					{0, canasta.Wild, canasta.Joker},
					{1, canasta.Wild, canasta.Joker},
					{2, canasta.Wild, canasta.Joker},
				},
				Rank:      canasta.Wild,
				WildCount: 3,
			}},
			playerCHand:    make([]canasta.Card, 0),
			playerCStaging: make([]canasta.Meld, 0),
		},
		{
			name:        "partner has one staging meld",
			playerAHand: make([]canasta.Card, 0),
			playerAStaging: []canasta.Meld{{
				Id: 0,
				Cards: []canasta.Card{
					{0, canasta.Wild, canasta.Joker},
					{1, canasta.Wild, canasta.Joker},
					{2, canasta.Wild, canasta.Joker},
				},
				Rank:      canasta.Wild,
				WildCount: 3,
			}},
			playerCHand: make([]canasta.Card, 0),
			playerCStaging: []canasta.Meld{{
				Id: 3,
				Cards: []canasta.Card{
					{3, canasta.Diamonds, canasta.Seven},
					{4, canasta.Hearts, canasta.Seven},
					{5, canasta.Clubs, canasta.Seven},
				},
				Rank:      canasta.Seven,
				WildCount: 0,
			}},
		},
		{
			name:        "partner has multiple staging melds",
			playerAHand: make([]canasta.Card, 0),
			playerAStaging: []canasta.Meld{{
				Id: 0,
				Cards: []canasta.Card{
					{0, canasta.Wild, canasta.Joker},
					{1, canasta.Wild, canasta.Joker},
					{2, canasta.Wild, canasta.Joker},
				},
				Rank:      canasta.Wild,
				WildCount: 3,
			}},
			playerCHand: make([]canasta.Card, 0),
			playerCStaging: []canasta.Meld{
				{
					Id: 3,
					Cards: []canasta.Card{
						{3, canasta.Diamonds, canasta.Seven},
						{4, canasta.Hearts, canasta.Seven},
						{5, canasta.Clubs, canasta.Seven},
					},
					Rank:      canasta.Seven,
					WildCount: 0,
				},
				{
					Id: 3,
					Cards: []canasta.Card{
						{6, canasta.Diamonds, canasta.Four},
						{7, canasta.Hearts, canasta.Four},
						{8, canasta.Clubs, canasta.Four},
					},
					Rank:      canasta.Four,
					WildCount: 0,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			aHand := make(canasta.PlayerHand)
			cHand := make(canasta.PlayerHand)

			for _, card := range tt.playerAHand {
				aHand[card.GetId()] = card
			}
			for _, card := range tt.playerCHand {
				cHand[card.GetId()] = card
			}

			g := canasta.NewGame([]string{"A", "B", "C", "D"})

			a := g.Players[0]
			a.StagingMelds = tt.playerAStaging
			a.Hand = aHand

			c := g.Players[2]
			c.StagingMelds = tt.playerCStaging

			err := g.GoDown(a)

			if err != nil {
				t.Error(err)
			}

			if !a.Team.GoneDown {
				t.Error("Did not update team flag")
			}

			// Assert new melds created
			if len(a.Team.Melds) == 0 {
				t.Error("No meld(s) created")
			}

			// Assert player's hand has cards removed
			if len(a.Hand) != 0 {
				t.Log(a.Hand)
				t.Log("Player had cards added to their hand")
			}

			// Assert player's staging melds are removed
			if len(a.StagingMelds) != 0 {
				t.Error("Did not clear player's staging melds")
			}

			// Assert partner's hand had staaging melds moved to their hand
			partnerCardCount := 0
			for _, meld := range tt.playerCStaging {
				partnerCardCount += len(meld.Cards)
			}
			if len(tt.playerCHand)+partnerCardCount != len(c.Hand) {
				t.Errorf("Expected %d cards to be added to partner's hand, %d added", partnerCardCount, len(c.Hand)-len(tt.playerCHand))
			}

			// Assert partner's staging hand is empty
			if len(c.StagingMelds) != 0 {
				t.Error("Did not remove teamamate's staging melds")
			}

			// Assert a canasta was not created
			if len(a.Team.Canastas) != 0 {
				t.Error("Should not have created a canasta")
			}
		})
	}
}

func TestStagingMeldToCanasta(t *testing.T) {
	tests := []struct {
		name           string
		playerAStaging []canasta.Meld
		playerAHand    []canasta.Card
	}{
		{
			name:        "straight to a canasta",
			playerAHand: make([]canasta.Card, 0),
			playerAStaging: []canasta.Meld{{
				Id: 0,
				Cards: []canasta.Card{
					{0, canasta.Wild, canasta.Joker},
					{1, canasta.Wild, canasta.Joker},
					{2, canasta.Hearts, canasta.Two},
					{3, canasta.Wild, canasta.Joker},
					{4, canasta.Wild, canasta.Joker},
					{5, canasta.Hearts, canasta.Two},
					{6, canasta.Clubs, canasta.Two},
				},
				Rank:      canasta.Wild,
				WildCount: 7,
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			aHand := make(canasta.PlayerHand)

			for _, card := range tt.playerAHand {
				aHand[card.GetId()] = card
			}

			g := canasta.NewGame([]string{"A", "B", "C", "D"})

			a := g.Players[0]
			a.StagingMelds = tt.playerAStaging
			a.Hand = aHand

			err := g.GoDown(a)

			if err != nil {
				t.Error(err)
			}

			if len(a.StagingMelds) != 0 {
				t.Error("Should not have created a staging meld")
			}

			if len(a.Team.Canastas) == 0 {
				t.Error("Expected canasta to be created")
			}
		})
	}
}

func TestInvalidGoDown(t *testing.T) {
	tests := []struct {
		name           string
		playerAStaging []canasta.Meld
		playerAHand    []canasta.Card
		playerCStaging []canasta.Meld
		playerCHand    []canasta.Card
	}{
		{
			name:        "not enough points",
			playerAHand: make([]canasta.Card, 0),
			playerAStaging: []canasta.Meld{{
				Id: 0,
				Cards: []canasta.Card{
					{0, canasta.Hearts, canasta.Five},
					{1, canasta.Hearts, canasta.Five},
					{2, canasta.Hearts, canasta.Five},
				},
				Rank:      canasta.Five,
				WildCount: 0,
			}},
			playerCHand:    make([]canasta.Card, 0),
			playerCStaging: make([]canasta.Meld, 0),
		},
		{
			name:        "partner has one staging meld",
			playerAHand: make([]canasta.Card, 0),
			playerAStaging: []canasta.Meld{{
				Id: 0,
				Cards: []canasta.Card{
					{0, canasta.Hearts, canasta.Five},
					{1, canasta.Hearts, canasta.Five},
					{2, canasta.Hearts, canasta.Five},
				},
				Rank:      canasta.Wild,
				WildCount: 3,
			}},
			playerCHand: make([]canasta.Card, 0),
			playerCStaging: []canasta.Meld{{
				Id: 3,
				Cards: []canasta.Card{
					{3, canasta.Diamonds, canasta.Seven},
					{4, canasta.Hearts, canasta.Seven},
					{5, canasta.Clubs, canasta.Seven},
				},
				Rank:      canasta.Seven,
				WildCount: 0,
			}},
		},
		{
			name:        "partner has multiple staging melds",
			playerAHand: make([]canasta.Card, 0),
			playerAStaging: []canasta.Meld{{
				Id: 0,
				Cards: []canasta.Card{
					{0, canasta.Hearts, canasta.Five},
					{1, canasta.Hearts, canasta.Five},
					{2, canasta.Hearts, canasta.Five},
				},
				Rank:      canasta.Wild,
				WildCount: 3,
			}},
			playerCHand: make([]canasta.Card, 0),
			playerCStaging: []canasta.Meld{
				{
					Id: 3,
					Cards: []canasta.Card{
						{3, canasta.Diamonds, canasta.Seven},
						{4, canasta.Hearts, canasta.Seven},
						{5, canasta.Clubs, canasta.Seven},
					},
					Rank:      canasta.Seven,
					WildCount: 0,
				},
				{
					Id: 3,
					Cards: []canasta.Card{
						{6, canasta.Diamonds, canasta.Four},
						{7, canasta.Hearts, canasta.Four},
						{8, canasta.Clubs, canasta.Four},
					},
					Rank:      canasta.Four,
					WildCount: 0,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			aHand := make(canasta.PlayerHand)
			cHand := make(canasta.PlayerHand)

			for _, card := range tt.playerAHand {
				aHand[card.GetId()] = card
			}
			for _, card := range tt.playerCHand {
				cHand[card.GetId()] = card
			}

			g := canasta.NewGame([]string{"A", "B", "C", "D"})

			a := g.Players[0]
			a.StagingMelds = tt.playerAStaging
			a.Hand = aHand

			c := g.Players[2]
			c.StagingMelds = tt.playerCStaging

			err := g.GoDown(a)

			if err == nil {
				t.Log("Expected error")
			}

			if a.Team.GoneDown {
				t.Error("Team flag updated")
			}

			// Assert no new melds created
			if len(a.Team.Melds) != 0 {
				t.Log(a.Team.Melds)
				t.Error("Meld(s) created")
			}

			if len(a.StagingMelds) == 0 {
				t.Error("Should not clear player's staging melds")
			}

			// Assert player's hand has no cards removed
			if len(tt.playerAHand) != len(a.Hand) {
				t.Error("Should not change player's hand")
			}

			// Assert partner's hand had staaging melds moved to their hand
			if len(tt.playerCHand) != len(c.Hand) {
				t.Error("Should not change partner's hand")
			}

			// Assert partner's staging hand is empty
			if len(tt.playerCStaging) != 0 && len(c.StagingMelds) == 0 {
				t.Error("Should not remove teamamate's staging melds")
			}

			// Assert a canasta was not created
			if len(a.Team.Canastas) != 0 {
				t.Error("Should not have created a canasta")
			}
		})
	}
}
