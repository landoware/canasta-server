package canasta

import "testing"

// TestPlayRedThreeFromHand tests playing red threes from initial hand with replacement draw
func TestPlayRedThreeFromHand(t *testing.T) {
	game := NewGame("test", []string{"Alice", "Bob", "Carol", "Dave"})
	game.Deal()
	player := game.Players[0]

	// Give player 2 red threes in hand
	redThree1 := Card{Id: 1001, Rank: Three, Suit: Hearts}
	redThree2 := Card{Id: 1002, Rank: Three, Suit: Diamonds}
	player.Hand[1001] = redThree1
	player.Hand[1002] = redThree2

	initialHandSize := len(player.Hand)
	initialDeckSize := game.Hand.Deck.Count()
	initialRedThreeCount := len(player.Team.RedThrees)

	// Play both red threes (from hand, not foot)
	err := game.PlayRedThree(player, []int{1001, 1002}, false)
	if err != nil {
		t.Fatalf("Playing red threes should succeed: %v", err)
	}

	// Verify red threes moved to team pile
	if len(player.Team.RedThrees) != initialRedThreeCount+2 {
		t.Errorf("Expected %d red threes on team pile, got %d",
			initialRedThreeCount+2, len(player.Team.RedThrees))
	}

	// Verify red threes removed from hand
	if _, exists := player.Hand[1001]; exists {
		t.Error("Red three 1001 should be removed from hand")
	}
	if _, exists := player.Hand[1002]; exists {
		t.Error("Red three 1002 should be removed from hand")
	}

	// Verify replacement cards drawn (hand size unchanged)
	// Why unchanged: -2 red threes, +2 replacement = net 0
	if len(player.Hand) != initialHandSize {
		t.Errorf("Expected hand size %d, got %d (should draw 2 replacements)",
			initialHandSize, len(player.Hand))
	}

	// Verify deck decreased by 2
	if game.Hand.Deck.Count() != initialDeckSize-2 {
		t.Errorf("Expected deck size %d, got %d", initialDeckSize-2, game.Hand.Deck.Count())
	}

	// Verify still in drawing phase
	if game.Phase != PhaseDrawing {
		t.Error("Should still be in drawing phase after playing red threes")
	}
}

// TestPlayRedThreeFromFoot tests playing red threes from foot with NO replacement draw
func TestPlayRedThreeFromFoot(t *testing.T) {
	game := NewGame("test", []string{"Alice", "Bob", "Carol", "Dave"})
	game.Deal()
	player := game.Players[0]

	// Simulate player picking up foot with red threes
	// For this test, just put red threes in hand and mark as "from foot"
	redThree := Card{Id: 2001, Rank: Three, Suit: Hearts}
	player.Hand[2001] = redThree

	initialHandSize := len(player.Hand)
	initialDeckSize := game.Hand.Deck.Count()

	// Play red three from foot (fromFoot = true)
	err := game.PlayRedThree(player, []int{2001}, true)
	if err != nil {
		t.Fatalf("Playing foot red three should succeed: %v", err)
	}

	// Verify red three moved to team pile
	found := false
	for _, card := range player.Team.RedThrees {
		if card.Id == 2001 {
			found = true
			break
		}
	}
	if !found {
		t.Error("Red three should be in team pile")
	}

	// Verify NO replacement draw (hand size decreased by 1)
	if len(player.Hand) != initialHandSize-1 {
		t.Errorf("Expected hand size %d (no replacement), got %d",
			initialHandSize-1, len(player.Hand))
	}

	// Verify deck unchanged
	if game.Hand.Deck.Count() != initialDeckSize {
		t.Error("Deck should not change when playing foot red threes")
	}
}

// TestPlayRedThreeWrongPhase tests that red threes can only be played during drawing phase
func TestPlayRedThreeWrongPhase(t *testing.T) {
	game := NewGame("test", []string{"Alice", "Bob", "Carol", "Dave"})
	game.Deal()
	player := game.Players[0]

	// Give player a red three
	redThree := Card{Id: 3001, Rank: Three, Suit: Hearts}
	player.Hand[3001] = redThree

	// Change to playing phase
	game.Phase = PhasePlaying

	err := game.PlayRedThree(player, []int{3001}, false)
	if err == nil {
		t.Error("Should not allow playing red threes during playing phase")
	}
	if err.Error() != "WRONG_PHASE: Red threes must be played at start of turn (drawing phase)" {
		t.Errorf("Wrong error: %v", err)
	}
}

// TestPlayRedThreeInvalidCard tests trying to play non-red-three cards
func TestPlayRedThreeInvalidCard(t *testing.T) {
	game := NewGame("test", []string{"Alice", "Bob", "Carol", "Dave"})
	game.Deal()
	player := game.Players[0]

	// Give player a black three (not allowed)
	blackThree := Card{Id: 4001, Rank: Three, Suit: Spades}
	player.Hand[4001] = blackThree

	err := game.PlayRedThree(player, []int{4001}, false)
	if err == nil {
		t.Error("Should not allow playing black threes")
	}

	// Give player a normal card
	normalCard := Card{Id: 4002, Rank: Queen, Suit: Hearts}
	player.Hand[4002] = normalCard

	err = game.PlayRedThree(player, []int{4002}, false)
	if err == nil {
		t.Error("Should not allow playing non-three cards")
	}
}

// TestPlayRedThreeMultiple tests playing multiple red threes at once
func TestPlayRedThreeMultiple(t *testing.T) {
	game := NewGame("test", []string{"Alice", "Bob", "Carol", "Dave"})
	game.Deal()
	player := game.Players[0]

	// Give player 3 red threes
	player.Hand[5001] = Card{Id: 5001, Rank: Three, Suit: Hearts}
	player.Hand[5002] = Card{Id: 5002, Rank: Three, Suit: Diamonds}
	player.Hand[5003] = Card{Id: 5003, Rank: Three, Suit: Hearts}

	initialHandSize := len(player.Hand)

	// Play all 3 at once (from hand)
	err := game.PlayRedThree(player, []int{5001, 5002, 5003}, false)
	if err != nil {
		t.Fatalf("Should allow playing multiple red threes: %v", err)
	}

	// Verify all moved to pile
	if len(player.Team.RedThrees) < 3 {
		t.Error("All 3 red threes should be in team pile")
	}

	// Verify 3 replacements drawn (hand size unchanged)
	if len(player.Hand) != initialHandSize {
		t.Errorf("Expected hand size %d (3 replacements), got %d",
			initialHandSize, len(player.Hand))
	}
}

// TestPlayRedThreeMixed tests mix of hand and foot red threes
func TestPlayRedThreeMixed(t *testing.T) {
	// This documents that you can't mix - must specify fromFoot consistently
	game := NewGame("test", []string{"Alice", "Bob", "Carol", "Dave"})
	game.Deal()
	player := game.Players[0]

	// Give 2 red threes
	player.Hand[6001] = Card{Id: 6001, Rank: Three, Suit: Hearts}
	player.Hand[6002] = Card{Id: 6002, Rank: Three, Suit: Diamonds}

	// Play both with fromFoot=false (both from hand)
	err := game.PlayRedThree(player, []int{6001, 6002}, false)
	if err != nil {
		t.Fatalf("Should succeed: %v", err)
	}

	// Note: If player has mix of hand/foot red threes, they make two separate moves
	// - First: play_red_three with fromFoot=false for hand red threes
	// - Second: play_red_three with fromFoot=true for foot red threes
}

// TestExecuteMovePlayRedThree tests the handler integration
func TestExecuteMovePlayRedThree(t *testing.T) {
	game := NewGame("test", []string{"Alice", "Bob", "Carol", "Dave"})
	game.Deal()
	player := game.Players[0]

	// Give red three
	redThree := Card{Id: 7001, Rank: Three, Suit: Hearts}
	player.Hand[7001] = redThree

	// Set current player
	game.CurrentPlayer = 0
	game.Phase = PhaseDrawing

	move := Move{
		PlayerId: 0,
		Type:     MovePlayRedThree,
		Ids:      []int{7001},
		FromFoot: false,
	}

	response := game.ExecuteMove(move)
	if !response.Success {
		t.Errorf("Move should succeed: %v", response.Message)
	}

	// Verify red three was moved
	found := false
	for _, card := range player.Team.RedThrees {
		if card.Id == 7001 {
			found = true
			break
		}
	}
	if !found {
		t.Error("Red three should be in team pile")
	}
}
