package room

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"

	"canasta-server/internal/canasta"
	"canasta-server/internal/protocol"
)

// maxNameLength caps how long a player's display name may be.
const maxNameLength = 24

// State is the room's lifecycle stage.
type State string

const (
	StateLobby    State = "lobby"
	StatePlaying  State = "playing"
	StateFinished State = "finished"
)

// disconnectGraceForAway is how long a dropped connection gets before the
// other players are told the seat is "away". The seat's token remains
// valid for reconnection indefinitely after that — there is no bot or
// replacement-player mechanism, so a hard forfeit would strand the game.
const disconnectGraceForAway = 2 * time.Minute

// ErrNameRequired is returned by Join when the given name is empty (after
// trimming whitespace).
var ErrNameRequired = errors.New("a name is required to join")

// ErrRoomFull is returned by Join when the name doesn't match an existing
// seat and there's no open seat left to claim (all four are taken, or the
// game has already started).
var ErrRoomFull = errors.New("room is full")

// ErrRoomClosed is returned by Join if the room's actor has already
// stopped (e.g. reaped) before the join could be processed.
var ErrRoomClosed = errors.New("room is closed")

// Room is a single 4-player game and its four seats. One actor goroutine
// (Run) owns the *canasta.Game and the seat table; every other goroutine
// (websocket read loops, the reaper) only ever sends events into inbox.
type Room struct {
	Code string

	inbox chan any
	done  chan struct{}

	seats [4]*Seat
	state State
	game  *canasta.Game

	lastActivityUnix atomic.Int64
	connectedCount   atomic.Int32
}

// NewRoom allocates a room in the Lobby state with four unclaimed seats.
func NewRoom(code string) *Room {
	r := &Room{
		Code:  code,
		inbox: make(chan any, 32),
		done:  make(chan struct{}),
		state: StateLobby,
	}
	for i := range r.seats {
		r.seats[i] = &Seat{Index: i}
	}
	r.lastActivityUnix.Store(time.Now().Unix())
	return r
}

// LastActivity reports when the room's actor last processed an event.
// Safe to call from any goroutine (used by the reaper).
func (r *Room) LastActivity() time.Time {
	return time.Unix(r.lastActivityUnix.Load(), 0)
}

// ConnectedCount reports how many seats currently have a live connection.
// Safe to call from any goroutine (used by the reaper).
func (r *Room) ConnectedCount() int {
	return int(r.connectedCount.Load())
}

// Run processes room events until ctx is canceled. It is the only
// goroutine that ever touches r.game or r.seats.
func (r *Room) Run(ctx context.Context) {
	defer close(r.done)
	for {
		select {
		case ev := <-r.inbox:
			r.handle(ev)
			r.lastActivityUnix.Store(time.Now().Unix())
		case <-ctx.Done():
			return
		}
	}
}

func (r *Room) submit(ev any) {
	select {
	case r.inbox <- ev:
	case <-r.done:
	}
}

type joinEvent struct {
	name   string
	result chan joinResult
}

type joinResult struct {
	seatIndex int
	err       error
}

type attachEvent struct {
	seatIndex int
	conn      Conn
}

type commandEvent struct {
	seatIndex int
	msg       protocol.ClientMessage
}

type disconnectEvent struct {
	seatIndex int
	conn      Conn
}

type awayTimeoutEvent struct {
	seatIndex int
	conn      Conn
}

// Join resolves name to a seat index — an existing seat if the name
// (matched case-insensitively) is already taken, otherwise the next open
// seat if the room is still in the Lobby. It does not attach any
// connection; call Attach once a live Conn is ready (e.g. after a
// websocket upgrade succeeds), so a join failure can be reported as a
// plain HTTP status instead of a socket that opens and immediately
// closes. It blocks until the room's actor has processed the join.
func (r *Room) Join(name string) (seatIndex int, err error) {
	resultCh := make(chan joinResult, 1)
	r.submit(joinEvent{name: name, result: resultCh})
	select {
	case res := <-resultCh:
		return res.seatIndex, res.err
	case <-r.done:
		return -1, ErrRoomClosed
	}
}

// Attach wires a live connection to a seat index already returned by
// Join, closing any existing live connection on that seat first
// (last-writer-wins — a page refresh, a second tab, or a reconnect).
func (r *Room) Attach(seatIndex int, conn Conn) {
	r.submit(attachEvent{seatIndex: seatIndex, conn: conn})
}

// Submit forwards a decoded client message from seatIndex's read loop.
func (r *Room) Submit(seatIndex int, msg protocol.ClientMessage) {
	r.submit(commandEvent{seatIndex: seatIndex, msg: msg})
}

// Disconnect notifies the room that seatIndex's read loop has ended. conn
// identifies which connection dropped, so a stale notification for a
// connection that's already been replaced by a reconnect is ignored.
func (r *Room) Disconnect(seatIndex int, conn Conn) {
	r.submit(disconnectEvent{seatIndex: seatIndex, conn: conn})
}

func (r *Room) handle(ev any) {
	switch e := ev.(type) {
	case joinEvent:
		r.handleJoin(e)
	case attachEvent:
		r.handleAttach(e)
	case commandEvent:
		r.handleCommand(e)
	case disconnectEvent:
		r.handleDisconnect(e)
	case awayTimeoutEvent:
		r.handleAwayTimeout(e)
	case func():
		// A synchronization probe (used by tests to wait until every
		// previously submitted event has been processed by this actor).
		e()
	}
}

func (r *Room) handleJoin(e joinEvent) {
	name := normalizeName(e.name)
	if name == "" {
		e.result <- joinResult{-1, ErrNameRequired}
		return
	}

	idx := r.seatIndexForName(name)
	if idx == -1 {
		idx = r.claimOpenSeat(name)
		if idx == -1 {
			e.result <- joinResult{-1, ErrRoomFull}
			return
		}
		if r.allSeatsNamed() {
			r.startGame()
		}
	}

	e.result <- joinResult{idx, nil}
}

func (r *Room) handleAttach(e attachEvent) {
	seat := r.seats[e.seatIndex]
	wasConnected := seat.Connected
	if seat.Conn != nil && wasConnected {
		_ = seat.Conn.Close(websocket.StatusNormalClosure, "replaced by new connection")
	}

	seat.Conn = e.conn
	seat.Connected = true
	seat.DisconnectedAt = time.Time{}
	r.recalculateConnectedCount()

	r.sendTo(e.seatIndex, protocol.NewServerMessage(protocol.TypeWelcome, protocol.WelcomePayload{
		SeatIndex: e.seatIndex,
		RoomCode:  r.Code,
		RoomState: string(r.state),
	}))

	if r.state == StateLobby {
		r.broadcastLobby()
		return
	}

	if !wasConnected {
		r.broadcastExcept(e.seatIndex, protocol.NewServerMessage(protocol.TypePlayerReconnected, protocol.PlayerStatusPayload{SeatIndex: e.seatIndex, Status: "connected"}))
	}
	if r.game != nil {
		r.sendTo(e.seatIndex, protocol.NewServerMessage(protocol.TypeState, protocol.NewStateMessage(r.game, e.seatIndex)))
	}
}

func (r *Room) handleCommand(e commandEvent) {
	if r.state != StatePlaying {
		r.sendError(e.seatIndex, protocol.ErrRoomNotPlaying, "room is not currently playing")
		return
	}

	if errPayload := r.applyCommand(e.seatIndex, e.msg); errPayload != nil {
		r.sendTo(e.seatIndex, protocol.NewServerMessage(protocol.TypeError, *errPayload))
		return
	}

	r.broadcastState()

	if r.game.GameOver {
		r.state = StateFinished
	}
}

// startGame builds the canasta.Game once all four seats have named
// themselves. WithFixedTeamOrder is required here: NewGame's default
// random team order shuffles its playerNames slice in place, which would
// break the seat-index-equals-player-index invariant this room relies on
// throughout (GetClientState, CurrentPlayer, turn enforcement).
func (r *Room) startGame() {
	names := make([]string, 4)
	for i, s := range r.seats {
		names[i] = s.Name
	}

	game := canasta.NewGame(r.Code, names, canasta.WithFixedTeamOrder())
	game.NewHand()

	r.game = &game
	r.state = StatePlaying
	r.broadcastState()
}

func (r *Room) handleDisconnect(e disconnectEvent) {
	seat := r.seats[e.seatIndex]
	if seat.Conn != e.conn {
		// A newer connection already replaced this one; ignore the stale
		// disconnect notification from the old read loop.
		return
	}

	seat.Connected = false
	seat.DisconnectedAt = time.Now()
	r.recalculateConnectedCount()

	if r.state == StateLobby {
		r.broadcastLobby()
	} else {
		r.broadcastExcept(e.seatIndex, protocol.NewServerMessage(protocol.TypePlayerDisconnected, protocol.PlayerStatusPayload{SeatIndex: e.seatIndex, Status: "disconnected"}))
	}

	conn := e.conn
	time.AfterFunc(disconnectGraceForAway, func() {
		r.submit(awayTimeoutEvent{seatIndex: e.seatIndex, conn: conn})
	})
}

func (r *Room) handleAwayTimeout(e awayTimeoutEvent) {
	seat := r.seats[e.seatIndex]
	if seat.Conn != e.conn || seat.Connected {
		return // reconnected, or replaced by a newer connection, before the grace period elapsed
	}
	r.broadcastExcept(e.seatIndex, protocol.NewServerMessage(protocol.TypePlayerStatus, protocol.PlayerStatusPayload{SeatIndex: e.seatIndex, Status: "away"}))
}

// normalizeName trims whitespace and caps the length of a player-supplied
// name. Matching between names is case-insensitive (see seatIndexForName).
func normalizeName(raw string) string {
	name := strings.TrimSpace(raw)
	if len(name) > maxNameLength {
		name = name[:maxNameLength]
	}
	return name
}

func (r *Room) seatIndexForName(name string) int {
	for i, s := range r.seats {
		if s.Name != "" && strings.EqualFold(s.Name, name) {
			return i
		}
	}
	return -1
}

// claimOpenSeat assigns name to the next unclaimed seat, if the room is
// still in the Lobby and has one. Returns -1 if there's no seat to claim.
func (r *Room) claimOpenSeat(name string) int {
	if r.state != StateLobby {
		return -1
	}
	for i, s := range r.seats {
		if s.Name == "" {
			s.Name = name
			return i
		}
	}
	return -1
}

func (r *Room) allSeatsNamed() bool {
	for _, s := range r.seats {
		if s.Name == "" {
			return false
		}
	}
	return true
}

func (r *Room) recalculateConnectedCount() {
	n := 0
	for _, s := range r.seats {
		if s.Connected {
			n++
		}
	}
	r.connectedCount.Store(int32(n))
}

func (r *Room) sendTo(seatIdx int, msg protocol.ServerMessage) {
	seat := r.seats[seatIdx]
	if !seat.Connected || seat.Conn == nil {
		return
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := seat.Conn.Write(ctx, data); err != nil {
		seat.Connected = false
		seat.DisconnectedAt = time.Now()
		r.recalculateConnectedCount()
	}
}

func (r *Room) broadcastExcept(exceptSeat int, msg protocol.ServerMessage) {
	for i := range r.seats {
		if i == exceptSeat {
			continue
		}
		r.sendTo(i, msg)
	}
}

func (r *Room) broadcastState() {
	for i := range r.seats {
		r.sendTo(i, protocol.NewServerMessage(protocol.TypeState, protocol.NewStateMessage(r.game, i)))
	}
}

func (r *Room) broadcastLobby() {
	seats := make([]protocol.LobbySeat, len(r.seats))
	for i, s := range r.seats {
		seats[i] = protocol.LobbySeat{SeatIndex: i, Name: s.Name, Connected: s.Connected}
	}

	msg := protocol.NewServerMessage(protocol.TypePlayersLobby, protocol.PlayersLobbyPayload{Seats: seats})
	for i := range r.seats {
		r.sendTo(i, msg)
	}
}

func (r *Room) sendError(seatIdx int, code protocol.ErrorCode, message string) {
	r.sendTo(seatIdx, protocol.NewServerMessage(protocol.TypeError, protocol.ErrorPayload{Code: string(code), Message: message}))
}
