package application

import (
	"context"
	"fmt"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/bidding"
)

func (s *Service) SubmitLiveHandBidWithID(ctx context.Context, actor domain.AccountID,
	handID domain.HandID, commandID string, expectedLiveVersion uint64,score bidding.Score,
)(LiveHandCommandResult,error){
	return s.submitLiveBid(ctx,actor,handID,commandID,expectedLiveVersion,score)
}

func (s *Service) SubmitLiveHandBid(
	ctx context.Context,
	actor domain.AccountID,
	handID domain.HandID,
	expectedLiveVersion uint64,
	score bidding.Score,
) (LiveHandCommandResult, error) {
	return s.submitLiveBid(ctx,actor,handID,"",expectedLiveVersion,score)
}
func (s *Service) submitLiveBid(ctx context.Context,actor domain.AccountID,handID domain.HandID,
	commandID string,expectedLiveVersion uint64,score bidding.Score,
)(LiveHandCommandResult,error){
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
	if durable,ok:=s.liveHands.(interface{BidCommitted(context.Context,domain.HandID,domain.AccountID,string,uint64,bidding.Score)(LiveHandCommandResult,error)});ok {
		if commandID=="" {return LiveHandCommandResult{},fmt.Errorf("%w: missing durable live command ID",ErrInvalidCommand)}
		return durable.BidCommitted(ctx,snapshot.ID,actor,commandID,expectedLiveVersion,score)
	}
	return s.liveHands.Bid(ctx, snapshot.ID, actor, expectedLiveVersion, score)
}
