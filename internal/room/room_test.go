package room

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"

	"github.com/coder/websocket"

	"canasta-server/internal/canasta"
	"canasta-server/internal/protocol"
)

// fakeConn is an in-memory stand-in for a websocket connection, letting
// room tests run without any real socket.
type fakeConn struct {
	mu      sync.Mutex
	written []json.RawMessage
	closed  bool
}

func (f *fakeConn) Read(ctx context.Context) ([]byte, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

func (f *fakeConn) Write(_ context.Context, data []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.closed {
		return errors.New("write to closed conn")
	}
	cp := make(json.RawMessage, len(data))
	copy(cp, data)
	f.written = append(f.written, cp)
	return nil
}

func (f *fakeConn) Close(websocket.StatusCode, string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.closed = true
	return nil
}

func (f *fakeConn) isClosed() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.closed
}

func (f *fakeConn) messages(t *testing.T) []protocol.ServerMessage {
	t.Helper()
	f.mu.Lock()
	defer f.mu.Unlock()

	msgs := make([]protocol.ServerMessage, 0, len(f.written))
	for _, raw := range f.written {
		var msg protocol.ServerMessage
		if err := json.Unmarshal(raw, &msg); err != nil {
			t.Fatalf("failed to unmarshal server message: %v", err)
		}
		msgs = append(msgs, msg)
	}
	return msgs
}

// last returns the most recently written message of the given type.
func (f *fakeConn) last(t *testing.T, msgType protocol.MessageType) (protocol.ServerMessage, bool) {
	t.Helper()
	msgs := f.messages(t)
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Type == msgType {
			return msgs[i], true
		}
	}
	return protocol.ServerMessage{}, false
}

func decodeData[T any](t *testing.T, msg protocol.ServerMessage) T {
	t.Helper()
	raw, err := json.Marshal(msg.Data)
	if err != nil {
		t.Fatalf("failed to re-marshal message data: %v", err)
	}
	var v T
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatalf("failed to unmarshal message data: %v", err)
	}
	return v
}

// drain blocks until the room's actor has processed every event submitted
// before this call, giving tests a deterministic sync point against the
// actor's asynchronous goroutine.
func drain(r *Room) {
	done := make(chan struct{})
	r.submit(func() { close(done) })
	<-done
}

func startRoom(t *testing.T) *Room {
	t.Helper()
	r := NewRoom("TEST01")
	ctx, cancel := context.WithCancel(context.Background())
	go r.Run(ctx)
	t.Cleanup(cancel)
	return r
}

// joinAndAttach resolves name to a seat and wires up a fresh fakeConn,
// mirroring what the HTTP layer does across the pre-upgrade Join call and
// the post-upgrade Attach call.
func joinAndAttach(t *testing.T, r *Room, name string) (int, *fakeConn) {
	t.Helper()
	seatIdx, err := r.Join(name)
	if err != nil {
		t.Fatalf("join %q: unexpected error: %v", name, err)
	}
	conn := &fakeConn{}
	r.Attach(seatIdx, conn)
	return seatIdx, conn
}

// joinAll joins all four seats with the given names (in seat order) and
// drains the actor, returning each seat's fakeConn.
func joinAll(t *testing.T, r *Room, names [4]string) [4]*fakeConn {
	t.Helper()
	var conns [4]*fakeConn
	for i, name := range names {
		seatIdx, conn := joinAndAttach(t, r, name)
		if seatIdx != i {
			t.Fatalf("expected seat index %d, got %d", i, seatIdx)
		}
		conns[i] = conn
	}
	drain(r)
	return conns
}

func cmd(t *testing.T, msgType protocol.MessageType, payload any) protocol.ClientMessage {
	t.Helper()
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload for %s: %v", msgType, err)
	}
	return protocol.ClientMessage{Type: msgType, Data: data}
}

func TestJoinEmptyNameRejected(t *testing.T) {
	r := startRoom(t)
	_, err := r.Join("   ")
	if !errors.Is(err, ErrNameRequired) {
		t.Errorf("expected ErrNameRequired, got %v", err)
	}
}

func TestJoinFullRoomRejected(t *testing.T) {
	r := startRoom(t)
	joinAll(t, r, [4]string{"Alice", "Bob", "Carol", "Dave"})

	_, err := r.Join("Eve")
	if !errors.Is(err, ErrRoomFull) {
		t.Errorf("expected ErrRoomFull for a 5th distinct name, got %v", err)
	}
}

func TestJoinIsCaseInsensitive(t *testing.T) {
	r := startRoom(t)
	joinAll(t, r, [4]string{"Alice", "Bob", "Carol", "Dave"})

	seatIdx, err := r.Join("bOB")
	if err != nil {
		t.Fatalf("unexpected error rejoining with different casing: %v", err)
	}
	if seatIdx != 1 {
		t.Errorf("expected differently-cased name to resolve to Bob's seat (1), got %d", seatIdx)
	}
}

func TestNewConnectionWinsOnCollision(t *testing.T) {
	r := startRoom(t)
	conns := joinAll(t, r, [4]string{"Alice", "Bob", "Carol", "Dave"})

	_, newConn := joinAndAttach(t, r, "Alice")
	drain(r)

	if !conns[0].isClosed() {
		t.Error("expected the original connection to be closed when a new one claims the same name")
	}
	if newConn.isClosed() {
		t.Error("expected the new connection to remain open")
	}
	if r.ConnectedCount() != 4 {
		t.Errorf("expected 4 seats still connected (one replaced, not lost), got %d", r.ConnectedCount())
	}
}

func TestLobbyStartsGameOnFourthJoin(t *testing.T) {
	r := startRoom(t)
	conns := joinAll(t, r, [4]string{"Alice", "Bob", "Carol", "Dave"})

	welcome, ok := conns[0].last(t, protocol.TypeWelcome)
	if !ok {
		t.Fatal("expected a welcome message")
	}
	w := decodeData[protocol.WelcomePayload](t, welcome)
	if w.SeatIndex != 0 || w.RoomCode != "TEST01" {
		t.Errorf("unexpected welcome payload: %+v", w)
	}

	// All four seats should have received an initial state broadcast once
	// the game started.
	for i, c := range conns {
		state, ok := c.last(t, protocol.TypeState)
		if !ok {
			t.Fatalf("seat %d: expected a state message after game start", i)
		}
		s := decodeData[protocol.StateMessage](t, state)
		if s.SeatIndex != i {
			t.Errorf("seat %d: expected SeatIndex %d, got %d", i, i, s.SeatIndex)
		}
		if s.Name != [4]string{"Alice", "Bob", "Carol", "Dave"}[i] {
			t.Errorf("seat %d: expected player name to match seat order, got %q", i, s.Name)
		}
		if s.Phase != phaseDrawingForTest {
			t.Errorf("seat %d: expected drawing phase at hand start, got %q", i, s.Phase)
		}
	}
}

// phaseDrawingForTest avoids importing internal/canasta's constant name
// directly in every assertion above.
const phaseDrawingForTest = "drawing"

func TestNotYourTurnRejected(t *testing.T) {
	r := startRoom(t)
	conns := joinAll(t, r, [4]string{"Alice", "Bob", "Carol", "Dave"})

	state := decodeData[protocol.StateMessage](t, mustLast(t, conns[0], protocol.TypeState))
	nonCurrent := (state.CurrentPlayer + 1) % 4

	r.Submit(nonCurrent, cmd(t, protocol.TypeDrawFromDeck, struct{}{}))
	drain(r)

	errMsg, ok := conns[nonCurrent].last(t, protocol.TypeError)
	if !ok {
		t.Fatal("expected an error message for an out-of-turn command")
	}
	e := decodeData[protocol.ErrorPayload](t, errMsg)
	if e.Code != string(protocol.ErrNotYourTurn) {
		t.Errorf("expected code %s, got %s", protocol.ErrNotYourTurn, e.Code)
	}
}

func TestPickUpFootBypassesTurnOrder(t *testing.T) {
	r := startRoom(t)
	conns := joinAll(t, r, [4]string{"Alice", "Bob", "Carol", "Dave"})

	state := decodeData[protocol.StateMessage](t, mustLast(t, conns[0], protocol.TypeState))
	nonCurrent := (state.CurrentPlayer + 1) % 4

	r.Submit(nonCurrent, cmd(t, protocol.TypePickUpFoot, struct{}{}))
	drain(r)

	// pick_up_foot is exempt from the generic "must be your turn" check
	// (see dispatch.go's turnAndPhaseExempt) — a fresh game still
	// rejects it, but for a *game* reason (no canasta made yet), not
	// ErrNotYourTurn, proving the exemption actually took effect.
	errMsg, ok := conns[nonCurrent].last(t, protocol.TypeError)
	if !ok {
		t.Fatal("expected an error message (no canasta made yet)")
	}
	e := decodeData[protocol.ErrorPayload](t, errMsg)
	if e.Code == string(protocol.ErrNotYourTurn) {
		t.Error("pick_up_foot should not be rejected for being out of turn")
	}
	if e.Code != "NO_CANASTA" {
		t.Errorf("expected code NO_CANASTA, got %s", e.Code)
	}
}

func TestWrongPhaseRejected(t *testing.T) {
	r := startRoom(t)
	conns := joinAll(t, r, [4]string{"Alice", "Bob", "Carol", "Dave"})

	state := decodeData[protocol.StateMessage](t, mustLast(t, conns[0], protocol.TypeState))
	current := state.CurrentPlayer

	// The hand always starts in the drawing phase, so discarding
	// immediately (before drawing) must be rejected as the wrong phase.
	r.Submit(current, cmd(t, protocol.TypeDiscard, protocol.DiscardPayload{CardId: 0}))
	drain(r)

	errMsg, ok := conns[current].last(t, protocol.TypeError)
	if !ok {
		t.Fatal("expected an error message for a wrong-phase command")
	}
	e := decodeData[protocol.ErrorPayload](t, errMsg)
	if e.Code != string(protocol.ErrWrongPhase) {
		t.Errorf("expected code %s, got %s", protocol.ErrWrongPhase, e.Code)
	}
}

func TestMeldAllowedDuringDrawPhaseBeforeGoingDown(t *testing.T) {
	r := startRoom(t)
	conns := joinAll(t, r, [4]string{"Alice", "Bob", "Carol", "Dave"})

	state := decodeData[protocol.StateMessage](t, mustLast(t, conns[0], protocol.TypeState))
	current := state.CurrentPlayer

	// Give the current player three matching-rank cards to meld. This
	// mutates r.game, so it must run on the actor goroutine — only it may
	// ever touch game state (see this package's -race invariant) — the
	// same way drain's own closure does.
	done := make(chan struct{})
	r.submit(func() {
		p := r.game.Players[current]
		p.Hand[9001] = canasta.Card{Id: 9001, Suit: canasta.Hearts, Rank: canasta.Four}
		p.Hand[9002] = canasta.Card{Id: 9002, Suit: canasta.Diamonds, Rank: canasta.Four}
		p.Hand[9003] = canasta.Card{Id: 9003, Suit: canasta.Clubs, Rank: canasta.Four}
		close(done)
	})
	<-done

	// The hand always starts in the drawing phase (see TestWrongPhaseRejected)
	// — confirm NewMeld now succeeds there for a player who hasn't gone
	// down, via dispatch.go's meldAllowedInDrawPhase carve-out.
	r.Submit(current, cmd(t, protocol.TypeNewMeld, protocol.NewMeldPayload{CardIds: []int{9001, 9002, 9003}}))
	drain(r)

	if errMsg, ok := conns[current].last(t, protocol.TypeError); ok {
		e := decodeData[protocol.ErrorPayload](t, errMsg)
		t.Fatalf("expected NewMeld to succeed during the draw phase pre-go-down, got %s: %s", e.Code, e.Message)
	}
	newState := decodeData[protocol.StateMessage](t, mustLast(t, conns[current], protocol.TypeState))
	if len(newState.OurMelds) == 0 {
		t.Error("expected the staging meld to appear in state")
	}
}

func TestMeldStillWrongPhaseDuringDrawAfterGoingDown(t *testing.T) {
	r := startRoom(t)
	conns := joinAll(t, r, [4]string{"Alice", "Bob", "Carol", "Dave"})

	state := decodeData[protocol.StateMessage](t, mustLast(t, conns[0], protocol.TypeState))
	current := state.CurrentPlayer

	// Mark the current player's team as already gone down, and give them
	// three matching-rank cards — same actor-goroutine constraint as
	// above.
	done := make(chan struct{})
	r.submit(func() {
		p := r.game.Players[current]
		p.Team.GoneDown = true
		p.Hand[9001] = canasta.Card{Id: 9001, Suit: canasta.Hearts, Rank: canasta.Four}
		p.Hand[9002] = canasta.Card{Id: 9002, Suit: canasta.Diamonds, Rank: canasta.Four}
		p.Hand[9003] = canasta.Card{Id: 9003, Suit: canasta.Clubs, Rank: canasta.Four}
		close(done)
	})
	<-done

	// meldAllowedInDrawPhase is GoneDown-scoped, not a blanket phase
	// change — a player extending their team's real, already-gone-down
	// melds still needs PhasePlaying.
	r.Submit(current, cmd(t, protocol.TypeNewMeld, protocol.NewMeldPayload{CardIds: []int{9001, 9002, 9003}}))
	drain(r)

	errMsg, ok := conns[current].last(t, protocol.TypeError)
	if !ok {
		t.Fatal("expected NewMeld to still be rejected during the draw phase once gone down")
	}
	e := decodeData[protocol.ErrorPayload](t, errMsg)
	if e.Code != string(protocol.ErrWrongPhase) {
		t.Errorf("expected code %s, got %s", protocol.ErrWrongPhase, e.Code)
	}
}

func TestDrawAdvancesStateForAllSeats(t *testing.T) {
	r := startRoom(t)
	conns := joinAll(t, r, [4]string{"Alice", "Bob", "Carol", "Dave"})

	state := decodeData[protocol.StateMessage](t, mustLast(t, conns[0], protocol.TypeState))
	current := state.CurrentPlayer

	r.Submit(current, cmd(t, protocol.TypeDrawFromDeck, struct{}{}))
	drain(r)

	newState := decodeData[protocol.StateMessage](t, mustLast(t, conns[current], protocol.TypeState))
	if newState.Phase != "playing" {
		t.Errorf("expected phase to advance to playing after drawing, got %q", newState.Phase)
	}
	if len(newState.Hand) != 17 {
		t.Errorf("expected 17 cards in hand after drawing 2, got %d", len(newState.Hand))
	}

	// Every seat, not just the acting one, should have received a fresh
	// broadcast.
	for i, c := range conns {
		s := decodeData[protocol.StateMessage](t, mustLast(t, c, protocol.TypeState))
		if s.Phase != "playing" {
			t.Errorf("seat %d: expected to observe the phase change too, got %q", i, s.Phase)
		}
	}
}

func TestGrantPermissionRequiresPartner(t *testing.T) {
	r := startRoom(t)
	conns := joinAll(t, r, [4]string{"Alice", "Bob", "Carol", "Dave"})

	state := decodeData[protocol.StateMessage](t, mustLast(t, conns[0], protocol.TypeState))
	current := state.CurrentPlayer
	nonPartner := (current + 1) % 4 // adjacent seat, not the partner at +2

	r.Submit(nonPartner, cmd(t, protocol.TypeGrantPermissionToGoOut, struct{}{}))
	drain(r)

	errMsg, ok := conns[nonPartner].last(t, protocol.TypeError)
	if !ok {
		t.Fatal("expected an error for a non-partner permission grant")
	}
	e := decodeData[protocol.ErrorPayload](t, errMsg)
	if e.Code != string(protocol.ErrNotPartner) {
		t.Errorf("expected code %s, got %s", protocol.ErrNotPartner, e.Code)
	}
}

func TestDisconnectReconnectResumesSeat(t *testing.T) {
	r := startRoom(t)
	conns := joinAll(t, r, [4]string{"Alice", "Bob", "Carol", "Dave"})

	stateBefore := decodeData[protocol.StateMessage](t, mustLast(t, conns[1], protocol.TypeState))

	r.Disconnect(1, conns[1])
	drain(r)

	if r.ConnectedCount() != 3 {
		t.Errorf("expected 3 connected seats after a disconnect, got %d", r.ConnectedCount())
	}

	// Reconnect with the same name (and no separate credential) on a new
	// connection.
	seatIdx, newConn := joinAndAttach(t, r, "Bob")
	if seatIdx != 1 {
		t.Fatalf("expected to resume seat 1, got %d", seatIdx)
	}
	drain(r)

	if r.ConnectedCount() != 4 {
		t.Errorf("expected all 4 seats connected after reconnect, got %d", r.ConnectedCount())
	}

	stateAfter := decodeData[protocol.StateMessage](t, mustLast(t, newConn, protocol.TypeState))
	if stateAfter.HandNumber != stateBefore.HandNumber {
		t.Error("expected the same hand/game state to be resumed after reconnect")
	}
	if len(stateAfter.Hand) != len(stateBefore.Hand) {
		t.Errorf("expected hand size to be preserved across reconnect, got %d want %d", len(stateAfter.Hand), len(stateBefore.Hand))
	}
}

func TestDisconnectThenAwayStatus(t *testing.T) {
	r := startRoom(t)
	conns := joinAll(t, r, [4]string{"Alice", "Bob", "Carol", "Dave"})

	r.Disconnect(2, conns[2])
	drain(r)

	// Manually fire the away-timeout event rather than sleeping for the
	// real grace period.
	r.submit(awayTimeoutEvent{seatIndex: 2, conn: conns[2]})
	drain(r)

	statusMsg, ok := conns[0].last(t, protocol.TypePlayerStatus)
	if !ok {
		t.Fatal("expected other seats to be notified of the away status")
	}
	status := decodeData[protocol.PlayerStatusPayload](t, statusMsg)
	if status.SeatIndex != 2 || status.Status != "away" {
		t.Errorf("unexpected away payload: %+v", status)
	}
}

func mustLast(t *testing.T, c *fakeConn, msgType protocol.MessageType) protocol.ServerMessage {
	t.Helper()
	msg, ok := c.last(t, msgType)
	if !ok {
		t.Fatalf("expected a %s message", msgType)
	}
	return msg
}
