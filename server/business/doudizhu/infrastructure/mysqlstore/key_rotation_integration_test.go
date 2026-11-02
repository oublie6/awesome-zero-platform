//go:build integration

package mysqlstore_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/infrastructure/mysqlstore"
	"github.com/oublie6/awesome-zero-platform/server/business/gamecore"
)

// Real MySQL rotation is fail-closed: only an epoch owner can rewrap an
// authenticated checkpoint; the original fairness seed is never regenerated.
func TestProtectedLiveSecretsRotateWithoutChangingAcknowledgedGame(t *testing.T) {
	if os.Getenv("APP_API_INTEGRATION")!="1"{t.Skip("real MySQL integration disabled")}
	db:=openIntegrationDB(t)
	ctx:=context.Background()
	id:=domain.HandID(fmt.Sprintf("rotate-live-%d",time.Now().UnixNano()))
	t.Cleanup(func(){
		for _,table:=range []string{"doudizhu_live_checkpoints","doudizhu_live_actions",
			"doudizhu_live_writers","doudizhu_protected_server_seeds"}{
			_,_=db.Exec("DELETE FROM "+table+" WHERE hand_id=?",id)
		}
	})
	oldRoot:=bytes.Repeat([]byte{0x5a},32)
	newRoot:=bytes.Repeat([]byte{0x7b},32)
	seed,genesis,_:=coordinatorCrashFixture(t,id)
	oldSeeds,err:=mysqlstore.NewProtectedSeedStore(db,oldRoot)
	if err!=nil{t.Fatal(err)}
	if err:=oldSeeds.Save(ctx,id,seed);err!=nil{t.Fatal(err)}
	oldLog,err:=mysqlstore.NewLiveJournal(db)
	if err!=nil{t.Fatal(err)}
	if err:=oldLog.ConfigureCheckpoints(oldRoot);err!=nil{t.Fatal(err)}
	if err:=oldLog.Create(ctx,genesis);err!=nil{t.Fatal(err)}
	oldEpoch,_,_,err:=oldLog.Acquire(ctx,id)
	if err!=nil{t.Fatal(err)}
	command:=gamecore.Command{ActorPosition:1,ExpectedVersion:1,
		Payload:[]byte(`{"v":"doudizhu-live-bid-command-v1","score":1}`)}
	outcome:=gamecore.CommandOutcome{Version:2,Payload:[]byte(`{"stateVersion":2}`)}
	if err:=oldLog.Append(ctx,id,oldEpoch,"rotation-command-1","crash-player-1",command,outcome);err!=nil{t.Fatal(err)}
	checkpoint:=[]byte(`{"v":"doudizhu-live-recovery-checkpoint-v1","private":"secret-proof"}`)
	if err:=oldLog.SaveCheckpoint(ctx,id,oldEpoch,2,checkpoint);err!=nil{t.Fatal(err)}

	roots:=map[string][]byte{"old":oldRoot,"next":newRoot}
	rotating,err:=mysqlstore.NewLiveJournal(db)
	if err!=nil{t.Fatal(err)}
	if err:=rotating.ConfigureCheckpointKeyring("next",roots);err!=nil{t.Fatal(err)}
	rotatingSeeds,err:=mysqlstore.NewProtectedSeedStoreKeyring(db,"next",roots)
	if err!=nil{t.Fatal(err)}
	if loaded,err:=rotatingSeeds.Load(ctx,id);err!=nil||loaded!=seed{
		t.Fatalf("old seed not recoverable during rotation: %v",err)
	}
	if cp,err:=rotating.LoadCheckpoint(ctx,id);err!=nil||!bytes.Equal(cp.Payload,checkpoint){
		t.Fatalf("old checkpoint not recoverable during rotation: %v",err)
	}
	currentEpoch,committed,_,err:=rotating.Acquire(ctx,id)
	if err!=nil||committed!=2{t.Fatalf("rotating owner not fenced: %d %v",committed,err)}
	if err:=rotating.ReencryptCheckpoint(ctx,id,oldEpoch);!errors.Is(err,mysqlstore.ErrLiveWriterFenced){
		t.Fatalf("old epoch was allowed to rewrite protected checkpoint: %v",err)
	}
	if err:=rotatingSeeds.ReencryptSeed(ctx,id,db);err!=nil{t.Fatal(err)}
	if err:=rotating.ReencryptCheckpoint(ctx,id,currentEpoch);err!=nil{t.Fatal(err)}
	newOnly,err:=mysqlstore.NewProtectedSeedStore(db,newRoot)
	if err!=nil{t.Fatal(err)}
	if recovered,err:=newOnly.Load(ctx,id);err!=nil||recovered!=seed{
		t.Fatalf("active key did not recover unchanged original seed: %v",err)
	}
	newLog,err:=mysqlstore.NewLiveJournal(db)
	if err!=nil{t.Fatal(err)}
	if err:=newLog.ConfigureCheckpoints(newRoot);err!=nil{t.Fatal(err)}
	if cp,err:=newLog.LoadCheckpoint(ctx,id);err!=nil||cp.Version!=2||!bytes.Equal(cp.Payload,checkpoint){
		t.Fatalf("active key did not recover rewrapped checkpoint: %+v %v",cp,err)
	}
	if _,err:=oldSeeds.Load(ctx,id);err==nil{t.Fatal("old-only key unexpectedly decrypted newly rotated seed")}
	if _,err:=oldLog.LoadCheckpoint(ctx,id);!errors.Is(err,mysqlstore.ErrLiveReplayIntegrity){
		t.Fatalf("old-only checkpoint key decrypted newly rotated data: %v",err)
	}
	if actions,err:=rotating.Read(ctx,id);err!=nil||len(actions)!=1||actions[0].Version!=2{
		t.Fatalf("rotation changed authoritative log: %+v %v",actions,err)
	}
}
