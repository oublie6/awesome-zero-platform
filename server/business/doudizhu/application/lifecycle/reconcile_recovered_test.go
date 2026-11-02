package lifecycle

import (
	"context"
	"testing"
	"time"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/livehand"
)

func TestRestartReconciliationResumesPlayingAndFinishesCommittedFinal(t *testing.T) {
	hands := &testHands{snapshots: map[domain.HandID]domain.HandSnapshot{
		"play": testHand("play", "player-1", domain.HandBidding, 8),
		"done": testHand("done", "player-2", domain.HandPlaying, 13),
		"bid":  testHand("bid", "player-3", domain.HandBidding, 2),
	}}
	term := &phaseTestTerminal{testTerminal: &testTerminal{}, hands: hands}
	supervisor := newTestSupervisor(t, hands, term, &testClock{now: time.Unix(1800000000, 0)}, &testIDs{prefix: "recover"})
	if err := supervisor.ReconcileRecovered(context.Background(), map[domain.HandID]string{
		"play": livehand.PhasePlaying,
		"done": livehand.PhaseCompleted,
		"bid":  livehand.PhaseBidding,
	}); err != nil {
		t.Fatal(err)
	}
	if hands.snapshots["play"].Phase != domain.HandPlaying || hands.snapshots["done"].Phase != domain.HandCompleted ||
		len(term.commands) != 3 {
		t.Fatalf("recovery not durable, commands=%+v", term.commands)
	}
	if _, ok := supervisor.entries["play"]; !ok {
		t.Fatal("restored playing deadline missing")
	}
	if _, ok := supervisor.entries["bid"]; !ok {
		t.Fatal("restored bidding deadline missing")
	}
	if _, ok := supervisor.entries["done"]; ok {
		t.Fatal("finished hand timer still running")
	}
}
func TestRestartFailsClosedOnImpossiblePhase(t *testing.T) {
	hands := &testHands{snapshots: map[domain.HandID]domain.HandSnapshot{
		"invalid": testHand("invalid", "owner", domain.HandPlaying, 9),
	}}
	term := &phaseTestTerminal{testTerminal: &testTerminal{}, hands: hands}
	supervisor := newTestSupervisor(t, hands, term, &testClock{now: time.Unix(1800000000, 0)}, &testIDs{prefix: "recover"})
	if err := supervisor.ReconcileRecovered(context.Background(), map[domain.HandID]string{"invalid": livehand.PhaseBidding}); err == nil {
		t.Fatal("recovery started impossible bidding phase")
	}
}
