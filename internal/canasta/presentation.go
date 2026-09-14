package canasta

type ClientState struct {
	DeckCount      int        `json:"deckCount"`
	DiscardCount   int        `json:"discardCount"`
	DiscardTopCard *Card      `json:"discardTopCard"` // Pointer so we can send nil when pile is empty
	Name           string     `json:"name"`
	Hand           PlayerHand `json:"hand"`
	HasFoot        bool       `json:"hasFoot"`
	// Whether this player has completed their own first canasta yet —
	// see PickUpFoot in moves.go, whose sole eligibility check this
	// mirrors. Per-player, not per-team: a partner going down or making
	// a canasta doesn't earn this player their own foot.
	MadeCanasta bool `json:"madeCanasta"`
	// True while this player is blocked from picking up their foot
	// because they made their first canasta this same turn and haven't
	// discarded yet — see PickUpFoot in moves.go.
	CanastaMadeThisTurn bool               `json:"canastaMadeThisTurn"`
	Players             []OtherPlayerState `json:"players"`
	OurScore            int                `json:"ourScore"`
	// GoneDown tells the client whether OurMelds is the team's official
	// melds (true) or the requesting player's own not-yet-committed
	// staging melds (false) — see NewMeld/GoDown in moves.go. There's no
	// per-meld distinction: OurMelds is always entirely one or the other.
	GoneDown       bool      `json:"goneDown"`
	OurMelds       []Meld    `json:"ourMelds"`
	OurCanastas    []Canasta `json:"ourCanastas"`
	OurRedThrees   []Card    `json:"ourRedThrees"`
	OtherScore     int       `json:"otherScore"`
	OtherMelds     []Meld    `json:"otherMelds"`
	OtherCanastas  []Canasta `json:"otherCanastas"`
	OtherRedThrees []Card    `json:"otherRedThrees"`
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
		DeckCount:           g.Hand.Deck.Count(),
		DiscardCount:        len(g.Hand.DiscardPile),
		DiscardTopCard:      topCard,
		Name:                player.Name,
		Hand:                player.Hand,
		HasFoot:             len(player.Foot) != 0,
		MadeCanasta:         player.MadeCanasta,
		CanastaMadeThisTurn: player.CanastaMadeThisTurn,
		Players:             otherStates,
		OurScore:            player.Team.Score,
		GoneDown:            player.Team.GoneDown,
		OurMelds:            melds,
		OurCanastas:         player.Team.Canastas,
		OurRedThrees:        player.Team.RedThrees,
		OtherScore:          opposingTeam.Score,
		OtherMelds:          opposingTeam.Melds,
		OtherCanastas:       opposingTeam.Canastas,
		OtherRedThrees:      opposingTeam.RedThrees,
	}
}

func GetOtherPlayerState(p *Player) OtherPlayerState {
	// Cards moved into a staging meld are removed from p.Hand immediately
	// (see NewMeld/AddToMeld in moves.go), even though the meld isn't real
	// until the player's team goes down. Counting them back in keeps a
	// staging meld invisible to opponents/teammates via hand count — it
	// stays hidden the same way the staging meld's contents already are.
	handLength := len(p.Hand)
	for _, meld := range p.StagingMelds {
		handLength += len(meld.Cards)
	}

	return OtherPlayerState{
		Name:       p.Name,
		HandLength: handLength,
		HasFoot:    len(p.Foot) != 0,
	}
}
