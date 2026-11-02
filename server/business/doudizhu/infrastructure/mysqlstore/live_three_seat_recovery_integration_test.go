//go:build integration

package mysqlstore_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/application"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/bidding"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/livehand"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/infrastructure/mysqlstore"
	"github.com/oublie6/awesome-zero-platform/server/business/gamecore"
)

// Three distinct durable account identities drive every authoritative turn
// against real MySQL. At the 32-action checkpoint a new coordinator takes over
// and replays both the protected prefix and the live suffix, then a second
// restart verifies final evidence and exact lost-response idempotency.
// This is a service/storage E2E; HTTPS/WSS and actual devices are separate gates.
func TestRealMySQLThreeSeatFullHandSurvivesTakeoverAndFinalRestart(t *testing.T) {
	if os.Getenv("APP_API_INTEGRATION") != "1" {
		t.Skip("MySQL integration disabled")
	}
	db := openIntegrationDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	id := domain.HandID(fmt.Sprintf("three-recovery-%d", time.Now().UnixNano()))
	t.Cleanup(func() {
		_, _ = db.Exec("DELETE FROM doudizhu_live_checkpoints WHERE hand_id=?", id)
		_, _ = db.Exec("DELETE FROM doudizhu_live_actions WHERE hand_id=?", id)
		_, _ = db.Exec("DELETE FROM doudizhu_live_writers WHERE hand_id=?", id)
		_, _ = db.Exec("DELETE FROM doudizhu_protected_server_seeds WHERE hand_id=?", id)
	})
	seed, genesis, persisted := coordinatorCrashFixture(t, id)
	seeds, err := mysqlstore.NewProtectedSeedStore(db, coordinatorCrashKey())
	if err != nil {
		t.Fatal(err)
	}
	if err := seeds.Save(ctx, id, seed); err != nil {
		t.Fatal(err)
	}
	journal, err := mysqlstore.NewLiveJournal(db)
	if err != nil {
		t.Fatal(err)
	}
	if err := journal.ConfigureCheckpoints(coordinatorCrashKey()); err != nil {
		t.Fatal(err)
	}
	if err := journal.Create(ctx, genesis); err != nil {
		t.Fatal(err)
	}
	load := func(_ context.Context, target domain.HandID) (domain.HandSnapshot, error) {
		if target != id {
			return domain.HandSnapshot{}, application.ErrNotFound
		}
		return persisted, nil
	}
	current := newCoordinatorCrashRuntime(t, db)
	if err := current.RecoverActive(ctx, load); err != nil {
		t.Fatal(err)
	}
	actor := func(seat uint8) domain.AccountID {
		return domain.AccountID(fmt.Sprintf("crash-player-%d", seat))
	}
	var first uint8
	var lastSeat uint8
	var lastCard string
	var lastID string
	var finalVersion uint64
	start := time.Now()
	for action := 1; action <= 59; action++ {
		public, err := current.PublicView(ctx, id, actor(1))
		if err != nil {
			t.Fatalf("action %d read: %v", action, err)
		}
		var state livehand.PublicView
		if err := json.Unmarshal(public.Payload, &state); err != nil {
			t.Fatal(err)
		}
		requestID := fmt.Sprintf("three-recovery-action-%d", action)
		var result application.LiveHandCommandResult
		if state.Phase == livehand.PhaseBidding {
			if action != 1 {
				t.Fatalf("unexpected bidding action %d", action)
			}
			first = state.Bidding.FirstBidder
			result, err = current.BidCommitted(ctx, id, actor(first), requestID, public.Version, bidding.ScoreThree)
		} else {
			if state.Phase != livehand.PhasePlaying || state.Playing == nil {
				t.Fatalf("unexpected active phase %s", state.Phase)
			}
			seat := state.Playing.CurrentSeat
			if seat == state.LandlordSeat {
				private, privateErr := current.PrivateView(ctx, id, actor(seat))
				if privateErr != nil {
					t.Fatal(privateErr)
				}
				var hand livehand.PrivateView
				if err := json.Unmarshal(private.Payload, &hand); err != nil {
					t.Fatal(err)
				}
				if len(hand.Cards) == 0 {
					t.Fatal("landlord's final card disappeared")
				}
				result, err = current.PlayCommitted(ctx, id, actor(seat), requestID, public.Version, hand.Cards[:1])
				lastSeat, lastCard = seat, hand.Cards[0]
			} else {
				result, err = current.PassCommitted(ctx, id, actor(seat), requestID, public.Version)
			}
		}
		if err != nil || result.Version != uint64(action+1) {
			t.Fatalf("action %d not durably acknowledged: %+v %v", action, result, err)
		}
		lastID = requestID
		finalVersion = result.Version
		if action == 32 {
			// SaveCheckpoint runs asynchronously. Verify the sealed record was
			// actually committed before the writer is fenced by takeover.
			deadline := time.Now().Add(5 * time.Second)
			for {
				checkpoint, checkpointErr := journal.LoadCheckpoint(ctx, id)
				if checkpointErr != nil {
					t.Fatal(checkpointErr)
				}
				if checkpoint.Version == 33 {
					break
				}
				if time.Now().After(deadline) {
					t.Fatal("protected midpoint checkpoint not published")
				}
				time.Sleep(10 * time.Millisecond)
			}
			successor := newCoordinatorCrashRuntime(t, db)
			if err := successor.RecoverActive(ctx, load); err != nil {
				t.Fatalf("takeover after 32 accepted actions: %v", err)
			}
			// Old owner is still in memory, but MySQL must reject its next
			// otherwise valid transition after the successor acquires epoch.
			staleView, err := current.PublicView(ctx, id, actor(1))
			if err != nil {
				t.Fatal(err)
			}
			var stale livehand.PublicView
			if err := json.Unmarshal(staleView.Payload, &stale); err != nil {
				t.Fatal(err)
			}
			nextSeat := stale.Playing.CurrentSeat
			if nextSeat == stale.LandlordSeat {
				private, err := current.PrivateView(ctx, id, actor(nextSeat))
				if err != nil {
					t.Fatal(err)
				}
				var hand livehand.PrivateView
				if err := json.Unmarshal(private.Payload, &hand); err != nil {
					t.Fatal(err)
				}
				_, err = current.PlayCommitted(ctx, id, actor(nextSeat), "stale-writer", staleView.Version, hand.Cards[:1])
			} else {
				_, err = current.PassCommitted(ctx, id, actor(nextSeat), "stale-writer", staleView.Version)
			}
			if !errors.Is(err, mysqlstore.ErrLiveWriterFenced) {
				t.Fatalf("fenced previous coordinator still wrote: %v", err)
			}
			replayed, err := successor.BidCommitted(ctx, id, actor(first), "three-recovery-action-1", 1, bidding.ScoreThree)
			if err != nil || replayed.Version != 2 {
				t.Fatalf("checkpoint forgot exactly acknowledged bid: %+v %v", replayed, err)
			}
			current = successor
		}
	}
	if finalVersion != 60 || lastSeat == 0 || lastCard == "" {
		t.Fatalf("not fully settled: version=%d last=%d/%s", finalVersion, lastSeat, lastCard)
	}
	actions, err := journal.Read(ctx, id)
	if err != nil || len(actions) != 59 || actions[len(actions)-1].Version != 60 {
		t.Fatalf("durable final journal incomplete: %d %v", len(actions), err)
	}
	third := newCoordinatorCrashRuntime(t, db)
	if err := third.RecoverActive(ctx, load); err != nil {
		t.Fatalf("second full-process-style restoration: %v", err)
	}
	if got := third.RecoveredPhases()[id]; got != livehand.PhaseCompleted {
		t.Fatalf("recovery did not archive final evidence: phase=%s", got)
	}
	record, err := third.Abort(ctx, id, "ignored-after-completion")
	if err != nil || record.Status() != gamecore.FinalStatusCompleted ||
		record.Version() != 60 || record.Validate() != nil {
		t.Fatalf("recovered final record corrupt: %+v %v", record, err)
	}
	replay, err := third.PlayCommitted(ctx, id, actor(lastSeat), lastID, 59, []string{lastCard})
	if err != nil || replay.Version != 60 {
		t.Fatalf("lost terminal response was replayed as a new command: %+v %v", replay, err)
	}
	_, err = third.PlayCommitted(ctx, id, actor(lastSeat), lastID, 59, []string{lastCard, lastCard})
	if !errors.Is(err, gamecore.ErrVerificationFailed) {
		t.Fatalf("changed terminal request accepted using same command ID: %v", err)
	}
	t.Logf("real MySQL three-seat game, 59 committed actions, sealed takeover and final replay: %s (CI runner, not production SLA)", time.Since(start))
}
