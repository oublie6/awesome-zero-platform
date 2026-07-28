package playing

import (
	"errors"
	"testing"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/carddeck"
)

func TestAnalyzeRecognizesVersionedPatterns(t *testing.T) {
	tests := []struct {
		name           string
		codes          []string
		kind           Kind
		mainRank       carddeck.Rank
		sequenceLength uint8
	}{
		{name: "single", codes: []string{"C3"}, kind: KindSingle, mainRank: carddeck.RankThree, sequenceLength: 1},
		{name: "pair", codes: []string{"C3", "D3"}, kind: KindPair, mainRank: carddeck.RankThree, sequenceLength: 1},
		{name: "triple", codes: []string{"C3", "D3", "H3"}, kind: KindTriple, mainRank: carddeck.RankThree, sequenceLength: 1},
		{name: "triple with single", codes: []string{"C3", "D3", "H3", "C4"}, kind: KindTripleWithSingle, mainRank: carddeck.RankThree, sequenceLength: 1},
		{name: "triple with pair", codes: []string{"C3", "D3", "H3", "C4", "D4"}, kind: KindTripleWithPair, mainRank: carddeck.RankThree, sequenceLength: 1},
		{name: "straight", codes: []string{"C3", "C4", "C5", "C6", "C7"}, kind: KindStraight, mainRank: carddeck.RankSeven, sequenceLength: 5},
		{name: "consecutive pairs", codes: []string{"C3", "D3", "C4", "D4", "C5", "D5"}, kind: KindConsecutivePairs, mainRank: carddeck.RankFive, sequenceLength: 3},
		{name: "airplane", codes: []string{"C3", "D3", "H3", "C4", "D4", "H4"}, kind: KindAirplane, mainRank: carddeck.RankFour, sequenceLength: 2},
		{name: "airplane with repeated single wing rank", codes: []string{"C3", "D3", "H3", "C4", "D4", "H4", "C5", "D5"}, kind: KindAirplaneWithSingles, mainRank: carddeck.RankFour, sequenceLength: 2},
		{name: "airplane with pair wings", codes: []string{"C3", "D3", "H3", "C4", "D4", "H4", "C5", "D5", "C6", "D6"}, kind: KindAirplaneWithPairs, mainRank: carddeck.RankFour, sequenceLength: 2},
		{name: "four with paired singles", codes: []string{"C3", "D3", "H3", "S3", "C4", "D4"}, kind: KindFourWithTwoSingles, mainRank: carddeck.RankThree, sequenceLength: 1},
		{name: "four with two pairs", codes: []string{"C3", "D3", "H3", "S3", "C4", "D4", "C5", "D5"}, kind: KindFourWithTwoPairs, mainRank: carddeck.RankThree, sequenceLength: 1},
		{name: "bomb", codes: []string{"C3", "D3", "H3", "S3"}, kind: KindBomb, mainRank: carddeck.RankThree, sequenceLength: 1},
		{name: "rocket", codes: []string{"XJ", "YJ"}, kind: KindRocket, mainRank: carddeck.RankBigJoker, sequenceLength: 1},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cards := parseCards(t, test.codes...)
			got, err := Analyze(cards)
			if err != nil {
				t.Fatal(err)
			}
			if got.Version != RulesVersion || got.Kind != test.kind || got.MainRank != test.mainRank || got.SequenceLength != test.sequenceLength || int(got.CardCount) != len(cards) {
				t.Fatalf("pattern=%#v", got)
			}
		})
	}
}

func TestAnalyzeRejectsInvalidPatterns(t *testing.T) {
	tests := []struct {
		name  string
		cards []carddeck.Card
	}{
		{name: "empty"},
		{name: "duplicate physical card", cards: parseCards(t, "C3", "C3")},
		{name: "invalid card", cards: []carddeck.Card{carddeck.Card(carddeck.DeckSize)}},
		{name: "unrelated singles", cards: parseCards(t, "C3", "C4")},
		{name: "straight gap", cards: parseCards(t, "C3", "C4", "C5", "C7", "C8")},
		{name: "straight includes two", cards: parseCards(t, "C10", "CJ", "CQ", "CK", "CA", "C2")},
		{name: "only two consecutive pairs", cards: parseCards(t, "C3", "D3", "C4", "D4")},
		{name: "airplane body includes two", cards: parseCards(t, "CA", "DA", "HA", "C2", "D2", "H2")},
		{name: "airplane wing reuses body rank", cards: parseCards(t, "C3", "D3", "H3", "S3", "C4", "D4", "H4", "C5")},
		{name: "four with two pairs needs distinct pairs", cards: parseCards(t, "C3", "D3", "H3", "S3", "C4", "D4", "H4", "S4")},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := Analyze(test.cards); !errors.Is(err, ErrInvalidPattern) {
				t.Fatalf("error=%v", err)
			}
		})
	}
}

func TestCanBeatUsesDoudizhuOverridesAndStructure(t *testing.T) {
	tests := []struct {
		name      string
		candidate []string
		incumbent []string
		want      bool
	}{
		{name: "higher single", candidate: []string{"C4"}, incumbent: []string{"C3"}, want: true},
		{name: "lower single", candidate: []string{"C3"}, incumbent: []string{"C4"}, want: false},
		{name: "different ordinary kind", candidate: []string{"C3", "D3"}, incumbent: []string{"C4"}, want: false},
		{name: "higher equal length straight", candidate: []string{"C4", "C5", "C6", "C7", "C8"}, incumbent: []string{"C3", "C4", "C5", "C6", "C7"}, want: true},
		{name: "different straight length", candidate: []string{"C4", "C5", "C6", "C7", "C8", "C9"}, incumbent: []string{"C3", "C4", "C5", "C6", "C7"}, want: false},
		{name: "bomb beats ordinary", candidate: []string{"C3", "D3", "H3", "S3"}, incumbent: []string{"C2"}, want: true},
		{name: "higher bomb", candidate: []string{"C4", "D4", "H4", "S4"}, incumbent: []string{"C3", "D3", "H3", "S3"}, want: true},
		{name: "lower bomb", candidate: []string{"C3", "D3", "H3", "S3"}, incumbent: []string{"C4", "D4", "H4", "S4"}, want: false},
		{name: "rocket beats bomb", candidate: []string{"XJ", "YJ"}, incumbent: []string{"C2", "D2", "H2", "S2"}, want: true},
		{name: "bomb cannot beat rocket", candidate: []string{"C2", "D2", "H2", "S2"}, incumbent: []string{"XJ", "YJ"}, want: false},
		{name: "rocket cannot beat itself", candidate: []string{"XJ", "YJ"}, incumbent: []string{"XJ", "YJ"}, want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			candidate := mustAnalyze(t, test.candidate...)
			incumbent := mustAnalyze(t, test.incumbent...)
			got, err := CanBeat(candidate, incumbent)
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Fatalf("CanBeat(%#v, %#v)=%v want=%v", candidate, incumbent, got, test.want)
			}
		})
	}
}

func TestCanBeatRejectsFabricatedPattern(t *testing.T) {
	valid := mustAnalyze(t, "C3")
	invalid := valid
	invalid.Version = "future"
	if _, err := CanBeat(invalid, valid); !errors.Is(err, ErrInvalidPatternValue) {
		t.Fatalf("error=%v", err)
	}
}

func parseCards(t *testing.T, codes ...string) []carddeck.Card {
	t.Helper()
	cards := make([]carddeck.Card, len(codes))
	for index, code := range codes {
		card, err := carddeck.ParseCard(code)
		if err != nil {
			t.Fatal(err)
		}
		cards[index] = card
	}
	return cards
}

func mustAnalyze(t *testing.T, codes ...string) Pattern {
	t.Helper()
	result, err := Analyze(parseCards(t, codes...))
	if err != nil {
		t.Fatal(err)
	}
	return result
}
