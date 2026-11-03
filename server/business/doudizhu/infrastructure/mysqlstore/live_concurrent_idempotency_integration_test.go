//go:build integration

package mysqlstore_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/bidding"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/infrastructure/mysqlstore"
	"github.com/oublie6/awesome-zero-platform/server/business/gamecore"
)

// Unlike simple sequential replay, this starts multiple independent clients'
// requests at the same instant against a real coordinator and durable MySQL.
func TestConcurrentIdenticalLiveRequestRemainsIdempotent(t *testing.T) {
	if os.Getenv("APP_API_INTEGRATION") != "1" {
		t.Skip("real MySQL integration disabled")
	}
	db := openIntegrationDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	id := domain.HandID(fmt.Sprintf("concurrent-receipt-%d", time.Now().UnixNano()))
	t.Cleanup(func() {
		_, _ = db.Exec("DELETE FROM doudizhu_live_checkpoints WHERE hand_id=?", id)
		_, _ = db.Exec("DELETE FROM doudizhu_live_actions WHERE hand_id=?", id)
		_, _ = db.Exec("DELETE FROM doudizhu_live_writers WHERE hand_id=?", id)
		_, _ = db.Exec("DELETE FROM doudizhu_protected_server_seeds WHERE hand_id=?", id)
	})
	seed, genesis, persisted := coordinatorCrashFixture(t, id)
	seedStore, err := mysqlstore.NewProtectedSeedStore(db, coordinatorCrashKey())
	if err != nil {
		t.Fatal(err)
	}
	if err := seedStore.Save(ctx, id, seed); err != nil {
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
	current := newCoordinatorCrashRuntime(t, db)
	if err := current.RecoverActive(ctx, func(_ context.Context, target domain.HandID) (domain.HandSnapshot, error) {
		if target != id {
			return domain.HandSnapshot{}, fmt.Errorf("unexpected hand %s", target)
		}
		return persisted, nil
	}); err != nil {
		t.Fatal(err)
	}
	public, err := current.PublicView(ctx, id, "crash-player-1")
	if err != nil {
		t.Fatal(err)
	}
	var state struct {
		Bidding struct {
			FirstBidder uint8 `json:"firstBidder"`
		} `json:"bidding"`
	}
	if err := json.Unmarshal(public.Payload, &state); err != nil {
		t.Fatal(err)
	}
	actor := domain.AccountID(fmt.Sprintf("crash-player-%d", state.Bidding.FirstBidder))
	const concurrent = 16
	results := make([]uint64, concurrent)
	errs := make([]error, concurrent)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < concurrent; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			<-start
			accepted, err := current.BidCommitted(ctx, id, actor,
				"one-immutable-id", 1, bidding.ScoreThree)
			results[index] = accepted.Version
			errs[index] = err
		}(i)
	}
	close(start)
	wg.Wait()
	for i := range errs {
		if errs[i] != nil || results[i] != 2 {
			t.Fatalf("concurrent same request %d: version=%d err=%v", i, results[i], errs[i])
		}
	}
	actions, err := journal.Read(ctx, id)
	if err != nil || len(actions) != 1 {
		t.Fatalf("retries created multiple writes: count=%d err=%v", len(actions), err)
	}
	if _, err := current.BidCommitted(ctx, id, "crash-player-1",
		"one-immutable-id", 1, bidding.ScoreTwo); !errors.Is(err, gamecore.ErrVerificationFailed) {
		t.Fatalf("reused ID with changed actor/payload accepted: %v", err)
	}
	successor := newCoordinatorCrashRuntime(t, db)
	if err := successor.RecoverActive(ctx, func(_ context.Context, target domain.HandID) (domain.HandSnapshot, error) {
		if target != id {
			return domain.HandSnapshot{}, fmt.Errorf("unexpected hand %s", target)
		}
		return persisted, nil
	}); err != nil {
		t.Fatal(err)
	}
	replayed, err := successor.BidCommitted(ctx, id, actor, "one-immutable-id", 1, bidding.ScoreThree)
	if err != nil || replayed.Version != 2 {
		t.Fatalf("accepted exact request lost after takeover: %+v %v", replayed, err)
	}
	actions, err = journal.Read(ctx, id)
	if err != nil || len(actions) != 1 {
		t.Fatalf("takeover duplicated already committed command: %d %v", len(actions), err)
	}
}
