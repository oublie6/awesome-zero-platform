//go:build integration

package mysqlstore_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/carddeck"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/infrastructure/mysqlstore"
	"github.com/oublie6/awesome-zero-platform/server/business/gamecore"
	"os"
)

func TestLiveJournalFenceContiguousLogAndProtectedSeedInMySQL(t *testing.T) {
	if os.Getenv("APP_API_INTEGRATION")!="1" {t.Skip("mysql integration disabled")}
	db:=openIntegrationDB(t)
	ctx:=context.Background()
	id:=domain.HandID(fmt.Sprintf("live-journal-%d",time.Now().UnixNano()))
	defer func(){
		db.Exec("DELETE FROM doudizhu_live_actions WHERE hand_id=?",id)
		db.Exec("DELETE FROM doudizhu_live_writers WHERE hand_id=?",id)
		db.Exec("DELETE FROM doudizhu_protected_server_seeds WHERE hand_id=?",id)
	}()
	seats:=[3]domain.HandSeat{
		{Seat:1,AccountID:"player-1"},{Seat:2,AccountID:"player-2"},{Seat:3,AccountID:"player-3"},
	}
	server:=domain.ServerCommitment{1}
	hand,_,err:=domain.NewHand(id,domain.RoomID("room-test"),seats,server,"test-key",domain.BeaconPlan{Provider:"test-provider",Round:"test-round"})
	if err!=nil{t.Fatal(err)}
	for index,seat:=range seats {
		var contribution domain.ContributionDigest
		contribution[0]=byte(index+1)
		commitment:=domain.ComputeClientCommitment(id,seat.Seat,contribution)
		if _,err:=hand.SubmitCommit(seat.AccountID,commitment,hand.Snapshot().Version);err!=nil{t.Fatal(err)}
	}
	for index,seat:=range seats {
		var contribution domain.ContributionDigest
		contribution[0]=byte(index+1)
		if _,err:=hand.SubmitReveal(seat.AccountID,contribution,fmt.Sprintf("record-%d",index),hand.Snapshot().Version);err!=nil{t.Fatal(err)}
	}
	beacon:=domain.BeaconDigest{3}
	if _,err:=hand.LockPublicBeacon(domain.BeaconValue{Provider:"test-provider",Round:"test-round",Digest:beacon,ProofRef:"proof-test"},hand.Snapshot().Version);err!=nil{t.Fatal(err)}
	if hand.Snapshot().Phase!=domain.HandDealing{t.Fatalf("genesis not dealing")}
	j,err:=mysqlstore.NewLiveJournal(db)
	if err!=nil{t.Fatal(err)}
	if err:=j.Create(ctx,hand.Snapshot());err!=nil{t.Fatal(err)}
	key:=bytes.Repeat([]byte{0x5a},32)
	seeds,err:=mysqlstore.NewProtectedSeedStore(db,key)
	if err!=nil{t.Fatal(err)}
	var original carddeck.Seed
	for i:=range original {original[i]=byte(i+1)}
	if err:=seeds.Save(ctx,id,original);err!=nil{t.Fatal(err)}
	restored,err:=mysqlstore.NewProtectedSeedStore(db,key)
	if err!=nil{t.Fatal(err)}
	recovered,err:=restored.Load(ctx,id)
	if err!=nil || recovered!=original{t.Fatalf("seed not restored: %v",err)}
	var stored []byte
	if err:=db.QueryRow("SELECT ciphertext FROM doudizhu_protected_server_seeds WHERE hand_id=?",id).Scan(&stored);err!=nil{t.Fatal(err)}
	if bytes.Contains(stored,original[:]){t.Fatal("unencrypted seed persisted")}
	epoch,version,genesis,err:=j.Acquire(ctx,id)
	if err!=nil || epoch!=2 || version!=1 || genesis.ID!=id{t.Fatalf("takeover epoch=%d version=%d genesis=%s err=%v",epoch,version,genesis.ID,err)}
	action:=gamecore.Command{ActorPosition:1,ExpectedVersion:1,Payload:[]byte(`{"v":"doudizhu-live-bid-command-v1","score":1}`)}
	result:=gamecore.CommandOutcome{Version:2,Payload:[]byte(`{"ok":true}`)}
	if err:=j.Append(ctx,id,epoch,"act1","player-1",action,result);err!=nil{t.Fatal(err)}
	actions,err:=j.Read(ctx,id)
	if err!=nil || len(actions)!=1 || actions[0].Version!=2 || actions[0].CommandID!="act1"{
		t.Fatalf("log=%+v err=%v",actions,err)
	}
	newEpoch,watermark,_,err:=j.Acquire(ctx,id)
	if err!=nil || newEpoch!=3 || watermark!=2{t.Fatalf("second owner epoch=%d watermark=%d err=%v",newEpoch,watermark,err)}
	next:=gamecore.Command{ActorPosition:2,ExpectedVersion:2,Payload:[]byte(`{"v":"doudizhu-live-pass-command-v1"}`)}
	if err:=j.Append(ctx,id,epoch,"stale","player-2",next,gamecore.CommandOutcome{Version:3,Payload:[]byte(`{"ok":true}`)});!errors.Is(err,mysqlstore.ErrLiveWriterFenced){
		t.Fatalf("old owner not fenced: %v",err)
	}
	if err:=j.Append(ctx,id,newEpoch,"gap","player-2",gamecore.Command{ActorPosition:2,ExpectedVersion:4,Payload:next.Payload},gamecore.CommandOutcome{Version:5,Payload:[]byte(`{"ok":true}`)});!errors.Is(err,mysqlstore.ErrLiveWatermarkConflict){
		t.Fatalf("version gap accepted: %v",err)
	}
	if _,err:=db.Exec("UPDATE doudizhu_live_actions SET result_digest=REPEAT(0x00,32) WHERE hand_id=? AND state_version=2",id);err!=nil{t.Fatal(err)}
	// The loader returns hashes, and recovery must validate them against
	// deterministic Go Apply output before exposing a hand.
	tampered,err:=j.Read(ctx,id)
	if err!=nil || len(tampered)!=1 || tampered[0].ResultDigest==actions[0].ResultDigest {t.Fatal("tamper was not observable to replay verifier")}
	if _,err:=db.Exec("UPDATE doudizhu_protected_server_seeds SET ciphertext=REPEAT(0x00,48) WHERE hand_id=?",id);err!=nil{t.Fatal(err)}
	if _,err:=restored.Load(ctx,id);err==nil{t.Fatal("tampered encrypted seed accepted")}
}
