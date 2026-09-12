package protocol_test

import (
	"encoding/json"
	"testing"

	"canasta-server/internal/canasta"
	"canasta-server/internal/protocol"
)

func TestNewStateMessage(t *testing.T) {
	g := canasta.NewGame("ABCE", []string{"A", "B", "C", "D"}, canasta.WithFixedTeamOrder())
	g.NewHand()
	g.CurrentPlayer = 2

	msg := protocol.NewStateMessage(&g, 2)

	if !msg.IsYourTurn {
		t.Error("expected IsYourTurn to be true for the current player")
	}
	if msg.SeatIndex != 2 {
		t.Errorf("expected SeatIndex 2, got %d", msg.SeatIndex)
	}
	if msg.CurrentPlayer != 2 {
		t.Errorf("expected CurrentPlayer 2, got %d", msg.CurrentPlayer)
	}
	if msg.Phase != canasta.PhaseDrawing {
		t.Errorf("expected phase %q, got %q", canasta.PhaseDrawing, msg.Phase)
	}
	if msg.Name != g.Players[2].Name {
		t.Errorf("expected embedded ClientState.Name %q, got %q", g.Players[2].Name, msg.Name)
	}

	otherMsg := protocol.NewStateMessage(&g, 0)
	if otherMsg.IsYourTurn {
		t.Error("expected IsYourTurn to be false for a non-current player")
	}

	// The embedded canasta.ClientState fields must flatten into the JSON
	// object rather than nesting under a "ClientState" key.
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("failed to marshal StateMessage: %v", err)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("failed to unmarshal StateMessage: %v", err)
	}
	for _, field := range []string{"deckCount", "hand", "seatIndex", "isYourTurn", "phase"} {
		if _, ok := raw[field]; !ok {
			t.Errorf("expected flattened field %q in JSON output, got keys %v", field, keys(raw))
		}
	}
}

func keys(m map[string]any) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	return ks
}
