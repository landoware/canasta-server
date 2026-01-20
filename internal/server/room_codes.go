package server

import (
	"errors"
	"math/rand"
	"strings"
)

func GenerateRoomCode(usedCodes map[string]bool) string {
	for {
		code := make([]byte, 4)
		for i := range code {
			// Generate random uppercase letter (A-Z)
			code[i] = 'A' + byte(rand.Intn(26))
		}
		roomCode := string(code)

		// Check if code is already in use
		if !usedCodes[roomCode] {
			return roomCode
		}
	}
}

func ValidateRoomCode(code string) error {
	if len(code) != 4 {
		return errors.New("Room code must be exactly 4 characters")
	}

	code = strings.ToUpper(code)
	for _, ch := range code {
		if ch < 'A' || ch > 'Z' {
			return errors.New("Room code must contain only letters A-Z")
		}
	}

	return nil
}

func NormalizeRoomCode(code string) string {
	return strings.ToUpper(code)
}
