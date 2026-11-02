package runtime

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/application"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/bidding"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/livehand"
	"github.com/oublie6/awesome-zero-platform/server/business/gamecore"
)

func (c *LiveHandCoordinator) BidCommitted(ctx context.Context, handID domain.HandID,
	actor domain.AccountID, commandID string, expectedVersion uint64, score bidding.Score,
) (application.LiveHandCommandResult, error) {
	return c.bidInternal(ctx, handID, actor, commandID, expectedVersion, score)
}

func (c *LiveHandCoordinator) Bid(
	ctx context.Context,
	handID domain.HandID,
	actor domain.AccountID,
	expectedVersion uint64,
	score bidding.Score,
) (application.LiveHandCommandResult, error) {
	return c.bidInternal(ctx, handID, actor, "", expectedVersion, score)
}
func (c *LiveHandCoordinator) bidInternal(ctx context.Context, handID domain.HandID, actor domain.AccountID,
	commandID string, expectedVersion uint64, score bidding.Score,
) (application.LiveHandCommandResult, error) {
	payload, err := json.Marshal(livehand.BidCommand{Version: livehand.BidCommandVersion, Score: score})
	if err != nil {
		return application.LiveHandCommandResult{}, fmt.Errorf("encode Doudizhu bid: %w", err)
	}
	lock := c.commandStripe(handID)
	lock.Lock()
	defer lock.Unlock()
	known := gamecore.Command{ExpectedVersion: expectedVersion, Payload: payload}
	if outcome, found, err := c.loadReceipt(ctx, handID, actor, commandID, known); err != nil {
		return application.LiveHandCommandResult{}, err
	} else if found {
		return checkedBidResult(handID, outcome)
	}
	game, err := c.game(ctx, handID)
	if err != nil {
		return application.LiveHandCommandResult{}, err
	}
	position, err := game.PositionForAccount(actor)
	if err != nil {
		return application.LiveHandCommandResult{}, err
	}
	known.ActorPosition = position
	outcome, err := c.applyLive(ctx, handID, actor, commandID, known)
	if err != nil {
		return application.LiveHandCommandResult{}, err
	}
	return checkedBidResult(handID, outcome)
}
func checkedBidResult(handID domain.HandID, outcome gamecore.CommandOutcome) (application.LiveHandCommandResult, error) {
	var result livehand.BidResult
	if err := json.Unmarshal(outcome.Payload, &result); err != nil {
		return application.LiveHandCommandResult{}, fmt.Errorf("decode Doudizhu bid result: %w", err)
	}
	if result.Version != livehand.BidResultVersion || result.HandID != string(handID) || result.StateVersion != outcome.Version || result.Phase == "" {
		return application.LiveHandCommandResult{}, fmt.Errorf("%w: invalid Doudizhu bid result", gamecore.ErrVerificationFailed)
	}
	return application.LiveHandCommandResult{
		Version:             outcome.Version,
		Payload:             append([]byte(nil), outcome.Payload...),
		RequiresTermination: result.RequiresTermination,
	}, nil
}
