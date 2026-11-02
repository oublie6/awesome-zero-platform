package lifecycle

import (
 "context"
 "fmt"
 "time"

 "github.com/oublie6/awesome-zero-platform/server/business/doudizhu/application"
 "github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
)

// Phase writes use the same durable application command ledger as room and
// fairness mutations. This interface is optional for older isolated lifecycle
// test stubs, but is implemented by the production *application.Service.
type phaseAdvancer interface {
 StartHandPlaying(context.Context,domain.AccountID,application.Command)(application.CommandResult,error)
 StartHandSettlement(context.Context,domain.AccountID,application.Command)(application.CommandResult,error)
 CompleteHand(context.Context,domain.AccountID,application.Command)(application.CommandResult,error)
}

func (s *Supervisor) advanceLiveHand(ctx context.Context,id domain.HandID,complete bool)error {
 service,ok:=s.terminal.(phaseAdvancer)
 if !ok{return nil}
 // Each accepted stage has its own durable command and monotonic system actor
 // sequence. Re-read before every step to reconcile an ACK lost after COMMIT.
 for step:=0;step<4;step++{
  snapshot,err:=s.hands.LoadHand(ctx,id)
  if err!=nil{return fmt.Errorf("read lifecycle phase: %w",err)}
  if snapshot.Phase.Terminal() {return nil}
  name:=""
  var perform func(context.Context,domain.AccountID,application.Command)(application.CommandResult,error)
  switch snapshot.Phase{
  case domain.HandBidding:
   name=application.CommandHandPlayStart
   perform=service.StartHandPlaying
  case domain.HandPlaying:
   if !complete{return nil}
   name=application.CommandHandSettlement
   perform=service.StartHandSettlement
  case domain.HandSettling:
   if !complete{return nil}
   name=application.CommandHandComplete
   perform=service.CompleteHand
  default:
   return fmt.Errorf("unexpected durable hand phase %s",snapshot.Phase)
  }
  idValue,err:=s.ids.NewID()
  if err!=nil{return fmt.Errorf("generate durable phase command: %w",err)}
  now:=s.clock.Now().UTC().Truncate(time.Millisecond)
  cmd:=application.Command{Version:application.CommandProtocolV1,Name:name,CommandID:idValue,
   AggregateType:domain.AggregateHand,AggregateID:string(id),
   ClientSeq:snapshot.Version,ExpectedVersion:snapshot.Version,
   IssuedAt:now,ExpiresAt:now.Add(s.config.CommandTTL)}
  result,err:=perform(ctx,SystemActor,cmd)
  if err!=nil {
   // A COMMIT could have succeeded while its response was lost; the caller
   // must retry via an authoritative fresh hand query, never assume rollback.
   return fmt.Errorf("durable phase %s outcome unknown: %w",name,err)
  }
  if !result.Accepted {
   return fmt.Errorf("durable phase %s rejected: %v",name,result.Failure)
  }
  if !complete {return nil}
 }
 return fmt.Errorf("durable hand phase advance exceeded bounded steps")
}
