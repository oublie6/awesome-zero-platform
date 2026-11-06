package application

import (
	"context"
	"testing"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
)

func TestRoomQueryDoesNotLeakSeatsToNonMembers(t *testing.T) {
	room, _, err := domain.NewRoom("private-room", "account-owner")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := room.Join("account-second", room.Snapshot().Version); err != nil {
		t.Fatal(err)
	}
	store := newTestStore()
	store.state.rooms["private-room"] = room.Snapshot()
	store.state.sequences[testSequenceKey(domain.AggregateRoom, "private-room", "account-second")] = 13
	service := &Service{store: store}
	member, err := service.GetRoomView(context.Background(), "account-second", "private-room")
	if err != nil {
		t.Fatal(err)
	}
	if !member.Member || len(member.Seats) != 3 || member.Occupied != 2 || member.Version != room.Snapshot().Version || member.LastClientSequence != 13 {
		t.Fatalf("unexpected authorized view: %+v", member)
	}
	if member.Seats[0].AccountID != "account-owner" || member.Seats[1].AccountID != "account-second" {
		t.Fatalf("wrong member seats: %+v", member.Seats)
	}
	outsider, err := service.GetRoomView(context.Background(), "account-third", "private-room")
	if err != nil {
		t.Fatal(err)
	}
	if outsider.Member || len(outsider.Seats) != 0 || outsider.ActiveHandID != "" ||
		outsider.Version != member.Version || outsider.Occupied != 2 || outsider.LastClientSequence != 0 {
		t.Fatalf("outsider received private room data: %+v", outsider)
	}
	if _, err := service.GetRoomView(context.Background(), "", "private-room"); err == nil {
		t.Fatal("anonymous query accepted")
	}
	if _, err := service.GetRoomView(context.Background(), "account-third", " private-room"); err == nil {
		t.Fatal("malformed room accepted")
	}
}

func TestRoomViewLastHandIsScopedToHistoricalParticipantsAfterSeatReplacement(t *testing.T) {
	ctx := context.Background()
	const roomID domain.RoomID = "history-room"
	const handID domain.HandID = "history-hand"
	room, _, err := domain.NewRoom(roomID, "account-A")
	if err != nil { t.Fatal(err) }
	for _, actor := range []domain.AccountID{"account-B", "account-C"} {
		if _, err := room.Join(actor, room.Snapshot().Version); err != nil { t.Fatal(err) }
	}
	for _, actor := range []domain.AccountID{"account-A", "account-B", "account-C"} {
		if _, err := room.SetReady(actor, true, room.Snapshot().Version); err != nil { t.Fatal(err) }
	}
	if _, err := room.StartHand("account-A", handID, room.Snapshot().Version); err != nil { t.Fatal(err) }
	if _, err := room.FinishHand(handID, room.Snapshot().Version, true); err != nil { t.Fatal(err) }
	if _, err := room.Leave("account-C", room.Snapshot().Version); err != nil { t.Fatal(err) }
	if _, err := room.Join("account-D", room.Snapshot().Version); err != nil { t.Fatal(err) }
	seats := [3]domain.HandSeat{
		{Seat: 1, AccountID: "account-A"},
		{Seat: 2, AccountID: "account-B"},
		{Seat: 3, AccountID: "account-C"},
	}
	hand, _, err := domain.NewHand(handID, roomID, seats, domain.ServerCommitment{1},
		"history-reveal-key", domain.BeaconPlan{Provider:"history-beacon", Round:"history-round"})
	if err != nil { t.Fatal(err) }
	if _, err := hand.Abort("history test archive", hand.Snapshot().Version); err != nil { t.Fatal(err) }
	store := newTestStore()
	store.state.rooms[string(roomID)] = room.Snapshot()
	store.state.hands[string(handID)] = hand.Snapshot()
	service := &Service{store: store}
	for _, actor := range []domain.AccountID{"account-A","account-B"} {
		view, err := service.GetRoomView(ctx, actor, roomID)
		if err != nil || !view.Member || view.LastHandID != string(handID) {
			t.Fatalf("original hand member %s lost archive locator: %+v %v", actor, view, err)
		}
	}
	replacement, err := service.GetRoomView(ctx, "account-D", roomID)
	if err != nil || !replacement.Member || replacement.LastHandID != "" || replacement.Occupied != 3 || len(replacement.Seats) != 3 {
		t.Fatalf("new room member was offered inaccessible archive: %+v %v", replacement, err)
	}
	former, err := service.GetRoomView(ctx, "account-C", roomID)
	if err != nil || former.Member || former.LastHandID != "" || len(former.Seats) != 0 {
		t.Fatalf("departed actor retained private room projection: %+v %v", former, err)
	}
	// A legitimately pruned/absent previous hand must not block a newly
	// admitted player or make the controller attempt an impossible archive.
	delete(store.state.hands, string(handID))
	view, err := service.GetRoomView(ctx, "account-A", roomID)
	if err != nil || view.LastHandID != "" {
		t.Fatalf("missing retained archive blocks room query: %+v %v", view, err)
	}
}
