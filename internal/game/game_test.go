package game_test

import (
	"canasta-server/internal/game"
	"testing"
)

func TestDeal(t *testing.T) {
	names := []string{"One", "Two", "Three", "Four"}
	gameObj := game.NewGame(names)

	gameObj.Deal()

	for _, player := range gameObj.State.Players {
		if len(player.Hand) != 15 {
			t.Errorf("Player %s has %d cards in their hand, 15 expected", player.Name, len(player.Hand))
		}
		if len(player.Foot) != 11 {
			t.Errorf("Player %s has %d cards in their foot, 11 expected", player.Name, len(player.Foot))
		}
	}

	if len(gameObj.State.DiscardPile) != 1 {
		t.Errorf("Only one card should be discarded. %d given.", len(gameObj.State.DiscardPile))
	}

	if gameObj.State.Deck.Count() != 111 {
		t.Errorf("Too many cards delt. Have %d in deck, 111 expected.", gameObj.State.Deck.Count())
	}

}
