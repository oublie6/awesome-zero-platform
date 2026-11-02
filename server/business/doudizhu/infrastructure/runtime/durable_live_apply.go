package runtime

import (
	"context"
	"fmt"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/livehand"
	"github.com/oublie6/awesome-zero-platform/server/business/gamecore"
)

func (c *LiveHandCoordinator) DurabilityEnforced() bool { return c!=nil && c.journal!=nil }

// applyLive requires a stable command ID for every durably acknowledged live
// mutation. The durable path uses a clone, an epoch-fenced MySQL transaction,
// then publishes the committed candidate. A failed/ambiguous commit quarantines
// the directory entry until explicit takeover/replay.
func (c *LiveHandCoordinator) applyLive(
	ctx context.Context,handID domain.HandID,actor domain.AccountID,commandID string,command gamecore.Command,
)(gamecore.CommandOutcome,error) {
	if c.journal==nil {
		return c.directory.Apply(gamecore.InstanceID(handID),command)
	}
	if commandID=="" {return gamecore.CommandOutcome{},fmt.Errorf("durable live command ID required")}
	c.mu.RLock()
	epoch:=c.epochs[handID]
	c.mu.RUnlock()
	if epoch==0 {return gamecore.CommandOutcome{},fmt.Errorf("live hand has no durable owner epoch")}
	outcome,next,err:=c.directory.ApplyCommitted(gamecore.InstanceID(handID),command,func(result gamecore.CommandOutcome)error{
		return c.journal.Append(ctx,handID,epoch,commandID,actor,command,result)
	})
	if err!=nil{return outcome,err}
	if next!=nil {
		game,ok:=next.(*livehand.Game)
		if !ok {return outcome,fmt.Errorf("%w: invalid committed Doudizhu candidate",gamecore.ErrVerificationFailed)}
		c.mu.Lock()
		c.games[handID]=game
		c.mu.Unlock()
	}
	return outcome,nil
}
