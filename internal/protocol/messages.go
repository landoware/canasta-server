// Package protocol defines the websocket wire format shared by the room
// actor and the transport layer: message envelopes, per-command payloads,
// state snapshots, and error codes. It has no I/O of its own.
package protocol

import "encoding/json"

// MessageType identifies what kind of envelope a ClientMessage/ServerMessage
// carries. A named type (rather than a bare string) lets these values be
// generated as a real TypeScript enum by tygo.
type MessageType string

// ClientMessage is the envelope for every message a client sends.
type ClientMessage struct {
	Type MessageType     `json:"type"`
	Data json.RawMessage `json:"data"`
}

// ServerMessage is the envelope for every message sent to a client.
type ServerMessage struct {
	Type MessageType `json:"type"`
	Data any         `json:"data"`
}

func NewServerMessage(msgType MessageType, data any) ServerMessage {
	return ServerMessage{Type: msgType, Data: data}
}

// Client -> server message types. One per internal/canasta mutator, plus
// the permission-to-go-out handshake. Identity (name) is established at
// connection time (see internal/room.Room.Join), not via an in-band
// message.
const (
	TypeDrawFromDeck           MessageType = "draw_from_deck"
	TypePickUpDiscardPile      MessageType = "pick_up_discard_pile"
	TypeNewMeld                MessageType = "new_meld"
	TypeAddToMeld              MessageType = "add_to_meld"
	TypeBurnCards              MessageType = "burn_cards"
	TypeGoDown                 MessageType = "go_down"
	TypeDiscard                MessageType = "discard"
	TypePickUpFoot             MessageType = "pick_up_foot"
	TypePlayRedThree           MessageType = "play_red_three"
	TypeGrantPermissionToGoOut MessageType = "grant_permission_to_go_out"
	TypeAskToGoOut             MessageType = "ask_to_go_out"
)

// Server -> client message types.
const (
	TypeWelcome            MessageType = "welcome"
	TypeState              MessageType = "state"
	TypePlayersLobby       MessageType = "players_lobby"
	TypePlayerDisconnected MessageType = "player_disconnected"
	TypePlayerReconnected  MessageType = "player_reconnected"
	TypePlayerStatus       MessageType = "player_status"
	TypeGoOutRequested     MessageType = "go_out_requested"
	TypeError              MessageType = "error"
)

// CreateRoomResponse is the body of a successful POST /rooms. It's the
// only HTTP (non-websocket) wire type, but lives here rather than in
// internal/server so all wire types stay in one place for tygo.
type CreateRoomResponse struct {
	RoomCode string `json:"roomCode"`
}

// PickUpDiscardPilePayload claims the discard pile by forming a new meld
// from the player's own cards plus the top discard card.
type PickUpDiscardPilePayload struct {
	CardIds []int `json:"cardIds"`
}

// NewMeldPayload starts a new meld (or staging meld) from hand cards.
type NewMeldPayload struct {
	CardIds []int `json:"cardIds"`
}

// AddToMeldPayload adds hand cards to an existing team meld.
type AddToMeldPayload struct {
	CardIds []int `json:"cardIds"`
	MeldId  int   `json:"meldId"`
}

// BurnCardsPayload adds hand cards to an existing completed canasta.
type BurnCardsPayload struct {
	CardIds   []int `json:"cardIds"`
	CanastaId int   `json:"canastaId"`
}

// DiscardPayload discards a single card, ending the player's turn.
type DiscardPayload struct {
	CardId int `json:"cardId"`
}

// PlayRedThreePayload plays one or more red threes from hand or foot.
type PlayRedThreePayload struct {
	CardIds  []int `json:"cardIds"`
	FromFoot bool  `json:"fromFoot"`
}

// draw_from_deck, go_down, pick_up_foot, grant_permission_to_go_out, and
// ask_to_go_out take no payload beyond the envelope.

// GoOutRequestedPayload notifies exactly one seat — the asker's
// partner — that their partner wants to go out. See applyAskToGoOut in
// internal/room/dispatch.go.
type GoOutRequestedPayload struct {
	AskerName string `json:"askerName"`
}

// WelcomePayload is sent immediately after a successful join/reconnect.
type WelcomePayload struct {
	SeatIndex int    `json:"seatIndex"`
	RoomCode  string `json:"roomCode"`
	RoomState string `json:"roomState"`
}

// LobbySeat describes one seat while a room is still in its Lobby state.
type LobbySeat struct {
	SeatIndex int    `json:"seatIndex"`
	Name      string `json:"name"`
	Connected bool   `json:"connected"`
}

// PlayersLobbyPayload is broadcast to all connected seats while a room
// waits for its remaining players to join.
type PlayersLobbyPayload struct {
	Seats []LobbySeat `json:"seats"`
}

// PlayerStatusPayload notifies clients of a seat's connection status, e.g.
// "disconnected", "reconnected", or "away" after the grace period elapses.
type PlayerStatusPayload struct {
	SeatIndex int    `json:"seatIndex"`
	Status    string `json:"status"`
}

// ErrorPayload is sent to a single requester when their command is
// rejected; no broadcast happens since no mutation occurred.
type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
