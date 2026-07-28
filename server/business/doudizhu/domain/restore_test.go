package domain

import (
	"errors"
	"testing"
)

func TestRestoreRoomPreservesValidatedSnapshotWithoutEvents(t *testing.T) {
	room, _, err := NewRoom("room-restore", "owner")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := room.Join("guest-2", room.Snapshot().Version); err != nil {
		t.Fatal(err)
	}
	snapshot := room.Snapshot()
	restored, err := RestoreRoom(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if got := restored.Snapshot(); got != snapshot {
		t.Fatalf("restored=%#v want=%#v", got, snapshot)
	}
}

func TestRestoreRoomRejectsInconsistentSnapshot(t *testing.T) {
	room, _, _ := NewRoom("room-restore", "owner")
	snapshot := room.Snapshot()
	snapshot.Status = RoomReady
	if _, err := RestoreRoom(snapshot); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("RestoreRoom error=%v", err)
	}
}

func TestRestoreCompletedHandWithoutTerminationReason(t *testing.T) {
	hand := newRestoreTestHand(t)
	advanceRestoreHandToBeacon(t, hand)
	var beacon BeaconDigest
	beacon[0] = 9
	if _, err := hand.LockPublicBeacon(BeaconValue{Provider: restoreBeaconPlan().Provider, Round: restoreBeaconPlan().Round, Digest: beacon, ProofRef: "proof"}, hand.Snapshot().Version); err != nil {
		t.Fatal(err)
	}
	if _, err := hand.MarkDealt(hand.Snapshot().Version); err != nil {
		t.Fatal(err)
	}
	if _, err := hand.StartPlaying(hand.Snapshot().Version); err != nil {
		t.Fatal(err)
	}
	if _, err := hand.StartSettling(hand.Snapshot().Version); err != nil {
		t.Fatal(err)
	}
	if _, err := hand.Complete(hand.Snapshot().Version); err != nil {
		t.Fatal(err)
	}
	snapshot := hand.Snapshot()
	if snapshot.TerminationReason != "" {
		t.Fatalf("completion reason=%q", snapshot.TerminationReason)
	}
	restored, err := RestoreHand(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if got := restored.Snapshot(); got.Phase != HandCompleted || got.Version != snapshot.Version {
		t.Fatalf("restored=%#v", got)
	}
}

func TestRestoreHandRejectsRevealWithoutRecord(t *testing.T) {
	hand := newRestoreTestHand(t)
	snapshot := hand.Snapshot()
	snapshot.Phase = HandFairnessRevealing
	for index := range snapshot.Contributions {
		digest := ContributionDigest{byte(index + 1)}
		snapshot.Contributions[index].Committed = true
		snapshot.Contributions[index].Commitment = ComputeClientCommitment(snapshot.ID, snapshot.Contributions[index].Seat, digest)
	}
	snapshot.Contributions[0].Revealed = true
	snapshot.Contributions[0].Digest = ContributionDigest{1}
	if _, err := RestoreHand(snapshot); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("RestoreHand error=%v", err)
	}
}

func newRestoreTestHand(t *testing.T) *Hand {
	t.Helper()
	seats := [3]HandSeat{
		{Seat: SeatOne, AccountID: "restore-account-1"},
		{Seat: SeatTwo, AccountID: "restore-account-2"},
		{Seat: SeatThree, AccountID: "restore-account-3"},
	}
	var server ServerCommitment
	server[0] = 1
	hand, _, err := NewHand("restore-hand", "restore-room", seats, server, "restore-key", restoreBeaconPlan())
	if err != nil {
		t.Fatal(err)
	}
	return hand
}

func restoreBeaconPlan() BeaconPlan {
	return BeaconPlan{Provider: "restore-beacon", Round: "restore-round"}
}

func advanceRestoreHandToBeacon(t *testing.T, hand *Hand) {
	t.Helper()
	actors := []AccountID{"restore-account-1", "restore-account-2", "restore-account-3"}
	digests := []ContributionDigest{{1}, {2}, {3}}
	for index, actor := range actors {
		commitment := ComputeClientCommitment(hand.Snapshot().ID, Seat(index+1), digests[index])
		if _, err := hand.SubmitCommit(actor, commitment, hand.Snapshot().Version); err != nil {
			t.Fatal(err)
		}
	}
	for index, actor := range actors {
		if _, err := hand.SubmitReveal(actor, digests[index], "restore-record-"+string(rune('1'+index)), hand.Snapshot().Version); err != nil {
			t.Fatal(err)
		}
	}
}
