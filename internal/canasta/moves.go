package canasta

import (
	"errors"
	"fmt"
	"slices"
)

func (g *Game) DrawFromDeck(p *Player) (exhausted bool) {
	cards, exhausted := g.Hand.Deck.Draw(2)

	// Keep drawing replacement cards for red threes
	for slices.ContainsFunc(cards, func(c Card) bool {
		return c.Rank == Three && !c.Suit.isBlack()
	}) {
		// Find and process red threes
		var remainingCards []Card
		for _, card := range cards {
			if card.Rank == Three && !card.Suit.isBlack() {
				// Add red three to team's collection
				p.Team.RedThrees = append(p.Team.RedThrees, card)
				// Draw a replacement card
				replacement, _ := g.Hand.Deck.Draw(1)
				remainingCards = append(remainingCards, replacement...)
			} else {
				remainingCards = append(remainingCards, card)
			}
		}
		cards = remainingCards
	}

	// Add all non-red-three cards to player's hand
	for _, card := range cards {
		p.Hand[card.GetId()] = card
	}

	g.Phase = PhasePlaying
	return exhausted
}

func (g *Game) PickUpDiscardPile(p *Player, cardIds []int) error {
	if len(cardIds) < 2 {
		return errors.New("INVALID_MELD: Must provide at least two cards to make a new meld")
	}

	topCard := g.Hand.DiscardPile[len(g.Hand.DiscardPile)-1]
	if topCard.Rank == Three {
		return errors.New("PILE_FROZEN: Cannot pickup the pile with a black three on top")
	}

	for _, cardId := range cardIds {
		if p.Hand[cardId].Rank != topCard.Rank && !p.Hand[cardId].IsWild() && !topCard.IsWild() {
			return fmt.Errorf("MELD_MISMATCH: New meld must be created with %ss", topCard.Rank.String())
		}
		if topCard.IsWild() && !p.Hand[cardId].IsWild() {
			return errors.New("MELD_MISMATCH: New meld must be created with wildcards")
		}
	}

	// Must meet meld requirements with staging meld point + this new meld's points
	if !p.Team.GoneDown {
		pointsRequired := meldRequirements[g.HandNumber]
		score := 0
		for _, meld := range p.StagingMelds {
			score += meld.Score()
		}
		for _, cardId := range cardIds {
			score += p.Hand[cardId].Value()
		}
		score += topCard.Value()

		if score < pointsRequired {
			return fmt.Errorf("GO_DOWN_REQUIREMENT_NOT_MET: Cannot go down with fewer than %d points. You have %d points between your staging melds and the new meld.", pointsRequired, score)
		}
	}

	p.Hand[topCard.GetId()] = topCard
	cardIds = append(cardIds, topCard.GetId())

	err := g.NewMeld(p, cardIds)
	if err != nil {
		// Take the card out of their hand
		delete(p.Hand, topCard.GetId())
		return err
	}

	if !p.Team.GoneDown {
		g.GoDown(p)
	}

	for _, card := range g.Hand.DiscardPile {
		p.Hand[card.GetId()] = card
	}
	delete(p.Hand, topCard.GetId())
	g.Hand.DiscardPile = []Card{}

	g.Phase = PhasePlaying
	return nil
}

func (g *Game) NewMeld(p *Player, cardIds []int) error {
	meld, err := p.ValidateMeld(cardIds)
	if err != nil {
		return err
	}

	// Cool let's do it then
	if p.Team.GoneDown {
		p.Team.Melds = append(p.Team.Melds, meld)

		if len(meld.Cards) >= 7 {
			p.NewCanasta(len(p.Team.Melds) - 1)
		}
	} else {
		// Add it to the player's "staging" melds.
		p.StagingMelds = append(p.StagingMelds, meld)
	}
	// No cards for you
	p.Hand.removeCards(cardIds)
	return nil
}

func (g *Game) AddToMeld(p *Player, cardIds []int, meldId int) error {
	var cards []Card

	// A meld being added to may be either an official team meld (post-go-down)
	// or one of the player's own not-yet-committed staging melds — try the
	// official melds first since that's the common case once a team is down.
	official := true
	meldIndex, err := findIndex(meldId, p.Team.Melds)
	if err != nil {
		official = false
		meldIndex, err = findIndex(meldId, p.StagingMelds)
		if err != nil {
			return err
		}
	}

	var meld *Meld
	if official {
		meld = &p.Team.Melds[meldIndex]
	} else {
		meld = &p.StagingMelds[meldIndex]
	}

	for _, cardId := range cardIds {
		card := p.Hand[cardId]
		if card.Rank != meld.Rank && !card.IsWild() {
			return errors.New("MELD_MISMATCH: Card does not match this meld")
		}
		if card.Rank == Three {
			return errors.New("MELD_MISMATCH: Cannot use threes in melds")
		}
		if meld.Rank == Seven && card.IsWild() {
			return errors.New("MELD_MISMATCH: Cannot use wildcards in a Sevens meld")
		}

		if card.IsWild() {
			meld.WildCount++
			// An all-wild meld has no wildcard cap, same as ValidateMeld's
			// "allWilds" allowance when the meld is first created.
			if meld.Rank != Wild && meld.WildCount > 3 {
				return errors.New("MELD_MISMATCH: Cannot add more wildcards to this Meld")
			}
		}
		cards = append(cards, p.Hand[cardId])
	}

	meld.Cards = append(meld.Cards, cards...)
	p.Hand.removeCards(cardIds)

	// A staging meld only ever becomes a canasta at GoDown time (see the
	// "7+ cards in a staging meld" handling there), never while still staged.
	if official && len(meld.Cards) >= 7 {
		p.NewCanasta(meldIndex)
	}

	return nil
}

func (g *Game) BurnCards(p *Player, cardIds []int, canastaId int) error {
	canastaIndex, err := findIndex(canastaId, p.Team.Canastas)
	if err != nil {
		return err
	}

	for _, cardId := range cardIds {
		card := p.Hand[cardId]
		if card.IsWild() && p.Team.Canastas[canastaIndex].Natural {
			return errors.New("MELD_MISMATCH: Cannot make a natural canasta unnatural")
		}
		if card.Rank != p.Team.Canastas[canastaIndex].Rank && !card.IsWild() {
			return errors.New("MELD_MISMATCH: Card does not match this meld")
		}
		if p.Team.Canastas[canastaIndex].Rank == Three {
			return errors.New("MELD_MISMATCH: Cannot use threes in melds")
		}
		if p.Team.Canastas[canastaIndex].Rank == Seven && card.IsWild() {
			return errors.New("MELD_MISMATCH: Cannot use wildcards in a Sevens meld")
		}

		wildcards := WildCount(p.Team.Canastas[canastaIndex].Cards)
		if card.IsWild() {
			wildcards++
			if wildcards > 3 {
				return errors.New("MELD_MISMATCH: Cannot add more wildcards to this Meld")
			}
		}

		p.Team.Canastas[canastaIndex].Cards = append(p.Team.Canastas[canastaIndex].Cards, p.Hand[cardId])
		p.Team.Canastas[canastaIndex].Count++
	}
	p.Hand.removeCards(cardIds)

	return nil
}

func (g *Game) GoDown(p *Player) error {
	pointsRequired := meldRequirements[g.HandNumber]
	score := 0
	for _, meld := range p.StagingMelds {
		score += meld.Score()
	}
	if score < pointsRequired {
		return fmt.Errorf("GO_DOWN_REQUIREMENT_NOT_MET: Cannot go down with fewer than %d points. You have played %d points.", pointsRequired, score)
	}

	p.Team.GoneDown = true

	// When a player goes down, put the partner's staging meld cards back in their hand
	t := p.partner
	partnerMelds := t.StagingMelds
	for _, meld := range partnerMelds {
		for _, card := range meld.Cards {
			t.Hand[card.GetId()] = card
		}
	}
	t.StagingMelds = []Meld{}

	for _, meld := range p.StagingMelds {
		p.Team.Melds = append(p.Team.Melds, meld)

		// Handle a player having 7+ cards in a staging meld
		if len(meld.Cards) >= 7 {
			p.NewCanasta(len(p.Team.Melds) - 1)
		}
	}
	p.StagingMelds = []Meld{}

	return nil
}

// GrantPermissionToGoOut lets a player's partner authorize them to discard
// their entire hand and end the round early.
func (g *Game) GrantPermissionToGoOut(partner *Player) error {
	if !partner.Team.GoneDown {
		return errors.New("CANNOT_GO_OUT: Team must go down before granting permission to go out")
	}
	if !partner.Team.MeetsGoOutRequirements() {
		return errors.New("CANASTA_REQUIREMENTS_NOT_MET: Team needs a natural canasta, an unnatural canasta, a canasta of sevens, and a canasta of wildcards before going out")
	}
	partner.Team.CanGoOut = true
	return nil
}

func (g *Game) Discard(p *Player, cardId int) error {
	// Are they allowed to go out?
	// If not they need at least two cards in their hand PRIOR to discarding.
	if !p.Team.CanGoOut {
		if len(p.Hand) < 2 {
			return errors.New("CANNOT_GO_OUT: Need permission from partner before going out")
		}
	}

	card := p.Hand[cardId]
	// A red three is never a real discard candidate — it's always played
	// via PlayRedThree instead (see there), immediately or at the start
	// of a later turn.
	if card.Rank == Three && !card.Suit.isBlack() {
		return errors.New("INVALID_CARD: Cannot discard a red three")
	}
	p.Hand.removeCards([]int{cardId})
	g.Hand.DiscardPile = append(g.Hand.DiscardPile, card)

	if p.Team.CanGoOut && len(p.Hand) == 0 {
		g.EndHand()
	}

	p.CanastaMadeThisTurn = false
	g.Phase = PhaseDrawing
	g.CurrentPlayer = (g.CurrentPlayer + 1) % 4
	return nil
}

func (g *Game) PickUpFoot(p *Player) error {
	// Must have completed a Canasta
	if !p.MadeCanasta {
		return errors.New("NO_CANASTA: Must complete a canasta before picking up foot")
	}
	// Can't pick up the same turn your first canasta was made — only
	// from the next turn onward, once you've discarded (see Discard,
	// which clears this).
	if p.CanastaMadeThisTurn {
		return errors.New("CANASTA_THIS_TURN: Cannot pick up foot until after discarding the turn your first canasta was made")
	}
	// Free to pick up on any other player's turn (see dispatch.go, which
	// exempts this command from the usual "must be your turn" check).
	// On your own turn, only before you've drawn — once you're in the
	// playing phase you have to wait until you discard (ending your
	// turn) to pick it up.
	if g.Players[g.CurrentPlayer] == p && g.Phase == PhasePlaying {
		return errors.New("WRONG_PHASE: Cannot pick up your foot after drawing on your own turn — wait until you discard")
	}

	for _, card := range p.Foot {
		p.Hand[card.GetId()] = card
	}
	p.Foot = []Card{}

	return nil
}

// PlayRedThree handles playing red threes from hand or foot
// Why separate from DrawFromDeck: Manual play at start of turn
// Behavior depends on source:
//   - From initial hand: Draw 1 replacement card per red three
//   - From foot: NO replacement draw (just add to pile)
func (g *Game) PlayRedThree(p *Player, cardIds []int, fromFoot bool) error {
	// Must be in drawing phase (start of turn, before normal draw)
	// Why: Red threes played first, then normal draw happens
	if g.Phase != PhaseDrawing {
		return errors.New("WRONG_PHASE: Red threes must be played at start of turn (drawing phase)")
	}

	if len(cardIds) == 0 {
		return errors.New("NO_CARDS: Must specify at least one card")
	}

	// Validate all cards are red threes
	for _, cardId := range cardIds {
		card, exists := p.Hand[cardId]
		if !exists {
			return fmt.Errorf("CARD_NOT_FOUND: Card %d not in hand", cardId)
		}
		if card.Rank != Three || card.Suit.isBlack() {
			return errors.New("INVALID_CARD: Can only play red threes with this move")
		}
	}

	// Move red threes from hand to team pile
	for _, cardId := range cardIds {
		card := p.Hand[cardId]
		p.Team.RedThrees = append(p.Team.RedThrees, card)
		delete(p.Hand, cardId)
	}

	// Draw replacement cards ONLY if from initial hand, NOT from foot
	// Why: Standard Canasta rules - foot red threes don't get replacements
	if !fromFoot {
		replacementCards, _ := g.Hand.Deck.Draw(len(cardIds))
		for _, card := range replacementCards {
			p.Hand[card.GetId()] = card
		}
	}

	// Stay in drawing phase - player still needs to draw/pickup
	// Why: Playing red threes doesn't count as the turn's draw action

	return nil
}

func (p *Player) ValidateMeld(cardIds []int) (meld Meld, err error) {
	if len(cardIds) < 3 {
		return meld, errors.New("INVALID_MELD: Melds require at least three cards.")
	}

	// Get the cards themselves without affecting the player's hand yet.
	// We'll check that they aren't trying to pull a fast one first.
	var cards []Card
	allWilds := true
	var rank Rank

	for _, cardId := range cardIds {
		card := p.Hand[cardId]

		// Can't use a three for a canasta
		if card.Rank == Three {
			return meld, errors.New("INVALID_MELD: Cannot use threes in melds")
		}

		// Set the rank based on the first non-wild
		if !card.IsWild() {
			if allWilds {
				rank = card.Rank
				allWilds = false
			} else {
				if card.Rank != rank {
					return meld, errors.New("INVALID_MELD: Cannot mix rank in a meld")
				}
			}
		}

		cards = append(cards, card)
	}

	wildCount := WildCount(cards)
	// Can't mix wilds with sevens
	if rank == Seven && wildCount > 0 {
		return meld, errors.New("INVALID_MELD: Cannot use wildcards for a sevens meld")
	}

	// Can't have majority wildcards
	if !allWilds && wildCount > 3 {
		return meld, errors.New("INVALID_MELD: Cannot use more than three wildcards in an unnatural meld")
	}

	if allWilds {
		rank = Wild
	}
	meld = Meld{
		Id:        cardIds[0],
		Rank:      rank,
		Cards:     cards,
		WildCount: wildCount,
	}

	return meld, nil
}

func (h *PlayerHand) removeCards(ids []int) {
	for _, id := range ids {
		delete(*h, id)
	}
}

func (p *Player) NewCanasta(meldIndex int) {
	// Only the player's genuinely *first* canasta starts the
	// same-turn foot-pickup block — a second one later this same turn
	// shouldn't re-trigger it, since MadeCanasta is already true.
	if !p.MadeCanasta {
		p.CanastaMadeThisTurn = true
	}

	meld := p.Team.Melds[meldIndex]
	natural := true
	if meld.WildCount > 0 {
		natural = false
	}

	p.Team.Canastas = append(p.Team.Canastas, Canasta{
		// Carries over the meld's own id (unique — see ValidateMeld, which
		// sets it to its first card's id) rather than leaving it unset:
		// a completed canasta is the same "thing" the meld was, just
		// finished, and the client needs a stable, distinct id per
		// canasta (e.g. to track which one it's interacting with).
		Id:      meld.Id,
		Rank:    meld.Rank,
		Cards:   meld.Cards,
		Count:   len(meld.Cards),
		Natural: natural,
	})

	remainingMelds := slices.Delete(p.Team.Melds, meldIndex, meldIndex+1)

	p.Team.Melds = remainingMelds
	p.MadeCanasta = true
}
