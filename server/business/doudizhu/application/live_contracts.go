package application

import (
	"context"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
	"github.com/oublie6/awesome-zero-platform/server/business/gamecore"
)

type HandSetupReleaser interface {
	ReleaseHand(context.Context, domain.HandID) error
}

type BeaconVerifier interface {
	Verify(context.Context, domain.BeaconPlan, domain.BeaconValue) (domain.BeaconValue, error)
}

type LiveHandView struct {
	Version uint64
	Payload []byte
}

type LiveHandRuntime interface {
	Start(context.Context, domain.HandSnapshot) error
	RollbackStart(context.Context, domain.HandID) error
	ReleasePrepared(context.Context, domain.HandID) error
	PublicView(context.Context, domain.HandID, domain.AccountID) (LiveHandView, error)
	PrivateView(context.Context, domain.HandID, domain.AccountID) (LiveHandView, error)
	Abort(context.Context, domain.HandID, string) (gamecore.FinalRecord, error)
	RetryArchive(context.Context, domain.HandID) (gamecore.FinalRecord, error)
	Contains(domain.HandID) bool
}
