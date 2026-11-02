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

func TestArchiveFailureKeepsOnlyCommittedTerminalReceipt(t *testing.T) {
	c := &LiveHandCoordinator{receipts: make(map[domain.HandID]map[string]liveReceipt)}
	hand := domain.HandID("hand-terminal-archive")
	command := gamecore.Command{ActorPosition: 1, ExpectedVersion: 40, Payload: []byte(`{"v":"doudizhu-live-play-command-v1"}`)}
	terminal := gamecore.CommandOutcome{
		Version: 41, Terminal: true,
		Payload: []byte(`{"result":"finished"}`), FinalPayload: []byte(`{"evidence":"committed"}`),
	}
	errArchive := errors.Join(gamecore.ErrArchiveFailed, errors.New("archive database unavailable"))
	if !c.retainArchivedFailureReceipt(hand, "player-1", "terminal-commit", command, terminal, errArchive) {
		t.Fatal("durably committed terminal action lost its retry receipt")
	}
	got, found, err := c.loadReceipt(context.Background(), hand, "player-1", "terminal-commit", command)
	if err != nil || !found || got.Version != terminal.Version || string(got.FinalPayload) != string(terminal.FinalPayload) {
		t.Fatalf("retry lost accepted terminal result: outcome=%+v found=%v err=%v", got, found, err)
	}
	for _, tt := range []struct {
		name   string
		result gamecore.CommandOutcome
		err    error
	}{
		{"uncommitted database failure", terminal, errors.New("ambiguous transaction commit")},
		{"nonterminal archive failure", gamecore.CommandOutcome{Version: 41, Payload: terminal.Payload}, errArchive},
		{"missing final evidence", gamecore.CommandOutcome{Version: 41, Terminal: true}, errArchive},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if c.retainArchivedFailureReceipt(hand, "player-1", "unsafe-"+tt.name, command, tt.result, tt.err) {
				t.Fatal("unconfirmed outcome became an accepted receipt")
			}
		})
	}
}

func TestMySQLJSONNormalizationDoesNotRejectLegitimateRequestRetry(t *testing.T) {
	c := &LiveHandCoordinator{receipts: make(map[domain.HandID]map[string]liveReceipt)}
	id := domain.HandID("json-normalization-hand")
	// The database JSON column may reorder fields and add whitespace.
	fromMySQL := gamecore.Command{ActorPosition: 2, ExpectedVersion: 7,
		Payload: []byte(`{"score": 1, "v": "doudizhu-live-bid-command-v1"}`)}
	original := gamecore.Command{ExpectedVersion: 7,
		Payload: []byte(`{"v":"doudizhu-live-bid-command-v1","score":1}`)}
	c.storeReceipt(id, "actor-2", "accepted-id", fromMySQL,
		gamecore.CommandOutcome{Version: 8, Payload: []byte(`{"ok":true}`)})
	accepted, found, err := c.loadReceipt(context.Background(), id, "actor-2", "accepted-id", original)
	if err != nil || !found || accepted.Version != 8 {
		t.Fatalf("valid MySQL-normalized retry rejected: result=%+v found=%v err=%v", accepted, found, err)
	}
	tampered := original
	tampered.Payload = []byte(`{"v":"doudizhu-live-bid-command-v1","score":2}`)
	if _, found, err := c.loadReceipt(context.Background(), id, "actor-2", "accepted-id", tampered); !found || !errors.Is(err, gamecore.ErrVerificationFailed) {
		t.Fatalf("semantically changed command was accepted: found=%v err=%v", found, err)
	}
	for _, malformed := range [][]byte{
		[]byte(`{"v":"doudizhu-live-bid-command-v1","score":1} {}`),
		[]byte(`{"v":"doudizhu-live-bid-command-v1","score":1`),
	} {
		original.Payload = malformed
		if _, found, err := c.loadReceipt(context.Background(), id, "actor-2", "accepted-id", original); !found || !errors.Is(err, gamecore.ErrVerificationFailed) {
			t.Fatalf("invalid JSON command was accepted: found=%v err=%v", found, err)
		}
	}
}
