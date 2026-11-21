//go:build integration

package mysqlstore_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/bidding"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/livehand"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/infrastructure/mysqlstore"
	"github.com/oublie6/awesome-zero-platform/server/business/gamecore"
)

func TestDQTurnDeadlineEnforcesHumanTimeoutAndAtomicTurnAdvance(t *testing.T) {
	if os.Getenv("APP_API_INTEGRATION") != "1" {
		t.Skip("real MySQL integration disabled")
	}
	db := openIntegrationDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	id := domain.HandID(fmt.Sprintf("g63-turn-%d", time.Now().UnixNano()))
	t.Cleanup(func() {
		_, _ = db.Exec("DELETE FROM doudizhu_turn_deadlines WHERE hand_id=?", id)
		_, _ = db.Exec("DELETE FROM doudizhu_live_actions WHERE hand_id=?", id)
		_, _ = db.Exec("DELETE FROM doudizhu_live_writers WHERE hand_id=?", id)
	})
	_, genesis, _ := coordinatorCrashFixture(t, id)
	journal, err := mysqlstore.NewLiveJournal(db)
	if err != nil {
		t.Fatal(err)
	}
	if err := journal.EnableTurnDeadlines(ctx, mysqlstore.TurnTimeouts{
		Bidding: time.Hour, Playing: time.Hour,
	}); err != nil {
		t.Fatal(err)
	}
	if err := journal.Create(ctx, genesis, 1); err != nil {
		t.Fatal(err)
	}
	initial, serverTime, err := journal.GetTurnDeadline(ctx, id)
	if err != nil || initial.Version != 1 || initial.Phase != livehand.PhaseBidding || initial.Seat != 1 ||
		!initial.Deadline.After(serverTime.Add(55*time.Minute)) {
		t.Fatalf("initial turn=%+v server=%v err=%v", initial, serverTime, err)
	}
	epoch, version, _, err := journal.Acquire(ctx, id)
	if err != nil || epoch != 2 || version != 1 {
		t.Fatalf("ownership epoch=%d version=%d err=%v", epoch, version, err)
	}
	nextBid := livehand.BidResult{Version: livehand.BidResultVersion, HandID: string(id),
		StateVersion: 2, Phase: livehand.PhaseBidding, Bidding: bidding.Snapshot{CurrentBidder: 2}}
	payload, err := json.Marshal(nextBid)
	if err != nil {
		t.Fatal(err)
	}
	cmd := gamecore.Command{ActorPosition: 1, ExpectedVersion: 1, Payload: []byte(`{"v":"doudizhu-live-bid-command-v1","score":0}`)}
	if err := journal.Append(ctx, id, epoch, "human-1", "crash-player-1", cmd,
		gamecore.CommandOutcome{Version: 2, Payload: payload}); err != nil {
		t.Fatal(err)
	}
	newer, _, err := journal.GetTurnDeadline(ctx, id)
	if err != nil || newer.Version != 2 || newer.Seat != 2 || !newer.Deadline.After(initial.Deadline.Add(-time.Minute)) {
		t.Fatalf("next turn=%+v err=%v", newer, err)
	}
	if _, err = db.ExecContext(ctx, `UPDATE doudizhu_turn_deadlines
SET deadline_at=DATE_SUB(UTC_TIMESTAMP(3),INTERVAL 1 SECOND)
WHERE hand_id=?`, id); err != nil {
		t.Fatal(err)
	}
	cmd = gamecore.Command{ActorPosition: 2, ExpectedVersion: 2, Payload: []byte(`{"v":"doudizhu-live-bid-command-v1","score":0}`)}
	finished := livehand.BidResult{Version: livehand.BidResultVersion, HandID: string(id),
		StateVersion: 3, Phase: livehand.PhaseNoLandlord, RequiresTermination: true}
	finishedPayload, err := json.Marshal(finished)
	if err != nil {
		t.Fatal(err)
	}
	outcome := gamecore.CommandOutcome{Version: 3, Payload: finishedPayload}
	if err := journal.Append(ctx, id, epoch, "human-late", "crash-player-2", cmd, outcome); !errors.Is(err, mysqlstore.ErrTurnExpired) || !errors.Is(err, gamecore.ErrCommitRejected) {
		t.Fatalf("human committed after deadline or lost pre-COMMIT rejection marker: %v", err)
	}
	if err := journal.Append(ctx, id, epoch, "turn-timeout:unit-2", "crash-player-2", cmd, outcome); err != nil {
		t.Fatalf("authoritative expired turn should be consumed: %v", err)
	}
	marker, _, err := journal.GetTurnDeadline(ctx, id)
	if err != nil || marker.Version != 3 || marker.Phase != livehand.PhaseNoLandlord {
		t.Fatalf("terminal commit lost durable lifecycle repair: marker=%+v err=%v", marker, err)
	}
	if err = journal.ClearTerminalRepair(ctx, marker); err != nil {
		t.Fatal(err)
	}
	if _, _, err = journal.GetTurnDeadline(ctx, id); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("terminal repair marker was not retired: %v", err)
	}
	log, err := journal.Read(ctx, id)
	if err != nil || len(log) != 2 || log[1].CommandID != "turn-timeout:unit-2" {
		t.Fatalf("live action journal mismatch: %d %v", len(log), err)
	}
}
