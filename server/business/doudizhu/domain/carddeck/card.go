package carddeck

import (
	"crypto/sha256"
	"fmt"
)

const (
	CardVersion       = "fair-doudizhu-card-v1"
	DeckVersion       = "fair-doudizhu-deck-v1"
	DeckSize          = 54
	StandardCardCount = 52
	CardsPerSeat      = 17
	LandlordCardCount = 3
)

type Card uint8

type Suit uint8

const (
	SuitClubs Suit = iota
	SuitDiamonds
	SuitHearts
	SuitSpades
)

type Rank uint8

const (
	RankThree Rank = iota
	RankFour
	RankFive
	RankSix
	RankSeven
	RankEight
	RankNine
	RankTen
	RankJack
	RankQueen
	RankKing
	RankAce
	RankTwo
	RankSmallJoker
	RankBigJoker
)

const (
	SmallJoker Card = 52
	BigJoker   Card = 53
)

type Digest [sha256.Size]byte
type DeckDigest Digest

type Deck [DeckSize]Card

var rankCodes = [...]string{"3", "4", "5", "6", "7", "8", "9", "10", "J", "Q", "K", "A", "2"}
var suitCodes = [...]byte{'C', 'D', 'H', 'S'}

func (c Card) Valid() bool { return c < DeckSize }

func (c Card) IsJoker() bool { return c == SmallJoker || c == BigJoker }

func (c Card) Rank() (Rank, error) {
	if !c.Valid() {
		return 0, fmt.Errorf("%w: card id %d", ErrInvalidArgument, c)
	}
	if c == SmallJoker {
		return RankSmallJoker, nil
	}
	if c == BigJoker {
		return RankBigJoker, nil
	}
	return Rank(uint8(c) / 4), nil
}

func (c Card) Suit() (Suit, error) {
	if !c.Valid() || c.IsJoker() {
		return 0, fmt.Errorf("%w: card %d has no suit", ErrInvalidArgument, c)
	}
	return Suit(uint8(c) % 4), nil
}

func (c Card) Code() (string, error) {
	if !c.Valid() {
		return "", fmt.Errorf("%w: card id %d", ErrInvalidArgument, c)
	}
	if c == SmallJoker {
		return "XJ", nil
	}
	if c == BigJoker {
		return "YJ", nil
	}
	rank, _ := c.Rank()
	suit, _ := c.Suit()
	return string(suitCodes[suit]) + rankCodes[rank], nil
}

func ParseCard(code string) (Card, error) {
	if code == "XJ" {
		return SmallJoker, nil
	}
	if code == "YJ" {
		return BigJoker, nil
	}
	if len(code) < 2 || len(code) > 3 {
		return 0, fmt.Errorf("%w: card code %q", ErrInvalidArgument, code)
	}
	var suit Suit
	switch code[0] {
	case 'C':
		suit = SuitClubs
	case 'D':
		suit = SuitDiamonds
	case 'H':
		suit = SuitHearts
	case 'S':
		suit = SuitSpades
	default:
		return 0, fmt.Errorf("%w: card suit %q", ErrInvalidArgument, code)
	}
	for rank, rankCode := range rankCodes {
		if code[1:] == rankCode {
			return Card(rank*4 + int(suit)), nil
		}
	}
	return 0, fmt.Errorf("%w: card rank %q", ErrInvalidArgument, code)
}

func CanonicalDeck() Deck {
	var deck Deck
	for index := range deck {
		deck[index] = Card(index)
	}
	return deck
}

func (d Deck) Validate() error {
	var seen [DeckSize]bool
	for index, card := range d {
		if !card.Valid() {
			return fmt.Errorf("%w: deck[%d]=%d", ErrInvalidArgument, index, card)
		}
		if seen[card] {
			return fmt.Errorf("%w: duplicate card %d", ErrInvalidArgument, card)
		}
		seen[card] = true
	}
	return nil
}

func (d Deck) Cards() []Card {
	cards := make([]Card, len(d))
	copy(cards, d[:])
	return cards
}

func (d Deck) Codes() ([]string, error) {
	codes := make([]string, len(d))
	for index, card := range d {
		code, err := card.Code()
		if err != nil {
			return nil, err
		}
		codes[index] = code
	}
	return codes, nil
}

func (d Deck) Digest() (DeckDigest, error) {
	if err := d.Validate(); err != nil {
		return DeckDigest{}, err
	}
	h := sha256.New()
	writeDomain(h, deckDigestDomain)
	writeString(h, DeckVersion)
	writeU16(h, DeckSize)
	for _, card := range d {
		_, _ = h.Write([]byte{byte(card)})
	}
	var digest DeckDigest
	copy(digest[:], h.Sum(nil))
	return digest, nil
}
