package server_test

import (
	"canasta-server/internal/server"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateRoomCodeFormat(t *testing.T) {
	assert := assert.New(t)
	usedCodes := make(map[string]bool)

	for range 100 {
		code := server.GenerateRoomCode(usedCodes)

		assert.Equal(4, len(code))

		for _, ch := range code {
			assert.True(ch >= 'A' && ch <= 'Z')
		}
	}
}

func TestGenerateRoomCodeUniqueness(t *testing.T) {
	usedCodes := make(map[string]bool)
	generatedCodes := make(map[string]bool)

	for range 1000 {
		code := server.GenerateRoomCode(usedCodes)

		assert.False(t, generatedCodes[code], "Code %s was generated twice", code)

		generatedCodes[code] = true
		usedCodes[code] = true
	}

	assert.Equal(t, 1000, len(generatedCodes))
}

func TestGenerateRoomCodeAvoidsUsedCodes(t *testing.T) {
	usedCodes := make(map[string]bool)

	usedCodes["AAAA"] = true
	usedCodes["ZZZZ"] = true
	usedCodes["TEST"] = true

	for range 100 {
		code := server.GenerateRoomCode(usedCodes)

		assert.NotEqual(t, "AAAA", code)
		assert.NotEqual(t, "ZZZZ", code)
		assert.NotEqual(t, "TEST", code)
	}
}

func TestValidateRoomCodeValidCodes(t *testing.T) {
	validCodes := []string{"BEAR", "GAME", "PLAY", "AAAA", "ZZZZ"}

	for _, code := range validCodes {
		err := server.ValidateRoomCode(code)
		assert.NoError(t, err, "Code %s should be valid", code)
	}
}

func TestValidateRoomCodeInvalidLength(t *testing.T) {
	invalidCodes := []string{"", "A", "AB", "ABC", "ABCDE", "ABCDEF"}

	for _, code := range invalidCodes {
		err := server.ValidateRoomCode(code)
		assert.Error(t, err, "Code %s should be invalid (wrong length)", code)
		assert.Contains(t, err.Error(), "exactly 4 characters")
	}
}

func TestValidateRoomCodeInvalidCharacters(t *testing.T) {
	invalidCodes := []string{
		"1234", // numbers
		"A1B2", // letters + numbers
		"A-B!", // special chars
		"T@ST", // special chars
		"A BC", // space
		" ABC", // leading space
	}

	for _, code := range invalidCodes {
		err := server.ValidateRoomCode(code)
		assert.Error(t, err, "Code %s should be invalid (bad characters)", code)
		assert.Contains(t, err.Error(), "only letters A-Z")
	}
}
