package playing

import (
	"errors"
	"fmt"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/carddeck"
)

const RulesVersion = "doudizhu-play-rules-v1"

type Kind string

const (
	KindSingle              Kind = "SINGLE"
	KindPair                Kind = "PAIR"
	KindTriple              Kind = "TRIPLE"
	KindTripleWithSingle    Kind = "TRIPLE_WITH_SINGLE"
	KindTripleWithPair      Kind = "TRIPLE_WITH_PAIR"
	KindStraight            Kind = "STRAIGHT"
	KindConsecutivePairs    Kind = "CONSECUTIVE_PAIRS"
	KindAirplane            Kind = "AIRPLANE"
	KindAirplaneWithSingles Kind = "AIRPLANE_WITH_SINGLES"
	KindAirplaneWithPairs   Kind = "AIRPLANE_WITH_PAIRS"
	KindFourWithTwoSingles  Kind = "FOUR_WITH_TWO_SINGLES"
	KindFourWithTwoPairs    Kind = "FOUR_WITH_TWO_PAIRS"
	KindBomb                Kind = "BOMB"
	KindRocket              Kind = "ROCKET"
)

var (
	ErrInvalidPattern = errors.New("doudizhu playing: invalid card pattern")
	ErrInvalidPatternValue = errors.New("doudizhu playing: invalid pattern value")
)

type Pattern struct {
	Version        string        `json:"v"`
	Kind           Kind          `json:"kind"`
	MainRank       carddeck.Rank `json:"mainRank"`
	SequenceLength uint8         `json:"sequenceLength"`
	CardCount      uint8         `json:"cardCount"`
}

func Analyze(cards []carddeck.Card) (Pattern, error) {
	if len(cards) == 0 || len(cards) > 20 {
		return Pattern{}, fmt.Errorf("%w: card count %d", ErrInvalidPattern, len(cards))
	}
	counts, err := rankCounts(cards)
	if err != nil {
		return Pattern{}, err
	}

	if len(cards) == 2 && counts[carddeck.RankSmallJoker] == 1 && counts[carddeck.RankBigJoker] == 1 {
		return pattern(KindRocket, carddeck.RankBigJoker, 1, len(cards)), nil
	}

	nonzero := ranksWithCards(counts)
	if len(nonzero) == 1 {
		rank := nonzero[0]
		switch len(cards) {
		case 1:
			return pattern(KindSingle, rank, 1, 1), nil
		case 2:
			return pattern(KindPair, rank, 1, 2), nil
		case 3:
			return pattern(KindTriple, rank, 1, 3), nil
		case 4:
			return pattern(KindBomb, rank, 1, 4), nil
		}
	}

	if len(cards) == 4 {
		if triple, ok := rankWithCount(counts, 3); ok {
			return pattern(KindTripleWithSingle, triple, 1, 4), nil
		}
	}
	if len(cards) == 5 {
		if triple, ok := rankWithCount(counts, 3); ok && countRanks(counts, 2) == 1 {
			return pattern(KindTripleWithPair, triple, 1, 5), nil
		}
	}

	if len(cards) >= 5 {
		if low, high, ok := exactSequence(counts, 1); ok && int(high-low)+1 == len(cards) {
			return pattern(KindStraight, high, len(cards), len(cards)), nil
		}
	}
	if len(cards) >= 6 && len(cards)%2 == 0 {
		length := len(cards) / 2
		if low, high, ok := exactSequence(counts, 2); ok && int(high-low)+1 == length && length >= 3 {
			return pattern(KindConsecutivePairs, high, length, len(cards)), nil
		}
	}
	if len(cards) >= 6 && len(cards)%3 == 0 {
		length := len(cards) / 3
		if low, high, ok := exactSequence(counts, 3); ok && int(high-low)+1 == length && length >= 2 {
			return pattern(KindAirplane, high, length, len(cards)), nil
		}
	}
	if len(cards) >= 8 && len(cards)%4 == 0 {
		length := len(cards) / 4
		if high, ok := airplaneBody(counts, length, false); ok {
			return pattern(KindAirplaneWithSingles, high, length, len(cards)), nil
		}
	}
	if len(cards) >= 10 && len(cards)%5 == 0 {
		length := len(cards) / 5
		if high, ok := airplaneBody(counts, length, true); ok {
			return pattern(KindAirplaneWithPairs, high, length, len(cards)), nil
		}
	}

	if len(cards) == 6 {
		if four, ok := rankWithCount(counts, 4); ok {
			return pattern(KindFourWithTwoSingles, four, 1, 6), nil
		}
	}
	if len(cards) == 8 {
		if four, ok := rankWithCount(counts, 4); ok && countRanks(counts, 2) == 2 {
			return pattern(KindFourWithTwoPairs, four, 1, 8), nil
		}
	}

	return Pattern{}, fmt.Errorf("%w: %d cards", ErrInvalidPattern, len(cards))
}

func CanBeat(candidate, incumbent Pattern) (bool, error) {
	if err := validatePattern(candidate); err != nil {
		return false, err
	}
	if err := validatePattern(incumbent); err != nil {
		return false, err
	}
	if candidate.Kind == KindRocket {
		return incumbent.Kind != KindRocket, nil
	}
	if incumbent.Kind == KindRocket {
		return false, nil
	}
	if candidate.Kind == KindBomb {
		if incumbent.Kind != KindBomb {
			return true, nil
		}
		return candidate.MainRank > incumbent.MainRank, nil
	}
	if incumbent.Kind == KindBomb {
		return false, nil
	}
	if candidate.Kind != incumbent.Kind || candidate.SequenceLength != incumbent.SequenceLength || candidate.CardCount != incumbent.CardCount {
		return false, nil
	}
	return candidate.MainRank > incumbent.MainRank, nil
}

func pattern(kind Kind, rank carddeck.Rank, sequenceLength, cardCount int) Pattern {
	return Pattern{
		Version:        RulesVersion,
		Kind:           kind,
		MainRank:       rank,
		SequenceLength: uint8(sequenceLength),
		CardCount:      uint8(cardCount),
	}
}

func rankCounts(cards []carddeck.Card) ([15]int, error) {
	var counts [15]int
	var seen [carddeck.DeckSize]bool
	for index, card := range cards {
		if !card.Valid() {
			return counts, fmt.Errorf("%w: card[%d]=%d", ErrInvalidPattern, index, card)
		}
		if seen[card] {
			return counts, fmt.Errorf("%w: duplicate card %d", ErrInvalidPattern, card)
		}
		seen[card] = true
		rank, err := card.Rank()
		if err != nil {
			return counts, err
		}
		counts[rank]++
	}
	return counts, nil
}

func ranksWithCards(counts [15]int) []carddeck.Rank {
	ranks := make([]carddeck.Rank, 0, len(counts))
	for rank, count := range counts {
		if count > 0 {
			ranks = append(ranks, carddeck.Rank(rank))
		}
	}
	return ranks
}

func rankWithCount(counts [15]int, wanted int) (carddeck.Rank, bool) {
	for rank, count := range counts {
		if count == wanted {
			return carddeck.Rank(rank), true
		}
	}
	return 0, false
}

func countRanks(counts [15]int, wanted int) int {
	result := 0
	for _, count := range counts {
		if count == wanted {
			result++
		}
	}
	return result
}

func exactSequence(counts [15]int, wanted int) (carddeck.Rank, carddeck.Rank, bool) {
	low := -1
	high := -1
	for rank, count := range counts {
		if count == 0 {
			continue
		}
		if rank > int(carddeck.RankAce) || count != wanted {
			return 0, 0, false
		}
		if low == -1 {
			low = rank
		}
		high = rank
	}
	if low == -1 {
		return 0, 0, false
	}
	for rank := low; rank <= high; rank++ {
		if counts[rank] != wanted {
			return 0, 0, false
		}
	}
	return carddeck.Rank(low), carddeck.Rank(high), true
}

func airplaneBody(counts [15]int, length int, pairWings bool) (carddeck.Rank, bool) {
	if length < 2 {
		return 0, false
	}
	for low := 0; low+length-1 <= int(carddeck.RankAce); low++ {
		remaining := counts
		bodyValid := true
		for rank := low; rank < low+length; rank++ {
			if remaining[rank] < 3 {
				bodyValid = false
				break
			}
			remaining[rank] -= 3
			if remaining[rank] != 0 {
				bodyValid = false
				break
			}
		}
		if !bodyValid {
			continue
		}
		if pairWings {
			if validPairWings(remaining, length) {
				return carddeck.Rank(low + length - 1), true
			}
			continue
		}
		if remainingCards(remaining) == length {
			return carddeck.Rank(low + length - 1), true
		}
	}
	return 0, false
}

func validPairWings(counts [15]int, pairs int) bool {
	seenPairs := 0
	for _, count := range counts {
		switch count {
		case 0:
		case 2:
			seenPairs++
		default:
			return false
		}
	}
	return seenPairs == pairs
}

func remainingCards(counts [15]int) int {
	total := 0
	for _, count := range counts {
		total += count
	}
	return total
}

func validatePattern(value Pattern) error {
	if value.Version != RulesVersion || !knownKind(value.Kind) || value.SequenceLength == 0 || value.CardCount == 0 || value.MainRank > carddeck.RankBigJoker {
		return fmt.Errorf("%w: %#v", ErrInvalidPatternValue, value)
	}
	return nil
}

func knownKind(kind Kind) bool {
	switch kind {
	case KindSingle, KindPair, KindTriple, KindTripleWithSingle, KindTripleWithPair,
		KindStraight, KindConsecutivePairs, KindAirplane, KindAirplaneWithSingles,
		KindAirplaneWithPairs, KindFourWithTwoSingles, KindFourWithTwoPairs,
		KindBomb, KindRocket:
		return true
	default:
		return false
	}
}
