package canasta

import "testing"

// TestMoveAskToGoOut tests the canasta validation when asking to go out
func TestMoveAskToGoOut(t *testing.T) {
	tests := []struct {
		name        string
		setupTeam   func(*Team)
		expectError string
	}{
		{
			name: "All four canastas present - success",
			setupTeam: func(team *Team) {
				// Wildcards canasta
				team.Canastas = append(team.Canastas, Canasta{Rank: Wild, Count: 7, Natural: false})
				// Sevens canasta
				team.Canastas = append(team.Canastas, Canasta{Rank: Seven, Count: 7, Natural: true})
				// Natural canasta (any rank)
				team.Canastas = append(team.Canastas, Canasta{Rank: Queen, Count: 7, Natural: true})
				// Unnatural canasta (mixed with wilds)
				team.Canastas = append(team.Canastas, Canasta{Rank: King, Count: 7, Natural: false})
			},
			expectError: "",
		},
		{
			name: "Missing wildcards canasta",
			setupTeam: func(team *Team) {
				team.Canastas = append(team.Canastas, Canasta{Rank: Seven, Count: 7, Natural: true})
				team.Canastas = append(team.Canastas, Canasta{Rank: Queen, Count: 7, Natural: true})
				team.Canastas = append(team.Canastas, Canasta{Rank: King, Count: 7, Natural: false})
			},
			expectError: "MISSING_CANASTA: Team needs a Wildcards canasta",
		},
		{
			name: "Missing sevens canasta",
			setupTeam: func(team *Team) {
				team.Canastas = append(team.Canastas, Canasta{Rank: Wild, Count: 7, Natural: false})
				team.Canastas = append(team.Canastas, Canasta{Rank: Queen, Count: 7, Natural: true})
				team.Canastas = append(team.Canastas, Canasta{Rank: King, Count: 7, Natural: false})
			},
			expectError: "MISSING_CANASTA: Team needs a Sevens canasta",
		},
		{
			name: "Missing natural canasta - mixed sevens doesn't count",
			setupTeam: func(team *Team) {
				team.Canastas = append(team.Canastas, Canasta{Rank: Wild, Count: 7, Natural: false})
				team.Canastas = append(team.Canastas, Canasta{Rank: Seven, Count: 7, Natural: false}) // Mixed sevens, not natural
				team.Canastas = append(team.Canastas, Canasta{Rank: King, Count: 7, Natural: false})
			},
			expectError: "MISSING_CANASTA: Team needs a Natural canasta",
		},
		{
			name: "Missing unnatural canasta",
			setupTeam: func(team *Team) {
				team.Canastas = append(team.Canastas, Canasta{Rank: Wild, Count: 7, Natural: false})
				team.Canastas = append(team.Canastas, Canasta{Rank: Seven, Count: 7, Natural: true})
				team.Canastas = append(team.Canastas, Canasta{Rank: Queen, Count: 7, Natural: true})
			},
			expectError: "MISSING_CANASTA: Team needs an Unnatural/Mixed canasta",
		},
		{
			name: "Only wildcards and sevens - missing unnatural (natural sevens covers natural requirement)",
			setupTeam: func(team *Team) {
				team.Canastas = append(team.Canastas, Canasta{Rank: Wild, Count: 7, Natural: false})
				team.Canastas = append(team.Canastas, Canasta{Rank: Seven, Count: 7, Natural: true}) // Covers both sevens AND natural
			},
			expectError: "MISSING_CANASTA: Team needs an Unnatural/Mixed canasta",
		},
		{
			name: "Natural seven counts as both sevens AND natural",
			setupTeam: func(team *Team) {
				team.Canastas = append(team.Canastas, Canasta{Rank: Wild, Count: 7, Natural: false})
				// Natural sevens canasta - counts as both requirements
				team.Canastas = append(team.Canastas, Canasta{Rank: Seven, Count: 7, Natural: true})
				team.Canastas = append(team.Canastas, Canasta{Rank: King, Count: 7, Natural: false})
			},
			expectError: "", // Should succeed - sevens canasta is also natural
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup game
			game := NewGame("test", []string{"Alice", "Bob", "Carol", "Dave"})
			player := game.Players[0]

			// Setup team with canastas
			tt.setupTeam(player.Team)

			// Try to ask to go out
			err := game.MoveAskToGoOut(player)

			if tt.expectError == "" {
				// Should succeed - creates permission request
				if err != nil {
					t.Errorf("Expected success, got error: %v", err)
				}
				// Check that permission request was created
				if !game.GoOutRequestPending {
					t.Error("GoOutRequestPending should be true after successful ask")
				}
				// CanGoOut is NOT set until partner approves
				// This is correct behavior with partner permission flow
			} else {
				// Should fail
				if err == nil {
					t.Errorf("Expected error containing '%s', got nil", tt.expectError)
				} else if err.Error() != tt.expectError && !contains(err.Error(), tt.expectError) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.expectError, err.Error())
				}
			}
		})
	}
}

// Helper function for substring matching
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr
}
