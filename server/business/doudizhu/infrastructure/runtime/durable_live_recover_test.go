package runtime

import (
	"errors"
	"testing"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/livehand"
	"github.com/oublie6/awesome-zero-platform/server/business/gamecore"
)

func TestRecoverLivePhasesAndFailClosedMismatch(t *testing.T) {
	for _, phase := range []domain.HandPhase{domain.HandBidding, domain.HandPlaying, domain.HandSettling} {
		if !recoveryPhaseNeedsReplay(phase) {
			t.Fatalf("acknowledged %s hand would be discarded", phase)
		}
	}
	for _, phase := range []domain.HandPhase{domain.HandCompleted, domain.HandAborted, domain.HandCancelled, domain.HandExpired, domain.HandDealing, domain.HandWaitingPublicBeacon} {
		if recoveryPhaseNeedsReplay(phase) {
			t.Fatalf("non-recoverable phase %s would be resurrected", phase)
		}
	}
	tests := []struct{
		name string
		persisted domain.HandPhase
		replayed string
		terminal bool
		allowed bool
	}{
		{"bidding-continues",domain.HandBidding,livehand.PhaseBidding,false,true},
		{"last-bid-committed-before-domain-transition",domain.HandBidding,livehand.PhasePlaying,false,true},
		{"crash-in-playing",domain.HandPlaying,livehand.PhasePlaying,false,true},
		{"playing-replayed-only-bids",domain.HandPlaying,livehand.PhaseBidding,false,false},
		{"settling-without-final",domain.HandSettling,livehand.PhasePlaying,false,false},
		{"completed-while-domain-playing",domain.HandPlaying,livehand.PhaseCompleted,true,true},
		{"completed-while-domain-bidding",domain.HandBidding,livehand.PhaseCompleted,true,true},
		{"settling-final",domain.HandSettling,livehand.PhaseCompleted,true,true},
		{"terminal-not-marked",domain.HandPlaying,livehand.PhaseCompleted,false,false},
		{"invalid-terminated-phase",domain.HandCompleted,livehand.PhaseCompleted,true,false},
	}
	for _,tt:=range tests {
		t.Run(tt.name,func(t *testing.T) {
			err:=validateReplayedPhase(tt.persisted,tt.replayed,tt.terminal)
			if tt.allowed && err!=nil {t.Fatalf("unexpected rejection: %v",err)}
			if !tt.allowed && !errors.Is(err,gamecore.ErrVerificationFailed) {
				t.Fatalf("unsafe recovery accepted: %v",err)
			}
		})
	}
}
