package application

import (
	"context"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/bidding"
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

type LiveHandCommandResult struct {
	// Historical is an internally verified old receipt. Transport must return
	// its original live result without re-running lifecycle side effects.
	Historical bool
	Version             uint64
	Payload             []byte
	RequiresTermination bool
}

type LiveHandRuntime interface {
	Start(context.Context, domain.HandSnapshot) error
	RollbackStart(context.Context, domain.HandID) error
	ReleasePrepared(context.Context, domain.HandID) error
	Bid(context.Context, domain.HandID, domain.AccountID, uint64, bidding.Score) (LiveHandCommandResult, error)
	Play(context.Context, domain.HandID, domain.AccountID, uint64, []string) (LiveHandCommandResult, error)
	Pass(context.Context, domain.HandID, domain.AccountID, uint64) (LiveHandCommandResult, error)
	PublicView(context.Context, domain.HandID, domain.AccountID) (LiveHandView, error)
	PrivateView(context.Context, domain.HandID, domain.AccountID) (LiveHandView, error)
	Abort(context.Context, domain.HandID, string) (gamecore.FinalRecord, error)
	RetryArchive(context.Context, domain.HandID) (gamecore.FinalRecord, error)
	Contains(domain.HandID) bool
}

// LiveHandReceiptReader is a read-only, actor/request-bound index of already
// committed live operations, rebuilt from the verified MySQL journal on startup.
// A missing receipt returns found=false; it MUST NOT execute a new action.
// This port is optional for legacy non-durable runtimes and avoids making the
// application depend on gamecore or storage implementation details.
type LiveHandReceiptReader interface {
	FindBidReceipt(context.Context, domain.HandID, domain.AccountID, string, uint64, bidding.Score) (LiveHandCommandResult, bool, error)
	FindPlayReceipt(context.Context, domain.HandID, domain.AccountID, string, uint64, []string) (LiveHandCommandResult, bool, error)
	FindPassReceipt(context.Context, domain.HandID, domain.AccountID, string, uint64) (LiveHandCommandResult, bool, error)
}
