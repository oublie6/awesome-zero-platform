package runtime

import (
	"context"
	"fmt"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/livehand"
	"github.com/oublie6/awesome-zero-platform/server/business/gamecore"
)

func (c *LiveHandCoordinator) DurabilityEnforced() bool { return c != nil && c.journal != nil }

// applyLive requires a stable command ID for every durably acknowledged live
// mutation. The durable path uses a clone, an epoch-fenced MySQL transaction,
// then publishes the committed candidate. A failed/ambiguous commit quarantines
// the directory entry until explicit takeover/replay.
func (c *LiveHandCoordinator) applyLive(
	ctx context.Context, handID domain.HandID, actor domain.AccountID, commandID string, command gamecore.Command,
) (gamecore.CommandOutcome, error) {
	if c.journal == nil {
		return c.directory.Apply(gamecore.InstanceID(handID), command)
	}
	if commandID == "" {
		return gamecore.CommandOutcome{}, fmt.Errorf("durable live command ID required")
	}
	c.mu.RLock()
	epoch := c.epochs[handID]
	c.mu.RUnlock()
	if epoch == 0 {
		return gamecore.CommandOutcome{}, fmt.Errorf("live hand has no durable owner epoch")
	}
	outcome, next, err := c.directory.ApplyCommitted(gamecore.InstanceID(handID), command, func(result gamecore.CommandOutcome) error {
		return c.journal.Append(ctx, handID, epoch, commandID, actor, command, result)
	})
	if err != nil {
		return outcome, err
	}
	if next != nil {
		game, ok := next.(*livehand.Game)
		if !ok {
			return outcome, fmt.Errorf("%w: invalid committed Doudizhu candidate", gamecore.ErrVerificationFailed)
		}
		if outcome.Terminal {
			// ApplyCommitted already archived the final record before return.
			// Dropping the active game here prevents a subsequent domain
			// CompleteHand from erroneously attempting to abort it.
			record, err := gamecore.NewFinalRecord(gamecore.InstanceID(handID), game.Descriptor(),
				gamecore.FinalStatusCompleted, outcome.Version, outcome.FinalPayload)
			if err != nil {
				return outcome, err
			}
			c.finish(handID, record)
		} else {
			c.mu.Lock()
			c.games[handID] = game
			c.mu.Unlock()
		}
	}
	c.storeReceipt(handID, actor, commandID, command, outcome)
	return outcome, nil
}
