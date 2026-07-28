package application

import (
	"context"
	"fmt"
	"strings"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
)

func (s *Service) SubmitLiveHandPlay(
	ctx context.Context,
	actor domain.AccountID,
	handID domain.HandID,
	expectedLiveVersion uint64,
	cards []string,
) (LiveHandCommandResult, error) {
	snapshot, err := s.authorizeHandViewer(ctx, actor, handID)
	if err != nil {
		return LiveHandCommandResult{}, err
	}
	if snapshot.Phase != domain.HandBidding {
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
	return s.liveHands.Play(ctx, snapshot.ID, actor, expectedLiveVersion, copied)
}

func (s *Service) SubmitLiveHandPass(
	ctx context.Context,
	actor domain.AccountID,
	handID domain.HandID,
	expectedLiveVersion uint64,
) (LiveHandCommandResult, error) {
	snapshot, err := s.authorizeHandViewer(ctx, actor, handID)
	if err != nil {
		return LiveHandCommandResult{}, err
	}
	if snapshot.Phase != domain.HandBidding {
		return LiveHandCommandResult{}, fmt.Errorf("%w: hand phase %s", domain.ErrWrongPhase, snapshot.Phase)
	}
	if expectedLiveVersion == 0 {
		return LiveHandCommandResult{}, fmt.Errorf("%w: live pass version", ErrInvalidCommand)
	}
	return s.liveHands.Pass(ctx, snapshot.ID, actor, expectedLiveVersion)
}
