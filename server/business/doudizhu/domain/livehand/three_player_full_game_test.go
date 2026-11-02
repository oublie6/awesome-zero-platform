package livehand

import (
 "encoding/json"
 "testing"

 "github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/bidding"
 "github.com/oublie6/awesome-zero-platform/server/business/gamecore"
)

// A deterministic three-seat rules simulation spanning bidding, 20 legal
// landlord plays, 38 opponent passes, and verified terminal settlement.
// This is a domain-level model test, not a substitute for three independent
// authenticated devices or MySQL fault-injection.
func TestThreeSeatGameFromBiddingThroughFinalSettlement(t *testing.T){
 game:=newDirectBiddingGame(t,1)
 game.transcript=validTranscriptForLiveTest(t,string(game.id))
 first,err:=game.Apply(bidCommand(t,1,game.version,bidding.ScoreThree))
 if err!=nil || first.Terminal || game.Phase()!=PhasePlaying {
  t.Fatalf("landlord auction failed: first=%+v err=%v",first,err)
 }
 if len(game.current[0])!=20 {t.Fatalf("landlord did not receive 3 bottom cards")}
 for round:=0;round<20;round++{
  card:=game.current[0][0]
  result,err:=game.Apply(livePlayCommand(t,1,game.version,card))
  if err!=nil{t.Fatalf("round %d landlord play: %v",round,err)}
  if round==19{
   if !result.Terminal || len(result.FinalPayload)==0 {
    t.Fatal("last card was not terminal with auditable final evidence")
   }
   var completed CompletedPayload
   if err:=json.Unmarshal(result.FinalPayload,&completed);err!=nil{t.Fatal(err)}
   if completed.Status!=string(gamecore.FinalStatusCompleted) || completed.WinnerSeat!=1 ||
     !completed.Settlement.LandlordWon || completed.StateVersion!=result.Version {
    t.Fatalf("invalid authoritative final result: %+v",completed)
   }
   break
  }
  if result.Terminal{t.Fatalf("game prematurely ended at round %d",round)}
  for _,seat:=range []uint8{2,3}{
   pass,err:=game.Apply(livePassCommand(t,seat,game.version))
   if err!=nil || pass.Terminal{
    t.Fatalf("round %d seat %d cannot pass: %+v %v",round,seat,pass,err)
   }
  }
  if game.Phase()!=PhasePlaying {
   t.Fatalf("round %d lost playing phase",round)
  }
 }
 if game.Phase()!=PhaseCompleted || game.version!=61{
  t.Fatalf("expected 1 bid + 20 plays + 38 passes; phase=%s version=%d",game.Phase(),game.version)
 }
 if len(game.current[0])!=0 || len(game.current[1])!=17 || len(game.current[2])!=17{
  t.Fatalf("incorrect final hand sizes: %d/%d/%d",len(game.current[0]),len(game.current[1]),len(game.current[2]))
 }
 if _,err:=game.Apply(livePassCommand(t,1,game.version));err==nil{
  t.Fatal("accepted a post-terminal extra command")
 }
}
