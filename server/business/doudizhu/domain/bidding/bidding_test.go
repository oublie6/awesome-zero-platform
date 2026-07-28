package bidding

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"reflect"
	"testing"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/carddeck"
)

func TestDeriveFirstBidderGolden(t *testing.T) {
	var digest carddeck.DealDigest
	for index := range digest {
		digest[index] = byte(index)
	}
	position, err := DeriveFirstBidder(digest)
	if err != nil {
		t.Fatal(err)
	}
	if position != 3 {
		t.Fatalf("position=%d want=3", position)
	}
	second, err := DeriveFirstBidder(digest)
	if err != nil || second != position {
		t.Fatalf("second=%d err=%v want=%d", second, err, position)
	}
}

func TestDeriveFirstBidderRejectsZeroDigest(t *testing.T) {
	if _, err := DeriveFirstBidder(carddeck.DealDigest{}); !errors.Is(err, ErrInvalidDealDigest) {
		t.Fatalf("error=%v want ErrInvalidDealDigest", err)
	}
}

func TestDeriveFirstBidderAlwaysReturnsStableSeat(t *testing.T) {
	for value := uint64(0); value < 1000; value++ {
		var encoded [8]byte
		binary.BigEndian.PutUint64(encoded[:], value)
		sum := sha256.Sum256(encoded[:])
		digest := carddeck.DealDigest(sum)
		first, err := DeriveFirstBidder(digest)
		if err != nil {
			t.Fatalf("value=%d error=%v", value, err)
		}
		second, err := DeriveFirstBidder(digest)
		if err != nil {
			t.Fatalf("value=%d second error=%v", value, err)
		}
		if first < 1 || first > 3 || second != first {
			t.Fatalf("value=%d first=%d second=%d", value, first, second)
		}
	}
}

func TestBiddingSelectsHighestPositiveBidder(t *testing.T) {
	state := newState(1)
	first := mustSubmit(t, state, 1, ScoreOne)
	if first.CurrentBidder != 2 || first.HighestScore != ScoreOne || first.HighestBidder != 1 || first.Complete {
		t.Fatalf("first=%#v", first)
	}
	second := mustSubmit(t, state, 2, ScorePass)
	if second.CurrentBidder != 3 || second.HighestBidder != 1 || second.Complete {
		t.Fatalf("second=%#v", second)
	}
	third := mustSubmit(t, state, 3, ScoreTwo)
	if !third.Complete || third.NoLandlord || third.Landlord != 3 || third.HighestScore != ScoreTwo || third.CurrentBidder != 0 {
		t.Fatalf("third=%#v", third)
	}
	wantActions := []Action{{Position: 1, Score: ScoreOne}, {Position: 2, Score: ScorePass}, {Position: 3, Score: ScoreTwo}}
	if !reflect.DeepEqual(third.Actions, wantActions) {
		t.Fatalf("actions=%#v want=%#v", third.Actions, wantActions)
	}
}

func TestScoreThreeEndsBiddingImmediately(t *testing.T) {
	state := newState(2)
	result := mustSubmit(t, state, 2, ScoreThree)
	if !result.Complete || result.Landlord != 2 || result.HighestBidder != 2 || result.HighestScore != ScoreThree {
		t.Fatalf("result=%#v", result)
	}
	if len(result.Actions) != 1 {
		t.Fatalf("actions=%d want=1", len(result.Actions))
	}
	if _, err := state.Submit(3, ScorePass); !errors.Is(err, ErrBiddingComplete) {
		t.Fatalf("post-complete error=%v", err)
	}
}

func TestThreePassesRequireNoLandlordTermination(t *testing.T) {
	state := newState(3)
	mustSubmit(t, state, 3, ScorePass)
	mustSubmit(t, state, 1, ScorePass)
	result := mustSubmit(t, state, 2, ScorePass)
	if !result.Complete || !result.NoLandlord || result.Landlord != 0 || result.HighestBidder != 0 || result.HighestScore != ScorePass {
		t.Fatalf("result=%#v", result)
	}
}

func TestRejectedBidDoesNotMutateState(t *testing.T) {
	tests := []struct {
		name  string
		state *State
		seat  uint8
		score Score
		want  error
	}{
		{name: "invalid position", state: newState(1), seat: 4, score: ScorePass, want: ErrInvalidPosition},
		{name: "wrong turn", state: newState(1), seat: 2, score: ScorePass, want: ErrWrongTurn},
		{name: "invalid score", state: newState(1), seat: 1, score: Score(4), want: ErrInvalidScore},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			before := test.state.Snapshot()
			if _, err := test.state.Submit(test.seat, test.score); !errors.Is(err, test.want) {
				t.Fatalf("error=%v want=%v", err, test.want)
			}
			after := test.state.Snapshot()
			if !reflect.DeepEqual(after, before) {
				t.Fatalf("state mutated: before=%#v after=%#v", before, after)
			}
		})
	}

	state := newState(1)
	mustSubmit(t, state, 1, ScoreTwo)
	before := state.Snapshot()
	if _, err := state.Submit(2, ScoreOne); !errors.Is(err, ErrBidNotHigher) {
		t.Fatalf("error=%v want ErrBidNotHigher", err)
	}
	if after := state.Snapshot(); !reflect.DeepEqual(after, before) {
		t.Fatalf("state mutated: before=%#v after=%#v", before, after)
	}
}

func TestSnapshotActionsAreCopyIsolated(t *testing.T) {
	state := newState(1)
	mustSubmit(t, state, 1, ScoreOne)
	first := state.Snapshot()
	first.Actions[0] = Action{Position: 3, Score: ScoreThree}
	second := state.Snapshot()
	if second.Actions[0] != (Action{Position: 1, Score: ScoreOne}) {
		t.Fatalf("snapshot mutation leaked: %#v", second.Actions)
	}
}

func mustSubmit(t *testing.T, state *State, position uint8, score Score) Snapshot {
	t.Helper()
	result, err := state.Submit(position, score)
	if err != nil {
		t.Fatalf("Submit(%d,%d) error=%v", position, score, err)
	}
	return result
}
