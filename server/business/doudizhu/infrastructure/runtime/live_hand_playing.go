package runtime

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/application"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/livehand"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/playing"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/settlement"
	"github.com/oublie6/awesome-zero-platform/server/business/gamecore"
)

func (c *LiveHandCoordinator) Play(
	ctx context.Context,
	handID domain.HandID,
	actor domain.AccountID,
	expectedVersion uint64,
	cards []string,
) (application.LiveHandCommandResult, error) {
	payload, err := json.Marshal(livehand.PlayCommand{
		Version: livehand.PlayCommandVersion,
		Cards:   append([]string(nil), cards...),
	})
	if err != nil {
		return application.LiveHandCommandResult{}, fmt.Errorf("encode Doudizhu play: %w", err)
	}
	return c.applyPlayingCommand(ctx, handID, actor, expectedVersion, payload)
}

func (c *LiveHandCoordinator) Pass(
	ctx context.Context,
	handID domain.HandID,
	actor domain.AccountID,
	expectedVersion uint64,
) (application.LiveHandCommandResult, error) {
	payload, err := json.Marshal(livehand.PassCommand{Version: livehand.PassCommandVersion})
	if err != nil {
		return application.LiveHandCommandResult{}, fmt.Errorf("encode Doudizhu pass: %w", err)
	}
	return c.applyPlayingCommand(ctx, handID, actor, expectedVersion, payload)
}

func (c *LiveHandCoordinator) applyPlayingCommand(
	ctx context.Context,
	handID domain.HandID,
	actor domain.AccountID,
	expectedVersion uint64,
	payload []byte,
) (application.LiveHandCommandResult, error) {
	game, err := c.game(ctx, handID)
	if err != nil {
		return application.LiveHandCommandResult{}, err
	}
	position, err := game.PositionForAccount(actor)
	if err != nil {
		return application.LiveHandCommandResult{}, err
	}
	outcome, err := c.directory.Apply(gamecore.InstanceID(handID), gamecore.Command{
		ActorPosition:   position,
		ExpectedVersion: expectedVersion,
		Payload:         append([]byte(nil), payload...),
	})
	if err != nil {
		return application.LiveHandCommandResult{}, err
	}
	var result livehand.PlayResult
	if err := json.Unmarshal(outcome.Payload, &result); err != nil {
		return application.LiveHandCommandResult{}, fmt.Errorf("decode Doudizhu play result: %w", err)
	}
	if err := validatePlayResult(handID, outcome, result); err != nil {
		return application.LiveHandCommandResult{}, err
	}
	return application.LiveHandCommandResult{
		Version: outcome.Version,
		Payload: append([]byte(nil), outcome.Payload...),
	}, nil
}

func validatePlayResult(handID domain.HandID, outcome gamecore.CommandOutcome, result livehand.PlayResult) error {
	if result.Version != livehand.PlayResultVersion || result.HandID != string(handID) || result.StateVersion != outcome.Version || result.Playing.Version != playing.StateVersion {
		return fmt.Errorf("%w: invalid Doudizhu play result identity", gamecore.ErrVerificationFailed)
	}
	switch result.Phase {
	case livehand.PhasePlaying:
		if outcome.Terminal || len(outcome.FinalPayload) != 0 || result.Playing.Complete || result.WinnerSeat != 0 || result.Settlement != nil {
			return fmt.Errorf("%w: invalid non-terminal Doudizhu play result", gamecore.ErrVerificationFailed)
		}
	case livehand.PhaseCompleted:
		if !outcome.Terminal || len(outcome.FinalPayload) == 0 || !result.Playing.Complete || result.WinnerSeat == 0 || result.Playing.WinnerSeat != result.WinnerSeat || result.Settlement == nil || result.Settlement.Version != settlement.RulesVersion || result.Settlement.WinnerSeat != result.WinnerSeat {
			return fmt.Errorf("%w: invalid completed Doudizhu play result", gamecore.ErrVerificationFailed)
		}
	default:
		return fmt.Errorf("%w: invalid Doudizhu play phase %q", gamecore.ErrVerificationFailed, result.Phase)
	}
	return nil
}
