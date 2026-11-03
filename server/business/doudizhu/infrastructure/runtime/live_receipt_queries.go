package runtime

import (
	"context"
	"encoding/json"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/application"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/bidding"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/livehand"
	"github.com/oublie6/awesome-zero-platform/server/business/gamecore"
)

var _ application.LiveHandReceiptReader = (*LiveHandCoordinator)(nil)

// These are read-only, bounded in-memory queries over the VERIFIED journal
// replay index. Never acquire a new owner epoch or execute a gameplay command
// from this path. Application authorizes the seated actor before calling it;
// the receipt additionally binds actor, exact command type/payload and version.
func (c *LiveHandCoordinator) FindBidReceipt(ctx context.Context, hand domain.HandID, actor domain.AccountID,
	commandID string, expectedVersion uint64, score bidding.Score, clientSeq ...uint64,
) (application.LiveHandCommandResult, bool, error) {
	payload, err := json.Marshal(livehand.BidCommand{Version: livehand.BidCommandVersion, Score: score})
	if err != nil {
		return application.LiveHandCommandResult{}, false, err
	}
	return c.findVerifiedReceipt(ctx, hand, actor, commandID, expectedVersion, payload, checkedBidResult, clientSeq...)
}

func (c *LiveHandCoordinator) FindPlayReceipt(ctx context.Context, hand domain.HandID, actor domain.AccountID,
	commandID string, expectedVersion uint64, cards []string, clientSeq ...uint64,
) (application.LiveHandCommandResult, bool, error) {
	payload, err := json.Marshal(livehand.PlayCommand{
		Version: livehand.PlayCommandVersion, Cards: append([]string(nil), cards...),
	})
	if err != nil {
		return application.LiveHandCommandResult{}, false, err
	}
	return c.findVerifiedReceipt(ctx, hand, actor, commandID, expectedVersion, payload, checkedPlayResult, clientSeq...)
}

func (c *LiveHandCoordinator) FindPassReceipt(ctx context.Context, hand domain.HandID, actor domain.AccountID,
	commandID string, expectedVersion uint64, clientSeq ...uint64,
) (application.LiveHandCommandResult, bool, error) {
	payload, err := json.Marshal(livehand.PassCommand{Version: livehand.PassCommandVersion})
	if err != nil {
		return application.LiveHandCommandResult{}, false, err
	}
	return c.findVerifiedReceipt(ctx, hand, actor, commandID, expectedVersion, payload, checkedPlayResult, clientSeq...)
}

func (c *LiveHandCoordinator) findVerifiedReceipt(ctx context.Context, hand domain.HandID, actor domain.AccountID,
	commandID string, expectedVersion uint64, payload []byte,
	check func(domain.HandID, gamecore.CommandOutcome) (application.LiveHandCommandResult, error), clientSeq ...uint64,
) (application.LiveHandCommandResult, bool, error) {
	if c == nil || commandID == "" {
		return application.LiveHandCommandResult{}, false, nil
	}
	// Synchronize with the transition+receipt-publication critical section.
	// A simultaneous successful COMMIT cannot be missed between verification
	// and a subsequent application-level phase check.
	stripe := c.commandStripe(hand)
	stripe.Lock()
	defer stripe.Unlock()
	outcome, found, err := c.loadReceipt(ctx, hand, actor, commandID,
		gamecore.Command{ExpectedVersion: expectedVersion, Payload: payload}, clientSeq...)
	if err != nil || !found {
		return application.LiveHandCommandResult{}, found, err
	}
	result, err := check(hand, outcome)
	if err != nil {
		return application.LiveHandCommandResult{}, true, err
	}
	result.Historical = true // Verified receipt provenance, independent of current phase.
	return result, true, nil
}
