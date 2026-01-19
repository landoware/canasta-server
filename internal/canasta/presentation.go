package canasta

type ClientState struct {
	DeckCount      int                `json:"deckCount"`
	DiscardCount   int                `json:"discardCount"`
	DiscardTopCard Card               `json:"discardTopCard"`
	Name           string             `json:"name"`
	Hand           PlayerHand         `json:"hand"`
	HasFoot        bool               `json:"hasFoot"`
	Players        []OtherPlayerState `json:"players"`
	OurScore       int                `json:"ourScore"`
	OurMelds       []Meld             `json:"ourMelds"`
	OurCanastas    []Canasta          `json:"ourCanastas"`
	OtherScore     int                `json:"otherScore"`
	OtherMelds     []Meld             `json:"otherMelds"`
	OtherCanastas  []Canasta          `json:"otherCanastas"`
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

	return &ClientState{
		DeckCount:      g.Hand.Deck.Count(),
		DiscardCount:   len(g.Hand.DiscardPile),
		DiscardTopCard: g.Hand.DiscardPile[len(g.Hand.DiscardPile)-1],
		Name:           player.Name,
		Hand:           player.Hand,
		HasFoot:        len(player.Foot) != 0,
		Players:        otherStates,
		OurScore:       player.Team.Score,
		OurMelds:       melds,
		OurCanastas:    player.Team.Canastas,
		OtherScore:     opposingTeam.Score,
		OtherMelds:     opposingTeam.Melds,
		OtherCanastas:  opposingTeam.Canastas,
	}
}

func GetOtherPlayerState(p *Player) OtherPlayerState {
	return OtherPlayerState{
		Name:       p.Name,
		HandLength: len(p.Hand),
		HasFoot:    len(p.Foot) != 0,
	}
}
