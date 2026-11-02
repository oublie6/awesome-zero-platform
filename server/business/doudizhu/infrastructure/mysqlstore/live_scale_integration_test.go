//go:build integration

package mysqlstore_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"runtime"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/application"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/bidding"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/infrastructure/mysqlstore"
)

// Exercises actual encrypted seeds, journal rows, 100 independent game states,
// fenced startup replay, and concurrent MySQL COMMIT-before-ACK transitions.
// Export DOUDIZHU_SCALE_PROFILE=1 to run a second 1,000-hand scenario. These
// observations are runner-specific and must never be described as an SLA.
func TestDurableActiveHandScaleProfile(t *testing.T) {
	if os.Getenv("APP_API_INTEGRATION") != "1" {
		t.Skip("real MySQL integration disabled")
	}
	counts := []int{100}
	if os.Getenv("DOUDIZHU_SCALE_PROFILE") == "1" {
		counts = append(counts, 1000)
	}
	db := openIntegrationDB(t)
	for _, count := range counts {
		t.Run(fmt.Sprintf("hands-%d", count), func(t *testing.T) {
			profileDurableHands(t, db, count)
		})
	}
}

func profileDurableHands(t *testing.T, db *sql.DB, count int) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	prefix := fmt.Sprintf("scale-%d-%d-", count, time.Now().UnixNano())
	pattern := prefix + "%"
	t.Cleanup(func() {
		for _, table := range []string{"doudizhu_live_checkpoints", "doudizhu_live_actions",
			"doudizhu_live_writers", "doudizhu_protected_server_seeds"} {
			if _, err := db.Exec("DELETE FROM "+table+" WHERE hand_id LIKE ?", pattern); err != nil {
				t.Errorf("clean scale fixtures from %s: %v", table, err)
			}
		}
	})
	seeds, err := mysqlstore.NewProtectedSeedStore(db, coordinatorCrashKey())
	if err != nil {
		t.Fatal(err)
	}
	journal, err := mysqlstore.NewLiveJournal(db)
	if err != nil {
		t.Fatal(err)
	}
	if err := journal.ConfigureCheckpoints(coordinatorCrashKey()); err != nil {
		t.Fatal(err)
	}
	persisted := make(map[domain.HandID]domain.HandSnapshot, count)
	ids := make([]domain.HandID, count)
	start := time.Now()
	for index := range ids {
		id := domain.HandID(fmt.Sprintf("%s%d", prefix, index))
		seed, genesis, hand := coordinatorCrashFixture(t, id)
		if err := seeds.Save(ctx, id, seed); err != nil {
			t.Fatal(err)
		}
		if err := journal.Create(ctx, genesis); err != nil {
			t.Fatal(err)
		}
		ids[index] = id
		persisted[id] = hand
	}
	fixtureTime := time.Since(start)
	runtime.GC()
	var before runtime.MemStats
	runtime.ReadMemStats(&before)
	current := newCoordinatorCrashRuntime(t, db)
	load := func(_ context.Context, target domain.HandID) (domain.HandSnapshot, error) {
		state, found := persisted[target]
		if !found {
			return domain.HandSnapshot{}, application.ErrNotFound
		}
		return state, nil
	}
	start = time.Now()
	if err := current.RecoverActive(ctx, load); err != nil {
		t.Fatal(err)
	}
	recovery := time.Since(start)
	if len(current.RecoveredPhases()) != count {
		t.Fatal("lost or duplicated a durable hand during recovery")
	}
	runtime.GC()
	var after runtime.MemStats
	runtime.ReadMemStats(&after)
	latencies := make([]time.Duration, count)
	work := make(chan int)
	errs := make(chan error, count)
	var wg sync.WaitGroup
	const workers = 16
	wg.Add(workers)
	for worker := 0; worker < workers; worker++ {
		go func() {
			defer wg.Done()
			for index := range work {
				id := ids[index]
				pub, err := current.PublicView(ctx, id, "crash-player-1")
				if err != nil {
					errs <- err
					continue
				}
				// First bidder is derived by the same verified shuffle as
				// the real game; no hard-coded seat or simulated outcome.
				var first struct {
					Bidding struct {
						FirstBidder uint8 `json:"firstBidder"`
					} `json:"bidding"`
				}
				if err := json.Unmarshal(pub.Payload, &first); err != nil {
					errs <- err
					continue
				}
				if first.Bidding.FirstBidder < 1 || first.Bidding.FirstBidder > 3 {
					errs <- errors.New("invalid first bidder in authoritative view")
					continue
				}
				actor := domain.AccountID(fmt.Sprintf("crash-player-%d", first.Bidding.FirstBidder))
				began := time.Now()
				ack, err := current.BidCommitted(ctx, id, actor, fmt.Sprintf("scale-first-bid-%d", index),
					pub.Version, bidding.ScoreThree)
				latencies[index] = time.Since(began)
				if err != nil || ack.Version != 2 {
					errs <- fmt.Errorf("hand %s failed durable commit: %+v %v", id, ack, err)
				}
			}
		}()
	}
	for index := range ids {
		work <- index
	}
	close(work)
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
	var committed int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM doudizhu_live_actions WHERE hand_id LIKE ?", pattern).Scan(&committed); err != nil {
		t.Fatal(err)
	}
	if committed != count {
		t.Fatalf("acknowledged transitions %d want %d", committed, count)
	}
	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
	pick := func(percent int) time.Duration {
		index := (len(latencies)-1)*percent/100
		return latencies[index]
	}
	// GC noise and SQL pool contention make this only a reproducible baseline,
	// not an absolute per-hand memory or fsync guarantee.
	var delta int64 = int64(after.Alloc) - int64(before.Alloc)
	t.Logf("MySQL live profile hands=%d setup=%s recovered=%s heapDelta=%d bytes "+
		"commit p50=%s p95=%s p99=%s max=%s; workers=%d (CI/hardware dependent)",
		count, fixtureTime, recovery, delta, pick(50), pick(95), pick(99), latencies[len(latencies)-1], workers)
}
