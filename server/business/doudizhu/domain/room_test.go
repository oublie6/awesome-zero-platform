package domain

import (
	"errors"
	"testing"
)

func TestRoomLifecycle(t *testing.T) {
	room, events, err := NewRoom("room-1", "owner")
	if err != nil {
		t.Fatalf("NewRoom() error = %v", err)
	}
	if len(events) != 1 || events[0].Version != 1 || events[0].Name != EventRoomCreated {
		t.Fatalf("unexpected create events: %#v", events)
	}

	version := room.Snapshot().Version
	if _, err := room.Join("player-2", version); err != nil {
		t.Fatalf("Join(player-2) error = %v", err)
	}
	version = room.Snapshot().Version
	if _, err := room.Join("player-3", version); err != nil {
		t.Fatalf("Join(player-3) error = %v", err)
	}
	if _, err := room.Join("player-4", room.Snapshot().Version); !errors.Is(err, ErrRoomFull) {
		t.Fatalf("Join(player-4) error = %v, want ErrRoomFull", err)
	}
	if _, err := room.Join("player-2", room.Snapshot().Version); !errors.Is(err, ErrAlreadySeated) {
		t.Fatalf("duplicate Join() error = %v, want ErrAlreadySeated", err)
	}

	for _, account := range []AccountID{"owner", "player-2", "player-3"} {
		if _, err := room.SetReady(account, true, room.Snapshot().Version); err != nil {
			t.Fatalf("SetReady(%s) error = %v", account, err)
		}
	}
	if got := room.Snapshot().Status; got != RoomReady {
		t.Fatalf("status = %s, want %s", got, RoomReady)
	}
	if _, err := room.StartHand("player-2", "hand-1", room.Snapshot().Version); !errors.Is(err, ErrForbidden) {
		t.Fatalf("StartHand(non-owner) error = %v, want ErrForbidden", err)
	}

	startEvents, err := room.StartHand("owner", "hand-1", room.Snapshot().Version)
	if err != nil {
		t.Fatalf("StartHand(owner) error = %v", err)
	}
	if len(startEvents) != 1 || startEvents[0].Name != EventRoomHandStarted {
		t.Fatalf("unexpected start events: %#v", startEvents)
	}
	snapshot := room.Snapshot()
	if snapshot.Status != RoomHandActive || snapshot.ActiveHandID != "hand-1" {
		t.Fatalf("unexpected active room snapshot: %#v", snapshot)
	}
	for _, seat := range snapshot.Seats {
		if seat.Ready {
			t.Fatalf("readiness should reset after hand start: %#v", snapshot.Seats)
		}
	}
	if _, err := room.Leave("player-2", snapshot.Version); !errors.Is(err, ErrHandActive) {
		t.Fatalf("Leave(active hand) error = %v, want ErrHandActive", err)
	}
	if _, err := room.FinishHand("other-hand", snapshot.Version); !errors.Is(err, ErrHandNotActive) {
		t.Fatalf("FinishHand(other) error = %v, want ErrHandNotActive", err)
	}
	if _, err := room.FinishHand("hand-1", snapshot.Version); err != nil {
		t.Fatalf("FinishHand() error = %v", err)
	}
	if got := room.Snapshot().Status; got != RoomWaitingPlayers {
		t.Fatalf("status after finish = %s, want %s", got, RoomWaitingPlayers)
	}
}

func TestRoomOwnerAndVersionInvariants(t *testing.T) {
	room, _, err := NewRoom("room-2", "owner")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := room.Join("guest", room.Snapshot().Version); err != nil {
		t.Fatal(err)
	}
	if _, err := room.Leave("owner", room.Snapshot().Version); !errors.Is(err, ErrOwnerMustRemain) {
		t.Fatalf("owner Leave() error = %v, want ErrOwnerMustRemain", err)
	}
	before := room.Snapshot()
	if _, err := room.SetReady("guest", true, before.Version-1); !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("stale SetReady() error = %v, want ErrVersionConflict", err)
	}
	after := room.Snapshot()
	if after.Version != before.Version || after.Seats != before.Seats {
		t.Fatalf("stale command changed state: before=%#v after=%#v", before, after)
	}
	if _, err := room.Leave("guest", after.Version); err != nil {
		t.Fatal(err)
	}
	if _, err := room.Leave("owner", room.Snapshot().Version); err != nil {
		t.Fatal(err)
	}
	if got := room.Snapshot().Status; got != RoomClosed {
		t.Fatalf("status = %s, want %s", got, RoomClosed)
	}
}

func TestRoomHandSeatsAreFixed(t *testing.T) {
	room, _, _ := NewRoom("room-3", "owner")
	_, _ = room.Join("player-2", room.Snapshot().Version)
	_, _ = room.Join("player-3", room.Snapshot().Version)
	seats, err := room.HandSeats()
	if err != nil {
		t.Fatal(err)
	}
	want := [3]HandSeat{
		{Seat: SeatOne, AccountID: "owner"},
		{Seat: SeatTwo, AccountID: "player-2"},
		{Seat: SeatThree, AccountID: "player-3"},
	}
	if seats != want {
		t.Fatalf("HandSeats() = %#v, want %#v", seats, want)
	}
}
