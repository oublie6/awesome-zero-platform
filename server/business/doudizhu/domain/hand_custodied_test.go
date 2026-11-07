package domain

import (
 "errors"
 "testing"
 "time"
)

func TestClientCustodiedHandPartialAndCompleteRecovery(t *testing.T){
 seats:=[3]HandSeat{{Seat:1,AccountID:"alice"},{Seat:2,AccountID:"bob"},{Seat:3,AccountID:"carol"}}
 binding:=RevealKeyBinding{PublicKeySHA256:RevealPublicKeyHash{1},BoundAt:time.Now().UTC()}
 hand,events,err:=NewClientCustodiedHand("hand-custody","room-custody",seats,
  ServerCommitment{1},"reveal-test",BeaconPlan{Provider:"drand-quicknet",Round:"135000000"},binding)
 if err!=nil{t.Fatal(err)}
 if hand.Snapshot().FairnessMode!=FairnessModeClientCustodied ||
   events[0].Payload.(HandCreatedPayload).FairnessMode!=FairnessModeClientCustodied {
  t.Fatal("missing immutable mode binding")
 }
 if _,err=hand.SubmitCommit("alice",Commitment{1},hand.Snapshot().Version);!errors.Is(err,ErrWrongPhase){
  t.Fatalf("custodied hand accepted legacy commit: %v",err)
 }
 if _,err=hand.SubmitReveal("alice",ContributionDigest{1},"legacy",hand.Snapshot().Version);!errors.Is(err,ErrWrongPhase){
  t.Fatalf("custodied hand accepted legacy reveal: %v",err)
 }
 for _,item:=range []struct{actor AccountID;digest ContributionDigest;ref string}{
  {"alice",ContributionDigest{11},"record-a"},
  {"bob",ContributionDigest{12},"record-b"},
  {"carol",ContributionDigest{13},"record-c"},
 }{
  _,err=hand.SubmitCustodiedContribution(item.actor,item.digest,item.ref,hand.Snapshot().Version)
  if err!=nil{t.Fatal(err)}
  restored,err:=RestoreHand(hand.Snapshot())
  if err!=nil{t.Fatalf("partial hand recovery: %v",err)}
  hand=restored
 }
 snapshot:=hand.Snapshot()
 if snapshot.Phase!=HandWaitingPublicBeacon || snapshot.FairnessMode!=FairnessModeClientCustodied {
  t.Fatalf("incorrect finished custodied phase: %+v",snapshot)
 }
 for _,c:=range snapshot.Contributions {
  if !c.Committed || !c.Revealed || c.RecordRef=="" || c.Commitment==(Commitment{}) {
   t.Fatalf("missing bound contribution: %+v",c)
  }
 }
 if _,err=hand.SubmitCustodiedContribution("alice",ContributionDigest{11},"record-z",snapshot.Version);err==nil {
  t.Fatal("duplicate client entropy accepted")
 }
}

func TestLegacyHandKeepsSeparateCommitReveal(t *testing.T){
 seats:=[3]HandSeat{{Seat:1,AccountID:"alice"},{Seat:2,AccountID:"bob"},{Seat:3,AccountID:"carol"}}
 hand,_,err:=NewHand("hand-legacy","room-legacy",seats,ServerCommitment{1},
 "reveal-test",BeaconPlan{Provider:"drand-quicknet",Round:"135000000"},
 RevealKeyBinding{PublicKeySHA256:RevealPublicKeyHash{1},BoundAt:time.Now().UTC()})
 if err!=nil{t.Fatal(err)}
 if hand.Snapshot().FairnessMode!=""{t.Fatal("changed legacy snapshot encoding")}
 digest:=ContributionDigest{5}
 c:=ComputeClientCommitment("hand-legacy",1,digest)
 if _,err=hand.SubmitCommit("alice",c,hand.Snapshot().Version);err!=nil{t.Fatal(err)}
 if hand.Snapshot().Contributions[0].Revealed{t.Fatal("legacy commit was implicitly revealed")}
}
