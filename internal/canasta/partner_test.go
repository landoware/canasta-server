package canasta

import "testing"

// TestPartnerPermissionFlow tests the complete partner permission workflow
func TestPartnerPermissionFlow(t *testing.T) {
	// Setup game with all 4 canastas
	game := NewGame("test", []string{"Alice", "Bob", "Carol", "Dave"})
	player0 := game.Players[0] // Alice (Team A)
	player1 := game.Players[1] // Bob (Team B)
	player2 := game.Players[2] // Carol (Team A, partner of Alice)

	// Give Team A all required canastas
	player0.Team.Canastas = append(player0.Team.Canastas,
		Canasta{Rank: Wild, Count: 7, Natural: false}, // Wildcards
		Canasta{Rank: Seven, Count: 7, Natural: true}, // Natural Sevens (counts as both)
		Canasta{Rank: King, Count: 7, Natural: false}, // Unnatural
	)

	// Set current player to Alice
	game.CurrentPlayer = 0
	game.Phase = PhasePlaying

	// Test 1: Alice asks permission
	err := game.MoveAskToGoOut(player0)
	if err != nil {
		t.Fatalf("Ask permission should succeed: %v", err)
	}

	// Verify request state
	if !game.GoOutRequestPending {
		t.Error("GoOutRequestPending should be true")
	}
	if game.GoOutRequester != 0 {
		t.Errorf("GoOutRequester should be 0, got %d", game.GoOutRequester)
	}
	if game.GoOutPartner != 2 {
		t.Errorf("GoOutPartner should be 2 (Carol), got %d", game.GoOutPartner)
	}

	// Test 2: Can't ask while request pending
	err = game.MoveAskToGoOut(player1)
	if err == nil {
		t.Error("Should not allow second request while one is pending")
	}

	// Test 3: Wrong person tries to respond
	err = game.RespondToGoOut(player1, true)
	if err == nil {
		t.Error("Should not allow non-partner to respond")
	}

	// Test 4: Carol (partner) approves
	err = game.RespondToGoOut(player2, true)
	if err != nil {
		t.Fatalf("Partner response should succeed: %v", err)
	}

	// Verify state cleared
	if game.GoOutRequestPending {
		t.Error("GoOutRequestPending should be false after response")
	}
	if game.GoOutRequester != -1 {
		t.Errorf("GoOutRequester should be -1, got %d", game.GoOutRequester)
	}
	if game.GoOutPartner != -1 {
		t.Errorf("GoOutPartner should be -1, got %d", game.GoOutPartner)
	}

	// Verify permission granted
	if !player0.Team.CanGoOut {
		t.Error("CanGoOut should be true after approval")
	}
}

// TestPartnerPermissionDenied tests denial flow
func TestPartnerPermissionDenied(t *testing.T) {
	game := NewGame("test", []string{"Alice", "Bob", "Carol", "Dave"})
	player0 := game.Players[0] // Alice
	player2 := game.Players[2] // Carol (partner)

	// Give all canastas
	player0.Team.Canastas = append(player0.Team.Canastas,
		Canasta{Rank: Wild, Count: 7, Natural: false},
		Canasta{Rank: Seven, Count: 7, Natural: true},
		Canasta{Rank: King, Count: 7, Natural: false},
	)

	game.CurrentPlayer = 0
	game.Phase = PhasePlaying

	// Alice asks
	err := game.MoveAskToGoOut(player0)
	if err != nil {
		t.Fatalf("Ask should succeed: %v", err)
	}

	// Carol denies
	err = game.RespondToGoOut(player2, false)
	if err != nil {
		t.Fatalf("Denial should succeed: %v", err)
	}

	// Verify permission NOT granted
	if player0.Team.CanGoOut {
		t.Error("CanGoOut should be false after denial")
	}

	// Verify state cleared
	if game.GoOutRequestPending {
		t.Error("Request should be cleared after response")
	}
}

// TestRespondWithoutRequest tests responding when no request exists
func TestRespondWithoutRequest(t *testing.T) {
	game := NewGame("test", []string{"Alice", "Bob", "Carol", "Dave"})
	player2 := game.Players[2]

	err := game.RespondToGoOut(player2, true)
	if err == nil {
		t.Error("Should not allow response when no request pending")
	}
	if err.Error() != "NO_REQUEST: No go-out request is pending" {
		t.Errorf("Wrong error message: %v", err)
	}
}

// TestAskWithoutAllCanastas tests asking without required canastas
func TestAskWithoutAllCanastas(t *testing.T) {
	game := NewGame("test", []string{"Alice", "Bob", "Carol", "Dave"})
	player0 := game.Players[0]

	// Only give 2 canastas (missing natural and unnatural)
	player0.Team.Canastas = append(player0.Team.Canastas,
		Canasta{Rank: Wild, Count: 7, Natural: false},
		Canasta{Rank: Seven, Count: 7, Natural: false}, // Mixed sevens
	)

	game.CurrentPlayer = 0
	game.Phase = PhasePlaying

	err := game.MoveAskToGoOut(player0)
	if err == nil {
		t.Error("Should not allow asking without all canastas")
	}

	// Should not have created a pending request
	if game.GoOutRequestPending {
		t.Error("Should not set pending request when validation fails")
	}
}

// TestExecuteMoveAskToGoOut tests the handler integration
func TestExecuteMoveAskToGoOut(t *testing.T) {
	game := NewGame("test", []string{"Alice", "Bob", "Carol", "Dave"})
	player0 := game.Players[0]

	// Setup canastas
	player0.Team.Canastas = append(player0.Team.Canastas,
		Canasta{Rank: Wild, Count: 7, Natural: false},
		Canasta{Rank: Seven, Count: 7, Natural: true},
		Canasta{Rank: King, Count: 7, Natural: false},
	)

	game.CurrentPlayer = 0
	game.Phase = PhasePlaying

	// Execute ask_to_go_out move
	move := Move{
		PlayerId: 0,
		Type:     MoveAskToGoOut,
	}

	response := game.ExecuteMove(move)
	if !response.Success {
		t.Errorf("Move should succeed: %v", response.Message)
	}

	if !game.GoOutRequestPending {
		t.Error("Request should be pending after move")
	}
}

// TestExecuteMoveRespondGoOut tests the response handler
func TestExecuteMoveRespondGoOut(t *testing.T) {
	game := NewGame("test", []string{"Alice", "Bob", "Carol", "Dave"})
	player0 := game.Players[0]
	player2 := game.Players[2]

	// Setup
	player0.Team.Canastas = append(player0.Team.Canastas,
		Canasta{Rank: Wild, Count: 7, Natural: false},
		Canasta{Rank: Seven, Count: 7, Natural: true},
		Canasta{Rank: King, Count: 7, Natural: false},
	)

	game.CurrentPlayer = 0
	game.Phase = PhasePlaying

	// Alice asks
	game.MoveAskToGoOut(player0)

	// Change current player (partner's turn or doesn't matter)
	game.CurrentPlayer = 2

	// Carol responds with approval (Id=1 means approve)
	move := Move{
		PlayerId: 2,
		Type:     MoveRespondGoOut,
		Id:       1, // 1 = approve, 0 = deny
	}

	response := game.ExecuteMove(move)
	if !response.Success {
		t.Errorf("Response should succeed: %v", response.Message)
	}

	if !player0.Team.CanGoOut {
		t.Error("CanGoOut should be true after approval")
	}

	// Test denial - reset state
	game.GoOutRequestPending = true
	game.GoOutRequester = 0
	game.GoOutPartner = 2
	player0.Team.CanGoOut = false
	player2.Team.CanGoOut = false // Also reset partner team

	move.Id = 0 // Deny
	response = game.ExecuteMove(move)
	if !response.Success {
		t.Errorf("Denial should succeed: %v", response.Message)
	}

	if player0.Team.CanGoOut {
		t.Error("CanGoOut should be false after denial")
	}
}
