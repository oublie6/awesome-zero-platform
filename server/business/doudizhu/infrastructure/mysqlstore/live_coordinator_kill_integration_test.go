//go:build integration

package mysqlstore_test

import (
	"bufio"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/bidding"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/carddeck"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/livehand"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/infrastructure/mysqlstore"
	ddzruntime "github.com/oublie6/awesome-zero-platform/server/business/doudizhu/infrastructure/runtime"
	"github.com/oublie6/awesome-zero-platform/server/business/gamecore"
)

const coordinatorCrashChildEnv = "DOUDIZHU_COORDINATOR_CRASH_CHILD"

type noopCoordinatorArchive struct{}

func (noopCoordinatorArchive) Archive(record gamecore.FinalRecord) error { return record.Validate() }

// Full live-hand coordinator restart from persisted MySQL journal and protected
// server seed after a real subprocess SIGKILL (not a mocked persistence call).
func TestDurableCoordinatorRecoversAfterProcessKill(t *testing.T) {
	if os.Getenv("APP_API_INTEGRATION") != "1" {
		t.Skip("MySQL integration disabled")
	}
	if os.Getenv(coordinatorCrashChildEnv) == "1" {
		runCoordinatorCrashChild(t)
		return
	}
	db := openIntegrationDB(t)
	ctx := context.Background()
	id := domain.HandID(fmt.Sprintf("crash-coord-%d", time.Now().UnixNano()))
	t.Cleanup(func() {
		_, _ = db.Exec("DELETE FROM doudizhu_live_checkpoints WHERE hand_id=?", id)
		_, _ = db.Exec("DELETE FROM doudizhu_live_actions WHERE hand_id=?", id)
		_, _ = db.Exec("DELETE FROM doudizhu_live_writers WHERE hand_id=?", id)
		_, _ = db.Exec("DELETE FROM doudizhu_protected_server_seeds WHERE hand_id=?", id)
	})
	seed, genesis, biddingHand := coordinatorCrashFixture(t, id)
	store, err := mysqlstore.NewProtectedSeedStore(db, coordinatorCrashKey())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Save(ctx, id, seed); err != nil {
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

	childCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	child := exec.CommandContext(childCtx, os.Args[0], "-test.run=^TestDurableCoordinatorRecoversAfterProcessKill$")
	child.Env = append(os.Environ(), coordinatorCrashChildEnv+"=1", "DOUDIZHU_CRASH_COORD_HAND="+string(id))
	output, err := child.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	child.Stderr = os.Stderr
	if err := child.Start(); err != nil {
		t.Fatal(err)
	}
	line, readErr := bufio.NewReader(output).ReadString('\n')
	if readErr != nil || strings.TrimSpace(line) != "CRASH_POINT_ACKED" {
		_ = child.Process.Kill()
		_ = child.Wait()
		t.Fatalf("child did not acknowledge durable action: marker=%q err=%v", line, readErr)
	}
	if err := child.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	if err := child.Wait(); err == nil {
		t.Fatal("writer exited normally, was not SIGKILLed")
	}
	if childCtx.Err() != nil {
		t.Fatalf("crash test timeout: %v", childCtx.Err())
	}

	actions, err := journal.Read(ctx, id)
	if err != nil || len(actions) != 1 || actions[0].Version != 2 || actions[0].CommandID != "crash-acked-bid" {
		t.Fatalf("acknowledged action not durable: %v %+v", err, actions)
	}
	fresh := newCoordinatorCrashRuntime(t, db)
	load := func(_ context.Context, target domain.HandID) (domain.HandSnapshot, error) {
		if target != id {
			return domain.HandSnapshot{}, errors.New("unexpected hand")
		}
		return biddingHand, nil
	}
	if err := fresh.RecoverActive(ctx, load); err != nil {
		t.Fatalf("full runtime recovery: %v", err)
	}
	for _, actor := range []domain.AccountID{"crash-player-1", "crash-player-2", "crash-player-3"} {
		snapshot, err := fresh.PrivateView(ctx, id, actor)
		if err != nil || snapshot.Version != 2 {
			t.Fatalf("missing private hand %s: %v %+v", actor, err, snapshot)
		}
		var decoded livehand.PrivateView
		if err := json.Unmarshal(snapshot.Payload, &decoded); err != nil || len(decoded.Cards) != 17 {
			t.Fatalf("private hand corrupted for %s: %v %+v", actor, err, decoded)
		}
	}
	originalActor := domain.AccountID(fmt.Sprintf("crash-player-%d", actions[0].ActorPosition))
	replay, err := fresh.BidCommitted(ctx, id, originalActor, "crash-acked-bid", 1, bidding.ScoreOne)
	if err != nil || replay.Version != 2 {
		t.Fatalf("exact accepted receipt lost: %+v %v", replay, err)
	}
	if _, err := fresh.BidCommitted(ctx, id, "unknown-actor", "crash-acked-bid", 1, bidding.ScoreOne); !errors.Is(err, gamecore.ErrVerificationFailed) {
		t.Fatalf("cross-account receipt reuse accepted: %v", err)
	}
	stale := gamecore.Command{ActorPosition: 1, ExpectedVersion: 2, Payload: []byte(`{"v":"doudizhu-live-bid-command-v1","score":3}`)}
	if err := journal.Append(ctx, id, 2, "old-owner-write", "crash-player-1", stale, gamecore.CommandOutcome{Version: 3, Payload: []byte(`{"ok":true}`)}); !errors.Is(err, mysqlstore.ErrLiveWriterFenced) {
		t.Fatalf("dead epoch could still append: %v", err)
	}
	public, err := fresh.PublicView(ctx, id, originalActor)
	if err != nil {
		t.Fatal(err)
	}
	var decoded livehand.PublicView
	if err := json.Unmarshal(public.Payload, &decoded); err != nil {
		t.Fatal(err)
	}
	next := domain.AccountID(fmt.Sprintf("crash-player-%d", decoded.Bidding.CurrentBidder))
	advanced, err := fresh.BidCommitted(ctx, id, next, "recovered-new-owner-bid", 2, bidding.ScoreThree)
	if err != nil || advanced.Version != 3 {
		t.Fatalf("new owner failed to resume bidding: %+v %v", advanced, err)
	}
}

func runCoordinatorCrashChild(t *testing.T) {
	id := domain.HandID(os.Getenv("DOUDIZHU_CRASH_COORD_HAND"))
	if id == "" {
		t.Fatal("missing process hand ID")
	}
	db := openIntegrationDB(t)
	_, _, biddingHand := coordinatorCrashFixture(t, id)
	current := newCoordinatorCrashRuntime(t, db)
	if err := current.RecoverActive(context.Background(), func(_ context.Context, target domain.HandID) (domain.HandSnapshot, error) {
		if target != id {
			return domain.HandSnapshot{}, errors.New("unknown hand")
		}
		return biddingHand, nil
	}); err != nil {
		t.Fatal(err)
	}
	public, err := current.PublicView(context.Background(), id, "crash-player-1")
	if err != nil {
		t.Fatal(err)
	}
	var decoded livehand.PublicView
	if err := json.Unmarshal(public.Payload, &decoded); err != nil {
		t.Fatal(err)
	}
	first := domain.AccountID(fmt.Sprintf("crash-player-%d", decoded.Bidding.FirstBidder))
	ack, err := current.BidCommitted(context.Background(), id, first, "crash-acked-bid", 1, bidding.ScoreOne)
	if err != nil || ack.Version != 2 {
		t.Fatalf("initial command was not durable: %+v %v", ack, err)
	}
	fmt.Fprintln(os.Stdout, "CRASH_POINT_ACKED")
	time.Sleep(time.Minute)
	t.Fatal("writer was not killed")
}

func coordinatorCrashKey() []byte {
	key := make([]byte, 32)
	for i := range key {
		key[i] = 0x5a
	}
	return key
}
func newCoordinatorCrashRuntime(t *testing.T, db *sql.DB) *ddzruntime.LiveHandCoordinator {
	t.Helper()
	protected, err := mysqlstore.NewProtectedSeedStore(db, coordinatorCrashKey())
	if err != nil {
		t.Fatal(err)
	}
	seeds, err := ddzruntime.NewPersistentSeedVault(nil, protected)
	if err != nil {
		t.Fatal(err)
	}
	directory, err := gamecore.NewLiveDirectory(noopCoordinatorArchive{})
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
	coordinator, err := ddzruntime.NewDurableLiveHandCoordinator(seeds, directory, journal)
	if err != nil {
		t.Fatal(err)
	}
	return coordinator
}
func coordinatorCrashFixture(t *testing.T, id domain.HandID) (carddeck.Seed, domain.HandSnapshot, domain.HandSnapshot) {
	t.Helper()
	var seed carddeck.Seed
	for i := range seed {
		seed[i] = byte(i + 1)
	}
	commitment, err := carddeck.ComputeServerCommitment(string(id), seed)
	if err != nil {
		t.Fatal(err)
	}
	var revealKey domain.RevealPublicKeyHash
	revealKey[0] = 10
	seats := [3]domain.HandSeat{
		{Seat: 1, AccountID: "crash-player-1"},
		{Seat: 2, AccountID: "crash-player-2"},
		{Seat: 3, AccountID: "crash-player-3"},
	}
	hand, _, err := domain.NewHand(id, "crash-room", seats, domain.ServerCommitment(commitment), "crash-reveal-key",
		domain.BeaconPlan{Provider: "test-beacon", Round: "fixed-round"},
		domain.RevealKeyBinding{PublicKeySHA256: revealKey, BoundAt: time.Unix(1, 0).UTC()})
	if err != nil {
		t.Fatal(err)
	}
	for i, seat := range seats {
		var digest domain.ContributionDigest
		digest[0] = byte(i + 1)
		if _, err := hand.SubmitCommit(seat.AccountID, domain.ComputeClientCommitment(id, seat.Seat, digest), hand.Snapshot().Version); err != nil {
			t.Fatal(err)
		}
	}
	for i, seat := range seats {
		var digest domain.ContributionDigest
		digest[0] = byte(i + 1)
		if _, err := hand.SubmitReveal(seat.AccountID, digest, fmt.Sprintf("crash-contribution-%d", i), hand.Snapshot().Version); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := hand.LockPublicBeacon(domain.BeaconValue{
		Provider: "test-beacon", Round: "fixed-round",
		Digest: domain.BeaconDigest(sha256.Sum256([]byte("fixed-test-beacon"))), ProofRef: "proof-test",
	}, hand.Snapshot().Version); err != nil {
		t.Fatal(err)
	}
	dealing := hand.Snapshot()
	if _, err := hand.MarkDealt(hand.Snapshot().Version); err != nil {
		t.Fatal(err)
	}
	return seed, dealing, hand.Snapshot()
}
