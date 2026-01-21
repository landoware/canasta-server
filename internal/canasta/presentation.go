package canasta

type ClientState struct {
	DeckCount      int                `json:"deckCount"`
	DiscardCount   int                `json:"discardCount"`
	DiscardTopCard *Card              `json:"discardTopCard"` // Pointer so we can send nil when pile is empty
	Name           string             `json:"name"`
	Hand           PlayerHand         `json:"hand"`
	HasFoot        bool               `json:"hasFoot"`
	Players        []OtherPlayerState `json:"players"`
	OurScore       int                `json:"ourScore"`
	OurMelds       []Meld             `json:"ourMelds"`
	OurCanastas    []Canasta          `json:"ourCanastas"`
	OurRedThrees   []Card             `json:"ourRedThrees"`
	OtherScore     int                `json:"otherScore"`
	OtherMelds     []Meld             `json:"otherMelds"`
	OtherCanastas  []Canasta          `json:"otherCanastas"`
	OtherRedThrees []Card             `json:"otherRedThrees"`
}

type OtherPlayerState struct {
	Name       string `json:"name"`
	HandLength int    `json:"handLength"`
	HasFoot    bool   `json:"hasFoot"`
}

func (g *Game) GetClientState(playerID int) *ClientState {
	player := g.Players[playerID]

	otherStates := []OtherPlayerState{}
	for id, p := range g.Players {
		if id != playerID {
			otherStates = append(otherStates, GetOtherPlayerState(p))
		}
	}

	melds := player.Team.Melds
	if !player.Team.GoneDown {
		melds = player.StagingMelds
	}

	opposingTeam := g.Players[(playerID+1)%4].Team

	// Handle empty discard pile (e.g., when a player picks up the entire pile)
	// Use pointer so we can send nil when pile is empty (instead of zero-value Card)
	var topCard *Card
	if len(g.Hand.DiscardPile) > 0 {
		card := g.Hand.DiscardPile[len(g.Hand.DiscardPile)-1]
		topCard = &card
	}

	return &ClientState{
		DeckCount:      g.Hand.Deck.Count(),
		DiscardCount:   len(g.Hand.DiscardPile),
		DiscardTopCard: topCard,
		Name:           player.Name,
		Hand:           player.Hand,
		HasFoot:        len(player.Foot) != 0,
		Players:        otherStates,
		OurScore:       player.Team.Score,
		OurMelds:       melds,
		OurCanastas:    player.Team.Canastas,
		OurRedThrees:   player.Team.RedThrees,
		OtherScore:     opposingTeam.Score,
		OtherMelds:     opposingTeam.Melds,
		OtherCanastas:  opposingTeam.Canastas,
		OtherRedThrees: opposingTeam.RedThrees,
	}
}

func GetOtherPlayerState(p *Player) OtherPlayerState {
	return OtherPlayerState{
		Name:       p.Name,
		HandLength: len(p.Hand),
		HasFoot:    len(p.Foot) != 0,
	}
}
