package application

import (
	"context"
	"fmt"
	"strings"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
)

func (s *Service) SubmitLiveHandPlayWithID(ctx context.Context, actor domain.AccountID, handID domain.HandID,
	commandID string, expectedLiveVersion uint64, cards []string,
) (LiveHandCommandResult, error) {
	return s.submitLivePlay(ctx, actor, handID, commandID, expectedLiveVersion, cards)
}
func (s *Service) SubmitLiveHandPassWithID(ctx context.Context, actor domain.AccountID, handID domain.HandID,
	commandID string, expectedLiveVersion uint64,
) (LiveHandCommandResult, error) {
	return s.submitLivePass(ctx, actor, handID, commandID, expectedLiveVersion)
}

func (s *Service) SubmitLiveHandPlay(
	ctx context.Context,
	actor domain.AccountID,
	handID domain.HandID,
	expectedLiveVersion uint64,
	cards []string,
) (LiveHandCommandResult, error) {
	return s.submitLivePlay(ctx, actor, handID, "", expectedLiveVersion, cards)
}
func (s *Service) submitLivePlay(ctx context.Context, actor domain.AccountID, handID domain.HandID,
	commandID string, expectedLiveVersion uint64, cards []string,
) (LiveHandCommandResult, error) {
	snapshot, err := s.authorizeHandViewer(ctx, actor, handID)
	if err != nil {
		return LiveHandCommandResult{}, err
	}
	if snapshot.Phase == domain.HandCompleted && commandID != "" {
		if durable, ok := s.liveHands.(interface {
			PlayCommitted(context.Context, domain.HandID, domain.AccountID, string, uint64, []string) (LiveHandCommandResult, error)
		}); ok {
			return durable.PlayCommitted(ctx, snapshot.ID, actor, commandID, expectedLiveVersion, cards)
		}
	}
	if snapshot.Phase != domain.HandBidding && snapshot.Phase != domain.HandPlaying {
		return LiveHandCommandResult{}, fmt.Errorf("%w: hand phase %s", domain.ErrWrongPhase, snapshot.Phase)
	}
	if expectedLiveVersion == 0 || len(cards) == 0 || len(cards) > 20 {
		return LiveHandCommandResult{}, fmt.Errorf("%w: live play version or card count", ErrInvalidCommand)
	}
	copied := make([]string, len(cards))
	for index, code := range cards {
		if strings.TrimSpace(code) == "" || len(code) > 3 {
			return LiveHandCommandResult{}, fmt.Errorf("%w: live play card %d", ErrInvalidCommand, index)
		}
		copied[index] = code
	}
	if durable, ok := s.liveHands.(interface {
		PlayCommitted(context.Context, domain.HandID, domain.AccountID, string, uint64, []string) (LiveHandCommandResult, error)
	}); ok {
		if commandID == "" {
			if enforce, yes := s.liveHands.(interface{ DurabilityEnforced() bool }); yes && enforce.DurabilityEnforced() {
				return LiveHandCommandResult{}, fmt.Errorf("%w: missing durable live command ID", ErrInvalidCommand)
			}
			// Legacy in-memory test/runtime adapters never claim B durability.
			return s.liveHands.Play(ctx, snapshot.ID, actor, expectedLiveVersion, copied)
		}
		return durable.PlayCommitted(ctx, snapshot.ID, actor, commandID, expectedLiveVersion, copied)
	}
	return s.liveHands.Play(ctx, snapshot.ID, actor, expectedLiveVersion, copied)
}

func (s *Service) SubmitLiveHandPass(
	ctx context.Context,
	actor domain.AccountID,
	handID domain.HandID,
	expectedLiveVersion uint64,
) (LiveHandCommandResult, error) {
	return s.submitLivePass(ctx, actor, handID, "", expectedLiveVersion)
}
func (s *Service) submitLivePass(ctx context.Context, actor domain.AccountID, handID domain.HandID,
	commandID string, expectedLiveVersion uint64,
) (LiveHandCommandResult, error) {
	snapshot, err := s.authorizeHandViewer(ctx, actor, handID)
	if err != nil {
		return LiveHandCommandResult{}, err
	}
	if snapshot.Phase == domain.HandCompleted && commandID != "" {
		if durable, ok := s.liveHands.(interface {
			PassCommitted(context.Context, domain.HandID, domain.AccountID, string, uint64) (LiveHandCommandResult, error)
		}); ok {
			return durable.PassCommitted(ctx, snapshot.ID, actor, commandID, expectedLiveVersion)
		}
	}
	if snapshot.Phase != domain.HandBidding && snapshot.Phase != domain.HandPlaying {
		return LiveHandCommandResult{}, fmt.Errorf("%w: hand phase %s", domain.ErrWrongPhase, snapshot.Phase)
	}
	if expectedLiveVersion == 0 {
		return LiveHandCommandResult{}, fmt.Errorf("%w: live pass version", ErrInvalidCommand)
	}
	if durable, ok := s.liveHands.(interface {
		PassCommitted(context.Context, domain.HandID, domain.AccountID, string, uint64) (LiveHandCommandResult, error)
	}); ok {
		if commandID == "" {
			if enforce, yes := s.liveHands.(interface{ DurabilityEnforced() bool }); yes && enforce.DurabilityEnforced() {
				return LiveHandCommandResult{}, fmt.Errorf("%w: missing durable live command ID", ErrInvalidCommand)
			}
			// Legacy in-memory test/runtime adapters never claim B durability.
			return s.liveHands.Pass(ctx, snapshot.ID, actor, expectedLiveVersion)
		}
		return durable.PassCommitted(ctx, snapshot.ID, actor, commandID, expectedLiveVersion)
	}
	return s.liveHands.Pass(ctx, snapshot.ID, actor, expectedLiveVersion)
}
