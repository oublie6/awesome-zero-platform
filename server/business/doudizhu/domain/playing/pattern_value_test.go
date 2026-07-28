package playing

import (
	"errors"
	"testing"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/carddeck"
)

func TestCanBeatRejectsStructurallyFabricatedPatterns(t *testing.T) {
	valid := mustAnalyze(t, "C3")
	tests := []Pattern{
		{Version: RulesVersion, Kind: KindSingle, MainRank: carddeck.RankThree, SequenceLength: 1, CardCount: 2},
		{Version: RulesVersion, Kind: KindPair, MainRank: carddeck.RankSmallJoker, SequenceLength: 1, CardCount: 2},
		{Version: RulesVersion, Kind: KindStraight, MainRank: carddeck.RankTwo, SequenceLength: 5, CardCount: 5},
		{Version: RulesVersion, Kind: KindStraight, MainRank: carddeck.RankFive, SequenceLength: 6, CardCount: 6},
		{Version: RulesVersion, Kind: KindConsecutivePairs, MainRank: carddeck.RankFive, SequenceLength: 2, CardCount: 4},
		{Version: RulesVersion, Kind: KindAirplane, MainRank: carddeck.RankFour, SequenceLength: 2, CardCount: 8},
		{Version: RulesVersion, Kind: KindAirplaneWithSingles, MainRank: carddeck.RankFour, SequenceLength: 2, CardCount: 10},
		{Version: RulesVersion, Kind: KindFourWithTwoPairs, MainRank: carddeck.RankThree, SequenceLength: 2, CardCount: 8},
		{Version: RulesVersion, Kind: KindBomb, MainRank: carddeck.RankBigJoker, SequenceLength: 1, CardCount: 4},
		{Version: RulesVersion, Kind: KindRocket, MainRank: carddeck.RankSmallJoker, SequenceLength: 1, CardCount: 2},
	}
	for index, fabricated := range tests {
		if _, err := CanBeat(fabricated, valid); !errors.Is(err, ErrInvalidPatternValue) {
			t.Fatalf("case %d pattern=%#v error=%v", index, fabricated, err)
		}
		if _, err := CanBeat(valid, fabricated); !errors.Is(err, ErrInvalidPatternValue) {
			t.Fatalf("incumbent case %d pattern=%#v error=%v", index, fabricated, err)
		}
	}
}
