package application

import (
	"context"
	"fmt"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/bidding"
)

func (s *Service) SubmitLiveHandBid(
	ctx context.Context,
	actor domain.AccountID,
	handID domain.HandID,
	expectedLiveVersion uint64,
	score bidding.Score,
) (LiveHandCommandResult, error) {
	snapshot, err := s.authorizeHandViewer(ctx, actor, handID)
	if err != nil {
		return LiveHandCommandResult{}, err
	}
	if snapshot.Phase != domain.HandBidding {
		return LiveHandCommandResult{}, fmt.Errorf("%w: hand phase %s", domain.ErrWrongPhase, snapshot.Phase)
	}
	if expectedLiveVersion == 0 || !score.Valid() {
		return LiveHandCommandResult{}, fmt.Errorf("%w: live bid version or score", ErrInvalidCommand)
	}
	return s.liveHands.Bid(ctx, snapshot.ID, actor, expectedLiveVersion, score)
}
