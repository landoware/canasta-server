package server_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"

	"canasta-server/internal/canasta"
	"canasta-server/internal/protocol"
	"canasta-server/internal/room"
	"canasta-server/internal/server"
)

// This is a black-box smoke test proving the HTTP + websocket + room-actor
// wiring works end-to-end, using real sockets against an httptest server.
// It doubles as the "scripted client" needed to exercise the design
// without a real frontend.

type testClient struct {
	t    *testing.T
	conn *websocket.Conn
}

func dialSeat(t *testing.T, wsURL, code, name string) *testClient {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dialURL := wsURL + "/rooms/" + code + "/ws?name=" + url.QueryEscape(name)
	c, _, err := websocket.Dial(ctx, dialURL, nil)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	return &testClient{t: t, conn: c}
}

func (c *testClient) send(msgType protocol.MessageType, payload any) {
	c.t.Helper()
	data, err := json.Marshal(payload)
	if err != nil {
		c.t.Fatalf("marshal payload: %v", err)
	}

	raw, err := json.Marshal(protocol.ClientMessage{Type: msgType, Data: data})
	if err != nil {
		c.t.Fatalf("marshal message: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := c.conn.Write(ctx, websocket.MessageText, raw); err != nil {
		c.t.Fatalf("write: %v", err)
	}
}

func (c *testClient) recv() protocol.ServerMessage {
	c.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, data, err := c.conn.Read(ctx)
	if err != nil {
		c.t.Fatalf("read: %v", err)
	}

	var msg protocol.ServerMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		c.t.Fatalf("unmarshal message: %v", err)
	}
	return msg
}

// recvUntil reads (and discards) messages until one of the given type
// arrives, so tests can skip incidental broadcasts (e.g. lobby updates)
// while waiting for the one they care about.
func (c *testClient) recvUntil(msgType protocol.MessageType) protocol.ServerMessage {
	c.t.Helper()
	for range 20 {
		msg := c.recv()
		if msg.Type == msgType {
			return msg
		}
	}
	c.t.Fatalf("did not receive a %s message in time", msgType)
	return protocol.ServerMessage{}
}

func decode[T any](t *testing.T, msg protocol.ServerMessage) T {
	t.Helper()
	raw, err := json.Marshal(msg.Data)
	if err != nil {
		t.Fatalf("re-marshal message data: %v", err)
	}
	var v T
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatalf("unmarshal message data: %v", err)
	}
	return v
}

func TestFourPlayerGameEndToEnd(t *testing.T) {
	mgr := room.NewManager()
	t.Cleanup(mgr.Close)

	srv := server.New(mgr, "")
	httpSrv := httptest.NewServer(srv.Routes())
	t.Cleanup(httpSrv.Close)

	resp, err := http.Post(httpSrv.URL+"/rooms", "application/json", nil)
	if err != nil {
		t.Fatalf("create room: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d", resp.StatusCode)
	}

	var created struct {
		RoomCode string `json:"roomCode"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatalf("decode create-room response: %v", err)
	}
	if created.RoomCode == "" {
		t.Fatal("expected a non-empty room code")
	}

	wsURL := "ws" + strings.TrimPrefix(httpSrv.URL, "http")
	names := [4]string{"Alice", "Bob", "Carol", "Dave"}

	clients := make([]*testClient, 4)
	for i, name := range names {
		c := dialSeat(t, wsURL, created.RoomCode, name)
		clients[i] = c

		welcome := decode[protocol.WelcomePayload](t, c.recvUntil(protocol.TypeWelcome))
		if welcome.SeatIndex != i {
			t.Fatalf("expected seat index %d, got %d", i, welcome.SeatIndex)
		}
	}

	var latest [4]protocol.StateMessage
	for i, c := range clients {
		latest[i] = decode[protocol.StateMessage](t, c.recvUntil(protocol.TypeState))
	}

	// Play several turns: whoever's turn it is draws, then discards a
	// card, and every seat should observe the resulting state change.
	for turn := range 10 {
		current := latest[0].CurrentPlayer
		actor := clients[current]

		actor.send(protocol.TypeDrawFromDeck, struct{}{})
		for i, c := range clients {
			latest[i] = decode[protocol.StateMessage](t, c.recvUntil(protocol.TypeState))
		}
		if latest[current].Phase != "playing" {
			t.Fatalf("turn %d: expected playing phase after draw, got %q", turn, latest[current].Phase)
		}

		// Any card except a red three (Discard now rejects those — see
		// PlayRedThree for how those actually leave a hand).
		var cardID int
		for id, card := range latest[current].Hand {
			isRedThree := card.Rank == canasta.Three && (card.Suit == canasta.Hearts || card.Suit == canasta.Diamonds)
			if isRedThree {
				continue
			}
			cardID = id
			break
		}

		actor.send(protocol.TypeDiscard, protocol.DiscardPayload{CardId: cardID})
		for i, c := range clients {
			latest[i] = decode[protocol.StateMessage](t, c.recvUntil(protocol.TypeState))
		}
		if latest[0].CurrentPlayer == current {
			t.Fatalf("turn %d: expected turn to advance after discard", turn)
		}
	}

	// Disconnect one seat and reconnect with just the room code and the
	// same name — no separate credential — and confirm the room resumes
	// the same hand rather than losing state.
	if err := clients[0].conn.Close(websocket.StatusNormalClosure, "test disconnect"); err != nil {
		t.Fatalf("close: %v", err)
	}
	time.Sleep(100 * time.Millisecond) // let the server observe the disconnect

	reconnected := dialSeat(t, wsURL, created.RoomCode, names[0])
	welcome := decode[protocol.WelcomePayload](t, reconnected.recvUntil(protocol.TypeWelcome))
	if welcome.SeatIndex != 0 {
		t.Fatalf("expected to resume seat 0, got %d", welcome.SeatIndex)
	}

	resumedState := decode[protocol.StateMessage](t, reconnected.recvUntil(protocol.TypeState))
	if resumedState.HandNumber != latest[0].HandNumber {
		t.Fatalf("expected reconnect to resume the same hand, got %d want %d", resumedState.HandNumber, latest[0].HandNumber)
	}
}

// TestNewMeldAllowedDuringDrawPhaseOverSockets proves dispatch.go's
// meldAllowedInDrawPhase carve-out actually takes effect over real
// sockets, without depending on the random deal producing a genuinely
// meldable hand: two arbitrary card ids are always too few to form a
// valid meld (ValidateMeld requires 3+), so if the wrong-phase gate had
// NOT been lifted for the draw phase, the rejection would be
// ErrWrongPhase; once lifted, the rejection reason is the meld's own
// (in)validity instead. The full positive-path scenario — staging a
// meld pre-draw, then picking up the pile to go down in the same turn —
// is already covered deterministically in internal/canasta/moves_test.go's
// TestGoingDownByPickingUpThePile and internal/room/room_test.go's
// TestMeldAllowedDuringDrawPhaseBeforeGoingDown, both of which can stage
// a known, controlled hand; a real deal here can't guarantee one.
func TestNewMeldAllowedDuringDrawPhaseOverSockets(t *testing.T) {
	mgr := room.NewManager()
	t.Cleanup(mgr.Close)

	srv := server.New(mgr, "")
	httpSrv := httptest.NewServer(srv.Routes())
	t.Cleanup(httpSrv.Close)

	resp, err := http.Post(httpSrv.URL+"/rooms", "application/json", nil)
	if err != nil {
		t.Fatalf("create room: %v", err)
	}
	var created struct {
		RoomCode string `json:"roomCode"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatalf("decode create-room response: %v", err)
	}
	resp.Body.Close()

	wsURL := "ws" + strings.TrimPrefix(httpSrv.URL, "http")
	names := [4]string{"Alice", "Bob", "Carol", "Dave"}

	clients := make([]*testClient, 4)
	for i, name := range names {
		c := dialSeat(t, wsURL, created.RoomCode, name)
		clients[i] = c
		c.recvUntil(protocol.TypeWelcome)
	}

	var latest [4]protocol.StateMessage
	for i, c := range clients {
		latest[i] = decode[protocol.StateMessage](t, c.recvUntil(protocol.TypeState))
	}

	current := latest[0].CurrentPlayer
	actor := clients[current]
	if latest[current].Phase != "drawing" {
		t.Fatalf("expected a fresh hand to start in the drawing phase, got %q", latest[current].Phase)
	}

	// Card ids must come from the acting player's own hand — each
	// client's state is redacted to its own perspective (see
	// presentation.go's GetClientState), so this must be latest[current],
	// not any other seat's.
	var ids []int
	for id := range latest[current].Hand {
		ids = append(ids, id)
		if len(ids) == 2 {
			break
		}
	}

	actor.send(protocol.TypeNewMeld, protocol.NewMeldPayload{CardIds: ids})
	errMsg := actor.recvUntil(protocol.TypeError)
	e := decode[protocol.ErrorPayload](t, errMsg)
	if e.Code == string(protocol.ErrWrongPhase) {
		t.Fatalf("expected NewMeld to be allowed during the draw phase pre-go-down, got %s: %s", e.Code, e.Message)
	}
}

// TestJoinRejectionsUseHTTPStatusCodes verifies that a rejected join is
// reported as a plain HTTP error before any websocket upgrade happens,
// rather than a socket that opens and immediately closes.
func TestJoinRejectionsUseHTTPStatusCodes(t *testing.T) {
	mgr := room.NewManager()
	t.Cleanup(mgr.Close)

	srv := server.New(mgr, "")
	httpSrv := httptest.NewServer(srv.Routes())
	t.Cleanup(httpSrv.Close)

	resp, err := http.Post(httpSrv.URL+"/rooms", "application/json", nil)
	if err != nil {
		t.Fatalf("create room: %v", err)
	}
	var created struct {
		RoomCode string `json:"roomCode"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatalf("decode create-room response: %v", err)
	}
	resp.Body.Close()

	wsPath := httpSrv.URL + "/rooms/" + created.RoomCode + "/ws"

	emptyNameResp, err := http.Get(wsPath + "?name=%20%20")
	if err != nil {
		t.Fatalf("request with empty name: %v", err)
	}
	emptyNameResp.Body.Close()
	if emptyNameResp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for an empty name, got %d", emptyNameResp.StatusCode)
	}

	names := [4]string{"Alice", "Bob", "Carol", "Dave"}
	wsURL := "ws" + strings.TrimPrefix(httpSrv.URL, "http")
	for _, name := range names {
		c := dialSeat(t, wsURL, created.RoomCode, name)
		c.recvUntil(protocol.TypeWelcome)
	}

	fullRoomResp, err := http.Get(wsPath + "?name=Eve")
	if err != nil {
		t.Fatalf("request for a 5th name: %v", err)
	}
	fullRoomResp.Body.Close()
	if fullRoomResp.StatusCode != http.StatusConflict {
		t.Errorf("expected 409 for a full room, got %d", fullRoomResp.StatusCode)
	}
}
