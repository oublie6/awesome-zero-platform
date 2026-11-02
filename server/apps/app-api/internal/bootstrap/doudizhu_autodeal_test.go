package bootstrap

import (
 "context"
 "errors"
 "testing"
 "time"

 "github.com/oublie6/awesome-zero-platform/server/business/doudizhu/application"
 "github.com/oublie6/awesome-zero-platform/server/business/doudizhu/application/lifecycle"
 "github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
 ddzruntime "github.com/oublie6/awesome-zero-platform/server/business/doudizhu/infrastructure/runtime"
)
type dealerStoreFixture struct{
 snaps map[domain.HandID]domain.HandSnapshot
 seq uint64
}
func (s *dealerStoreFixture) PendingDeals(_ context.Context,max int)([]domain.HandID,error){
 if max!=64 {panic("unbounded scan")}
 out:=make([]domain.HandID,0)
 for id:=range s.snaps{out=append(out,id)}
 return out,nil
}
func (s *dealerStoreFixture) LoadHand(_ context.Context,id domain.HandID)(domain.HandSnapshot,error){
 return s.snaps[id],nil
}
func (s *dealerStoreFixture) HandClientSequence(_ context.Context,id domain.HandID,actor domain.AccountID)(uint64,error){
 if actor!=lifecycle.SystemActor {panic("system actor not used")}
 return s.seq,nil
}
type dealerWriterFixture struct{
 store *dealerStoreFixture
 locked,dealt,aborted int
}
func (w *dealerWriterFixture) LockHandBeacon(_ context.Context,actor domain.AccountID,
 cmd application.Command,input application.LockBeaconInput)(application.CommandResult,error){
 if actor!=lifecycle.SystemActor || cmd.Name!=application.CommandHandBeaconLock ||
  cmd.ExpectedVersion!=4 || cmd.ClientSeq!=9 ||
  input.Value.Provider!=ddzruntime.DrandQuicknetProvider{panic("beacon not signed/system-scoped")}
 w.locked++
 snap:=w.store.snaps[domain.HandID(cmd.AggregateID)]
 snap.Phase=domain.HandDealing
 snap.Version++
 w.store.snaps[snap.ID]=snap
 w.store.seq=9
 return application.CommandResult{Accepted:true},nil
}
func (w *dealerWriterFixture) AbortHand(_ context.Context,actor domain.AccountID,
 cmd application.Command,input application.TerminateHandInput)(application.CommandResult,error){
 if actor!=lifecycle.SystemActor || input.ReasonCode!="public_beacon_round_expired" ||
  cmd.Name!=application.CommandHandAbort{panic("untrusted abort")}
 w.aborted++
 snap:=w.store.snaps[domain.HandID(cmd.AggregateID)]
 snap.Phase=domain.HandAborted
 snap.Version++
 w.store.snaps[snap.ID]=snap
 return application.CommandResult{Accepted:true},nil
}
func (w *dealerWriterFixture) MarkHandDealt(_ context.Context,actor domain.AccountID,
 cmd application.Command)(application.CommandResult,error){
 if actor!=lifecycle.SystemActor || cmd.Name!=application.CommandHandDealt ||
  cmd.ExpectedVersion!=5 || cmd.ClientSeq!=10 {panic("invalid durable dealing command")}
 w.dealt++
 snap:=w.store.snaps[domain.HandID(cmd.AggregateID)]
 snap.Phase=domain.HandBidding
 snap.Version++
 w.store.snaps[snap.ID]=snap
 return application.CommandResult{Accepted:true},nil
}
type dealerTrackerFixture struct{ called int }
func (t *dealerTrackerFixture) TrackBidding(_ context.Context,actor domain.AccountID,id domain.HandID)error{
 if actor!="player-1" || id!="hand-1" {panic("wrong tracked hand")}
 t.called++
 return nil
}
type dealerBeaconFixture struct {ready bool}
func (b *dealerBeaconFixture) Fetch(_ context.Context,plan domain.BeaconPlan)(domain.BeaconValue,error){
 if !b.ready {return domain.BeaconValue{},ddzruntime.ErrBeaconNotPublished}
 return domain.BeaconValue{Provider:plan.Provider,Round:plan.Round,Digest:domain.BeaconDigest{1},ProofRef:"drand-quicknet:verified-test"},nil
}
type dealerIDsFixture struct{n int}
func (d *dealerIDsFixture) NewID()(string,error){d.n++;return "autodeal-"+string(rune('0'+d.n)),nil}
type dealerClockFixture struct{}
func (dealerClockFixture) Now()time.Time{return time.UnixMilli(1800000000000)}
func TestAutoDealerWaitsForPinnedBeaconThenDealsExactlyOnce(t *testing.T){
 snap:=domain.HandSnapshot{ID:"hand-1",Phase:domain.HandWaitingPublicBeacon,Version:4,
  BeaconPlan:domain.BeaconPlan{Provider:ddzruntime.DrandQuicknetProvider,Round:"1234567"},
  Seats:[3]domain.HandSeat{{Seat:1,AccountID:"player-1"}}}
 store:=&dealerStoreFixture{snaps:map[domain.HandID]domain.HandSnapshot{"hand-1":snap},seq:8}
 writer:=&dealerWriterFixture{store:store}
 tracker:=&dealerTrackerFixture{}
 beacon:=&dealerBeaconFixture{}
 ids:=&dealerIDsFixture{}
 dealer,err:=newFairAutoDealer(store,writer,tracker,beacon,ids,dealerClockFixture{},time.Minute)
 if err!=nil{t.Fatal(err)}
 if progressed,err:=dealer.ProcessOnce(context.Background());err!=nil || progressed!=0 ||
  writer.locked!=0 || writer.dealt!=0{t.Fatalf("future round caused premature deal: %d %v",progressed,err)}
 beacon.ready=true
 if progressed,err:=dealer.ProcessOnce(context.Background());err!=nil || progressed!=1 ||
  writer.locked!=1 || writer.dealt!=0{t.Fatalf("verified beacon did not lock: %d %v",progressed,err)}
 if progressed,err:=dealer.ProcessOnce(context.Background());err!=nil || progressed!=1 ||
  writer.dealt!=1 || tracker.called!=1 {t.Fatalf("durable dealing failed: %d %v",progressed,err)}
 if progressed,err:=dealer.ProcessOnce(context.Background());err!=nil || progressed!=0 ||
  writer.dealt!=1 {t.Fatalf("hand dealt twice: %d %v",progressed,err)}
}
func TestAutoDealerNoFallbackToForgedBeacon(t *testing.T){
 s:=&dealerStoreFixture{snaps:map[domain.HandID]domain.HandSnapshot{"h":{
  ID:"h",Phase:domain.HandWaitingPublicBeacon,Version:3,
  BeaconPlan:domain.BeaconPlan{Provider:"legacy-hmac",Round:"123"},
 }}}
 w:=&dealerWriterFixture{store:s}
 d,err:=newFairAutoDealer(s,w,&dealerTrackerFixture{},&dealerBeaconFixture{ready:true},
  &dealerIDsFixture{},dealerClockFixture{},time.Minute)
 if err!=nil{t.Fatal(err)}
 if count,err:=d.ProcessOnce(context.Background());err!=nil || count!=0 || w.locked!=0 {
  t.Fatalf("untrusted beacon auto-accepted: %d %v",count,err)
 }
 if _,err:=newFairAutoDealer(s,w,&dealerTrackerFixture{},nil,
  &dealerIDsFixture{},dealerClockFixture{},time.Minute);err==nil{t.Fatal("missing verifier accepted")}
 if !errors.Is(ddzruntime.ErrBeaconNotPublished,ddzruntime.ErrBeaconNotPublished){t.Fatal("bad sentinel")}
}

func TestAutoDealerAbortsUnfinishedFutureBeaconAfterRoundIsPublished(t *testing.T){
 s:=&dealerStoreFixture{snaps:map[domain.HandID]domain.HandSnapshot{
  "expired":{ID:"expired",Phase:domain.HandFairnessCommitting,Version:2,
   BeaconPlan:domain.BeaconPlan{Provider:ddzruntime.DrandQuicknetProvider,Round:"1000"}},
 }}
 w:=&dealerWriterFixture{store:s}
 d,err:=newFairAutoDealer(s,w,&dealerTrackerFixture{},&dealerBeaconFixture{ready:true},
  &dealerIDsFixture{},dealerClockFixture{},time.Minute)
 if err!=nil{t.Fatal(err)}
 count,err:=d.ProcessOnce(context.Background())
 if err!=nil || count!=1 || w.aborted!=1 || w.locked!=0 || w.dealt!=0 ||
  s.snaps["expired"].Phase!=domain.HandAborted{
  t.Fatalf("expired reveal could bias known beacon: count=%d aborted=%d err=%v",count,w.aborted,err)
 }
}
