package carddeck

import "fmt"

const ShuffleVersion = "fair-doudizhu-fisher-yates-v1"

type ShuffleResult struct {
	Seed       Seed
	Deck       Deck
	DeckDigest DeckDigest
}

func Shuffle(input ShuffleInput) (ShuffleResult, error) {
	seed, err := input.DeriveSeed()
	if err != nil {
		return ShuffleResult{}, err
	}
	stream, err := NewStream(seed)
	if err != nil {
		return ShuffleResult{}, err
	}
	deck, err := ShuffleDeck(CanonicalDeck(), stream)
	if err != nil {
		return ShuffleResult{}, err
	}
	digest, err := deck.Digest()
	if err != nil {
		return ShuffleResult{}, err
	}
	return ShuffleResult{Seed: seed, Deck: deck, DeckDigest: digest}, nil
}

func ShuffleDeck(input Deck, source Uint64Source) (Deck, error) {
	if err := input.Validate(); err != nil {
		return Deck{}, err
	}
	if source == nil {
		return Deck{}, fmt.Errorf("%w: nil random source", ErrInvalidArgument)
	}
	deck := input
	for index := len(deck) - 1; index > 0; index-- {
		selected, err := Uniform(source, uint64(index+1))
		if err != nil {
			return Deck{}, err
		}
		deck[index], deck[selected] = deck[selected], deck[index]
	}
	return deck, nil
}
