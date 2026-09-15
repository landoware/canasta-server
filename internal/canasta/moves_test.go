package canasta_test

import (
	"canasta-server/internal/canasta"
	"testing"
)

func TestDraw(t *testing.T) {
	g := canasta.NewGame("ABCE", []string{"A", "B", "C", "D"})
	p := g.Players[0]

	// Fix the deck contents so no random shuffle result can put a red
	// three among the drawn cards and throw off the count assertions.
	g.Hand.Deck.Cards = []canasta.Card{
		{0, canasta.Clubs, canasta.Four},
		{1, canasta.Clubs, canasta.Five},
		{2, canasta.Clubs, canasta.Six},
	}

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

func TestDrawRedThree(t *testing.T) {
	g := canasta.NewGame("ABCE", []string{"A", "B", "C", "D"})
	p := g.Players[0]

	// Fix the entire deck: top card drawn is a red three, the second
	// card drawn and its replacement are both known non-red-threes, so
	// the outcome doesn't depend on the game's shuffle at all.
	g.Hand.Deck.Cards = []canasta.Card{
		{0, canasta.Clubs, canasta.Four},
		{1, canasta.Clubs, canasta.Five},
		{2, canasta.Hearts, canasta.Three},
	}

	startingHandLength := len(p.Hand)
	startingDeckLength := g.Hand.Deck.Count()

	g.DrawFromDeck(p)

	if len(p.Hand) != startingHandLength+2 {
		t.Errorf("Did not increase player's card count: got %d, want %d", len(p.Hand), startingHandLength+2)
	}
	if g.Hand.Deck.Count() != startingDeckLength-3 {
		t.Errorf("Cards remaining in deck: got %d, want %d (drew 2 cards, one was red three, drew 1 replacement)", g.Hand.Deck.Count(), startingDeckLength-3)
	}
	if g.Phase != canasta.PhasePlaying {
		t.Error("Phase did not advance")
	}
	if len(p.Team.RedThrees) != 1 {
		t.Errorf("Did not add Red three to team: got %d, want 1", len(p.Team.RedThrees))
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

			g := canasta.NewGame("ABCE", []string{"A", "B", "C", "D"})

			g.Players[0] = &canasta.Player{
				Name: tt.name,
				Hand: hand,
				Team: &canasta.Team{
					Melds:    make([]canasta.Meld, 0),
					GoneDown: tt.hasGoneDown,
					// This test is about meld validity, not the
					// hand-stranding rule (see TestWouldStrandHand) —
					// every case here plays its entire hand.
					CanGoOut: true,
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

			game := canasta.NewGame("ABCE", []string{"A", "B", "C", "D"})

			for _, card := range tt.hand {
				hand[card.GetId()] = card
			}

			player := game.Players[0]
			player.Hand = hand
			// This test is about meld validity, not the hand-stranding
			// rule (see TestWouldStrandHand) — every case here plays
			// its entire hand.
			player.Team.CanGoOut = true
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
			game := canasta.NewGame("ABCE", []string{"A", "B", "C", "D"})

			hand := make(canasta.PlayerHand)
			for _, card := range tt.hand {
				hand[card.GetId()] = card
			}

			player := game.Players[0]
			player.Hand = hand
			// This test is about canasta-completion mechanics, not the
			// hand-stranding rule (see TestWouldStrandHand) — every
			// case here plays its entire hand.
			player.Team.CanGoOut = true
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

func TestNewCanastaGetsAUniqueId(t *testing.T) {
	// Regression test: NewCanasta used to leave Canasta.Id unset (always
	// 0), so a team with more than one canasta couldn't tell them apart
	// client-side — e.g. clicking to expand one canasta would show
	// another's cards, since every canasta shared the same id.
	game := canasta.NewGame("ABCE", []string{"A", "B", "C", "D"})

	hand := canasta.PlayerHand{
		0: {0, canasta.Hearts, canasta.Queen},
		1: {1, canasta.Hearts, canasta.King},
	}

	player := game.Players[0]
	player.Hand = hand
	// This test is about canasta id uniqueness, not the hand-stranding
	// rule (see TestWouldStrandHand) — it plays its whole 2-card hand.
	player.Team.CanGoOut = true
	player.Team.Melds = append(player.Team.Melds,
		canasta.Meld{
			Id:   100,
			Rank: canasta.Queen,
			Cards: []canasta.Card{
				{10, canasta.Hearts, canasta.Queen},
				{11, canasta.Spades, canasta.Queen},
				{12, canasta.Spades, canasta.Queen},
				{13, canasta.Diamonds, canasta.Queen},
				{14, canasta.Diamonds, canasta.Queen},
				{15, canasta.Diamonds, canasta.Queen},
			},
		},
		canasta.Meld{
			Id:   200,
			Rank: canasta.King,
			Cards: []canasta.Card{
				{20, canasta.Hearts, canasta.King},
				{21, canasta.Spades, canasta.King},
				{22, canasta.Spades, canasta.King},
				{23, canasta.Diamonds, canasta.King},
				{24, canasta.Diamonds, canasta.King},
				{25, canasta.Diamonds, canasta.King},
			},
		},
	)

	// Complete the Queens meld into a canasta first, then the Kings —
	// mirrors a team finishing two canastas over the course of a game.
	if err := game.AddToMeld(player, []int{0}, 100); err != nil {
		t.Fatal(err)
	}
	if err := game.AddToMeld(player, []int{1}, 200); err != nil {
		t.Fatal(err)
	}

	if len(player.Team.Canastas) != 2 {
		t.Fatalf("expected 2 canastas, got %d", len(player.Team.Canastas))
	}

	queensCanasta, kingsCanasta := player.Team.Canastas[0], player.Team.Canastas[1]
	if queensCanasta.Id != 100 {
		t.Errorf("expected the Queens canasta to keep its meld's id 100, got %d", queensCanasta.Id)
	}
	if kingsCanasta.Id != 200 {
		t.Errorf("expected the Kings canasta to keep its meld's id 200, got %d", kingsCanasta.Id)
	}
	if queensCanasta.Id == kingsCanasta.Id {
		t.Error("two different canastas must not share an id")
	}
}

func TestAddToMeldOnStagingMeld(t *testing.T) {
	game := canasta.NewGame("ABCE", []string{"A", "B", "C", "D"})

	hand := canasta.PlayerHand{
		3: {3, canasta.Hearts, canasta.Queen},
	}

	player := game.Players[0]
	player.Hand = hand
	// This test is about staging-meld mechanics, not the hand-stranding
	// rule (see TestWouldStrandHand) — it plays its whole 1-card hand.
	player.Team.CanGoOut = true
	player.StagingMelds = append(player.StagingMelds, canasta.Meld{
		Id:   0,
		Rank: canasta.Queen,
		Cards: []canasta.Card{
			{0, canasta.Hearts, canasta.Queen},
			{1, canasta.Spades, canasta.Queen},
			{2, canasta.Diamonds, canasta.Queen},
		},
	})

	err := game.AddToMeld(player, []int{3}, player.StagingMelds[0].Id)
	if err != nil {
		t.Fatal(err)
	}

	if len(player.StagingMelds[0].Cards) != 4 {
		t.Error("Staging meld should have the new card")
	}

	if len(player.Hand) != 0 {
		t.Error("Player's hand did not have the card removed")
	}

	if len(player.Team.Melds) != 0 {
		t.Error("Adding to a staging meld should not touch the team's official melds")
	}
}

func TestAddToMeldOnStagingMeldDoesNotAutoCanasta(t *testing.T) {
	game := canasta.NewGame("ABCE", []string{"A", "B", "C", "D"})

	hand := canasta.PlayerHand{
		6: {6, canasta.Hearts, canasta.Queen},
	}

	player := game.Players[0]
	player.Hand = hand
	// This test is about staging-meld mechanics, not the hand-stranding
	// rule (see TestWouldStrandHand) — it plays its whole 1-card hand.
	player.Team.CanGoOut = true
	player.StagingMelds = append(player.StagingMelds, canasta.Meld{
		Id:   0,
		Rank: canasta.Queen,
		Cards: []canasta.Card{
			{0, canasta.Hearts, canasta.Queen},
			{1, canasta.Spades, canasta.Queen},
			{2, canasta.Diamonds, canasta.Queen},
			{3, canasta.Hearts, canasta.Queen},
			{4, canasta.Spades, canasta.Queen},
			{5, canasta.Diamonds, canasta.Queen},
		},
	})

	err := game.AddToMeld(player, []int{6}, player.StagingMelds[0].Id)
	if err != nil {
		t.Fatal(err)
	}

	if len(player.StagingMelds[0].Cards) != 7 {
		t.Error("Staging meld should have the new card")
	}

	if len(player.Team.Canastas) != 0 {
		t.Error("A staging meld should never become a canasta before going down")
	}
}

func TestAddToMeldOnAllWildMeldHasNoWildcardCap(t *testing.T) {
	game := canasta.NewGame("ABCE", []string{"A", "B", "C", "D"})

	hand := canasta.PlayerHand{
		3: {3, canasta.Wild, canasta.Joker},
	}

	player := game.Players[0]
	player.Hand = hand
	// This test is about the all-wild wildcard-cap exception, not the
	// hand-stranding rule (see TestWouldStrandHand) — it plays its
	// whole 1-card hand.
	player.Team.CanGoOut = true
	player.Team.Melds = append(player.Team.Melds, canasta.Meld{
		Id:        0,
		Rank:      canasta.Wild,
		WildCount: 3,
		Cards: []canasta.Card{
			{0, canasta.Hearts, canasta.Two},
			{1, canasta.Spades, canasta.Two},
			{2, canasta.Wild, canasta.Joker},
		},
	})

	// Already at the normal 3-wildcard cap — an all-wild meld should still
	// accept a 4th, unlike a mixed-rank meld (see TestAddToMeld's
	// "unnatural" cases, which do enforce that cap).
	err := game.AddToMeld(player, []int{3}, player.Team.Melds[0].Id)
	if err != nil {
		t.Fatal(err)
	}

	if player.Team.Melds[0].WildCount != 4 {
		t.Errorf("expected WildCount 4, got %d", player.Team.Melds[0].WildCount)
	}

	if len(player.Team.Melds[0].Cards) != 4 {
		t.Error("Meld should have the new card")
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
			game := canasta.NewGame("ABCE", []string{"A", "B", "C", "D"})

			hand := make(canasta.PlayerHand)
			for _, card := range tt.hand {
				hand[card.GetId()] = card
			}

			player := game.Players[0]
			player.Hand = hand
			player.Team.GoneDown = true
			// This test is about meld/canasta validity, not the
			// hand-stranding rule (see TestWouldStrandHand).
			player.Team.CanGoOut = true

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
			g := canasta.NewGame("ABCE", []string{"A", "B", "C", "D"})
			p := g.Players[0]
			p.Hand = hand
			// This test is about burn validity, not the hand-stranding
			// rule (see TestWouldStrandHand) — it plays its whole hand.
			p.Team.CanGoOut = true
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
			g := canasta.NewGame("ABCE", []string{"A", "B", "C", "D"})
			p := g.Players[0]
			p.Hand = hand
			// This test is about burn validity, not the hand-stranding
			// rule (see TestWouldStrandHand) — keep the rejection
			// reason the one each case actually means to exercise.
			p.Team.CanGoOut = true
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
			g := canasta.NewGame("ABCE", []string{"A", "B", "C", "D"})
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
			g := canasta.NewGame("ABCE", []string{"A", "B", "C", "D"})
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
			g := canasta.NewGame("ABCE", []string{"A", "B", "C", "D"})
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
			g := canasta.NewGame("ABCE", []string{"A", "B", "C", "D"})
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

func TestPickUpFootBlockedUntilDiscardOnFirstCanastaTurn(t *testing.T) {
	g := canasta.NewGame("ABCE", []string{"A", "B", "C", "D"})
	g.Deal()

	p := g.Players[0]
	p.Hand = canasta.PlayerHand{
		900: {900, canasta.Clubs, canasta.Ace},
		901: {901, canasta.Clubs, canasta.King},
	}

	sevenFours := make([]canasta.Card, 7)
	for i := range sevenFours {
		sevenFours[i] = canasta.Card{10 + i, canasta.Hearts, canasta.Four}
	}
	sevenFives := make([]canasta.Card, 7)
	for i := range sevenFives {
		sevenFives[i] = canasta.Card{20 + i, canasta.Hearts, canasta.Five}
	}
	p.Team.Melds = []canasta.Meld{
		{Id: 100, Rank: canasta.Four, Cards: sevenFours},
		{Id: 200, Rank: canasta.Five, Cards: sevenFives},
	}

	p.NewCanasta(0) // first canasta this turn
	if !p.CanastaMadeThisTurn {
		t.Fatal("expected CanastaMadeThisTurn to be true right after the first canasta")
	}

	// A second canasta later the same turn shouldn't clear the block —
	// only completing the *first* one ever starts it, and only that
	// turn's Discard ends it.
	p.NewCanasta(0) // index 0 now points at the fives meld, the fours having been removed
	if !p.CanastaMadeThisTurn {
		t.Error("a second canasta the same turn should not clear the same-turn block")
	}

	if err := g.PickUpFoot(p); err == nil {
		t.Error("expected PickUpFoot to be blocked on the turn the first canasta was made")
	}
	if len(p.Foot) == 0 {
		t.Fatal("test setup: player should still have a foot to pick up")
	}

	if err := g.Discard(p, 900); err != nil {
		t.Fatalf("unexpected error discarding: %v", err)
	}
	if p.CanastaMadeThisTurn {
		t.Error("expected CanastaMadeThisTurn to be cleared after discarding")
	}

	if err := g.PickUpFoot(p); err != nil {
		t.Errorf("expected PickUpFoot to succeed once discarded: %v", err)
	}
	if len(p.Foot) != 0 {
		t.Error("expected the foot to have been picked up")
	}
}

func TestPickUpFootTurnTiming(t *testing.T) {
	// Every case starts from the same eligible-to-pick-up baseline
	// (MadeCanasta true, CanastaMadeThisTurn false, still has a foot —
	// Deal() already dealt one) and only varies CurrentPlayer/Phase.
	setup := func() (canasta.Game, *canasta.Player) {
		g := canasta.NewGame("ABCE", []string{"A", "B", "C", "D"})
		g.Deal()
		p := g.Players[0]
		p.MadeCanasta = true
		return g, p
	}

	t.Run("succeeds on another player's turn, even mid-play", func(t *testing.T) {
		g, p := setup()
		g.CurrentPlayer = 1 // not p's seat
		g.Phase = canasta.PhasePlaying

		if err := g.PickUpFoot(p); err != nil {
			t.Errorf("expected pickup to succeed on another player's turn: %v", err)
		}
		if len(p.Foot) != 0 {
			t.Error("expected the foot to have been picked up")
		}
	})

	t.Run("succeeds on the player's own turn before they've drawn", func(t *testing.T) {
		g, p := setup()
		g.CurrentPlayer = 0 // p's own seat
		g.Phase = canasta.PhaseDrawing

		if err := g.PickUpFoot(p); err != nil {
			t.Errorf("expected pickup to succeed before drawing: %v", err)
		}
		if len(p.Foot) != 0 {
			t.Error("expected the foot to have been picked up")
		}
	})

	t.Run("fails on the player's own turn after they've drawn", func(t *testing.T) {
		g, p := setup()
		g.CurrentPlayer = 0 // p's own seat
		g.Phase = canasta.PhasePlaying
		startingFootLen := len(p.Foot)

		if err := g.PickUpFoot(p); err == nil {
			t.Error("expected pickup to be blocked after drawing on your own turn")
		}
		if len(p.Foot) != startingFootLen {
			t.Error("should not have touched the foot")
		}
	})
}

// TestWouldStrandHand covers the soft-lock this guards against: Discard
// refuses to discard from a 1-card hand without go-out permission (see
// TestDiscard's "going out too early" case), so nothing else may ever
// reduce a hand to 0 or 1 cards without that permission either — a
// player left holding one un-discardable card would never be able to
// end their turn again.
func TestWouldStrandHand(t *testing.T) {
	t.Run("NewMeld rejects leaving fewer than two cards without permission", func(t *testing.T) {
		g := canasta.NewGame("ABCE", []string{"A", "B", "C", "D"})
		p := g.Players[0]
		p.Team.GoneDown = true
		p.Hand = canasta.PlayerHand{
			0: {0, canasta.Hearts, canasta.Four},
			1: {1, canasta.Clubs, canasta.Four},
			2: {2, canasta.Diamonds, canasta.Four},
		}

		err := g.NewMeld(p, []int{0, 1, 2})

		if err == nil {
			t.Error("expected NewMeld to be rejected")
		}
		if len(p.Hand) != 3 {
			t.Error("hand should be untouched by a rejected meld")
		}
	})

	t.Run("NewMeld succeeds once granted permission to go out", func(t *testing.T) {
		g := canasta.NewGame("ABCE", []string{"A", "B", "C", "D"})
		p := g.Players[0]
		p.Team.GoneDown = true
		p.Team.CanGoOut = true
		p.Hand = canasta.PlayerHand{
			0: {0, canasta.Hearts, canasta.Four},
			1: {1, canasta.Clubs, canasta.Four},
			2: {2, canasta.Diamonds, canasta.Four},
		}

		if err := g.NewMeld(p, []int{0, 1, 2}); err != nil {
			t.Errorf("expected NewMeld to succeed with go-out permission: %v", err)
		}
	})

	t.Run("NewMeld succeeds when spare cards remain", func(t *testing.T) {
		g := canasta.NewGame("ABCE", []string{"A", "B", "C", "D"})
		p := g.Players[0]
		p.Team.GoneDown = true
		p.Hand = canasta.PlayerHand{
			0: {0, canasta.Hearts, canasta.Four},
			1: {1, canasta.Clubs, canasta.Four},
			2: {2, canasta.Diamonds, canasta.Four},
			3: {3, canasta.Hearts, canasta.Eight},
			4: {4, canasta.Clubs, canasta.Nine},
		}

		if err := g.NewMeld(p, []int{0, 1, 2}); err != nil {
			t.Errorf("expected NewMeld to succeed with 2 cards left over: %v", err)
		}
		if len(p.Hand) != 2 {
			t.Errorf("expected 2 cards left in hand, got %d", len(p.Hand))
		}
	})

	// Mirrors the exact reported bug: a 2-card hand, no go-out
	// permission, laying one card off on an existing meld.
	t.Run("AddToMeld rejects a 2-card hand laying off one card without permission", func(t *testing.T) {
		g := canasta.NewGame("ABCE", []string{"A", "B", "C", "D"})
		p := g.Players[0]
		p.Hand = canasta.PlayerHand{
			0: {0, canasta.Hearts, canasta.Four},
			1: {1, canasta.Clubs, canasta.Eight},
		}
		p.Team.Melds = append(p.Team.Melds, canasta.Meld{
			Id:   100,
			Rank: canasta.Four,
			Cards: []canasta.Card{
				{10, canasta.Hearts, canasta.Four},
				{11, canasta.Spades, canasta.Four},
				{12, canasta.Diamonds, canasta.Four},
			},
		})

		err := g.AddToMeld(p, []int{0}, 100)

		if err == nil {
			t.Error("expected AddToMeld to be rejected")
		}
		if len(p.Hand) != 2 {
			t.Error("hand should be untouched by a rejected add")
		}
	})

	t.Run("AddToMeld succeeds from a 2-card hand once granted permission", func(t *testing.T) {
		g := canasta.NewGame("ABCE", []string{"A", "B", "C", "D"})
		p := g.Players[0]
		p.Team.CanGoOut = true
		p.Hand = canasta.PlayerHand{
			0: {0, canasta.Hearts, canasta.Four},
			1: {1, canasta.Clubs, canasta.Eight},
		}
		p.Team.Melds = append(p.Team.Melds, canasta.Meld{
			Id:   100,
			Rank: canasta.Four,
			Cards: []canasta.Card{
				{10, canasta.Hearts, canasta.Four},
				{11, canasta.Spades, canasta.Four},
				{12, canasta.Diamonds, canasta.Four},
			},
		})

		if err := g.AddToMeld(p, []int{0}, 100); err != nil {
			t.Errorf("expected AddToMeld to succeed with go-out permission: %v", err)
		}
	})

	t.Run("BurnCards rejects leaving fewer than two cards without permission", func(t *testing.T) {
		g := canasta.NewGame("ABCE", []string{"A", "B", "C", "D"})
		p := g.Players[0]
		p.Hand = canasta.PlayerHand{
			7: {7, canasta.Clubs, canasta.Seven},
		}
		p.Team.Canastas = append(p.Team.Canastas, canasta.Canasta{
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
		})

		err := g.BurnCards(p, []int{7}, 0)

		if err == nil {
			t.Error("expected BurnCards to be rejected")
		}
		if len(p.Hand) != 1 {
			t.Error("hand should be untouched by a rejected burn")
		}
	})

	// PickUpDiscardPile has its own correct check: it refills the hand
	// from the rest of the discard pile right after building the meld,
	// so the danger is specifically an almost-empty pile, not a small
	// contribution on its own.
	t.Run("PickUpDiscardPile rejects a 2-card hand when the pile has only the top card", func(t *testing.T) {
		g := canasta.NewGame("ABCE", []string{"A", "B", "C", "D"})
		p := g.Players[0]
		p.Team.GoneDown = true
		p.Hand = canasta.PlayerHand{
			1: {1, canasta.Clubs, canasta.Six},
			2: {2, canasta.Hearts, canasta.Six},
		}
		g.Hand.DiscardPile = []canasta.Card{{0, canasta.Spades, canasta.Six}}

		err := g.PickUpDiscardPile(p, []int{1, 2})

		if err == nil {
			t.Error("expected PickUpDiscardPile to be rejected")
		}
		if len(p.Hand) != 2 {
			t.Error("hand should be untouched by a rejected pickup")
		}
		if len(g.Hand.DiscardPile) != 1 {
			t.Error("discard pile should be untouched by a rejected pickup")
		}
	})

	t.Run("PickUpDiscardPile succeeds when the rest of the pile refills the hand", func(t *testing.T) {
		g := canasta.NewGame("ABCE", []string{"A", "B", "C", "D"})
		p := g.Players[0]
		p.Team.GoneDown = true
		p.Hand = canasta.PlayerHand{
			1: {1, canasta.Clubs, canasta.Six},
			2: {2, canasta.Hearts, canasta.Six},
		}
		// Same 2-card contribution as above, but padded with 3 extra
		// pile cards this time — matches the existing convention in
		// TestValidPickupDiscardPile.
		for i := range 3 {
			g.Hand.DiscardPile = append(g.Hand.DiscardPile, canasta.Card{i + 10, canasta.Spades, canasta.Three})
		}
		g.Hand.DiscardPile = append(g.Hand.DiscardPile, canasta.Card{0, canasta.Spades, canasta.Six})

		if err := g.PickUpDiscardPile(p, []int{1, 2}); err != nil {
			t.Errorf("expected PickUpDiscardPile to succeed once the pile refills the hand: %v", err)
		}
		if len(p.Hand) != 3 {
			t.Errorf("expected 3 cards (the padding) left in hand, got %d", len(p.Hand))
		}
	})

	// The escape hatch: a move that would otherwise strand the hand may
	// still proceed when the move itself completes the team's last
	// required canasta type, leaving exactly one card — mirrors the
	// reported edge case (a 2-card hand: King + a wildcard, adding the
	// wildcard to an existing 6-card meld completes the missing
	// "unnatural" canasta type).
	t.Run("AddToMeld succeeds when it completes the team's last required canasta type, leaving exactly one card", func(t *testing.T) {
		g := canasta.NewGame("ABCE", []string{"A", "B", "C", "D"})
		p := g.Players[0]
		p.Team.GoneDown = true
		// Already covers natural, sevens, and wildcards — only
		// unnatural is missing.
		p.Team.Canastas = []canasta.Canasta{
			{Id: 1, Rank: canasta.Four, Cards: makeCards(1, canasta.Four, 7), Natural: true},
			{Id: 2, Rank: canasta.Seven, Cards: makeCards(201, canasta.Seven, 7), Natural: true},
			{Id: 3, Rank: canasta.Wild, Cards: makeCards(301, canasta.Two, 7), Natural: false},
		}
		// A 6-card Queens meld, one wildcard away from completing an
		// unnatural canasta.
		p.Team.Melds = []canasta.Meld{{Id: 100, Rank: canasta.Queen, Cards: makeCards(400, canasta.Queen, 6)}}
		p.Hand = canasta.PlayerHand{
			500: {500, canasta.Clubs, canasta.King},
			501: {501, canasta.Wild, canasta.Two},
		}

		err := g.AddToMeld(p, []int{501}, 100)

		if err != nil {
			t.Errorf("expected AddToMeld to succeed, completing the last required canasta type: %v", err)
		}
		if len(p.Hand) != 1 {
			t.Errorf("expected exactly 1 card left in hand, got %d", len(p.Hand))
		}
		if !p.Team.MeetsGoOutRequirements() {
			t.Error("expected the team to now meet the go-out requirements")
		}
	})

	t.Run("AddToMeld still rejects when the resulting canasta does not complete a missing requirement", func(t *testing.T) {
		g := canasta.NewGame("ABCE", []string{"A", "B", "C", "D"})
		p := g.Players[0]
		p.Team.GoneDown = true
		// Already has a natural, an unnatural, and a wildcards canasta —
		// only sevens is missing, but this meld isn't sevens, so
		// completing it doesn't help.
		p.Team.Canastas = []canasta.Canasta{
			{Id: 1, Rank: canasta.Four, Cards: makeCards(1, canasta.Four, 7), Natural: true},
			{Id: 2, Rank: canasta.Five, Cards: makeCards(101, canasta.Five, 7), Natural: false},
			{Id: 3, Rank: canasta.Wild, Cards: makeCards(301, canasta.Two, 7), Natural: false},
		}
		p.Team.Melds = []canasta.Meld{{Id: 100, Rank: canasta.Queen, Cards: makeCards(400, canasta.Queen, 6)}}
		p.Hand = canasta.PlayerHand{
			500: {500, canasta.Clubs, canasta.King},
			501: {501, canasta.Diamonds, canasta.Queen},
		}

		err := g.AddToMeld(p, []int{501}, 100)

		if err == nil {
			t.Error("expected AddToMeld to still be rejected — completing a second natural canasta doesn't help")
		}
		if len(p.Hand) != 2 {
			t.Error("hand should be untouched by a rejected add")
		}
	})

	t.Run("AddToMeld still rejects leaving zero cards, even completing the last required type", func(t *testing.T) {
		g := canasta.NewGame("ABCE", []string{"A", "B", "C", "D"})
		p := g.Players[0]
		p.Team.GoneDown = true
		p.Team.Canastas = []canasta.Canasta{
			{Id: 1, Rank: canasta.Four, Cards: makeCards(1, canasta.Four, 7), Natural: true},
			{Id: 2, Rank: canasta.Seven, Cards: makeCards(201, canasta.Seven, 7), Natural: true},
			{Id: 3, Rank: canasta.Wild, Cards: makeCards(301, canasta.Two, 7), Natural: false},
		}
		p.Team.Melds = []canasta.Meld{{Id: 100, Rank: canasta.Queen, Cards: makeCards(400, canasta.Queen, 6)}}
		// Only the one wildcard in hand — playing it leaves 0 cards, not 1.
		p.Hand = canasta.PlayerHand{
			501: {501, canasta.Wild, canasta.Two},
		}

		err := g.AddToMeld(p, []int{501}, 100)

		if err == nil {
			t.Error("expected AddToMeld to be rejected — would leave zero cards, not one")
		}
	})

	t.Run("AddToMeld still rejects a completing-size staging meld before going down", func(t *testing.T) {
		g := canasta.NewGame("ABCE", []string{"A", "B", "C", "D"})
		p := g.Players[0]
		// Not gone down — Team.Canastas can't exist yet, and even a
		// 7-card staging meld doesn't immediately become a canasta.
		p.StagingMelds = []canasta.Meld{{Id: 100, Rank: canasta.Queen, Cards: makeCards(400, canasta.Queen, 6)}}
		p.Hand = canasta.PlayerHand{
			500: {500, canasta.Clubs, canasta.King},
			501: {501, canasta.Wild, canasta.Two},
		}

		err := g.AddToMeld(p, []int{501}, 100)

		if err == nil {
			t.Error("expected AddToMeld to be rejected — staging melds never immediately complete a canasta")
		}
	})

	t.Run("NewMeld succeeds when it completes the team's last required canasta type, leaving exactly one card", func(t *testing.T) {
		g := canasta.NewGame("ABCE", []string{"A", "B", "C", "D"})
		p := g.Players[0]
		p.Team.GoneDown = true
		// Already covers natural, unnatural, and wildcards — only
		// sevens is missing.
		p.Team.Canastas = []canasta.Canasta{
			{Id: 1, Rank: canasta.Four, Cards: makeCards(1, canasta.Four, 7), Natural: true},
			{Id: 2, Rank: canasta.Five, Cards: makeCards(101, canasta.Five, 7), Natural: false},
			{Id: 3, Rank: canasta.Wild, Cards: makeCards(301, canasta.Two, 7), Natural: false},
		}
		sevens := makeCards(500, canasta.Seven, 7)
		hand := canasta.PlayerHand{}
		var cardIds []int
		for _, c := range sevens {
			hand[c.GetId()] = c
			cardIds = append(cardIds, c.GetId())
		}
		hand[600] = canasta.Card{600, canasta.Clubs, canasta.King} // spare card left over
		p.Hand = hand

		err := g.NewMeld(p, cardIds)

		if err != nil {
			t.Errorf("expected NewMeld to succeed, completing the last required canasta type: %v", err)
		}
		if len(p.Hand) != 1 {
			t.Errorf("expected exactly 1 card left in hand, got %d", len(p.Hand))
		}
	})

	t.Run("PickUpDiscardPile succeeds when it completes the team's last required canasta type, leaving exactly one card", func(t *testing.T) {
		g := canasta.NewGame("ABCE", []string{"A", "B", "C", "D"})
		p := g.Players[0]
		p.Team.GoneDown = true
		// Already covers natural, unnatural, and sevens — only
		// wildcards is missing.
		p.Team.Canastas = []canasta.Canasta{
			{Id: 1, Rank: canasta.Four, Cards: makeCards(1, canasta.Four, 7), Natural: true},
			{Id: 2, Rank: canasta.Five, Cards: makeCards(101, canasta.Five, 7), Natural: false},
			{Id: 3, Rank: canasta.Seven, Cards: makeCards(201, canasta.Seven, 7), Natural: true},
		}
		twos := makeCards(500, canasta.Two, 6)
		hand := canasta.PlayerHand{}
		var cardIds []int
		for _, c := range twos {
			hand[c.GetId()] = c
			cardIds = append(cardIds, c.GetId())
		}
		hand[600] = canasta.Card{600, canasta.Clubs, canasta.King} // spare card left over
		p.Hand = hand
		g.Hand.DiscardPile = []canasta.Card{{0, canasta.Wild, canasta.Joker}}

		err := g.PickUpDiscardPile(p, cardIds)

		if err != nil {
			t.Errorf("expected PickUpDiscardPile to succeed, completing the last required canasta type: %v", err)
		}
		if len(p.Hand) != 1 {
			t.Errorf("expected exactly 1 card left in hand, got %d", len(p.Hand))
		}
	})
}

func TestGrantPermissionToGoOut(t *testing.T) {
	t.Run("team has not gone down", func(t *testing.T) {
		g := canasta.NewGame("ABCE", []string{"A", "B", "C", "D"})
		player := g.Players[0]

		err := g.GrantPermissionToGoOut(player)

		if err == nil {
			t.Error("Expected an error when team has not gone down")
		}
		if player.Team.CanGoOut {
			t.Error("CanGoOut should not have been set")
		}
	})

	t.Run("team has gone down but has no canastas yet", func(t *testing.T) {
		g := canasta.NewGame("ABCE", []string{"A", "B", "C", "D"})
		player := g.Players[0]
		player.Team.GoneDown = true

		err := g.GrantPermissionToGoOut(player)

		if err == nil {
			t.Error("expected an error when the team has no canastas at all")
		}
		if player.Team.CanGoOut {
			t.Error("CanGoOut should not have been set")
		}
	})

	// A natural or unnatural canasta at a regular rank, a sevens canasta,
	// and a wildcards canasta — the four buckets MeetsGoOutRequirements
	// checks. Missing any one of the four should still fail.
	naturalCanasta := canasta.Canasta{Id: 1, Rank: canasta.Four, Cards: makeCards(1, canasta.Four, 7), Natural: true}
	unnaturalCanasta := canasta.Canasta{Id: 2, Rank: canasta.Five, Cards: makeCards(101, canasta.Five, 7), Natural: false}
	sevensCanasta := canasta.Canasta{Id: 3, Rank: canasta.Seven, Cards: makeCards(201, canasta.Seven, 7), Natural: true}
	wildcardsCanasta := canasta.Canasta{Id: 4, Rank: canasta.Wild, Cards: makeCards(301, canasta.Two, 7), Natural: false}
	allFour := []canasta.Canasta{naturalCanasta, unnaturalCanasta, sevensCanasta, wildcardsCanasta}

	missingCases := []struct {
		name     string
		canastas []canasta.Canasta
	}{
		{"missing the natural canasta", []canasta.Canasta{unnaturalCanasta, sevensCanasta, wildcardsCanasta}},
		{"missing the unnatural canasta", []canasta.Canasta{naturalCanasta, sevensCanasta, wildcardsCanasta}},
		{"missing the sevens canasta", []canasta.Canasta{naturalCanasta, unnaturalCanasta, wildcardsCanasta}},
		{"missing the wildcards canasta", []canasta.Canasta{naturalCanasta, unnaturalCanasta, sevensCanasta}},
		// A sevens canasta is technically "natural" (zero wildcards) but
		// must not double as satisfying the generic natural requirement.
		{"two sevens canastas do not substitute for a natural one", []canasta.Canasta{sevensCanasta, unnaturalCanasta, wildcardsCanasta}},
	}
	for _, tt := range missingCases {
		t.Run(tt.name, func(t *testing.T) {
			g := canasta.NewGame("ABCE", []string{"A", "B", "C", "D"})
			player := g.Players[0]
			player.Team.GoneDown = true
			player.Team.Canastas = tt.canastas

			err := g.GrantPermissionToGoOut(player)

			if err == nil {
				t.Error("expected an error when a required canasta type is missing")
			}
			if player.Team.CanGoOut {
				t.Error("CanGoOut should not have been set")
			}
		})
	}

	t.Run("team has gone down with all four required canasta types", func(t *testing.T) {
		g := canasta.NewGame("ABCE", []string{"A", "B", "C", "D"})
		player := g.Players[0]
		player.Team.GoneDown = true
		player.Team.Canastas = allFour

		err := g.GrantPermissionToGoOut(player)

		if err != nil {
			t.Error(err)
		}
		if !player.Team.CanGoOut {
			t.Error("Expected CanGoOut to be set to true")
		}
	})
}

// makeCards builds n cards of the given rank starting at startId, cycling
// through suits — good enough for tests that only care about rank/count.
func makeCards(startId int, rank canasta.Rank, n int) []canasta.Card {
	suits := []canasta.Suit{canasta.Hearts, canasta.Diamonds, canasta.Clubs, canasta.Spades}
	cards := make([]canasta.Card, n)
	for i := range n {
		cards[i] = canasta.Card{Id: startId + i, Suit: suits[i%len(suits)], Rank: rank}
	}
	return cards
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
		{
			name: "cannot discard a red three",
			hand: []canasta.Card{
				{0, canasta.Clubs, canasta.Ace},
				{1, canasta.Hearts, canasta.Three},
			},
			discardedCard: 1,
			canGoOut:      false,
			valid:         false,
		},
		{
			name: "cannot discard a red three even with permission to go out",
			hand: []canasta.Card{
				{1, canasta.Diamonds, canasta.Three},
			},
			discardedCard: 1,
			canGoOut:      true,
			valid:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			game := canasta.NewGame("ABCE", []string{"A", "B", "C", "D"})

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
