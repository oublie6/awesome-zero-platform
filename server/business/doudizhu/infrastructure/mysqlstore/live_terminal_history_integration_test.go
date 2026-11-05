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

// Persist a domain expiry after a live action COMMIT, but leave archive
// unfinished, then verify restart compensation and exact cold live receipts.
func TestExpiredHandColdJournalReceiptAndArchiveRecovery(t *testing.T) {
 if os.Getenv("APP_API_INTEGRATION") != "1" { t.Skip("MySQL integration disabled") }
 db:=openIntegrationDB(t)
 ctx,cancel:=context.WithTimeout(context.Background(),45*time.Second)
 defer cancel()
 id:=domain.HandID(fmt.Sprintf("expired-history-%d",time.Now().UnixNano()))
 roomID:=domain.RoomID("room-"+string(id))
 t.Cleanup(func(){
  _,_=db.Exec("DELETE FROM doudizhu_live_checkpoints WHERE hand_id=?",id)
  _,_=db.Exec("DELETE FROM doudizhu_live_actions WHERE hand_id=?",id)
  _,_=db.Exec("DELETE FROM doudizhu_live_writers WHERE hand_id=?",id)
  _,_=db.Exec("DELETE FROM doudizhu_protected_server_seeds WHERE hand_id=?",id)
  _,_=db.Exec("DELETE FROM doudizhu_hands WHERE hand_id=?",id)
  _,_=db.Exec("DELETE FROM doudizhu_rooms WHERE room_id=?",roomID)
 })
 seed,genesis,biddingHand:=coordinatorCrashFixture(t,id)
 genesis.RoomID=roomID
 biddingHand.RoomID=roomID
 room,_,err:=domain.NewRoom(roomID,"crash-player-1")
 if err!=nil {t.Fatal(err)}
 roomSnapshot:=room.Snapshot()
 roomJSON,err:=json.Marshal(roomSnapshot)
 if err!=nil {t.Fatal(err)}
 now:=time.Now().UTC()
 if _,err=db.ExecContext(ctx,`INSERT INTO doudizhu_rooms
 (room_id,owner_account_id,status,active_hand_id,aggregate_version,snapshot_json,created_at,updated_at)
 VALUES (?,?,?,?,?,?,?,?)`,roomID,roomSnapshot.OwnerID,roomSnapshot.Status,nil,roomSnapshot.Version,roomJSON,now,now);err!=nil {t.Fatal(err)}
 protected,err:=mysqlstore.NewProtectedSeedStore(db,coordinatorCrashKey())
 if err!=nil {t.Fatal(err)}
 if err=protected.Save(ctx,id,seed);err!=nil {t.Fatal(err)}
 journal,err:=mysqlstore.NewLiveJournal(db)
 if err!=nil {t.Fatal(err)}
 if err=journal.ConfigureCheckpoints(coordinatorCrashKey());err!=nil {t.Fatal(err)}
 if err=journal.Create(ctx,genesis);err!=nil {t.Fatal(err)}
 fresh:=newCoordinatorCrashRuntime(t,db)
 loadBidding:=func(_ context.Context,target domain.HandID)(domain.HandSnapshot,error) {
  if target!=id {return domain.HandSnapshot{},application.ErrNotFound}
  return biddingHand,nil
 }
 if err=fresh.RecoverActive(ctx,loadBidding);err!=nil {t.Fatal(err)}
 public,err:=fresh.PublicView(ctx,id,"crash-player-1")
 if err!=nil {t.Fatal(err)}
 var view livehand.PublicView
 if err=json.Unmarshal(public.Payload,&view);err!=nil {t.Fatal(err)}
 actor:=domain.AccountID(fmt.Sprintf("crash-player-%d",view.Bidding.FirstBidder))
 accepted,err:=fresh.BidCommitted(ctx,id,actor,"expired-cold-bid",1,bidding.ScoreOne)
 if err!=nil || accepted.Version!=2 {t.Fatalf("accepted %+v %v",accepted,err)}
 hand,err:=domain.RestoreHand(biddingHand)
 if err!=nil {t.Fatal(err)}
 if _,err=hand.Expire("bidding_timeout",hand.Snapshot().Version);err!=nil {t.Fatal(err)}
 terminal:=hand.Snapshot()
 encoded,err:=json.Marshal(terminal)
 if err!=nil {t.Fatal(err)}
 if _,err=db.ExecContext(ctx,`INSERT INTO doudizhu_hands
 (hand_id,room_id,phase,reveal_key_id,reveal_public_key_sha256,aggregate_version,snapshot_json,created_at,updated_at)
 VALUES (?,?,?,?,?,?,?,?,?)`,id,roomID,terminal.Phase,terminal.RevealKeyID,terminal.RevealPublicKeySHA256[:],
 terminal.Version,encoded,now,now);err!=nil {t.Fatal(err)}
 restored:=newCoordinatorCrashRuntime(t,db)
 if err=restored.RecoverActive(ctx,func(_ context.Context,target domain.HandID)(domain.HandSnapshot,error){
  if target!=id {return domain.HandSnapshot{},application.ErrNotFound}
  return terminal,nil
 });err!=nil {t.Fatalf("expired recovery/compensated archive: %v",err)}
 if restored.Contains(id) {t.Fatal("terminal game resurrected as active")}
 replay,err:=restored.BidCommitted(ctx,id,actor,"expired-cold-bid",1,bidding.ScoreOne)
 if err!=nil || !replay.Historical || replay.Version!=accepted.Version {
  t.Fatalf("expired hot receipt missing: %+v %v",replay,err)
 }
 // A new coordinator without a hot entry must authenticate and replay the
 // exact historical request directly from MySQL rather than re-execute it.
 third:=newCoordinatorCrashRuntime(t,db)
 replay,err=third.BidCommitted(ctx,id,actor,"expired-cold-bid",1,bidding.ScoreOne)
 if err!=nil || !replay.Historical || replay.Version!=accepted.Version {
  t.Fatalf("expired cold receipt missing: %+v %v",replay,err)
 }
 if _,err=third.BidCommitted(ctx,id,"wrong-player","expired-cold-bid",1,bidding.ScoreOne);
  !errors.Is(err,gamecore.ErrVerificationFailed) {t.Fatalf("wrong actor accepted cold receipt: %v",err)}
}
