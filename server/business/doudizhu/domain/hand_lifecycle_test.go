package domain

import (
	"errors"
	"testing"
)

func TestHandFairnessAndGameplayLifecycle(t *testing.T) {
	hand := newTestHand(t)
	accounts := []AccountID{"account-1", "account-2", "account-3"}
	digests := []ContributionDigest{digest(11), digest(22), digest(33)}

	for index, account := range accounts {
		seat := Seat(index + 1)
		commitment := ComputeClientCommitment("hand-1", seat, digests[index])
		events, err := hand.SubmitCommit(account, commitment, hand.Snapshot().Version)
		if err != nil {
			t.Fatalf("SubmitCommit(%s) error = %v", account, err)
		}
		if index < 2 && len(events) != 1 {
			t.Fatalf("commit events = %d, want 1", len(events))
		}
		if index == 2 && (len(events) != 2 || events[0].Version+1 != events[1].Version) {
			t.Fatalf("final commit should emit sequential accept and phase events: %#v", events)
		}
	}
	if got := hand.Snapshot().Phase; got != HandFairnessRevealing {
		t.Fatalf("phase = %s, want %s", got, HandFairnessRevealing)
	}

	for index, account := range accounts {
		events, err := hand.SubmitReveal(account, digests[index], "encrypted-record-"+string(rune('1'+index)), hand.Snapshot().Version)
		if err != nil {
			t.Fatalf("SubmitReveal(%s) error = %v", account, err)
		}
		if index == 2 && (len(events) != 2 || hand.Snapshot().Phase != HandWaitingPublicBeacon) {
			t.Fatalf("final reveal did not advance phase: events=%#v snapshot=%#v", events, hand.Snapshot())
		}
	}

	beacon := BeaconValue{
		Provider: "nist-randomness-beacon",
		Round:    "2026-07-28T00:00:00Z",
		Digest:   BeaconDigest(digest(77)),
		ProofRef: "beacon-proof-1",
	}
	beaconEvents, err := hand.LockPublicBeacon(beacon, hand.Snapshot().Version)
	if err != nil {
		t.Fatalf("LockPublicBeacon() error = %v", err)
	}
	if len(beaconEvents) != 2 || hand.Snapshot().Phase != HandDealing {
		t.Fatalf("unexpected beacon result: events=%#v snapshot=%#v", beaconEvents, hand.Snapshot())
	}

	steps := []struct {
		name string
		call func(uint64) ([]Event, error)
		want HandPhase
	}{
		{"dealt", hand.MarkDealt, HandBidding},
		{"playing", hand.StartPlaying, HandPlaying},
		{"settling", hand.StartSettling, HandSettling},
		{"completed", hand.Complete, HandCompleted},
	}
	for _, step := range steps {
		if _, err := step.call(hand.Snapshot().Version); err != nil {
			t.Fatalf("%s transition error = %v", step.name, err)
		}
		if got := hand.Snapshot().Phase; got != step.want {
			t.Fatalf("%s phase = %s, want %s", step.name, got, step.want)
		}
	}
	if _, err := hand.Abort("too late", hand.Snapshot().Version); !errors.Is(err, ErrHandTerminal) {
		t.Fatalf("Abort(completed) error = %v, want ErrHandTerminal", err)
	}
}
