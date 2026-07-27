package domain

import (
	"errors"
	"testing"
)

func TestHandRejectsInvalidContributionsAndPhases(t *testing.T) {
	hand := newTestHand(t)
	digest1 := digest(1)
	commit1 := ComputeClientCommitment("hand-1", SeatOne, digest1)
	if _, err := hand.SubmitCommit("outsider", commit1, hand.Snapshot().Version); !errors.Is(err, ErrNotSeated) {
		t.Fatalf("outsider commit error = %v, want ErrNotSeated", err)
	}
	if _, err := hand.SubmitCommit("account-1", commit1, hand.Snapshot().Version); err != nil {
		t.Fatal(err)
	}
	if _, err := hand.SubmitCommit("account-1", commit1, hand.Snapshot().Version); !errors.Is(err, ErrDuplicateContribution) {
		t.Fatalf("duplicate commit error = %v, want ErrDuplicateContribution", err)
	}
	if _, err := hand.SubmitReveal("account-1", digest1, "record", hand.Snapshot().Version); !errors.Is(err, ErrWrongPhase) {
		t.Fatalf("early reveal error = %v, want ErrWrongPhase", err)
	}

	for index, account := range []AccountID{"account-2", "account-3"} {
		seat := Seat(index + 2)
		d := digest(byte(index + 2))
		if _, err := hand.SubmitCommit(account, ComputeClientCommitment("hand-1", seat, d), hand.Snapshot().Version); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := hand.SubmitReveal("account-1", digest(99), "record", hand.Snapshot().Version); !errors.Is(err, ErrCommitmentMismatch) {
		t.Fatalf("mismatched reveal error = %v, want ErrCommitmentMismatch", err)
	}
	before := hand.Snapshot()
	if _, err := hand.SubmitReveal("account-1", digest1, "record", before.Version-1); !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("stale reveal error = %v, want ErrVersionConflict", err)
	}
	if after := hand.Snapshot(); after.Version != before.Version || after.Contributions != before.Contributions {
		t.Fatalf("stale reveal changed state: before=%#v after=%#v", before, after)
	}
}

func TestHandRejectsBeaconOutsideLockedPlan(t *testing.T) {
	hand := newTestHand(t)
	advanceHandToBeacon(t, hand)
	before := hand.Snapshot()

	wrong := BeaconValue{
		Provider: "another-provider",
		Round:    beaconPlan().Round,
		Digest:   BeaconDigest(digest(88)),
		ProofRef: "proof-ref",
	}
	if _, err := hand.LockPublicBeacon(wrong, before.Version); !errors.Is(err, ErrBeaconMismatch) {
		t.Fatalf("LockPublicBeacon(wrong plan) error = %v, want ErrBeaconMismatch", err)
	}
	after := hand.Snapshot()
	if after.Version != before.Version || after.Phase != before.Phase || after.Beacon != nil {
		t.Fatalf("mismatched beacon changed state: before=%#v after=%#v", before, after)
	}
}

func TestHandTerminalStates(t *testing.T) {
	hand := newTestHand(t)
	if _, err := hand.Cancel("players did not commit", hand.Snapshot().Version); err != nil {
		t.Fatal(err)
	}
	if got := hand.Snapshot().Phase; got != HandCancelled {
		t.Fatalf("phase = %s, want %s", got, HandCancelled)
	}
	if _, err := hand.Expire("again", hand.Snapshot().Version); !errors.Is(err, ErrHandTerminal) {
		t.Fatalf("Expire(cancelled) error = %v, want ErrHandTerminal", err)
	}

	aborted := newTestHandWithID(t, "hand-aborted")
	if events, err := aborted.Abort("integrity failure", aborted.Snapshot().Version); err != nil {
		t.Fatal(err)
	} else if len(events) != 2 || events[0].Version+1 != events[1].Version {
		t.Fatalf("unexpected abort events: %#v", events)
	}

	expired := newTestHandWithID(t, "hand-expired")
	if _, err := expired.Expire("commit timeout", expired.Snapshot().Version); err != nil {
		t.Fatal(err)
	}
	if got := expired.Snapshot().Phase; got != HandExpired {
		t.Fatalf("phase = %s, want %s", got, HandExpired)
	}
}

func TestCommitmentIsBoundToHandAndSeat(t *testing.T) {
	d := digest(42)
	base := ComputeClientCommitment("hand-a", SeatOne, d)
	if base == ComputeClientCommitment("hand-b", SeatOne, d) {
		t.Fatal("commitment must be bound to hand ID")
	}
	if base == ComputeClientCommitment("hand-a", SeatTwo, d) {
		t.Fatal("commitment must be bound to seat")
	}
}

func TestNewHandValidatesThreeUniqueFixedSeats(t *testing.T) {
	seats := testSeats()
	seats[2] = HandSeat{Seat: SeatTwo, AccountID: "account-3"}
	_, _, err := NewHand("hand-invalid", "room-1", seats, serverCommitment(), "key-1", beaconPlan())
	if !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("NewHand(duplicate seat) error = %v, want ErrInvalidArgument", err)
	}
}

func advanceHandToBeacon(t *testing.T, hand *Hand) {
	t.Helper()
	accounts := []AccountID{"account-1", "account-2", "account-3"}
	digests := []ContributionDigest{digest(51), digest(52), digest(53)}
	for index, account := range accounts {
		commitment := ComputeClientCommitment(hand.Snapshot().ID, Seat(index+1), digests[index])
		if _, err := hand.SubmitCommit(account, commitment, hand.Snapshot().Version); err != nil {
			t.Fatalf("SubmitCommit(%s) error = %v", account, err)
		}
	}
	for index, account := range accounts {
		if _, err := hand.SubmitReveal(account, digests[index], "record-"+string(rune('1'+index)), hand.Snapshot().Version); err != nil {
			t.Fatalf("SubmitReveal(%s) error = %v", account, err)
		}
	}
	if hand.Snapshot().Phase != HandWaitingPublicBeacon {
		t.Fatalf("phase = %s, want %s", hand.Snapshot().Phase, HandWaitingPublicBeacon)
	}
}

func newTestHand(t *testing.T) *Hand {
	t.Helper()
	return newTestHandWithID(t, "hand-1")
}

func newTestHandWithID(t *testing.T, id HandID) *Hand {
	t.Helper()
	hand, events, err := NewHand(id, "room-1", testSeats(), serverCommitment(), "reveal-key-2026-07", beaconPlan())
	if err != nil {
		t.Fatalf("NewHand() error = %v", err)
	}
	if len(events) != 1 || events[0].Version != 1 || hand.Snapshot().Phase != HandFairnessCommitting {
		t.Fatalf("unexpected new hand: events=%#v snapshot=%#v", events, hand.Snapshot())
	}
	return hand
}

func testSeats() [3]HandSeat {
	return [3]HandSeat{
		{Seat: SeatOne, AccountID: "account-1"},
		{Seat: SeatTwo, AccountID: "account-2"},
		{Seat: SeatThree, AccountID: "account-3"},
	}
}

func serverCommitment() ServerCommitment {
	var value ServerCommitment
	value[0] = 1
	return value
}

func beaconPlan() BeaconPlan {
	return BeaconPlan{Provider: "nist-randomness-beacon", Round: "2026-07-28T00:00:00Z"}
}

func digest(seed byte) ContributionDigest {
	var value ContributionDigest
	for index := range value {
		value[index] = seed + byte(index)
	}
	return value
}
