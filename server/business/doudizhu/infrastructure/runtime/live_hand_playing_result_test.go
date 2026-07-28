package runtime

import (
	"errors"
	"testing"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/livehand"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/playing"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/settlement"
	"github.com/oublie6/awesome-zero-platform/server/business/gamecore"
)

func TestValidatePlayResultAcceptsNonTerminalAndCompletedContracts(t *testing.T) {
	nonTerminal := livehand.PlayResult{
		Version:      livehand.PlayResultVersion,
		HandID:       "hand-1",
		StateVersion: 2,
		Phase:        livehand.PhasePlaying,
		Playing:      playing.Snapshot{Version: playing.StateVersion, CurrentSeat: 2},
	}
	if err := validatePlayResult("hand-1", gamecore.CommandOutcome{Version: 2}, nonTerminal); err != nil {
		t.Fatal(err)
	}

	completedSettlement := settlement.Result{Version: settlement.RulesVersion, WinnerSeat: 1}
	completed := livehand.PlayResult{
		Version:      livehand.PlayResultVersion,
		HandID:       "hand-1",
		StateVersion: 3,
		Phase:        livehand.PhaseCompleted,
		Playing:      playing.Snapshot{Version: playing.StateVersion, Complete: true, WinnerSeat: 1},
		WinnerSeat:   1,
		Settlement:   &completedSettlement,
	}
	if err := validatePlayResult("hand-1", gamecore.CommandOutcome{Version: 3, Terminal: true, FinalPayload: []byte("final")}, completed); err != nil {
		t.Fatal(err)
	}
}

func TestValidatePlayResultRejectsContractMismatches(t *testing.T) {
	base := livehand.PlayResult{
		Version:      livehand.PlayResultVersion,
		HandID:       "hand-1",
		StateVersion: 2,
		Phase:        livehand.PhasePlaying,
		Playing:      playing.Snapshot{Version: playing.StateVersion, CurrentSeat: 2},
	}
	tests := []struct {
		name    string
		outcome gamecore.CommandOutcome
		result  livehand.PlayResult
	}{
		{name: "wrong hand", outcome: gamecore.CommandOutcome{Version: 2}, result: withHand(base, "other")},
		{name: "nonterminal carries final", outcome: gamecore.CommandOutcome{Version: 2, FinalPayload: []byte("bad")}, result: base},
		{name: "nonterminal reports winner", outcome: gamecore.CommandOutcome{Version: 2}, result: withWinner(base, 1)},
		{name: "completed not terminal", outcome: gamecore.CommandOutcome{Version: 2}, result: completedResult(2, 1)},
		{name: "completed missing final payload", outcome: gamecore.CommandOutcome{Version: 2, Terminal: true}, result: completedResult(2, 1)},
		{name: "completed missing settlement", outcome: gamecore.CommandOutcome{Version: 2, Terminal: true, FinalPayload: []byte("final")}, result: completedWithoutSettlement(2, 1)},
		{name: "completed settlement winner mismatch", outcome: gamecore.CommandOutcome{Version: 2, Terminal: true, FinalPayload: []byte("final")}, result: completedResult(2, 2)},
		{name: "unknown phase", outcome: gamecore.CommandOutcome{Version: 2}, result: withPhase(base, "SETTLING")},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := validatePlayResult(domain.HandID("hand-1"), test.outcome, test.result); !errors.Is(err, gamecore.ErrVerificationFailed) {
				t.Fatalf("error=%v", err)
			}
		})
	}
}

func completedResult(version uint64, settlementWinner uint8) livehand.PlayResult {
	result := completedWithoutSettlement(version, 1)
	value := settlement.Result{Version: settlement.RulesVersion, WinnerSeat: settlementWinner}
	result.Settlement = &value
	return result
}

func completedWithoutSettlement(version uint64, winner uint8) livehand.PlayResult {
	return livehand.PlayResult{
		Version:      livehand.PlayResultVersion,
		HandID:       "hand-1",
		StateVersion: version,
		Phase:        livehand.PhaseCompleted,
		Playing:      playing.Snapshot{Version: playing.StateVersion, Complete: true, WinnerSeat: winner},
		WinnerSeat:   winner,
	}
}

func withHand(result livehand.PlayResult, handID string) livehand.PlayResult {
	result.HandID = handID
	return result
}

func withWinner(result livehand.PlayResult, winner uint8) livehand.PlayResult {
	result.WinnerSeat = winner
	return result
}

func withPhase(result livehand.PlayResult, phase string) livehand.PlayResult {
	result.Phase = phase
	return result
}
