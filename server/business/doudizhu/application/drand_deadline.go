package application

import (
 "fmt"
 "strconv"
 "time"

 "github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
)
const quicknetGenesisTime = int64(1692803367)
const quicknetRoundSeconds = int64(3)

// checkContributionDeadline ensures a player cannot choose or reveal their
// commitment after its precommitted public beacon becomes predictable.
// The auto dealer must abort expired unfinished fairness rounds, never use
// an already-released round as if it were future entropy.
func checkContributionDeadline(plan domain.BeaconPlan, now time.Time)error{
 if plan.Provider!="drand-quicknet"{return nil}
 round,err:=strconv.ParseUint(plan.Round,10,64)
 if err!=nil || round==0 || round>1<<52 || strconv.FormatUint(round,10)!=plan.Round {
  return fmt.Errorf("%w: invalid committed public beacon round",domain.ErrBeaconMismatch)
 }
 publishedAt:=time.Unix(quicknetGenesisTime+int64(round-1)*quicknetRoundSeconds,0)
 if !now.UTC().Before(publishedAt){
  return fmt.Errorf("%w: public beacon already known; restart fairness hand",domain.ErrBeaconMismatch)
 }
 return nil
}
