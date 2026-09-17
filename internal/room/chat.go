package room

import (
	"encoding/json"
	"strings"

	"canasta-server/internal/protocol"
)

// maxChatMessageLength caps a single chat message, mirroring
// maxNameLength's role for player names in room.go.
const maxChatMessageLength = 500

// applyChatMessage relays a chat message to every other connected seat.
// Unlike every other command it never touches r.game and runs
// regardless of room state (Lobby or Playing) — see handleCommand's
// early special-case, which calls this before the Lobby/Playing routing
// every other command is subject to. seatIdx is the sender's raw
// connection slot (matching PlayerStatusPayload/LobbySeat.SeatIndex's
// convention), not their table position.
func (r *Room) applyChatMessage(seatIdx int, msg protocol.ClientMessage) {
	var payload protocol.ChatMessagePayload
	if err := json.Unmarshal(msg.Data, &payload); err != nil {
		r.sendError(seatIdx, protocol.ErrInvalidPayload, "could not parse chat payload")
		return
	}

	text := strings.TrimSpace(payload.Text)
	if text == "" {
		return // silently drop empty/whitespace-only messages
	}
	if len(text) > maxChatMessageLength {
		r.sendError(seatIdx, protocol.ErrValidation, "message is too long")
		return
	}

	r.broadcastExcept(seatIdx, protocol.NewServerMessage(protocol.TypeChatMessage, protocol.ChatBroadcastPayload{
		SeatIndex: seatIdx,
		Name:      r.seats[seatIdx].Name,
		Text:      text,
	}))
}
