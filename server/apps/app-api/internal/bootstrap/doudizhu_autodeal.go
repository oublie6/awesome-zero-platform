package bootstrap

import (
 "context"
 "errors"
 "fmt"
 "time"

 "github.com/oublie6/awesome-zero-platform/server/business/doudizhu/application"
 "github.com/oublie6/awesome-zero-platform/server/business/doudizhu/application/lifecycle"
 "github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
 ddzruntime "github.com/oublie6/awesome-zero-platform/server/business/doudizhu/infrastructure/runtime"
)

type fairDealStore interface {
 PendingDeals(context.Context,int)([]domain.HandID,error)
 LoadHand(context.Context,domain.HandID)(domain.HandSnapshot,error)
 HandClientSequence(context.Context,domain.HandID,domain.AccountID)(uint64,error)
}
type fairDealWriter interface {
 LockHandBeacon(context.Context,domain.AccountID,application.Command,application.LockBeaconInput)(application.CommandResult,error)
 MarkHandDealt(context.Context,domain.AccountID,application.Command)(application.CommandResult,error)
}
type fairDealTracker interface {TrackBidding(context.Context,domain.AccountID,domain.HandID)error}
type drandReader interface {Fetch(context.Context,domain.BeaconPlan)(domain.BeaconValue,error)}
type fairAutoDealer struct {
 store fairDealStore
 writer fairDealWriter
 tracker fairDealTracker
 reader drandReader
 ids application.IDGenerator
 clock application.Clock
 ttl time.Duration
}
func newFairAutoDealer(store fairDealStore,writer fairDealWriter,tracker fairDealTracker,
 reader drandReader,ids application.IDGenerator,clock application.Clock,ttl time.Duration)(*fairAutoDealer,error){
 if store==nil || writer==nil || tracker==nil || reader==nil || ids==nil || clock==nil ||
  ttl<=0 || ttl>time.Hour{return nil,fmt.Errorf("invalid fair auto dealer dependencies")}
 return &fairAutoDealer{store:store,writer:writer,tracker:tracker,reader:reader,ids:ids,clock:clock,ttl:ttl},nil
}
func (a *fairAutoDealer) issueCommand(ctx context.Context,id domain.HandID,version uint64,
 name string,run func(application.Command)(application.CommandResult,error))error{
 seq,err:=a.store.HandClientSequence(ctx,id,lifecycle.SystemActor)
 if err!=nil{return err}
 commandID,err:=a.ids.NewID()
 if err!=nil{return err}
 now:=a.clock.Now().UTC().Truncate(time.Millisecond)
 cmd:=application.Command{Version:application.CommandProtocolV1,Name:name,CommandID:commandID,
  AggregateType:domain.AggregateHand,AggregateID:string(id),ClientSeq:seq+1,ExpectedVersion:version,
  IssuedAt:now,ExpiresAt:now.Add(a.ttl)}
 result,err:=run(cmd)
 if err!=nil{return fmt.Errorf("durable fair transition outcome unknown: %w",err)}
 if !result.Accepted {return fmt.Errorf("durable fair transition rejected: %+v",result.Failure)}
 return nil
}
// ProcessOnce never acknowledges an unverified value and never manufactures a
// new deck. The authoritative domain version decides whether another worker
// already progressed the hand. Bounded page scans limit queue pressure.
func (a *fairAutoDealer) ProcessOnce(ctx context.Context)(int,error){
 ids,err:=a.store.PendingDeals(ctx,64)
 if err!=nil{return 0,err}
 progressed:=0
 for _,id:=range ids {
  if err:=ctx.Err();err!=nil{return progressed,err}
  snap,err:=a.store.LoadHand(ctx,id)
  if err!=nil{return progressed,err}
  switch snap.Phase {
  case domain.HandWaitingPublicBeacon:
   if snap.BeaconPlan.Provider!=ddzruntime.DrandQuicknetProvider {
    // Legacy HMAC proof is never an automatic release-security fallback.
    continue
   }
   value,err:=a.reader.Fetch(ctx,snap.BeaconPlan)
   if errors.Is(err,ddzruntime.ErrBeaconNotPublished){continue}
   if err!=nil{return progressed,fmt.Errorf("read independently verified drand beacon for %s: %w",id,err)}
   err=a.issueCommand(ctx,id,snap.Version,application.CommandHandBeaconLock,
    func(cmd application.Command)(application.CommandResult,error){
     return a.writer.LockHandBeacon(ctx,lifecycle.SystemActor,cmd,application.LockBeaconInput{Value:value})
    })
   if err!=nil{return progressed,err}
   progressed++
  case domain.HandDealing:
   // Domain.MarkHandDealt creates the durably journalled live state and
   // compensates if its transaction fails; after a crash the same phase
   // is retried by a fresh instance, not replaced with a different deal.
   err=a.issueCommand(ctx,id,snap.Version,application.CommandHandDealt,
    func(cmd application.Command)(application.CommandResult,error){
     return a.writer.MarkHandDealt(ctx,lifecycle.SystemActor,cmd)
    })
   if err!=nil{return progressed,err}
   if snap.Seats[0].AccountID=="" {return progressed,fmt.Errorf("dealt hand has no authorized seat")}
   if err:=a.tracker.TrackBidding(ctx,snap.Seats[0].AccountID,id);err!=nil{
    return progressed,fmt.Errorf("track dealt hand %s: %w",id,err)
   }
   progressed++
  }
 }
 return progressed,nil
}
