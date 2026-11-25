package mysqlstore_test

import (
 "context"
 "encoding/json"
 "fmt"
 "sort"
 "sync"
 "testing"
 "time"
 "github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
 "github.com/oublie6/awesome-zero-platform/server/business/doudizhu/infrastructure/mysqlstore"
)
func TestRound9AuthorityConsistentReadAndCost(t *testing.T){
 db:=openIntegrationDB(t);ctx:=context.Background();prefix:=fmt.Sprintf("r9joint-%d",time.Now().UnixNano());actor:=domain.AccountID(prefix+"-A");rid:=domain.RoomID(prefix+"-room");hid:=domain.HandID(prefix+"-hand")
 defer cleanupIntegrationRows(t,db,prefix)
 room,_,err:=domain.NewRoom(rid,actor);if err!=nil{t.Fatal(err)}
 actors:=[]domain.AccountID{actor,domain.AccountID(prefix+"-B"),domain.AccountID(prefix+"-C")}
 for _,a:=range actors[1:]{if _,err=room.Join(a,room.Snapshot().Version);err!=nil{t.Fatal(err)}}
 for _,a:=range actors{if _,err=room.SetReady(a,true,room.Snapshot().Version);err!=nil{t.Fatal(err)}}
 if _,err=room.StartHand(actor,hid,room.Snapshot().Version);err!=nil{t.Fatal(err)}
 hand,_,err:=domain.NewHand(hid,rid,[3]domain.HandSeat{{Seat:1,AccountID:actors[0]},{Seat:2,AccountID:actors[1]},{Seat:3,AccountID:actors[2]}},domain.ServerCommitment{1},"joint-key",domain.BeaconPlan{Provider:"test",Round:"test"});if err!=nil{t.Fatal(err)}
 r,h:=room.Snapshot(),hand.Snapshot();r.Version=1;h.Version=1;rb,_:=json.Marshal(r);hb,_:=json.Marshal(h);now:=time.Now().UTC()
 if _,err=db.Exec(`INSERT INTO doudizhu_rooms(room_id,owner_account_id,status,active_hand_id,aggregate_version,snapshot_json,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?)`,rid,actor,r.Status,hid,1,rb,now,now);err!=nil{t.Fatal(err)}
 if _,err=db.Exec(`INSERT INTO doudizhu_hands(hand_id,room_id,phase,reveal_key_id,reveal_public_key_sha256,aggregate_version,snapshot_json,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?)`,hid,rid,h.Phase,"joint-key",make([]byte,32),1,hb,now,now);err!=nil{t.Fatal(err)}
 if _,err=db.Exec(`INSERT INTO doudizhu_client_sequences(aggregate_type,aggregate_id,actor_account_id,last_sequence,updated_at) VALUES('room',?,?,1,?)`,rid,actor,now);err!=nil{t.Fatal(err)}
 store,err:=mysqlstore.New(sqlDatabase{db});if err!=nil{t.Fatal(err)}
 query:=`SELECT r.snapshot_json AS room_snapshot, COALESCE(h.snapshot_json,'null') AS hand_snapshot, COALESCE(c.last_sequence,0) AS last_sequence FROM doudizhu_rooms r LEFT JOIN doudizhu_hands h ON h.hand_id=r.active_hand_id AND h.room_id=r.room_id LEFT JOIN doudizhu_client_sequences c ON c.aggregate_type='room' AND c.aggregate_id=r.room_id AND c.actor_account_id=? WHERE r.room_id=?`
 rows,err:=db.Query("EXPLAIN "+query,actor,rid);if err!=nil{t.Fatal(err)};cols,_:=rows.Columns();for rows.Next(){vals:=make([]interface{},len(cols));raw:=make([]interface{},len(cols));for i:=range raw{raw[i]=&vals[i]};if err=rows.Scan(raw...);err!=nil{t.Fatal(err)};entry:=map[string]string{};for i,v:=range vals{if b,ok:=v.([]byte);ok{entry[cols[i]]=string(b)}else{entry[cols[i]]=fmt.Sprint(v)}};t.Logf("EXPLAIN %v",entry);if entry["key"]!="PRIMARY"{t.Errorf("non-PK authority join: %v",entry)}};rows.Close()
 read:=func()error{a,b,seq,e:=store.LoadRoomAuthority(ctx,rid,actor);if e!=nil{return e};if b==nil||a.Version!=b.Version||a.Version!=seq{return fmt.Errorf("torn tuple room=%d hand=%v seq=%d",a.Version,b,seq)};return nil}
 errs:=make(chan error,8);var wg sync.WaitGroup
 wg.Add(1);go func(){defer wg.Done();for n:=uint64(2);n<=121;n++{rr,hh:=r,h;rr.Version=n;hh.Version=n;rb,_:=json.Marshal(rr);hb,_:=json.Marshal(hh);tx,e:=db.BeginTx(ctx,nil);if e!=nil{errs<-e;return};if _,e=tx.Exec(`UPDATE doudizhu_rooms SET snapshot_json=?,aggregate_version=? WHERE room_id=?`,rb,n,rid);e==nil{_,e=tx.Exec(`UPDATE doudizhu_hands SET snapshot_json=?,aggregate_version=? WHERE hand_id=?`,hb,n,hid)};if e==nil{_,e=tx.Exec(`UPDATE doudizhu_client_sequences SET last_sequence=? WHERE aggregate_type='room' AND aggregate_id=? AND actor_account_id=?`,n,rid,actor)};if e!=nil{tx.Rollback();errs<-e;return};if e=tx.Commit();e!=nil{errs<-e;return}}}()
 for j:=0;j<3;j++{wg.Add(1);go func(){defer wg.Done();for i:=0;i<160;i++{if e:=read();e!=nil{errs<-e;return}}}()};wg.Wait();close(errs);for e:=range errs{t.Error(e)}
 if err=read();err!=nil{t.Fatal(err)};t.Log("consistent tuples: 120 real InnoDB atomic updates / 480 concurrent production-store reads, zero torn observations")
 for _,concurrency:=range []int{1,4}{var mu sync.Mutex;var samples []int64;var w sync.WaitGroup;for j:=0;j<concurrency;j++{w.Add(1);go func(){defer w.Done();var local []int64;for i:=0;i<100;i++{start:=time.Now();e:=read();local=append(local,time.Since(start).Microseconds());if e!=nil{t.Error(e);break}};mu.Lock();samples=append(samples,local...);mu.Unlock()}()};w.Wait();sort.Slice(samples,func(i,j int)bool{return samples[i]<samples[j]});t.Logf("PRODUCTION_STORE_LATENCY concurrency=%d n=%d p50_us=%d p95_us=%d p99_us=%d room_json_bytes=%d hand_json_bytes=%d",concurrency,len(samples),samples[len(samples)*50/100],samples[len(samples)*95/100],samples[len(samples)*99/100],len(rb),len(hb))}
 if _,err=db.Exec(`UPDATE doudizhu_hands SET room_id='wrong-room' WHERE hand_id=?`,hid);err!=nil{t.Fatal(err)};if _,_,_,err=store.LoadRoomAuthority(ctx,rid,actor);err==nil{t.Error("mismatched joined hand accepted")}
}
