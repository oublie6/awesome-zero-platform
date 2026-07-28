package application

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
)

func TestSubmitLiveHandPlayUsesTrustedMembershipAndRuntimeOnly(t *testing.T) {
	store := &bidStore{snapshot: biddingHandSnapshot(domain.HandBidding)}
	runtime := &playRuntime{result: LiveHandCommandResult{Version: 9, Payload: []byte(`{"accepted":true}`)}}
	service := &Service{store: store, liveHands: runtime}
	cards := []string{"C3", "D3"}

	result, err := service.SubmitLiveHandPlay(context.Background(), "player-2", "hand-1", 8, cards)
	if err != nil {
		t.Fatal(err)
	}
	cards[0] = "YJ"
	if result.Version != 9 || string(result.Payload) != `{"accepted":true}` {
		t.Fatalf("result=%#v", result)
	}
	if store.loads != 1 || runtime.playCalls != 1 || runtime.passCalls != 0 {
		t.Fatalf("loads=%d play=%d pass=%d", store.loads, runtime.playCalls, runtime.passCalls)
	}
	if runtime.handID != "hand-1" || runtime.actor != "player-2" || runtime.expectedVersion != 8 || !reflect.DeepEqual(runtime.cards, []string{"C3", "D3"}) {
		t.Fatalf("runtime=%#v", runtime)
	}
}

func TestSubmitLiveHandPassUsesTrustedMembershipAndRuntimeOnly(t *testing.T) {
	store := &bidStore{snapshot: biddingHandSnapshot(domain.HandBidding)}
	runtime := &playRuntime{result: LiveHandCommandResult{Version: 10}}
	service := &Service{store: store, liveHands: runtime}

	result, err := service.SubmitLiveHandPass(context.Background(), "player-3", "hand-1", 9)
	if err != nil {
		t.Fatal(err)
	}
	if result.Version != 10 || runtime.passCalls != 1 || runtime.playCalls != 0 || runtime.actor != "player-3" || runtime.expectedVersion != 9 {
		t.Fatalf("result=%#v runtime=%#v", result, runtime)
	}
}

func TestSubmitLiveHandPlayingRejectsUnauthorizedOrInvalidRequests(t *testing.T) {
	tests := []struct {
		name string
		phase domain.HandPhase
		actor domain.AccountID
		version uint64
		cards []string
		pass bool
		want error
	}{
		{name: "outsider play", phase: domain.HandBidding, actor: "outsider", version: 1, cards: []string{"C3"}, want: domain.ErrNotSeated},
		{name: "wrong persisted phase", phase: domain.HandDealing, actor: "player-1", version: 1, cards: []string{"C3"}, want: domain.ErrWrongPhase},
		{name: "zero play version", phase: domain.HandBidding, actor: "player-1", cards: []string{"C3"}, want: ErrInvalidCommand},
		{name: "empty play cards", phase: domain.HandBidding, actor: "player-1", version: 1, want: ErrInvalidCommand},
		{name: "blank play card", phase: domain.HandBidding, actor: "player-1", version: 1, cards: []string{" "}, want: ErrInvalidCommand},
		{name: "zero pass version", phase: domain.HandBidding, actor: "player-1", pass: true, want: ErrInvalidCommand},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := &bidStore{snapshot: biddingHandSnapshot(test.phase)}
			runtime := &playRuntime{}
			service := &Service{store: store, liveHands: runtime}
			var err error
			if test.pass {
				_, err = service.SubmitLiveHandPass(context.Background(), test.actor, "hand-1", test.version)
			} else {
				_, err = service.SubmitLiveHandPlay(context.Background(), test.actor, "hand-1", test.version, test.cards)
			}
			if !errors.Is(err, test.want) {
				t.Fatalf("error=%v want=%v", err, test.want)
			}
			if runtime.playCalls != 0 || runtime.passCalls != 0 {
				t.Fatalf("runtime=%#v", runtime)
			}
		})
	}
}

type playRuntime struct {
	LiveHandRuntime
	result LiveHandCommandResult
	err error
	playCalls int
	passCalls int
	handID domain.HandID
	actor domain.AccountID
	expectedVersion uint64
	cards []string
}

func (r *playRuntime) Play(_ context.Context, handID domain.HandID, actor domain.AccountID, expectedVersion uint64, cards []string) (LiveHandCommandResult, error) {
	r.playCalls++
	r.handID = handID
	r.actor = actor
	r.expectedVersion = expectedVersion
	r.cards = append([]string(nil), cards...)
	return r.result, r.err
}

func (r *playRuntime) Pass(_ context.Context, handID domain.HandID, actor domain.AccountID, expectedVersion uint64) (LiveHandCommandResult, error) {
	r.passCalls++
	r.handID = handID
	r.actor = actor
	r.expectedVersion = expectedVersion
	return r.result, r.err
}
