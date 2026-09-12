# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

A Go backend for realtime, four-player, four-deck Canasta (Pierson family
rules — see `README.md` for the actual game rules: scoring, melds,
canastas, going out). The client is a **separate repo**; this repo is
server-only. There is no persistence — all game state lives in server
process memory for the life of the process.

## Commands

```
make build          # go build -o main ./cmd/api
make run            # go run ./cmd/api
make test           # go test ./... -v
go test ./... -race # run before considering any change to internal/room done —
                     # the room actor's whole correctness argument is "only one
                     # goroutine touches *canasta.Game", and -race is what catches
                     # a violation of that
go test ./internal/canasta/... -run TestName -v   # run a single test
make watch          # live-reload via air (make build + ./main on change)
make docker-run / docker-down
```

`make dev` runs `generate-types` (via `tygo`) before starting the server —
this is aspirational for the not-yet-existing client repo; there's no
`tygo.yaml` in this repo yet, so `generate-types` currently has nothing to
generate from. Ignore it unless you're specifically wiring that up.

Env vars (read via `github.com/joho/godotenv/autoload`, so a `.env` file
works): `PORT` (default 8080), `CLIENT_URL` (the allowed CORS/websocket
origin; leave unset for local dev, which allows any origin).

## Architecture

Four packages, each with a single responsibility, imported in this order
(no cycles):

```
internal/canasta   pure game-rules engine, no I/O, not concurrency-safe
internal/protocol  wire message types (envelopes, payloads, error codes) — no I/O
internal/room      the concurrency boundary: one actor goroutine per game
internal/server    net/http + websocket transport, wires room.Manager to HTTP
cmd/api            entrypoint
```

### `internal/canasta` — the rules engine

Plain data structs (`Game`, `Player`, `Team`, `Hand`, `Meld`, `Canasta`,
`Card`, `Deck`) mutated in place by methods in `moves.go`
(`DrawFromDeck`, `PickUpDiscardPile`, `NewMeld`, `AddToMeld`, `BurnCards`,
`GoDown`, `Discard`, `PickUpFoot`, `PlayRedThree`,
`GrantPermissionToGoOut`). **This package has zero internal locking and is
not safe for concurrent use** — it assumes one caller drives it turn by
turn. It also does **not** enforce turn order or game phase itself (only
`PlayRedThree` checks `Phase`) — that enforcement lives entirely in
`internal/room/dispatch.go`, deliberately, so this package's ~2500-line
test suite can keep driving mutators directly on arbitrary players without
staging `CurrentPlayer`/`Phase` to match.

Game errors are plain `errors.New("CODE: human message")` strings (e.g.
`PILE_FROZEN: ...`, `MELD_MISMATCH: ...`) — `internal/protocol.ClassifyGameError`
parses that convention. If you add a new failure case to a mutator, follow
the same `SHOUTING_CASE: message` shape or it'll silently fall back to a
generic `VALIDATION_ERROR` on the wire.

`presentation.go`'s `Game.GetClientState(playerID int)` builds the
per-player redacted view (own hand in full, opponents redacted to hand
length) — this is the payload `internal/protocol.NewStateMessage` wraps
for the wire. `playerID` is a plain index into `Game.Players`; seating is
fixed at `NewGame` time as `0/2 = TeamA`, `1/3 = TeamB` (partners are
always `i` and `(i+2)%4`), and this pairing invariant is relied on in
several places (`GetClientState`'s opposing-team lookup,
`dispatch.go`'s partner check for `grant_permission_to_go_out`).

**`canasta.NewGame(id, playerNames, ...)` mutates `playerNames` in place**
when randomizing team order (the default). `internal/room.Room.startGame`
always passes `canasta.WithFixedTeamOrder()` — without it, the shuffle
would break the seat-index-equals-player-index invariant the whole room
package depends on. Don't drop that option.

### `internal/room` — the concurrency boundary

One `Room` = one game + its four `Seat`s. A single actor goroutine
(`Room.Run`) is the *only* code that ever touches `*canasta.Game` or the
seat table; everything else (websocket read loops, the reaper, the HTTP
handlers) only ever sends events onto `Room`'s internal channel via
`Join`/`Attach`/`Submit`/`Disconnect` and gets a response back over a
result channel where one is needed. If you add a new kind of event, follow
that pattern — never reach into `Room` fields from another goroutine.

Identity within a room is the player's **name**, matched
case-insensitively — there are no per-seat tokens. `Room.Join(name)`
resolves a name to a seat index (existing seat if the name matches, else
the next open one) *without* touching any connection, specifically so the
HTTP layer can reject a bad join (empty name, full room) with a plain
status code before ever upgrading the socket. `Room.Attach(seatIndex,
conn)` is the separate step that actually wires up a live connection,
called only after a successful upgrade. Don't collapse these back into one
call — it's what lets `ws_handler.go` return 400/409 pre-upgrade.

On a name collision (same name joins while already connected — two tabs,
a flaky reconnect, someone else typing the same name) the newest
connection always wins and the old one is closed. This is a deliberate
trade-off: since the room code plus a name are the *only* things needed to
join or rejoin, there's no secret distinguishing "the real Alice" from
"anyone who knows the room code and her name."

`internal/room/manager.go`'s `Manager` owns the map of live rooms behind a
plain `sync.RWMutex` — that mutex protects only the map, never game state.
A background reaper deletes idle rooms (30 min with nobody connected, or a
12-hour hard cap regardless).

### `internal/protocol` — wire format

Pure types, no I/O. `ClientMessage`/`ServerMessage` are the `{"type":
"...", "data": {...}}` envelope; one payload struct per client command,
mirroring `internal/canasta`'s mutators 1:1. `StateMessage` embeds
`canasta.ClientState` and adds the session/turn fields it doesn't know
about (`seatIndex`, `currentPlayer`, `isYourTurn`, `phase`, `handNumber`,
`gameOver`, `winner`, `canGoOut`) — add new outward-facing fields here, not
by extending `canasta.ClientState` itself, to keep the rules engine free
of transport concerns.

### `internal/server` — transport

Stdlib `net/http.ServeMux` only, no router framework (matches this
project's existing convention — don't introduce one). `ws_handler.go`
resolves the seat via `Room.Join` *before* calling `websocket.Accept`, so
join failures are plain HTTP errors, then `Room.Attach`es the upgraded
connection and runs the read loop inline in the same handler goroutine for
the life of the connection (this is intentional, not an oversight — see
the coder/websocket docs on why the request context stays valid across the
handler's lifetime this way).

## Testing patterns

- `internal/room/room_test.go` is white-box (`package room`), using an
  in-memory `fakeConn` — no real sockets. It synchronizes with the actor
  goroutine via a `drain(r)` helper that submits a closure event and waits
  for it to be processed, rather than sleeping.
- `internal/server/e2e_test.go` is black-box (`package server_test`),
  using `httptest.NewServer` plus real `github.com/coder/websocket` client
  connections — this is the only place a full 4-player game is played over
  actual sockets end to end, and doubles as the project's "scripted
  client" in place of a real frontend.
- `internal/canasta`'s tests construct games directly
  (`canasta.NewGame(...)`) and often stage state by writing struct fields
  directly (e.g. `player.Team.CanGoOut = true`) rather than driving through
  the full move sequence — that's the existing convention, follow it
  rather than always going through mutators.
