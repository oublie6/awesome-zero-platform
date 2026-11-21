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
	ddzruntime "github.com/oublie6/awesome-zero-platform/server/business/doudizhu/infrastructure/runtime"
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
	// An early automated ticket is also a provably uncommitted rejection,
	// and must not quarantine the next legitimate human writer.
	if err := journal.Append(ctx, id, epoch, "turn-timeout:too-early", "crash-player-1", cmd,
		gamecore.CommandOutcome{Version: 2, Payload: payload}); !errors.Is(err, mysqlstore.ErrTurnExpired) || !errors.Is(err, gamecore.ErrCommitRejected) {
		t.Fatalf("early timeout was not rejected before COMMIT: %v", err)
	}
	if err := journal.Append(ctx, id, epoch, "human-1", "crash-player-1", cmd,
		gamecore.CommandOutcome{Version: 2, Payload: payload}); err != nil {
		t.Fatal(err)
	}
	newer, _, err := journal.GetTurnDeadline(ctx, id)
	if err != nil || newer.Version != 2 || newer.Seat != 2 || !newer.Deadline.After(initial.Deadline.Add(-time.Minute)) {
		t.Fatalf("next turn=%+v err=%v", newer, err)
	}
	// Fenced owners and stale committed versions must NEVER carry the
	// 'known uncommitted business rejection' marker.
	for _, tc := range []struct{name string; epoch uint64; cmd gamecore.Command; version uint64; expected error}{
		{"fenced owner", epoch-1, gamecore.Command{ActorPosition:2, ExpectedVersion:2, Payload:cmd.Payload}, 3, mysqlstore.ErrLiveWriterFenced},
		{"stale watermark", epoch, cmd, 2, mysqlstore.ErrLiveWatermarkConflict},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := journal.Append(ctx,id,tc.epoch,"stale-"+tc.name,"crash-player-2",tc.cmd,
				gamecore.CommandOutcome{Version:tc.version,Payload:payload})
			if !errors.Is(err,tc.expected) || errors.Is(err,gamecore.ErrCommitRejected) {
				t.Fatalf("unsafe stale/fenced state was classified as safe: %v",err)
			}
		})
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

 
// Round8 P1: exercise the *actual* candidate -> MySQL turn guard -> live
// directory -> DQ system command flow. The previous repository-only test
// missed that ApplyCommitted quarantined the entire in-memory game after a
// legal, late human bid was rejected before any SQL action could COMMIT.
func TestDQExpiredHumanBidKeepsCoordinatorLiveAndAutoBidCommits(t *testing.T) {
	if os.Getenv("APP_API_INTEGRATION") != "1" {
		t.Skip("real MySQL integration disabled")
	}
	db := openIntegrationDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
	defer cancel()
	id := domain.HandID(fmt.Sprintf("g63-r8-repair-%d", time.Now().UnixNano()))
	t.Cleanup(func() {
		for _, query := range []string{
			"DELETE FROM doudizhu_turn_deadlines WHERE hand_id=?",
			"DELETE FROM doudizhu_live_checkpoints WHERE hand_id=?",
			"DELETE FROM doudizhu_live_actions WHERE hand_id=?",
			"DELETE FROM doudizhu_live_writers WHERE hand_id=?",
			"DELETE FROM doudizhu_protected_server_seeds WHERE hand_id=?",
		} {
			if _, err := db.Exec(query, id); err != nil {
				t.Errorf("cleanup isolated live hand: %v", err)
			}
		}
	})
	seed, genesis, _ := coordinatorCrashFixture(t, id)
	protected, err := mysqlstore.NewProtectedSeedStore(db, coordinatorCrashKey())
	if err != nil { t.Fatal(err) }
	if err := protected.Save(ctx, id, seed); err != nil { t.Fatal(err) }
	seeds, err := ddzruntime.NewPersistentSeedVault(nil, protected)
	if err != nil { t.Fatal(err) }
	dir, err := gamecore.NewLiveDirectory(noopCoordinatorArchive{})
	if err != nil { t.Fatal(err) }
	journal, err := mysqlstore.NewLiveJournal(db)
	if err != nil { t.Fatal(err) }
	if err := journal.ConfigureCheckpoints(coordinatorCrashKey()); err != nil { t.Fatal(err) }
	if err := journal.EnableTurnDeadlines(ctx, mysqlstore.TurnTimeouts{Bidding:time.Hour, Playing:time.Hour}); err != nil { t.Fatal(err) }
	coordinator, err := ddzruntime.NewDurableLiveHandCoordinator(seeds, dir, journal)
	if err != nil { t.Fatal(err) }
	if err := coordinator.Start(ctx, genesis); err != nil { t.Fatal(err) }

	view, err := coordinator.PublicView(ctx, id, "crash-player-1")
	if err != nil { t.Fatal(err) }
	var public livehand.PublicView
	if err := json.Unmarshal(view.Payload,&public); err != nil { t.Fatal(err) }
	bidder := domain.AccountID(fmt.Sprintf("crash-player-%d", public.Bidding.CurrentBidder))
	if bidder == "" || public.Bidding.CurrentBidder < 1 || public.Bidding.CurrentBidder > 3 {
		t.Fatalf("invalid live bidder: %+v",public.Bidding)
	}
	before,_,err := journal.GetTurnDeadline(ctx,id)
	if err != nil || before.Version != 1 || before.Seat != public.Bidding.CurrentBidder {
		t.Fatalf("initial authoritative deadline disagrees with live game: %+v %v",before,err)
	}
	// No sleeping or queue timing dependency: reproduce the late-human edge
	// using the MySQL clock, just as the real worker checks it.
	if _,err := db.ExecContext(ctx, `UPDATE doudizhu_turn_deadlines
SET deadline_at=DATE_SUB(UTC_TIMESTAMP(3),INTERVAL 1 SECOND)
WHERE hand_id=? AND live_version=1`,id); err != nil { t.Fatal(err) }
	for seq:=uint64(1);seq<=2;seq++ {
		_,err := coordinator.BidCommittedWithSequence(ctx,id,bidder,
			fmt.Sprintf("r8-late-human-%d",seq),1,bidding.ScoreZero,seq)
		if !errors.Is(err,gamecore.ErrCommitRejected) || !errors.Is(err,mysqlstore.ErrTurnExpired) {
			t.Fatalf("late human request must be deterministic conflict, not ambiguous COMMIT: %v",err)
		}
		recovered,err := coordinator.PrivateView(ctx,id,bidder)
		if err != nil || recovered.Version!=1 {
			t.Fatalf("late human bid quarantined live game: %+v %v",recovered,err)
		}
		actions,err := journal.Read(ctx,id)
		if err!=nil || len(actions)!=0 { t.Fatalf("uncommitted action became durable: %+v %v",actions,err) }
	}
	timeout,err:=coordinator.BidCommitted(ctx,id,bidder,"turn-timeout:repair-round8",1,bidding.ScoreZero)
	if err!=nil || timeout.Version!=2 { t.Fatalf("server timeout failed to progress after rejection: %+v %v",timeout,err) }
	actions,err:=journal.Read(ctx,id)
	if err!=nil || len(actions)!=1 || actions[0].CommandID!="turn-timeout:repair-round8" ||
		actions[0].ExpectedVersion!=1 || actions[0].Version!=2 {
		t.Fatalf("timeout action was not committed exactly once: %+v %v",actions,err)
	}
	after,_,err:=journal.GetTurnDeadline(ctx,id)
	if err!=nil || after.Version!=2 || after.Seat==before.Seat || !after.Deadline.After(before.Deadline) {
		// We manually moved the original deadline into the past; the next
		// committed turn must have a fresh authoritative time and seat.
		t.Fatalf("next turn did not advance: before=%+v after=%+v err=%v",before,after,err)
	}
	next:=domain.AccountID(fmt.Sprintf("crash-player-%d",after.Seat))
	accepted,err:=coordinator.BidCommittedWithSequence(ctx,id,next,"r8-new-turn-human",2,bidding.ScoreOne,5)
	if err!=nil || accepted.Version!=3 { t.Fatalf("new current player cannot act: %+v %v",accepted,err) }
	actions,err=journal.Read(ctx,id)
	if err!=nil || len(actions)!=2 || actions[0].Version!=2 || actions[1].Version!=3 {
		t.Fatalf("newly accepted action not durable: %+v %v",actions,err)
	}
}
