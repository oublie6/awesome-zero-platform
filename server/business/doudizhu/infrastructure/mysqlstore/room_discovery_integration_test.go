//go:build integration

package mysqlstore_test

import (
 "context"
 "encoding/json"
 "fmt"
 "os"
 "testing"
 "time"

 "github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
 "github.com/oublie6/awesome-zero-platform/server/business/doudizhu/infrastructure/mysqlstore"
)

func TestRecoverableRoomDiscoveryWithRealMySQL(t *testing.T){
 if os.Getenv("APP_API_INTEGRATION")!="1"{t.Skip("APP_API_INTEGRATION is not enabled")}
 db:=openIntegrationDB(t)
 ctx:=context.Background()
 prefix:=fmt.Sprintf("recover-%d",time.Now().UnixNano())
 room,_,err:=domain.NewRoom(domain.RoomID(prefix+"-active"),domain.AccountID(prefix+"-A"))
 if err!=nil{t.Fatal(err)}
 joined:=domain.AccountID(prefix+"-B")
 if _,err=room.Join(joined,room.Snapshot().Version);err!=nil{t.Fatal(err)}
 for _,actor:=range []domain.AccountID{domain.AccountID(prefix+"-A"),joined}{
  if _,err=room.SetReady(actor,true,room.Snapshot().Version);err!=nil{t.Fatal(err)}
 }
 snapshot:=room.Snapshot()
 encoded,err:=json.Marshal(snapshot)
 if err!=nil{t.Fatal(err)}
 now:=time.Now().UTC()
 _,err=db.ExecContext(ctx,`
 INSERT INTO doudizhu_rooms(room_id,owner_account_id,status,active_hand_id,aggregate_version,
 snapshot_json,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?)`,
 snapshot.ID,snapshot.OwnerID,snapshot.Status,nil,snapshot.Version,encoded,now,now)
 if err!=nil{t.Fatal(err)}
 t.Cleanup(func(){_,_=db.Exec("DELETE FROM doudizhu_rooms WHERE room_id=?",snapshot.ID)})
 store,err:=mysqlstore.New(sqlDatabase{db});if err!=nil{t.Fatal(err)}
 mine,err:=store.FindRecoverableRoomIDs(ctx,joined)
 if err!=nil || len(mine)!=1 || mine[0]!=snapshot.ID{
  t.Fatalf("member discovery did not match Go snapshot JSON: ids=%v err=%v",mine,err)
 }
 // Similar-looking account names must not be accepted as a substring match.
 for _,stranger:=range []domain.AccountID{domain.AccountID(prefix+"-B2"),domain.AccountID(prefix+"-X")}{
  ids,queryErr:=store.FindRecoverableRoomIDs(ctx,stranger)
  if queryErr!=nil || len(ids)!=0{t.Fatalf("account %q recovered another actor's room: %v %v",stranger,ids,queryErr)}
 }
 _,err=db.ExecContext(ctx,"UPDATE doudizhu_rooms SET status='CLOSED' WHERE room_id=?",snapshot.ID)
 if err!=nil{t.Fatal(err)}
 ids,err:=store.FindRecoverableRoomIDs(ctx,joined)
 if err!=nil || len(ids)!=0{t.Fatalf("closed room discovered: %v %v",ids,err)}
}
