package settlement

import (
	"errors"
	"testing"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/bidding"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/carddeck"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/playing"
)

func TestCalculateLandlordSpring(t *testing.T) {
	result, err := Calculate(Input{
		LandlordSeat: 1,
		WinningScore: bidding.ScoreTwo,
		Playing: completedSnapshot(1,
			playAction(t, 1, 1, "C3"),
		),
	})
	if err != nil {
		t.Fatal(err)
	}
	assertResult(t, result, true, 0, 0, true, false, 2, 4, [3]int64{8, -4, -4})
}

func TestCalculateFarmerAntiSpring(t *testing.T) {
	result, err := Calculate(Input{
		LandlordSeat: 1,
		WinningScore: bidding.ScoreThree,
		Playing: completedSnapshot(2,
			playAction(t, 1, 1, "C3"),
			playAction(t, 2, 2, "C4"),
		),
	})
	if err != nil {
		t.Fatal(err)
	}
	assertResult(t, result, false, 0, 0, false, true, 2, 6, [3]int64{-12, 6, 6})
}

func TestCalculateBombRocketAndSpringMultipliers(t *testing.T) {
	result, err := Calculate(Input{
		LandlordSeat: 3,
		WinningScore: bidding.ScoreOne,
		Playing: completedSnapshot(3,
			playAction(t, 1, 3, "C3", "D3", "H3", "S3"),
			playAction(t, 2, 3, "XJ", "YJ"),
		),
	})
	if err != nil {
		t.Fatal(err)
	}
	assertResult(t, result, true, 1, 1, true, false, 8, 8, [3]int64{-8, -8, 16})
}

func TestCalculateFarmerWinWithoutAntiSpring(t *testing.T) {
	result, err := Calculate(Input{
		LandlordSeat: 2,
		WinningScore: bidding.ScoreOne,
		Playing: completedSnapshot(3,
			playAction(t, 1, 2, "C3"),
			playAction(t, 2, 1, "C4"),
			playAction(t, 3, 2, "C5"),
			playAction(t, 4, 3, "C6"),
		),
	})
	if err != nil {
		t.Fatal(err)
	}
	assertResult(t, result, false, 0, 0, false, false, 1, 1, [3]int64{1, -2, 1})
}

func TestCalculateIsZeroSum(t *testing.T) {
	for _, result := range []Result{
		mustCalculate(t, Input{LandlordSeat: 1, WinningScore: bidding.ScoreOne, Playing: completedSnapshot(1, playAction(t, 1, 1, "C3"))}),
		mustCalculate(t, Input{LandlordSeat: 2, WinningScore: bidding.ScoreTwo, Playing: completedSnapshot(3, playAction(t, 1, 2, "C3"), playAction(t, 2, 3, "C4"))}),
	} {
		if sum := result.SeatPoints[0] + result.SeatPoints[1] + result.SeatPoints[2]; sum != 0 {
			t.Fatalf("points=%v sum=%d", result.SeatPoints, sum)
		}
	}
}

func TestCalculateRejectsInvalidSnapshots(t *testing.T) {
	valid := Input{
		LandlordSeat: 1,
		WinningScore: bidding.ScoreOne,
		Playing: completedSnapshot(1, playAction(t, 1, 1, "C3")),
	}
	tests := []struct {
		name string
		mutate func(*Input)
	}{
		{name: "invalid landlord", mutate: func(input *Input) { input.LandlordSeat = 0 }},
		{name: "invalid score", mutate: func(input *Input) { input.WinningScore = bidding.ScorePass }},
		{name: "incomplete", mutate: func(input *Input) { input.Playing.Complete = false }},
		{name: "current seat retained", mutate: func(input *Input) { input.Playing.CurrentSeat = 2 }},
		{name: "revision mismatch", mutate: func(input *Input) { input.Playing.Revision++ }},
		{name: "winner mismatch", mutate: func(input *Input) { input.Playing.WinnerSeat = 2 }},
		{name: "bad action order", mutate: func(input *Input) { input.Playing.History[0].Number = 2 }},
		{name: "bad pattern", mutate: func(input *Input) { input.Playing.History[0].Pattern.Kind = playing.KindPair }},
		{name: "card reused", mutate: func(input *Input) {
			input.Playing = completedSnapshot(1,
				playAction(t, 1, 1, "C3"),
				playAction(t, 2, 1, "C3"),
			)
		}},
		{name: "pass carries cards", mutate: func(input *Input) {
			input.Playing = completedSnapshot(1,
				playing.Action{Number: 1, Type: playing.ActionPass, Seat: 2, Cards: []carddeck.Card{mustCard(t, "C4")}},
				playAction(t, 2, 1, "C3"),
			)
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := cloneInput(valid)
			test.mutate(&input)
			if _, err := Calculate(input); !errors.Is(err, ErrInvalidInput) {
				t.Fatalf("error=%v", err)
			}
		})
	}
}

func assertResult(t *testing.T, result Result, landlordWon bool, bombs, rockets uint8, spring, antiSpring bool, multiplier, stake uint64, points [3]int64) {
	t.Helper()
	if result.Version != RulesVersion || result.LandlordWon != landlordWon || result.BombCount != bombs || result.RocketCount != rockets || result.Spring != spring || result.AntiSpring != antiSpring || result.Multiplier != multiplier || result.FinalStake != stake || result.SeatPoints != points {
		t.Fatalf("result=%#v", result)
	}
}

func completedSnapshot(winner uint8, actions ...playing.Action) playing.Snapshot {
	return playing.Snapshot{
		Version:    playing.StateVersion,
		Revision:   uint64(len(actions)),
		Complete:   true,
		WinnerSeat: winner,
		History:    actions,
	}
}

func playAction(t *testing.T, number uint64, seat uint8, codes ...string) playing.Action {
	t.Helper()
	cards := make([]carddeck.Card, len(codes))
	for index, code := range codes {
		cards[index] = mustCard(t, code)
	}
	pattern, err := playing.Analyze(cards)
	if err != nil {
		t.Fatal(err)
	}
	return playing.Action{Number: number, Type: playing.ActionPlay, Seat: seat, Cards: cards, Pattern: &pattern}
}

func mustCard(t *testing.T, code string) carddeck.Card {
	t.Helper()
	card, err := carddeck.ParseCard(code)
	if err != nil {
		t.Fatal(err)
	}
	return card
}

func mustCalculate(t *testing.T, input Input) Result {
	t.Helper()
	result, err := Calculate(input)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func cloneInput(input Input) Input {
	result := input
	result.Playing.LeadingCards = append([]carddeck.Card(nil), input.Playing.LeadingCards...)
	result.Playing.History = make([]playing.Action, len(input.Playing.History))
	for index, action := range input.Playing.History {
		result.Playing.History[index] = action
		result.Playing.History[index].Cards = append([]carddeck.Card(nil), action.Cards...)
		if action.Pattern != nil {
			pattern := *action.Pattern
			result.Playing.History[index].Pattern = &pattern
		}
	}
	return result
}
