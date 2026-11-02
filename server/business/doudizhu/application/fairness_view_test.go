package application

import (
	"context"
	"testing"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
)

func TestFairnessProjectionAndRevealBindingAreActorScoped(t *testing.T) {
	seats := [3]domain.HandSeat{
		{Seat: domain.SeatOne, AccountID: "alice"},
		{Seat: domain.SeatTwo, AccountID: "bob"},
		{Seat: domain.SeatThree, AccountID: "charlie"},
	}
	hand, _, err := domain.NewHand("hand-real", "room-real", seats, domain.ServerCommitment{1},
		"key-live-v1", domain.BeaconPlan{Provider: "trusted", Round: "fixed-round"})
	if err != nil {
		t.Fatal(err)
	}
	store := newTestStore()
	store.state.hands["hand-real"] = hand.Snapshot()
	service := &Service{store: store}
	if _, err := service.GetRevealBinding(context.Background(), "alice", "hand-real"); err == nil {
		t.Fatal("reveal ticket issued before all commitments")
	}
	commitments := [3]domain.Commitment{{1}, {2}, {3}}
	for index, seat := range seats {
		if _, err := hand.SubmitCommit(seat.AccountID, commitments[index], hand.Snapshot().Version); err != nil {
			t.Fatal(err)
		}
	}
	store.state.hands["hand-real"] = hand.Snapshot()
	store.state.sequences[testSequenceKey(domain.AggregateHand, "hand-real", "bob")] = 17
	if hand.Snapshot().Phase != domain.HandFairnessRevealing {
		t.Fatal("did not enter reveal phase")
	}
	own, err := service.GetFairnessView(context.Background(), "bob", "hand-real")
	if err != nil {
		t.Fatal(err)
	}
	if own.Seat != 2 || !own.Committed || own.Revealed ||
		own.LastClientSequence != 17 || own.Commitment == "" || own.Version != hand.Snapshot().Version {
		t.Fatalf("wrong fair view: %+v", own)
	}
	alice, err := service.GetFairnessView(context.Background(), "alice", "hand-real")
	if err != nil {
		t.Fatal(err)
	}
	if alice.Commitment == own.Commitment || alice.LastClientSequence != 0 {
		t.Fatalf("cross-account contribution leaked: %+v", alice)
	}
	binding, err := service.GetRevealBinding(context.Background(), "bob", "hand-real")
	if err != nil || binding.Seat != domain.SeatTwo || binding.Commitment != commitments[1] {
		t.Fatalf("incorrect ticket binding: %+v %v", binding, err)
	}
	if _, err := service.GetFairnessView(context.Background(), "outsider", "hand-real"); err == nil {
		t.Fatal("nonmember read fairness projection")
	}
	if _, err := service.GetRevealBinding(context.Background(), "outsider", "hand-real"); err == nil {
		t.Fatal("nonmember obtained reveal binding")
	}
}
