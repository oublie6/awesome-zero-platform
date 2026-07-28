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

func (testLiveRuntime) Play(context.Context, domain.HandID, domain.AccountID, uint64, []string) (LiveHandCommandResult, error) {
	return LiveHandCommandResult{}, errors.New("not used")
}

func (testLiveRuntime) Pass(context.Context, domain.HandID, domain.AccountID, uint64) (LiveHandCommandResult, error) {
	return LiveHandCommandResult{}, errors.New("not used")
}
