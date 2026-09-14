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

func TestOtherPlayerStateHidesStagingMeld(t *testing.T) {
	assert := assert.New(t)

	game := canasta.NewGame("ABCD", []string{"A", "B", "C", "D"}, canasta.WithFixedTeamOrder())
	game.NewHand()

	player := game.Players[0]
	before := canasta.GetOtherPlayerState(player)

	// Move a card out of hand and into a staging meld, exactly as
	// NewMeld/AddToMeld do — before the player's team has gone down, this
	// must not visibly shrink the hand count opponents/teammates see.
	var moved canasta.Card
	for id, card := range player.Hand {
		moved = card
		delete(player.Hand, id)
		break
	}
	player.StagingMelds = append(player.StagingMelds, canasta.Meld{
		Id:    0,
		Rank:  moved.Rank,
		Cards: []canasta.Card{moved},
	})

	after := canasta.GetOtherPlayerState(player)

	assert.Equal(before.HandLength, after.HandLength)
}

func TestClientStateReflectsMadeCanasta(t *testing.T) {
	assert := assert.New(t)

	g := canasta.NewGame("ABCD", []string{"A", "B", "C", "D"}, canasta.WithFixedTeamOrder())
	g.NewHand()

	assert.False(g.GetClientState(0).MadeCanasta)

	g.Players[0].MadeCanasta = true

	assert.True(g.GetClientState(0).MadeCanasta)
	// Per-player, not per-team — a partner having made a canasta doesn't
	// earn this player their own foot pickup.
	assert.False(g.GetClientState(2).MadeCanasta)
}

func TestClientStateReflectsCanastaMadeThisTurn(t *testing.T) {
	assert := assert.New(t)

	g := canasta.NewGame("ABCD", []string{"A", "B", "C", "D"}, canasta.WithFixedTeamOrder())
	g.NewHand()

	assert.False(g.GetClientState(0).CanastaMadeThisTurn)

	g.Players[0].CanastaMadeThisTurn = true

	assert.True(g.GetClientState(0).CanastaMadeThisTurn)
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
