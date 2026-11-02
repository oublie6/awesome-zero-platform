package application

import (
	"context"
	"testing"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
)

func TestRoomQueryDoesNotLeakSeatsToNonMembers(t *testing.T) {
	room, _, err := domain.NewRoom("private-room", "account-owner")
	if err != nil { t.Fatal(err) }
	if _, err := room.Join("account-second", room.Snapshot().Version); err != nil { t.Fatal(err) }
	store := newTestStore()
	store.state.rooms["private-room"] = room.Snapshot()
	service := &Service{store:store}
	member, err := service.GetRoomView(context.Background(), "account-second", "private-room")
	if err != nil { t.Fatal(err) }
	if !member.Member || len(member.Seats) != 3 || member.Occupied != 2 || member.Version != room.Snapshot().Version {
		t.Fatalf("unexpected authorized view: %+v", member)
	}
	if member.Seats[0].AccountID != "account-owner" || member.Seats[1].AccountID != "account-second" {
		t.Fatalf("wrong member seats: %+v", member.Seats)
	}
	outsider, err := service.GetRoomView(context.Background(), "account-third", "private-room")
	if err != nil { t.Fatal(err) }
	if outsider.Member || len(outsider.Seats) != 0 || outsider.ActiveHandID != "" ||
		outsider.Version != member.Version || outsider.Occupied != 2 {
		t.Fatalf("outsider received private room data: %+v", outsider)
	}
	if _, err := service.GetRoomView(context.Background(), "", "private-room"); err == nil { t.Fatal("anonymous query accepted") }
	if _, err := service.GetRoomView(context.Background(), "account-third", " private-room"); err == nil { t.Fatal("malformed room accepted") }
}
