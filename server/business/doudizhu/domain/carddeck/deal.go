package carddeck

import (
	"crypto/sha256"
	"fmt"
)

const DealVersion = "fair-doudizhu-deal-v1"

type DealDigest Digest

type DealResult struct {
	hands         [3][CardsPerSeat]Card
	landlordCards [LandlordCardCount]Card
	digest        DealDigest
}

func Deal(deck Deck) (DealResult, error) {
	if err := deck.Validate(); err != nil {
		return DealResult{}, err
	}
	var result DealResult
	for round := 0; round < CardsPerSeat; round++ {
		for seat := 0; seat < 3; seat++ {
			result.hands[seat][round] = deck[round*3+seat]
		}
	}
	copy(result.landlordCards[:], deck[DeckSize-LandlordCardCount:])
	digest, err := result.computeDigest()
	if err != nil {
		return DealResult{}, err
	}
	result.digest = digest
	return result, nil
}

func (result DealResult) Hand(seat uint8) ([CardsPerSeat]Card, error) {
	if seat < 1 || seat > 3 {
		return [CardsPerSeat]Card{}, fmt.Errorf("%w: seat %d", ErrInvalidArgument, seat)
	}
	return result.hands[seat-1], nil
}

func (result DealResult) Hands() [3][CardsPerSeat]Card { return result.hands }

func (result DealResult) LandlordCards() [LandlordCardCount]Card { return result.landlordCards }

func (result DealResult) Digest() DealDigest { return result.digest }

func (result DealResult) ReconstructDeck() Deck {
	var deck Deck
	for round := 0; round < CardsPerSeat; round++ {
		for seat := 0; seat < 3; seat++ {
			deck[round*3+seat] = result.hands[seat][round]
		}
	}
	copy(deck[DeckSize-LandlordCardCount:], result.landlordCards[:])
	return deck
}

func (result DealResult) Validate() error {
	deck := result.ReconstructDeck()
	if err := deck.Validate(); err != nil {
		return err
	}
	digest, err := result.computeDigest()
	if err != nil {
		return err
	}
	if result.digest != (DealDigest{}) && result.digest != digest {
		return fmt.Errorf("%w: deal digest mismatch", ErrVerificationFailed)
	}
	return nil
}

func (result DealResult) computeDigest() (DealDigest, error) {
	deck := result.ReconstructDeck()
	if err := deck.Validate(); err != nil {
		return DealDigest{}, err
	}
	h := sha256.New()
	writeDomain(h, dealDigestDomain)
	writeString(h, DealVersion)
	for seat, hand := range result.hands {
		_, _ = h.Write([]byte{byte(seat + 1)})
		writeU16(h, len(hand))
		for _, card := range hand {
			_, _ = h.Write([]byte{byte(card)})
		}
	}
	writeU16(h, len(result.landlordCards))
	for _, card := range result.landlordCards {
		_, _ = h.Write([]byte{byte(card)})
	}
	var digest DealDigest
	copy(digest[:], h.Sum(nil))
	return digest, nil
}
