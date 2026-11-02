package runtime

import (
	"context"
	"errors"
	"testing"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
	"github.com/oublie6/awesome-zero-platform/server/business/gamecore"
)

func TestExactAcceptedReceiptDoesNotReplayDuplicateGameplay(t *testing.T) {
	c := &LiveHandCoordinator{receipts: make(map[domain.HandID]map[string]liveReceipt)}
	hand := domain.HandID("hand-retry")
	action := gamecore.Command{ExpectedVersion: 4, ActorPosition: 2, Payload: []byte(`{"v":"doudizhu-live-pass-command-v1"}`)}
	outcome := gamecore.CommandOutcome{Version: 5, Payload: []byte(`{"v":"doudizhu-live-play-result-v1"}`)}
	c.storeReceipt(hand, "player-2", "exact-id", action, outcome)
	request := gamecore.Command{ExpectedVersion: 4, Payload: append([]byte(nil), action.Payload...)}
	got, found, err := c.loadReceipt(context.Background(), hand, "player-2", "exact-id", request)
	if err != nil || !found || got.Version != 5 {
		t.Fatalf("receipt=%+v found=%v err=%v", got, found, err)
	}
	got.Payload[0] = 'X'
	again, found, err := c.loadReceipt(context.Background(), hand, "player-2", "exact-id", request)
	if err != nil || !found || again.Payload[0] == 'X' {
		t.Fatal("caller mutated internal receipt")
	}
	if _, found, err := c.loadReceipt(context.Background(), hand, "player-1", "exact-id", request); !found || !errors.Is(err, gamecore.ErrVerificationFailed) {
		t.Fatalf("other actor reused receipt: found=%v err=%v", found, err)
	}
	wrongVersion := request
	wrongVersion.ExpectedVersion = 6
	if _, found, err := c.loadReceipt(context.Background(), hand, "player-2", "exact-id", wrongVersion); !found || !errors.Is(err, gamecore.ErrVerificationFailed) {
		t.Fatalf("wrong version reused ID: found=%v err=%v", found, err)
	}
	wrongPayload := request
	wrongPayload.Payload = []byte(`{"v":"doudizhu-live-pass-command-v1","fake":true}`)
	if _, found, err := c.loadReceipt(context.Background(), hand, "player-2", "exact-id", wrongPayload); !found || !errors.Is(err, gamecore.ErrVerificationFailed) {
		t.Fatalf("changed payload reused ID: found=%v err=%v", found, err)
	}
	if _, found, err := c.loadReceipt(context.Background(), hand, "player-2", "another-id", request); found || err != nil {
		t.Fatalf("unknown request treated as accepted: found=%v err=%v", found, err)
	}
}
