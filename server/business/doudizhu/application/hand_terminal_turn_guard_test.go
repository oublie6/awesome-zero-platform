package application

import (
	"errors"
	"testing"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/livehand"
)

func TestGuardCommittedTerminalTurnRejectsConflictingCancellation(t *testing.T) {
	cases := []struct {
		name, phase, command, reason string
		want error
	}{
		{"cancel final winning play", livehand.PhaseCompleted, CommandHandCancel, "participant_cancelled", domain.ErrHandTerminal},
		{"expire final winning play", livehand.PhaseCompleted, CommandHandExpire, "playing_timeout", domain.ErrHandTerminal},
		{"abort final winning play", livehand.PhaseCompleted, CommandHandAbort, "no_landlord", domain.ErrHandTerminal},
		{"allow genuine completion", livehand.PhaseCompleted, CommandHandComplete, "", nil},
		{"cancel all no bids", livehand.PhaseNoLandlord, CommandHandCancel, "participant_cancelled", domain.ErrHandTerminal},
		{"expire all no bids", livehand.PhaseNoLandlord, CommandHandExpire, "bidding_timeout", domain.ErrHandTerminal},
		{"incorrect abort reason", livehand.PhaseNoLandlord, CommandHandAbort, "operator_cancel", domain.ErrHandTerminal},
		{"allow no landlord", livehand.PhaseNoLandlord, CommandHandAbort, "no_landlord", nil},
		{"complete unfinished game", livehand.PhasePlaying, CommandHandComplete, "", domain.ErrWrongPhase},
		{"cancel ongoing game", livehand.PhasePlaying, CommandHandCancel, "participant_cancelled", nil},
		{"allow legacy without clock", "", CommandHandCancel, "participant_cancelled", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := guardCommittedTerminalTurn(tc.phase, tc.command, TerminateHandInput{ReasonCode: tc.reason})
			if !errors.Is(err, tc.want) || (tc.want == nil && err != nil) {
				t.Fatalf("phase %q command %q reason %q: got %v want %v", tc.phase, tc.command, tc.reason, err, tc.want)
			}
		})
	}
}
