package application

import (
 "errors"
 "testing"
 "time"

 "github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
)
func TestPublicBeaconContributionsMustPrecedePublicRound(t *testing.T){
 plan:=domain.BeaconPlan{Provider:"drand-quicknet",Round:"1000"}
 due,err:=QuicknetRoundPublication(plan)
 if err!=nil{t.Fatal(err)}
 expected:=time.Unix(1692803367+999*3,0)
 if !due.Equal(expected){t.Fatalf("wrong drand genesis round emission: %s",due)}
 if err:=checkContributionDeadline(plan,due.Add(-time.Nanosecond));err!=nil{
  t.Fatalf("rejected still-unpredictable contribution: %v",err)
 }
 for _,now:=range []time.Time{due,due.Add(time.Minute)}{
  if err:=checkContributionDeadline(plan,now);!errors.Is(err,domain.ErrBeaconMismatch){
   t.Fatalf("accepted contribution after public release at %s: %v",now,err)
  }
 }
 for _,round:=range []string{"0","01","auto","-1","18446744073709551615"}{
  if err:=checkContributionDeadline(domain.BeaconPlan{Provider:"drand-quicknet",Round:round},
   due.Add(-time.Hour));!errors.Is(err,domain.ErrBeaconMismatch){
   t.Fatalf("accepted malformed pinned round %s: %v",round,err)
  }
 }
 if err:=checkContributionDeadline(domain.BeaconPlan{Provider:"test-only",Round:"whatever"},due.Add(time.Hour));err!=nil{
  t.Fatalf("legacy isolated test fixture rejected: %v",err)
 }
}
