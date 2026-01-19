package canasta_test

import (
	"canasta-server/internal/canasta"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOtherStatesMatch(t *testing.T) {
	assert := assert.New(t)

	g := canasta.NewGame("ABCD", []string{"A", "B", "C", "D"}, canasta.WithFixedTeamOrder())
	g.NewHand()

	state1 := g.GetClientState(0)
	assert.NotContains(state1.Players, canasta.GetOtherPlayerState(g.Players[0]))

	state2 := g.GetClientState(1)
	assert.NotContains(state2.Players, canasta.GetOtherPlayerState(g.Players[1]))

	state3 := g.GetClientState(2)
	assert.NotContains(state3.Players, canasta.GetOtherPlayerState(g.Players[2]))

	state4 := g.GetClientState(3)
	assert.NotContains(state4.Players, canasta.GetOtherPlayerState(g.Players[3]))

	assert.Equal(state1.DeckCount, state2.DeckCount, state3.DeckCount, state4.DeckCount)
}

func TestStagingMeldsAreShown(t *testing.T) {
}

func TestMovesChangeState(t *testing.T) {
	assert := assert.New(t)

	g := canasta.NewGame("ABCD", []string{"A", "B", "C", "D"}, canasta.WithFixedTeamOrder())
	g.NewHand()

	stateA := g.GetClientState(0)

	g.DrawFromDeck(g.Players[0])

	stateB := g.GetClientState(0)

	assert.NotEqual(stateA, stateB)
	assert.Greater(stateA.DeckCount, stateB.DeckCount)
}
