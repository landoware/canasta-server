package room

import (
	"encoding/json"

	"canasta-server/internal/protocol"
)

// handleLobbyCommand dispatches a commandEvent received while the room is
// still in the Lobby state. Kept separate from dispatch.go's applyCommand,
// which is entirely oriented around in-game turn/phase rules.
func (r *Room) handleLobbyCommand(e commandEvent) {
	switch e.msg.Type {
	case protocol.TypeSetReady:
		r.applySetReady(e.seatIndex, e.msg.Data)
	case protocol.TypeReorderSeats:
		r.applyReorderSeats(e.seatIndex, e.msg.Data)
	case protocol.TypeStartGame:
		r.applyStartGame(e.seatIndex)
	default:
		r.sendError(e.seatIndex, protocol.ErrUnknownType, "unknown lobby command: "+string(e.msg.Type))
	}
}

func (r *Room) applySetReady(seatIdx int, data json.RawMessage) {
	var payload protocol.SetReadyPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		r.sendError(seatIdx, protocol.ErrInvalidPayload, "could not parse set_ready payload")
		return
	}
	r.seats[seatIdx].Ready = payload.Ready
	r.broadcastLobby()
}

// applyReorderSeats lets the host rearrange table order before the game
// starts. Order[pos] is the connection slot that should occupy table
// position pos — see posForConnSlot/startGame for where tableOrder is
// consumed.
func (r *Room) applyReorderSeats(seatIdx int, data json.RawMessage) {
	if !r.seats[seatIdx].IsHost {
		r.sendError(seatIdx, protocol.ErrNotHost, "only the host can reorder seats")
		return
	}

	var payload protocol.ReorderSeatsPayload
	if err := json.Unmarshal(data, &payload); err != nil || !isPermutationOfFour(payload.Order) {
		r.sendError(seatIdx, protocol.ErrInvalidPayload, "order must be a permutation of the 4 seat indices")
		return
	}

	r.tableOrder = [4]int{payload.Order[0], payload.Order[1], payload.Order[2], payload.Order[3]}
	r.broadcastLobby()
}

func (r *Room) applyStartGame(seatIdx int) {
	if !r.seats[seatIdx].IsHost {
		r.sendError(seatIdx, protocol.ErrNotHost, "only the host can start the game")
		return
	}
	if !r.allSeatsNamed() {
		r.sendError(seatIdx, protocol.ErrSeatsNotFull, "all four seats must be filled")
		return
	}
	if !r.allSeatsReady() {
		r.sendError(seatIdx, protocol.ErrNotAllReady, "all players must be ready")
		return
	}
	r.startGame()
}

func isPermutationOfFour(order []int) bool {
	if len(order) != 4 {
		return false
	}
	var seen [4]bool
	for _, v := range order {
		if v < 0 || v > 3 || seen[v] {
			return false
		}
		seen[v] = true
	}
	return true
}
