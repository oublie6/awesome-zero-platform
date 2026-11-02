package runtime

import (
	"bytes"
	"context"
	"fmt"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
	"github.com/oublie6/awesome-zero-platform/server/business/gamecore"
)

// Live command responses are reconstructed and hash-checked during startup
// replay; a retry of the exact same accepted request ID returns the same
// authoritative result rather than executing an action twice.
type liveReceipt struct {
	actor   domain.AccountID
	command gamecore.Command
	outcome gamecore.CommandOutcome
}

func duplicateCommandReceipt(old, current gamecore.Command, previousActor, actor domain.AccountID) bool {
	return previousActor == actor && (current.ActorPosition == 0 || old.ActorPosition == current.ActorPosition) &&
		old.ExpectedVersion == current.ExpectedVersion && bytes.Equal(old.Payload, current.Payload)
}
func cloneLiveOutcome(v gamecore.CommandOutcome) gamecore.CommandOutcome {
	v.Payload = append([]byte(nil), v.Payload...)
	v.FinalPayload = append([]byte(nil), v.FinalPayload...)
	return v
}
func (c *LiveHandCoordinator) loadReceipt(ctx context.Context, id domain.HandID, actor domain.AccountID,
	commandID string, command gamecore.Command) (gamecore.CommandOutcome, bool, error) {
	if c == nil || commandID == "" {
		return gamecore.CommandOutcome{}, false, nil
	}
	if err := ctx.Err(); err != nil {
		return gamecore.CommandOutcome{}, false, err
	}
	c.mu.RLock()
	receipt, found := c.receipts[id][commandID]
	c.mu.RUnlock()
	if !found {
		return gamecore.CommandOutcome{}, false, nil
	}
	if !duplicateCommandReceipt(receipt.command, command, receipt.actor, actor) {
		return gamecore.CommandOutcome{}, true, fmt.Errorf("%w: command ID bound to different live request", gamecore.ErrVerificationFailed)
	}
	return cloneLiveOutcome(receipt.outcome), true, nil
}
func (c *LiveHandCoordinator) storeReceipt(id domain.HandID, actor domain.AccountID,
	commandID string, command gamecore.Command, outcome gamecore.CommandOutcome) {
	if c == nil || commandID == "" {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.receipts[id] == nil {
		c.receipts[id] = make(map[string]liveReceipt)
	}
	command.Payload = append([]byte(nil), command.Payload...)
	c.receipts[id][commandID] = liveReceipt{actor: actor, command: command, outcome: cloneLiveOutcome(outcome)}
}
