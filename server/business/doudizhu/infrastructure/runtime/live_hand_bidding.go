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

func (c *LiveHandCoordinator) Bid(
	ctx context.Context,
	handID domain.HandID,
	actor domain.AccountID,
	expectedVersion uint64,
	score bidding.Score,
) (application.LiveHandCommandResult, error) {
	game, err := c.game(ctx, handID)
	if err != nil {
		return application.LiveHandCommandResult{}, err
	}
	position, err := game.PositionForAccount(actor)
	if err != nil {
		return application.LiveHandCommandResult{}, err
	}
	payload, err := json.Marshal(livehand.BidCommand{Version: livehand.BidCommandVersion, Score: score})
	if err != nil {
		return application.LiveHandCommandResult{}, fmt.Errorf("encode Doudizhu bid: %w", err)
	}
	outcome, err := c.directory.Apply(gamecore.InstanceID(handID), gamecore.Command{
		ActorPosition:   position,
		ExpectedVersion: expectedVersion,
		Payload:         payload,
	})
	if err != nil {
		return application.LiveHandCommandResult{}, err
	}
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
