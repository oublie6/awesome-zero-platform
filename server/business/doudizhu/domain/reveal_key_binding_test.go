package domain

import (
	"testing"
	"time"
)

func TestHandLocksRevealKeyIDHashAndBindTime(t *testing.T) {
	seats := [3]HandSeat{{Seat: SeatOne, AccountID: "a"}, {Seat: SeatTwo, AccountID: "b"}, {Seat: SeatThree, AccountID: "c"}}
	serverCommitment := ServerCommitment{1}
	hash := RevealPublicKeyHash{9, 8, 7}
	boundAt := time.Date(2026, 7, 28, 4, 0, 0, 0, time.UTC)
	hand, events, err := NewHand("hand-1", "room-1", seats, serverCommitment, "reveal-key-1", BeaconPlan{Provider: "provider", Round: "round"}, RevealKeyBinding{PublicKeySHA256: hash, BoundAt: boundAt})
	if err != nil {
		t.Fatal(err)
	}
	snapshot := hand.Snapshot()
	if snapshot.RevealKeyID != "reveal-key-1" || snapshot.RevealPublicKeySHA256 != hash || !snapshot.RevealKeyBoundAt.Equal(boundAt) {
		t.Fatalf("snapshot=%#v", snapshot)
	}
	payload, ok := events[0].Payload.(HandCreatedPayload)
	if !ok || payload.RevealPublicKeySHA256 != hash || !payload.RevealKeyBoundAt.Equal(boundAt) {
		t.Fatalf("event=%#v", events[0])
	}
	restored, err := RestoreHand(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if restored.Snapshot().RevealPublicKeySHA256 != hash {
		t.Fatal("restored key hash changed")
	}
}

func TestRestoreHandRejectsMissingRevealKeyHash(t *testing.T) {
	seats := [3]HandSeat{{Seat: SeatOne, AccountID: "a"}, {Seat: SeatTwo, AccountID: "b"}, {Seat: SeatThree, AccountID: "c"}}
	hand, _, err := NewHand("hand-1", "room-1", seats, ServerCommitment{1}, "reveal-key-1", BeaconPlan{Provider: "provider", Round: "round"})
	if err != nil {
		t.Fatal(err)
	}
	snapshot := hand.Snapshot()
	snapshot.RevealPublicKeySHA256 = RevealPublicKeyHash{}
	if _, err := RestoreHand(snapshot); err == nil {
		t.Fatal("missing reveal key hash was accepted")
	}
}
