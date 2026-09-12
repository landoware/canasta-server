package protocol_test

import (
	"errors"
	"testing"

	"canasta-server/internal/protocol"
)

func TestClassifyGameError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		expectCode protocol.ErrorCode
		expectMsg  string
	}{
		{
			name:       "nil error",
			err:        nil,
			expectCode: "",
			expectMsg:  "",
		},
		{
			name:       "prefixed error",
			err:        errors.New("PILE_FROZEN: Cannot pickup the pile with a black three on top"),
			expectCode: "PILE_FROZEN",
			expectMsg:  "Cannot pickup the pile with a black three on top",
		},
		{
			name:       "prefixed error with underscores",
			err:        errors.New("GO_DOWN_REQUIREMENT_NOT_MET: Cannot go down with fewer than 50 points."),
			expectCode: "GO_DOWN_REQUIREMENT_NOT_MET",
			expectMsg:  "Cannot go down with fewer than 50 points.",
		},
		{
			name:       "unprefixed error falls back to validation",
			err:        errors.New("something went wrong"),
			expectCode: protocol.ErrValidation,
			expectMsg:  "something went wrong",
		},
		{
			name:       "colon without shouting-case prefix is not treated as a code",
			err:        errors.New("ratio: 3 to 1 favors team A"),
			expectCode: protocol.ErrValidation,
			expectMsg:  "ratio: 3 to 1 favors team A",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, msg := protocol.ClassifyGameError(tt.err)
			if code != tt.expectCode {
				t.Errorf("expected code %q, got %q", tt.expectCode, code)
			}
			if msg != tt.expectMsg {
				t.Errorf("expected message %q, got %q", tt.expectMsg, msg)
			}
		})
	}
}
