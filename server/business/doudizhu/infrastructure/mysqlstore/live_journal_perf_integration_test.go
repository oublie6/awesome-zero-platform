//go:build integration

package mysqlstore_test

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/infrastructure/mysqlstore"
	"github.com/oublie6/awesome-zero-platform/server/business/gamecore"
)

func TestDurableLiveAppendLatencyPercentiles(t *testing.T) {
	if os.Getenv("APP_API_INTEGRATION") != "1" {
		t.Skip("MySQL integration disabled")
	}
	db := openIntegrationDB(t)
	ctx := context.Background()
	id := domain.HandID(fmt.Sprintf("journal-perf-%d", time.Now().UnixNano()))
	t.Cleanup(func() {
		_, _ = db.Exec("DELETE FROM doudizhu_live_actions WHERE hand_id=?", id)
		_, _ = db.Exec("DELETE FROM doudizhu_live_writers WHERE hand_id=?", id)
	})
	seats := [3]domain.HandSeat{
		{Seat: 1, AccountID: "perf-1"}, {Seat: 2, AccountID: "perf-2"}, {Seat: 3, AccountID: "perf-3"},
	}
	hand, _, err := domain.NewHand(id, "room-perf", seats, domain.ServerCommitment{1}, "test-key",
		domain.BeaconPlan{Provider: "perf", Round: "fixed"})
	if err != nil {
		t.Fatal(err)
	}
	for i, seat := range seats {
		digest := domain.ContributionDigest{byte(i + 1)}
		if _, err := hand.SubmitCommit(seat.AccountID, domain.ComputeClientCommitment(id, seat.Seat, digest), hand.Snapshot().Version); err != nil {
			t.Fatal(err)
		}
	}
	for i, seat := range seats {
		if _, err := hand.SubmitReveal(seat.AccountID, domain.ContributionDigest{byte(i + 1)},
			fmt.Sprintf("perf-contribution-%d", i), hand.Snapshot().Version); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := hand.LockPublicBeacon(domain.BeaconValue{Provider: "perf", Round: "fixed",
		Digest: domain.BeaconDigest{1}, ProofRef: "fixture"}, hand.Snapshot().Version); err != nil {
		t.Fatal(err)
	}
	journal, err := mysqlstore.NewLiveJournal(db)
	if err != nil {
		t.Fatal(err)
	}
	if err := journal.Create(ctx, hand.Snapshot()); err != nil {
		t.Fatal(err)
	}
	epoch, version, _, err := journal.Acquire(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	const total = 80
	timings := make([]time.Duration, 0, total)
	for i := 0; i < total; i++ {
		command := gamecore.Command{ActorPosition: uint8(i%3 + 1), ExpectedVersion: version,
			Payload: []byte(fmt.Sprintf(`{"v":"perf","index":%d}`, i))}
		outcome := gamecore.CommandOutcome{Version: version + 1, Payload: json.RawMessage(`{"ok":true}`)}
		start := time.Now()
		if err := journal.Append(ctx, id, epoch, fmt.Sprintf("perf-%d", i), seats[i%3].AccountID, command, outcome); err != nil {
			t.Fatal(err)
		}
		timings = append(timings, time.Since(start))
		version++
	}
	actions, err := journal.Read(ctx, id)
	if err != nil || len(actions) != total {
		t.Fatalf("record loss after acknowledged writes: %d %v", len(actions), err)
	}
	for i, act := range actions {
		raw, _ := json.Marshal(gamecore.CommandOutcome{Version: act.Version, Payload: json.RawMessage(`{"ok":true}`)})
		if act.ResultDigest != sha256.Sum256(raw) {
			t.Fatalf("result evidence corrupted at %d", i)
		}
	}
	sort.Slice(timings, func(i, j int) bool { return timings[i] < timings[j] })
	percentile := func(p float64) time.Duration {
		i := int(float64(len(timings)-1) * p)
		return timings[i]
	}
	t.Logf("MySQL COMMIT-before-ACK sample=%d p50=%s p95=%s p99=%s max=%s (CI runner, not production SLA)",
		total, percentile(0.50), percentile(0.95), percentile(0.99), timings[len(timings)-1])
}
