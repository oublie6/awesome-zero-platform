package runtime

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
	"github.com/oublie6/awesome-zero-platform/server/business/gamecore"
)

// Live command responses are reconstructed and hash-checked during startup
// replay; a retry of the exact same accepted request ID returns the same
// authoritative result rather than executing an action twice.
type liveReceipt struct {
	clientSeq uint64
	actor     domain.AccountID
	command   gamecore.Command
	outcome   gamecore.CommandOutcome
}

func duplicateCommandReceipt(old, current gamecore.Command, previousActor, actor domain.AccountID) bool {
	return previousActor == actor && (current.ActorPosition == 0 || old.ActorPosition == current.ActorPosition) &&
		old.ExpectedVersion == current.ExpectedVersion && sameJSONCommandPayload(old.Payload, current.Payload)
}
func cloneLiveOutcome(v gamecore.CommandOutcome) gamecore.CommandOutcome {
	v.Payload = append([]byte(nil), v.Payload...)
	v.FinalPayload = append([]byte(nil), v.FinalPayload...)
	return v
}
func (c *LiveHandCoordinator) loadReceipt(ctx context.Context, id domain.HandID, actor domain.AccountID,
	commandID string, command gamecore.Command, clientSeq ...uint64) (gamecore.CommandOutcome, bool, error) {
	if c == nil || commandID == "" {
		return gamecore.CommandOutcome{}, false, nil
	}
	if err := ctx.Err(); err != nil {
		return gamecore.CommandOutcome{}, false, err
	}
	c.mu.RLock()
	receipt, found := c.receipts[id][commandID]
	c.mu.RUnlock()
	if !found && c.journal != nil {
		// Active hands have a complete RAM index. Retired hands are bounded
		// and must use an exact, verified journal replay after eviction.
		c.mu.RLock()
		_, active := c.games[id]
		c.mu.RUnlock()
		if !active {
			var err error
			receipt, found, err = c.loadTerminalReceipt(ctx, id, commandID)
			if err != nil {
				return gamecore.CommandOutcome{}, false, err
			}
		}
	}
	if !found {
		return gamecore.CommandOutcome{}, false, nil
	}
	var seq uint64
	if len(clientSeq) > 0 {
		seq = clientSeq[0]
	}
	if receipt.clientSeq != seq || !duplicateCommandReceipt(receipt.command, command, receipt.actor, actor) {
		return gamecore.CommandOutcome{}, true, fmt.Errorf("%w: command ID bound to different live request", gamecore.ErrVerificationFailed)
	}
	return cloneLiveOutcome(receipt.outcome), true, nil
}
func (c *LiveHandCoordinator) storeReceipt(id domain.HandID, actor domain.AccountID,
	commandID string, command gamecore.Command, outcome gamecore.CommandOutcome, clientSeq ...uint64) {
	if c == nil || commandID == "" {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.receipts[id] == nil {
		c.receipts[id] = make(map[string]liveReceipt)
	}
	command.Payload = append([]byte(nil), command.Payload...)
	var seq uint64
	if len(clientSeq) > 0 {
		seq = clientSeq[0]
	}
	c.receipts[id][commandID] = liveReceipt{actor: actor, clientSeq: seq, command: command, outcome: cloneLiveOutcome(outcome)}
}

// MySQL JSON columns normalize object-key order and insignificant whitespace.
// Live request DTOs are marshalled from typed Go commands; recovery must bind
// exact semantic values rather than the database's byte-level presentation.
// UseNumber preserves the actual numeric lexical representation and avoids
// converting large JSON integers through float64 on duplicate retries.
func sameJSONCommandPayload(left, right []byte) bool {
	if bytes.Equal(left, right) {
		return true
	}
	canonical := func(raw []byte) ([]byte, bool) {
		decoder := json.NewDecoder(bytes.NewReader(raw))
		decoder.UseNumber()
		var value any
		if decoder.Decode(&value) != nil {
			return nil, false
		}
		var trailing any
		if decoder.Decode(&trailing) != io.EOF {
			return nil, false
		}
		encoded, err := json.Marshal(value)
		return encoded, err == nil
	}
	a, ok := canonical(left)
	if !ok {
		return false
	}
	b, ok := canonical(right)
	return ok && bytes.Equal(a, b)
}
