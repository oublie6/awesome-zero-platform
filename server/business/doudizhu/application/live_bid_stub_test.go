package application

import (
	"context"
	"errors"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/bidding"
)

func (testLiveRuntime) Bid(context.Context, domain.HandID, domain.AccountID, uint64, bidding.Score) (LiveHandCommandResult, error) {
	return LiveHandCommandResult{}, errors.New("not used")
}
