package protocol

import "strings"

// ErrorCode identifies why a command was rejected.
type ErrorCode string

const (
	// Server-native codes, produced outside internal/canasta. Note:
	// there's no code here for a bad room code, a missing/blank name, or a
	// full room — those are all rejected as plain HTTP errors before a
	// websocket connection (and thus any ErrorPayload) can exist at all;
	// see internal/server/ws_handler.go.
	ErrNotYourTurn    ErrorCode = "NOT_YOUR_TURN"
	ErrWrongPhase     ErrorCode = "WRONG_PHASE"
	ErrNotPartner     ErrorCode = "NOT_PARTNER"
	ErrRoomNotPlaying ErrorCode = "ROOM_NOT_PLAYING"
	ErrUnknownType    ErrorCode = "UNKNOWN_MESSAGE_TYPE"
	ErrInvalidPayload ErrorCode = "INVALID_PAYLOAD"

	// Fallback for any internal/canasta error that doesn't follow the
	// "CODE: message" convention.
	ErrValidation ErrorCode = "VALIDATION_ERROR"
)

// ClassifyGameError splits an internal/canasta error of the form
// "CODE: message" into its typed code and remaining message. Errors that
// don't follow the convention are classified as ErrValidation, carrying
// their full text as the message.
func ClassifyGameError(err error) (ErrorCode, string) {
	if err == nil {
		return "", ""
	}

	msg := err.Error()
	if idx := strings.Index(msg, ": "); idx > 0 && isShoutingCode(msg[:idx]) {
		return ErrorCode(msg[:idx]), msg[idx+2:]
	}

	return ErrValidation, msg
}

// isShoutingCode reports whether s looks like a "CODE"-style prefix:
// non-empty, upper-case letters and underscores only.
func isShoutingCode(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r == '_' {
			continue
		}
		if r < 'A' || r > 'Z' {
			return false
		}
	}
	return true
}
