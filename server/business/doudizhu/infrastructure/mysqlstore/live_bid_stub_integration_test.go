//go:build integration

package mysqlstore_test

import (
	"context"
	"errors"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/application"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/bidding"
)

func (integrationLiveRuntime) Bid(context.Context, domain.HandID, domain.AccountID, uint64, bidding.Score) (application.LiveHandCommandResult, error) {
	return application.LiveHandCommandResult{}, errors.New("not used")
}

func (integrationLiveRuntime) Play(context.Context, domain.HandID, domain.AccountID, uint64, []string) (application.LiveHandCommandResult, error) {
	return application.LiveHandCommandResult{}, errors.New("not used")
}

func (integrationLiveRuntime) Pass(context.Context, domain.HandID, domain.AccountID, uint64) (application.LiveHandCommandResult, error) {
	return application.LiveHandCommandResult{}, errors.New("not used")
}
