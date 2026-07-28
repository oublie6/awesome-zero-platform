package application

import (
	"context"
	"errors"
	"testing"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/bidding"
)

func TestSubmitLiveHandBidUsesTrustedSeatMembershipAndRuntimeOnly(t *testing.T) {
	store := &bidStore{snapshot: biddingHandSnapshot(domain.HandBidding)}
	runtime := &bidRuntime{result: LiveHandCommandResult{Version: 8, Payload: []byte(`{"accepted":true}`)}}
	service := &Service{store: store, liveHands: runtime}

	result, err := service.SubmitLiveHandBid(context.Background(), "player-2", "hand-1", 7, bidding.ScoreTwo)
	if err != nil {
		t.Fatal(err)
	}
	if result.Version != 8 || string(result.Payload) != `{"accepted":true}` {
		t.Fatalf("result=%#v", result)
	}
	if store.loads != 1 || runtime.calls != 1 {
		t.Fatalf("loads=%d runtime calls=%d", store.loads, runtime.calls)
	}
	if runtime.handID != "hand-1" || runtime.actor != "player-2" || runtime.expectedVersion != 7 || runtime.score != bidding.ScoreTwo {
		t.Fatalf("runtime call=%#v", runtime)
	}
}

func TestSubmitLiveHandBidRejectsNonSeatedActorBeforeRuntime(t *testing.T) {
	store := &bidStore{snapshot: biddingHandSnapshot(domain.HandBidding)}
	runtime := &bidRuntime{}
	service := &Service{store: store, liveHands: runtime}

	_, err := service.SubmitLiveHandBid(context.Background(), "outsider", "hand-1", 1, bidding.ScorePass)
	if !errors.Is(err, domain.ErrNotSeated) {
		t.Fatalf("error=%v want ErrNotSeated", err)
	}
	if runtime.calls != 0 {
		t.Fatalf("runtime calls=%d want=0", runtime.calls)
	}
}

func TestSubmitLiveHandBidRequiresPersistedBiddingPhase(t *testing.T) {
	store := &bidStore{snapshot: biddingHandSnapshot(domain.HandDealing)}
	runtime := &bidRuntime{}
	service := &Service{store: store, liveHands: runtime}

	_, err := service.SubmitLiveHandBid(context.Background(), "player-1", "hand-1", 1, bidding.ScorePass)
	if !errors.Is(err, domain.ErrWrongPhase) {
		t.Fatalf("error=%v want ErrWrongPhase", err)
	}
	if runtime.calls != 0 {
		t.Fatalf("runtime calls=%d want=0", runtime.calls)
	}
}

func TestSubmitLiveHandBidRejectsInvalidLiveVersionAndScore(t *testing.T) {
	for _, test := range []struct {
		name    string
		version uint64
		score   bidding.Score
	}{
		{name: "zero version", version: 0, score: bidding.ScorePass},
		{name: "invalid score", version: 1, score: bidding.Score(4)},
	} {
		t.Run(test.name, func(t *testing.T) {
			store := &bidStore{snapshot: biddingHandSnapshot(domain.HandBidding)}
			runtime := &bidRuntime{}
			service := &Service{store: store, liveHands: runtime}
			_, err := service.SubmitLiveHandBid(context.Background(), "player-1", "hand-1", test.version, test.score)
			if !errors.Is(err, ErrInvalidCommand) {
				t.Fatalf("error=%v want ErrInvalidCommand", err)
			}
			if runtime.calls != 0 {
				t.Fatalf("runtime calls=%d want=0", runtime.calls)
			}
		})
	}
}

func biddingHandSnapshot(phase domain.HandPhase) domain.HandSnapshot {
	return domain.HandSnapshot{
		ID:    "hand-1",
		Phase: phase,
		Seats: [3]domain.HandSeat{
			{Seat: 1, AccountID: "player-1"},
			{Seat: 2, AccountID: "player-2"},
			{Seat: 3, AccountID: "player-3"},
		},
	}
}

type bidStore struct {
	Store
	snapshot domain.HandSnapshot
	loads    int
	err      error
}

func (s *bidStore) LoadHand(context.Context, domain.HandID) (domain.HandSnapshot, error) {
	s.loads++
	return s.snapshot, s.err
}

type bidRuntime struct {
	LiveHandRuntime
	result          LiveHandCommandResult
	err             error
	calls           int
	handID          domain.HandID
	actor           domain.AccountID
	expectedVersion uint64
	score           bidding.Score
}

func (r *bidRuntime) Bid(_ context.Context, handID domain.HandID, actor domain.AccountID, expectedVersion uint64, score bidding.Score) (LiveHandCommandResult, error) {
	r.calls++
	r.handID = handID
	r.actor = actor
	r.expectedVersion = expectedVersion
	r.score = score
	return r.result, r.err
}
