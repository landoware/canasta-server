package room

import "crypto/rand"

// roomCodeAlphabet omits visually ambiguous characters (0/O, 1/I/L).
const roomCodeAlphabet = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"

const roomCodeLength = 6

// newRoomCode generates a short, human-shareable code for a room.
func newRoomCode() string {
	b := make([]byte, roomCodeLength)
	buf := make([]byte, roomCodeLength)
	_, err := rand.Read(buf)
	if err != nil {
		panic(err)
	}
	for i, v := range buf {
		b[i] = roomCodeAlphabet[int(v)%len(roomCodeAlphabet)]
	}
	return string(b)
}
