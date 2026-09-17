package room

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
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

// joinAll joins all four seats with the given names (in seat order), then
// has every seat ready up and the host (always seat 0, since it's always
// the first to join) start the game, mirroring the real client flow now
// that the room no longer auto-starts on the 4th join. Returns each seat's
// fakeConn once the game has started.
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

	for i := range conns {
		r.Submit(i, cmd(t, protocol.TypeSetReady, protocol.SetReadyPayload{Ready: true}))
	}
	r.Submit(0, cmd(t, protocol.TypeStartGame, struct{}{}))
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

func TestFourthJoinDoesNotAutoStart(t *testing.T) {
	r := startRoom(t)
	var conns [4]*fakeConn
	for i, name := range [4]string{"Alice", "Bob", "Carol", "Dave"} {
		seatIdx, conn := joinAndAttach(t, r, name)
		if seatIdx != i {
			t.Fatalf("expected seat index %d, got %d", i, seatIdx)
		}
		conns[i] = conn
	}
	drain(r)

	for i, c := range conns {
		if _, ok := c.last(t, protocol.TypeState); ok {
			t.Errorf("seat %d: expected no state message before the host starts the game", i)
		}
		lobby, ok := c.last(t, protocol.TypePlayersLobby)
		if !ok {
			t.Fatalf("seat %d: expected a players_lobby broadcast", i)
		}
		payload := decodeData[protocol.PlayersLobbyPayload](t, lobby)
		if len(payload.Seats) != 4 {
			t.Fatalf("seat %d: expected 4 lobby seats, got %d", i, len(payload.Seats))
		}
		for _, s := range payload.Seats {
			if s.Name == "" {
				t.Errorf("seat %d: expected all seats named, got %+v", i, payload.Seats)
			}
		}
	}
}

func TestFirstJoinerIsHost(t *testing.T) {
	r := startRoom(t)
	conns := [4]*fakeConn{}
	for i, name := range [4]string{"Alice", "Bob", "Carol", "Dave"} {
		_, conn := joinAndAttach(t, r, name)
		conns[i] = conn
	}
	drain(r)

	lobby := decodeData[protocol.PlayersLobbyPayload](t, mustLast(t, conns[0], protocol.TypePlayersLobby))
	for _, s := range lobby.Seats {
		if s.Name == "Alice" && !s.IsHost {
			t.Error("expected the first joiner (Alice) to be host")
		}
		if s.Name != "Alice" && s.IsHost {
			t.Errorf("expected only Alice to be host, but %q is host too", s.Name)
		}
	}
}

func TestSetReadyMarksSeatReady(t *testing.T) {
	r := startRoom(t)
	conns := [4]*fakeConn{}
	for i, name := range [4]string{"Alice", "Bob", "Carol", "Dave"} {
		_, conn := joinAndAttach(t, r, name)
		conns[i] = conn
	}
	drain(r)

	r.Submit(1, cmd(t, protocol.TypeSetReady, protocol.SetReadyPayload{Ready: true}))
	drain(r)

	lobby := decodeData[protocol.PlayersLobbyPayload](t, mustLast(t, conns[0], protocol.TypePlayersLobby))
	for _, s := range lobby.Seats {
		wantReady := s.Name == "Bob"
		if s.Ready != wantReady {
			t.Errorf("seat %q: expected Ready=%v, got %v", s.Name, wantReady, s.Ready)
		}
	}
}

func TestReorderSeatsHostOnly(t *testing.T) {
	r := startRoom(t)
	conns := [4]*fakeConn{}
	for i, name := range [4]string{"Alice", "Bob", "Carol", "Dave"} {
		_, conn := joinAndAttach(t, r, name)
		conns[i] = conn
	}
	drain(r)

	r.Submit(1, cmd(t, protocol.TypeReorderSeats, protocol.ReorderSeatsPayload{Order: []int{1, 0, 2, 3}}))
	drain(r)

	errMsg, ok := conns[1].last(t, protocol.TypeError)
	if !ok {
		t.Fatal("expected an error for a non-host reorder attempt")
	}
	e := decodeData[protocol.ErrorPayload](t, errMsg)
	if e.Code != string(protocol.ErrNotHost) {
		t.Errorf("expected code %s, got %s", protocol.ErrNotHost, e.Code)
	}

	lobby := decodeData[protocol.PlayersLobbyPayload](t, mustLast(t, conns[0], protocol.TypePlayersLobby))
	if lobby.Seats[0].Name != "Alice" {
		t.Errorf("expected table order unchanged after rejected reorder, got %+v", lobby.Seats)
	}
}

func TestReorderSeatsInvalidPayloadRejected(t *testing.T) {
	r := startRoom(t)
	conns := [4]*fakeConn{}
	for i, name := range [4]string{"Alice", "Bob", "Carol", "Dave"} {
		_, conn := joinAndAttach(t, r, name)
		conns[i] = conn
	}
	drain(r)

	r.Submit(0, cmd(t, protocol.TypeReorderSeats, protocol.ReorderSeatsPayload{Order: []int{0, 0, 2, 3}}))
	drain(r)

	errMsg, ok := conns[0].last(t, protocol.TypeError)
	if !ok {
		t.Fatal("expected an error for a non-permutation reorder payload")
	}
	e := decodeData[protocol.ErrorPayload](t, errMsg)
	if e.Code != string(protocol.ErrInvalidPayload) {
		t.Errorf("expected code %s, got %s", protocol.ErrInvalidPayload, e.Code)
	}
}

func TestReorderSeatsAppliesNewTableOrder(t *testing.T) {
	r := startRoom(t)
	conns := [4]*fakeConn{}
	for i, name := range [4]string{"Alice", "Bob", "Carol", "Dave"} {
		_, conn := joinAndAttach(t, r, name)
		conns[i] = conn
	}
	drain(r)

	// Swap Alice (conn slot 0) and Bob (conn slot 1) at the table.
	r.Submit(0, cmd(t, protocol.TypeReorderSeats, protocol.ReorderSeatsPayload{Order: []int{1, 0, 2, 3}}))
	for i := range conns {
		r.Submit(i, cmd(t, protocol.TypeSetReady, protocol.SetReadyPayload{Ready: true}))
	}
	r.Submit(0, cmd(t, protocol.TypeStartGame, struct{}{}))
	drain(r)

	// conn slot 0 (Alice) now sits at table position 1; conn slot 1 (Bob) at 0.
	aliceState := decodeData[protocol.StateMessage](t, mustLast(t, conns[0], protocol.TypeState))
	if aliceState.Name != "Alice" || aliceState.SeatIndex != 1 {
		t.Errorf("expected Alice at table position 1, got name=%q seatIndex=%d", aliceState.Name, aliceState.SeatIndex)
	}
	bobState := decodeData[protocol.StateMessage](t, mustLast(t, conns[1], protocol.TypeState))
	if bobState.Name != "Bob" || bobState.SeatIndex != 0 {
		t.Errorf("expected Bob at table position 0, got name=%q seatIndex=%d", bobState.Name, bobState.SeatIndex)
	}

	// Turn order and command routing must follow the new table order: it's
	// table position 0's (Bob's) turn, so only conn slot 1 may act.
	if bobState.CurrentPlayer != 0 {
		t.Fatalf("expected table position 0 to start, got CurrentPlayer=%d", bobState.CurrentPlayer)
	}
	r.Submit(0, cmd(t, protocol.TypeDrawFromDeck, struct{}{})) // Alice, conn slot 0, table position 1 — not her turn
	drain(r)
	errMsg, ok := conns[0].last(t, protocol.TypeError)
	if !ok {
		t.Fatal("expected an out-of-turn error for Alice after reorder")
	}
	e := decodeData[protocol.ErrorPayload](t, errMsg)
	if e.Code != string(protocol.ErrNotYourTurn) {
		t.Errorf("expected code %s, got %s", protocol.ErrNotYourTurn, e.Code)
	}
}

func TestStartGameRequiresAllSeatsFilled(t *testing.T) {
	r := startRoom(t)
	_, conn := joinAndAttach(t, r, "Alice")
	drain(r)

	r.Submit(0, cmd(t, protocol.TypeStartGame, struct{}{}))
	drain(r)

	errMsg, ok := conn.last(t, protocol.TypeError)
	if !ok {
		t.Fatal("expected an error when starting with empty seats")
	}
	e := decodeData[protocol.ErrorPayload](t, errMsg)
	if e.Code != string(protocol.ErrSeatsNotFull) {
		t.Errorf("expected code %s, got %s", protocol.ErrSeatsNotFull, e.Code)
	}
}

func TestStartGameRequiresAllReady(t *testing.T) {
	r := startRoom(t)
	conns := [4]*fakeConn{}
	for i, name := range [4]string{"Alice", "Bob", "Carol", "Dave"} {
		_, conn := joinAndAttach(t, r, name)
		conns[i] = conn
	}
	drain(r)

	r.Submit(0, cmd(t, protocol.TypeStartGame, struct{}{}))
	drain(r)

	errMsg, ok := conns[0].last(t, protocol.TypeError)
	if !ok {
		t.Fatal("expected an error when starting before everyone is ready")
	}
	e := decodeData[protocol.ErrorPayload](t, errMsg)
	if e.Code != string(protocol.ErrNotAllReady) {
		t.Errorf("expected code %s, got %s", protocol.ErrNotAllReady, e.Code)
	}
}

func TestStartGameHostOnly(t *testing.T) {
	r := startRoom(t)
	conns := [4]*fakeConn{}
	for i, name := range [4]string{"Alice", "Bob", "Carol", "Dave"} {
		_, conn := joinAndAttach(t, r, name)
		conns[i] = conn
	}
	for i := range conns {
		r.Submit(i, cmd(t, protocol.TypeSetReady, protocol.SetReadyPayload{Ready: true}))
	}
	drain(r)

	r.Submit(1, cmd(t, protocol.TypeStartGame, struct{}{}))
	drain(r)

	errMsg, ok := conns[1].last(t, protocol.TypeError)
	if !ok {
		t.Fatal("expected an error for a non-host start attempt")
	}
	e := decodeData[protocol.ErrorPayload](t, errMsg)
	if e.Code != string(protocol.ErrNotHost) {
		t.Errorf("expected code %s, got %s", protocol.ErrNotHost, e.Code)
	}
	if _, ok := conns[1].last(t, protocol.TypeState); ok {
		t.Error("expected the game not to have started")
	}
}

func TestStartGameSucceedsWhenReadyAndHost(t *testing.T) {
	r := startRoom(t)
	conns := joinAll(t, r, [4]string{"Alice", "Bob", "Carol", "Dave"})

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

func TestHostStaysHostAfterDisconnect(t *testing.T) {
	r := startRoom(t)
	conns := [4]*fakeConn{}
	for i, name := range [4]string{"Alice", "Bob", "Carol", "Dave"} {
		_, conn := joinAndAttach(t, r, name)
		conns[i] = conn
	}
	drain(r)

	r.Disconnect(0, conns[0])
	drain(r)

	r.Submit(1, cmd(t, protocol.TypeStartGame, struct{}{}))
	drain(r)
	errMsg, ok := conns[1].last(t, protocol.TypeError)
	if !ok || decodeData[protocol.ErrorPayload](t, errMsg).Code != string(protocol.ErrNotHost) {
		t.Fatal("expected a non-host to still be rejected while the host is disconnected")
	}

	seatIdx, newConn := joinAndAttach(t, r, "Alice")
	if seatIdx != 0 {
		t.Fatalf("expected Alice to reclaim seat 0, got %d", seatIdx)
	}
	for i := range conns {
		if i == 0 {
			continue
		}
		r.Submit(i, cmd(t, protocol.TypeSetReady, protocol.SetReadyPayload{Ready: true}))
	}
	r.Submit(0, cmd(t, protocol.TypeSetReady, protocol.SetReadyPayload{Ready: true}))
	r.Submit(0, cmd(t, protocol.TypeStartGame, struct{}{}))
	drain(r)

	if _, ok := newConn.last(t, protocol.TypeState); !ok {
		t.Error("expected the reconnected host to be able to start the game")
	}
}

func TestReadySeatResetOnDisconnectInLobby(t *testing.T) {
	r := startRoom(t)
	conns := [4]*fakeConn{}
	for i, name := range [4]string{"Alice", "Bob", "Carol", "Dave"} {
		_, conn := joinAndAttach(t, r, name)
		conns[i] = conn
	}
	drain(r)

	r.Submit(1, cmd(t, protocol.TypeSetReady, protocol.SetReadyPayload{Ready: true}))
	drain(r)

	r.Disconnect(1, conns[1])
	drain(r)

	lobby := decodeData[protocol.PlayersLobbyPayload](t, mustLast(t, conns[0], protocol.TypePlayersLobby))
	for _, s := range lobby.Seats {
		if s.Name == "Bob" && s.Ready {
			t.Error("expected Bob's ready flag to reset after disconnecting")
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

func TestAddToMeldRejectsStrandingTheHand(t *testing.T) {
	r := startRoom(t)
	conns := joinAll(t, r, [4]string{"Alice", "Bob", "Carol", "Dave"})

	state := decodeData[protocol.StateMessage](t, mustLast(t, conns[0], protocol.TypeState))
	current := state.CurrentPlayer

	// Mirrors the reported soft-lock: a 2-card hand, no go-out
	// permission, laying one card off on an existing meld — leaving a
	// single un-discardable card. Mutates r.game, so it must run on the
	// actor goroutine.
	done := make(chan struct{})
	r.submit(func() {
		p := r.game.Players[current]
		p.Hand = canasta.PlayerHand{
			9001: {Id: 9001, Suit: canasta.Hearts, Rank: canasta.Four},
			9002: {Id: 9002, Suit: canasta.Clubs, Rank: canasta.Eight},
		}
		p.Team.Melds = []canasta.Meld{{
			Id:   100,
			Rank: canasta.Four,
			Cards: []canasta.Card{
				{Id: 1, Suit: canasta.Hearts, Rank: canasta.Four},
				{Id: 2, Suit: canasta.Spades, Rank: canasta.Four},
				{Id: 3, Suit: canasta.Diamonds, Rank: canasta.Four},
			},
		}}
		r.game.Phase = canasta.PhasePlaying
		close(done)
	})
	<-done

	r.Submit(current, cmd(t, protocol.TypeAddToMeld, protocol.AddToMeldPayload{CardIds: []int{9001}, MeldId: 100}))
	drain(r)

	errMsg, ok := conns[current].last(t, protocol.TypeError)
	if !ok {
		t.Fatal("expected AddToMeld to be rejected")
	}
	e := decodeData[protocol.ErrorPayload](t, errMsg)
	if e.Code != "WOULD_STRAND_HAND" {
		t.Errorf("expected code WOULD_STRAND_HAND, got %s", e.Code)
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

// stageGoOutEligibleTeam marks seatIdx's team as gone down with all four
// canasta types MeetsGoOutRequirements checks for — must run on the
// actor goroutine (see the r.submit(func(){...}) callers below), since
// only it may ever touch r.game.
func stageGoOutEligibleTeam(g *canasta.Game, seatIdx int) {
	team := g.Players[seatIdx].Team
	team.GoneDown = true
	team.Canastas = []canasta.Canasta{
		{Id: 1, Rank: canasta.Four, Natural: true, Cards: make([]canasta.Card, 7)},
		{Id: 2, Rank: canasta.Five, Natural: false, Cards: make([]canasta.Card, 7)},
		{Id: 3, Rank: canasta.Seven, Natural: true, Cards: make([]canasta.Card, 7)},
		{Id: 4, Rank: canasta.Wild, Natural: false, Cards: make([]canasta.Card, 7)},
	}
}

func TestAskToGoOutRejectedBeforeGoingDown(t *testing.T) {
	r := startRoom(t)
	conns := joinAll(t, r, [4]string{"Alice", "Bob", "Carol", "Dave"})

	r.Submit(0, cmd(t, protocol.TypeAskToGoOut, struct{}{}))
	drain(r)

	errMsg, ok := conns[0].last(t, protocol.TypeError)
	if !ok {
		t.Fatal("expected an error before the team has gone down")
	}
	e := decodeData[protocol.ErrorPayload](t, errMsg)
	if e.Code != "CANNOT_GO_OUT" {
		t.Errorf("expected code CANNOT_GO_OUT, got %s", e.Code)
	}
}

func TestAskToGoOutRejectedMissingCanastaType(t *testing.T) {
	r := startRoom(t)
	conns := joinAll(t, r, [4]string{"Alice", "Bob", "Carol", "Dave"})

	done := make(chan struct{})
	r.submit(func() {
		team := r.game.Players[0].Team
		team.GoneDown = true
		// Only three of the four required types.
		team.Canastas = []canasta.Canasta{
			{Id: 1, Rank: canasta.Four, Natural: true, Cards: make([]canasta.Card, 7)},
			{Id: 2, Rank: canasta.Seven, Natural: true, Cards: make([]canasta.Card, 7)},
			{Id: 3, Rank: canasta.Wild, Natural: false, Cards: make([]canasta.Card, 7)},
		}
		close(done)
	})
	<-done

	r.Submit(0, cmd(t, protocol.TypeAskToGoOut, struct{}{}))
	drain(r)

	errMsg, ok := conns[0].last(t, protocol.TypeError)
	if !ok {
		t.Fatal("expected an error when a required canasta type is missing")
	}
	e := decodeData[protocol.ErrorPayload](t, errMsg)
	if e.Code != "CANASTA_REQUIREMENTS_NOT_MET" {
		t.Errorf("expected code CANASTA_REQUIREMENTS_NOT_MET, got %s", e.Code)
	}
}

func TestAskToGoOutRejectedOnceAlreadyGranted(t *testing.T) {
	r := startRoom(t)
	conns := joinAll(t, r, [4]string{"Alice", "Bob", "Carol", "Dave"})

	done := make(chan struct{})
	r.submit(func() {
		stageGoOutEligibleTeam(r.game, 0)
		r.game.Players[0].Team.CanGoOut = true
		close(done)
	})
	<-done

	r.Submit(0, cmd(t, protocol.TypeAskToGoOut, struct{}{}))
	drain(r)

	errMsg, ok := conns[0].last(t, protocol.TypeError)
	if !ok {
		t.Fatal("expected an error when permission was already granted")
	}
	e := decodeData[protocol.ErrorPayload](t, errMsg)
	if e.Code != "ALREADY_GRANTED" {
		t.Errorf("expected code ALREADY_GRANTED, got %s", e.Code)
	}
}

func TestAskToGoOutNotifiesOnlyThePartner(t *testing.T) {
	r := startRoom(t)
	conns := joinAll(t, r, [4]string{"Alice", "Bob", "Carol", "Dave"})

	done := make(chan struct{})
	r.submit(func() {
		stageGoOutEligibleTeam(r.game, 0)
		close(done)
	})
	<-done

	// Not turn-restricted — asking works from any seat on the eligible
	// team, whether or not it's currently their turn.
	r.Submit(0, cmd(t, protocol.TypeAskToGoOut, struct{}{}))
	drain(r)

	if errMsg, ok := conns[0].last(t, protocol.TypeError); ok {
		e := decodeData[protocol.ErrorPayload](t, errMsg)
		t.Fatalf("expected the ask to succeed, got %s: %s", e.Code, e.Message)
	}

	// Seat 2 is (0+2)%4 — the asker's partner.
	notifyMsg, ok := conns[2].last(t, protocol.TypeGoOutRequested)
	if !ok {
		t.Fatal("expected the partner to receive a go_out_requested notification")
	}
	payload := decodeData[protocol.GoOutRequestedPayload](t, notifyMsg)
	if payload.AskerName != "Alice" {
		t.Errorf("expected askerName %q, got %q", "Alice", payload.AskerName)
	}

	// Neither of the opposing team's seats should have been notified.
	for _, seatIdx := range []int{1, 3} {
		if _, ok := conns[seatIdx].last(t, protocol.TypeGoOutRequested); ok {
			t.Errorf("seat %d should not have received a go_out_requested notification", seatIdx)
		}
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

func TestChatRelayedInLobby(t *testing.T) {
	r := startRoom(t)
	_, aliceConn := joinAndAttach(t, r, "Alice")
	_, bobConn := joinAndAttach(t, r, "Bob")
	drain(r)

	r.Submit(0, cmd(t, protocol.TypeChatMessage, protocol.ChatMessagePayload{Text: "hi from the lobby"}))
	drain(r)

	msg, ok := bobConn.last(t, protocol.TypeChatMessage)
	if !ok {
		t.Fatal("expected the other seat to receive the chat broadcast while still in the lobby")
	}
	payload := decodeData[protocol.ChatBroadcastPayload](t, msg)
	if payload.SeatIndex != 0 || payload.Name != "Alice" || payload.Text != "hi from the lobby" {
		t.Errorf("unexpected chat payload: %+v", payload)
	}

	if _, ok := aliceConn.last(t, protocol.TypeChatMessage); ok {
		t.Error("sender should not receive an echo of their own chat message")
	}
}

func TestChatRelayedInGame(t *testing.T) {
	r := startRoom(t)
	conns := joinAll(t, r, [4]string{"Alice", "Bob", "Carol", "Dave"})

	r.Submit(1, cmd(t, protocol.TypeChatMessage, protocol.ChatMessagePayload{Text: "nice hand"}))
	drain(r)

	for _, seatIdx := range []int{0, 2, 3} {
		msg, ok := conns[seatIdx].last(t, protocol.TypeChatMessage)
		if !ok {
			t.Fatalf("expected seat %d to receive the chat broadcast", seatIdx)
		}
		payload := decodeData[protocol.ChatBroadcastPayload](t, msg)
		if payload.SeatIndex != 1 || payload.Name != "Bob" || payload.Text != "nice hand" {
			t.Errorf("unexpected chat payload for seat %d: %+v", seatIdx, payload)
		}
	}

	if _, ok := conns[1].last(t, protocol.TypeChatMessage); ok {
		t.Error("sender should not receive an echo of their own chat message")
	}
}

func TestChatNotSentToDisconnectedSeat(t *testing.T) {
	r := startRoom(t)
	conns := joinAll(t, r, [4]string{"Alice", "Bob", "Carol", "Dave"})

	r.Disconnect(2, conns[2])
	drain(r)
	before := len(conns[2].messages(t))

	r.Submit(0, cmd(t, protocol.TypeChatMessage, protocol.ChatMessagePayload{Text: "anyone there?"}))
	drain(r)

	after := len(conns[2].messages(t))
	if after != before {
		t.Errorf("disconnected seat should not receive new messages, got %d new", after-before)
	}
}

func TestChatEmptyMessageDropped(t *testing.T) {
	r := startRoom(t)
	conns := joinAll(t, r, [4]string{"Alice", "Bob", "Carol", "Dave"})

	r.Submit(0, cmd(t, protocol.TypeChatMessage, protocol.ChatMessagePayload{Text: "   "}))
	drain(r)

	for i, c := range conns {
		if _, ok := c.last(t, protocol.TypeChatMessage); ok {
			t.Errorf("seat %d should not have received a broadcast for a blank message", i)
		}
		if _, ok := c.last(t, protocol.TypeError); ok {
			t.Errorf("seat %d should not have received an error for a blank message", i)
		}
	}
}

func TestChatOverLengthRejected(t *testing.T) {
	r := startRoom(t)
	conns := joinAll(t, r, [4]string{"Alice", "Bob", "Carol", "Dave"})

	tooLong := strings.Repeat("a", maxChatMessageLength+1)
	r.Submit(0, cmd(t, protocol.TypeChatMessage, protocol.ChatMessagePayload{Text: tooLong}))
	drain(r)

	errMsg := mustLast(t, conns[0], protocol.TypeError)
	errPayload := decodeData[protocol.ErrorPayload](t, errMsg)
	if errPayload.Code != string(protocol.ErrValidation) {
		t.Errorf("expected %s, got %s", protocol.ErrValidation, errPayload.Code)
	}

	for _, seatIdx := range []int{1, 2, 3} {
		if _, ok := conns[seatIdx].last(t, protocol.TypeChatMessage); ok {
			t.Errorf("seat %d should not have received an over-length message", seatIdx)
		}
	}
}

func TestChatMalformedPayloadRejected(t *testing.T) {
	r := startRoom(t)
	conns := joinAll(t, r, [4]string{"Alice", "Bob", "Carol", "Dave"})

	r.Submit(0, protocol.ClientMessage{Type: protocol.TypeChatMessage, Data: json.RawMessage(`{not valid json`)})
	drain(r)

	errMsg := mustLast(t, conns[0], protocol.TypeError)
	errPayload := decodeData[protocol.ErrorPayload](t, errMsg)
	if errPayload.Code != string(protocol.ErrInvalidPayload) {
		t.Errorf("expected %s, got %s", protocol.ErrInvalidPayload, errPayload.Code)
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
