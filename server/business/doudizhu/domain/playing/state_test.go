package playing

import (
	"errors"
	"reflect"
	"testing"
)

func TestStatePlayAdvancesTurnAndTracksLeader(t *testing.T) {
	state := mustState(t, 1)
	snapshot, err := state.Play(1, parseCards(t, "C3"), false)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Revision != 1 || snapshot.CurrentSeat != 2 || snapshot.LeadingSeat != 1 || snapshot.PassCount != 0 || snapshot.Complete {
		t.Fatalf("snapshot=%#v", snapshot)
	}
	if snapshot.LeadingPattern == nil || snapshot.LeadingPattern.Kind != KindSingle || len(snapshot.History) != 1 || snapshot.History[0].Type != ActionPlay {
		t.Fatalf("snapshot=%#v", snapshot)
	}
}

func TestStateRejectsInvalidActionsWithoutMutation(t *testing.T) {
	tests := []struct {
		name    string
		prepare func(*testing.T, *State)
		run     func(*testing.T, *State) error
		want    error
	}{
		{
			name: "pass empty trick",
			run: func(t *testing.T, state *State) error {
				_, err := state.Pass(1)
				return err
			},
			want: ErrCannotPass,
		},
		{
			name: "wrong turn",
			run: func(t *testing.T, state *State) error {
				_, err := state.Play(2, parseCards(t, "C3"), false)
				return err
			},
			want: ErrWrongTurn,
		},
		{
			name: "duplicate card",
			run: func(t *testing.T, state *State) error {
				cards := parseCards(t, "C3", "C3")
				_, err := state.Play(1, cards, false)
				return err
			},
			want: ErrInvalidPattern,
		},
		{
			name: "does not beat leader",
			prepare: func(t *testing.T, state *State) {
				if _, err := state.Play(1, parseCards(t, "C4"), false); err != nil {
					t.Fatal(err)
				}
			},
			run: func(t *testing.T, state *State) error {
				_, err := state.Play(2, parseCards(t, "C3"), false)
				return err
			},
			want: ErrDoesNotBeat,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state := mustState(t, 1)
			if test.prepare != nil {
				test.prepare(t, state)
			}
			before := state.Snapshot()
			if err := test.run(t, state); !errors.Is(err, test.want) {
				t.Fatalf("error=%v want=%v", err, test.want)
			}
			if after := state.Snapshot(); !reflect.DeepEqual(after, before) {
				t.Fatalf("state mutated: before=%#v after=%#v", before, after)
			}
		})
	}
}

func TestStateBombOverridesOrdinaryLeader(t *testing.T) {
	state := mustState(t, 1)
	if _, err := state.Play(1, parseCards(t, "C2"), false); err != nil {
		t.Fatal(err)
	}
	snapshot, err := state.Play(2, parseCards(t, "C3", "D3", "H3", "S3"), false)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.LeadingSeat != 2 || snapshot.LeadingPattern == nil || snapshot.LeadingPattern.Kind != KindBomb || snapshot.CurrentSeat != 3 {
		t.Fatalf("snapshot=%#v", snapshot)
	}
}

func TestStateTwoPassesResetTrickToLeadingSeat(t *testing.T) {
	state := mustState(t, 1)
	if _, err := state.Play(1, parseCards(t, "C3"), false); err != nil {
		t.Fatal(err)
	}
	firstPass, err := state.Pass(2)
	if err != nil {
		t.Fatal(err)
	}
	if firstPass.CurrentSeat != 3 || firstPass.LeadingSeat != 1 || firstPass.PassCount != 1 {
		t.Fatalf("firstPass=%#v", firstPass)
	}
	secondPass, err := state.Pass(3)
	if err != nil {
		t.Fatal(err)
	}
	if secondPass.CurrentSeat != 1 || secondPass.LeadingSeat != 0 || secondPass.LeadingPattern != nil || len(secondPass.LeadingCards) != 0 || secondPass.PassCount != 0 {
		t.Fatalf("secondPass=%#v", secondPass)
	}
	if len(secondPass.History) != 3 || secondPass.History[1].Type != ActionPass || secondPass.History[2].Type != ActionPass {
		t.Fatalf("history=%#v", secondPass.History)
	}
	if _, err := state.Pass(1); !errors.Is(err, ErrCannotPass) {
		t.Fatalf("empty trick pass error=%v", err)
	}
}

func TestStateWinningPlayCompletesPlaying(t *testing.T) {
	state := mustState(t, 2)
	snapshot, err := state.Play(2, parseCards(t, "YJ"), true)
	if err != nil {
		t.Fatal(err)
	}
	if !snapshot.Complete || snapshot.WinnerSeat != 2 || snapshot.CurrentSeat != 0 {
		t.Fatalf("snapshot=%#v", snapshot)
	}
	if _, err := state.Play(2, parseCards(t, "C3"), true); !errors.Is(err, ErrPlayingComplete) {
		t.Fatalf("post-complete play error=%v", err)
	}
	if _, err := state.Pass(2); !errors.Is(err, ErrPlayingComplete) {
		t.Fatalf("post-complete pass error=%v", err)
	}
}

func TestStateSnapshotsAreCopyIsolated(t *testing.T) {
	state := mustState(t, 1)
	if _, err := state.Play(1, parseCards(t, "C3"), false); err != nil {
		t.Fatal(err)
	}
	first := state.Snapshot()
	first.LeadingCards[0] = parseCards(t, "YJ")[0]
	first.History[0].Cards[0] = parseCards(t, "XJ")[0]
	first.History[0].Pattern.Kind = KindRocket
	first.LeadingPattern.Kind = KindBomb

	second := state.Snapshot()
	if second.LeadingPattern == nil || second.LeadingPattern.Kind != KindSingle || second.History[0].Pattern == nil || second.History[0].Pattern.Kind != KindSingle {
		t.Fatalf("second=%#v", second)
	}
	if code, _ := second.LeadingCards[0].Code(); code != "C3" {
		t.Fatalf("leading card=%s", code)
	}
	if code, _ := second.History[0].Cards[0].Code(); code != "C3" {
		t.Fatalf("history card=%s", code)
	}
}

func TestNewStateRejectsInvalidFirstSeat(t *testing.T) {
	for _, seat := range []uint8{0, 4} {
		if _, err := NewState(seat); !errors.Is(err, ErrInvalidSeat) {
			t.Fatalf("seat=%d error=%v", seat, err)
		}
	}
}

func mustState(t *testing.T, firstSeat uint8) *State {
	t.Helper()
	state, err := NewState(firstSeat)
	if err != nil {
		t.Fatal(err)
	}
	return state
}
