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
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/infrastructure/mysqlstore"
	"github.com/oublie6/awesome-zero-platform/server/business/gamecore"
)

const crashChildEnvironment = "DOUDIZHU_LIVE_JOURNAL_CRASH_CHILD"

// A real SIGKILL separates the database writer from the recovering process.
// This tests the journal's storage boundary, not full game state replay.
func TestLiveJournalSurvivesActualWriterProcessKill(t *testing.T) {
	if os.Getenv("APP_API_INTEGRATION") != "1" {
		t.Skip("MySQL integration disabled")
	}
	if mode := os.Getenv(crashChildEnvironment); mode != "" {
		runLiveJournalCrashChild(t, mode)
		return
	}
	db := openIntegrationDB(t)
	ctx := context.Background()
	id := domain.HandID(fmt.Sprintf("journal-kill-%d", time.Now().UnixNano()))
	t.Cleanup(func() {
		_, _ = db.Exec("DELETE FROM doudizhu_live_actions WHERE hand_id=?", id)
		_, _ = db.Exec("DELETE FROM doudizhu_live_writers WHERE hand_id=?", id)
	})
	seats := [3]domain.HandSeat{
		{Seat: 1, AccountID: "kill-1"},
		{Seat: 2, AccountID: "kill-2"},
		{Seat: 3, AccountID: "kill-3"},
	}
	hand, _, err := domain.NewHand(id, domain.RoomID("room-kill"), seats, domain.ServerCommitment{1}, "test-key", domain.BeaconPlan{Provider: "test", Round: "round"})
	if err != nil {
		t.Fatal(err)
	}
	for i, seat := range seats {
		contribution := domain.ContributionDigest{byte(i + 1)}
		if _, err := hand.SubmitCommit(seat.AccountID, domain.ComputeClientCommitment(id, seat.Seat, contribution), hand.Snapshot().Version); err != nil {
			t.Fatal(err)
		}
	}
	for i, seat := range seats {
		if _, err := hand.SubmitReveal(seat.AccountID, domain.ContributionDigest{byte(i + 1)}, fmt.Sprintf("kill-contribution-%d", i), hand.Snapshot().Version); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := hand.LockPublicBeacon(domain.BeaconValue{Provider: "test", Round: "round", Digest: domain.BeaconDigest{1}, ProofRef: "fixture"}, hand.Snapshot().Version); err != nil {
		t.Fatal(err)
	}
	journal, err := mysqlstore.NewLiveJournal(db)
	if err != nil {
		t.Fatal(err)
	}
	if err := journal.Create(ctx, hand.Snapshot()); err != nil {
		t.Fatal(err)
	}
	epoch, watermark, _, err := journal.Acquire(ctx, id)
	if err != nil || epoch != 2 || watermark != 1 {
		t.Fatalf("initial owner=%d watermark=%d err=%v", epoch, watermark, err)
	}

	// SIGKILL before issuing a transaction: no phantom committed mutation.
	killLiveJournalChildAtMarker(t, "before-commit", id, epoch)
	actions, err := journal.Read(ctx, id)
	if err != nil || len(actions) != 0 {
		t.Fatalf("pre-commit kill created an action: actions=%v err=%v", actions, err)
	}

	// The worker has inserted an action and advanced the watermark inside
	// an open transaction, but has not COMMITted. SIGKILL must roll both
	// changes back together, never exposing a half-accepted operation.
	killLiveJournalChildAtMarker(t, "during-transaction", id, epoch)
	actions, err = journal.Read(ctx, id)
	if err != nil || len(actions) != 0 {
		t.Fatalf("killed uncommitted transaction leaked action: %+v %v", actions, err)
	}
	var uncommittedWatermark uint64
	if err := db.QueryRowContext(ctx, "SELECT committed_version FROM doudizhu_live_writers WHERE hand_id=?", id).Scan(&uncommittedWatermark); err != nil || uncommittedWatermark != 1 {
		t.Fatalf("uncommitted watermark survived kill: %d %v", uncommittedWatermark, err)
	}

	// SIGKILL after Append returned success: a new process must see the exact
	// persisted operation and fence the dead process's epoch.
	killLiveJournalChildAtMarker(t, "after-commit", id, epoch)
	actions, err = journal.Read(ctx, id)
	if err != nil || len(actions) != 1 || actions[0].CommandID != "kill-command-1" || actions[0].Version != 2 {
		t.Fatalf("acknowledged action lost on SIGKILL: actions=%+v err=%v", actions, err)
	}
	newEpoch, recoveredVersion, _, err := journal.Acquire(ctx, id)
	if err != nil || newEpoch <= epoch || recoveredVersion != 2 {
		t.Fatalf("recovered owner=%d version=%d err=%v", newEpoch, recoveredVersion, err)
	}
	next := gamecore.Command{ActorPosition: 2, ExpectedVersion: 2, Payload: []byte(`{"v":"doudizhu-live-pass-command-v1"}`)}
	result := gamecore.CommandOutcome{Version: 3, Payload: []byte(`{"ok":true}`)}
	if err := journal.Append(ctx, id, epoch, "old-owner", "kill-2", next, result); !errors.Is(err, mysqlstore.ErrLiveWriterFenced) {
		t.Fatalf("killed owner was not fenced: %v", err)
	}
	if err := journal.Append(ctx, id, newEpoch, "new-owner", "kill-2", next, result); err != nil {
		t.Fatalf("new owner cannot advance: %v", err)
	}
	actions, err = journal.Read(ctx, id)
	if err != nil || len(actions) != 2 || actions[1].Version != 3 {
		t.Fatalf("lost new owner append: %v %+v", err, actions)
	}
}

func killLiveJournalChildAtMarker(t *testing.T, mode string, id domain.HandID, epoch uint64) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	child := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestLiveJournalSurvivesActualWriterProcessKill$")
	child.Env = append(os.Environ(), crashChildEnvironment+"="+mode, "DOUDIZHU_CRASH_HAND="+string(id), "DOUDIZHU_CRASH_EPOCH="+strconv.FormatUint(epoch, 10))
	out, err := child.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	child.Stderr = os.Stderr
	if err := child.Start(); err != nil {
		t.Fatal(err)
	}
	line, readErr := bufio.NewReader(out).ReadString('\n')
	if readErr != nil || strings.TrimSpace(line) != "CRASH_POINT_"+mode {
		_ = child.Process.Kill()
		_ = child.Wait()
		t.Fatalf("child did not reach %s: marker=%q err=%v", mode, line, readErr)
	}
	if err := child.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	if err := child.Wait(); err == nil {
		t.Fatal("child exited normally instead of SIGKILL")
	}
	if ctx.Err() != nil {
		t.Fatalf("process kill timed out: %v", ctx.Err())
	}
}

func runLiveJournalCrashChild(t *testing.T, mode string) {
	id := domain.HandID(os.Getenv("DOUDIZHU_CRASH_HAND"))
	epoch, err := strconv.ParseUint(os.Getenv("DOUDIZHU_CRASH_EPOCH"), 10, 64)
	if err != nil || id == "" {
		t.Fatal("invalid crash child identity")
	}
	if mode != "before-commit" && mode != "during-transaction" && mode != "after-commit" {
		t.Fatal("invalid crash child mode")
	}
	if mode == "during-transaction" {
		db := openIntegrationDB(t)
		tx, err := db.BeginTx(context.Background(), &sql.TxOptions{Isolation: sql.LevelReadCommitted})
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback()
		var storedEpoch, version uint64
		if err := tx.QueryRowContext(context.Background(), `SELECT owner_epoch, committed_version FROM doudizhu_live_writers
WHERE hand_id=? FOR UPDATE`, id).Scan(&storedEpoch, &version); err != nil || storedEpoch != epoch || version != 1 {
			t.Fatalf("child writer not authoritative: epoch %d version %d err %v", storedEpoch, version, err)
		}
		outcome := gamecore.CommandOutcome{Version: 2, Payload: []byte(`{"ok":true}`)}
		encoded, err := json.Marshal(outcome)
		if err != nil {
			t.Fatal(err)
		}
		digest := sha256.Sum256(encoded)
		if _, err := tx.ExecContext(context.Background(), `INSERT INTO doudizhu_live_actions
(hand_id,state_version,owner_epoch,command_id,actor_account_id,actor_position,
expected_version,command_payload,result_digest,committed_at)
VALUES(?,2,?,'uncommitted-command-1','kill-1',1,1,?, ?,UTC_TIMESTAMP(6))`,
			id, epoch, []byte(`{"v":"doudizhu-live-bid-command-v1","score":1}`), digest[:]); err != nil {
			t.Fatal(err)
		}
		if _, err := tx.ExecContext(context.Background(), `UPDATE doudizhu_live_writers
SET committed_version=2 WHERE hand_id=? AND owner_epoch=? AND committed_version=1`, id, epoch); err != nil {
			t.Fatal(err)
		}
	}
	if mode == "after-commit" {
		db := openIntegrationDB(t)
		journal, err := mysqlstore.NewLiveJournal(db)
		if err != nil {
			t.Fatal(err)
		}
		command := gamecore.Command{ActorPosition: 1, ExpectedVersion: 1, Payload: []byte(`{"v":"doudizhu-live-bid-command-v1","score":1}`)}
		outcome := gamecore.CommandOutcome{Version: 2, Payload: []byte(`{"ok":true}`)}
		if err := journal.Append(context.Background(), id, epoch, "kill-command-1", "kill-1", command, outcome); err != nil {
			t.Fatal(err)
		}
	}
	fmt.Fprintln(os.Stdout, "CRASH_POINT_"+mode)
	time.Sleep(time.Minute)
	t.Fatal("child was not killed")
}
