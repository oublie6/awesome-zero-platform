package application

import (
	"context"
	"time"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
)

type Store interface {
	WithinCommand(context.Context, domain.AccountID, string, func(context.Context, Transaction) error) error
	LoadHand(context.Context, domain.HandID) (domain.HandSnapshot, error)
}

type Transaction interface {
	ClaimCommand(context.Context, domain.AccountID, Command, time.Time) (StoredCommandResult, bool, error)
	CompleteCommand(context.Context, domain.AccountID, string, CommandResult, time.Time) error
	LockClientSequence(context.Context, domain.AggregateType, string, domain.AccountID) (uint64, error)
	SaveClientSequence(context.Context, domain.AggregateType, string, domain.AccountID, uint64, time.Time) error

	InsertRoom(context.Context, domain.RoomSnapshot, time.Time) error
	LoadRoomForUpdate(context.Context, domain.RoomID) (domain.RoomSnapshot, error)
	UpdateRoom(context.Context, domain.RoomSnapshot, uint64, time.Time) error

	InsertHand(context.Context, domain.HandSnapshot, time.Time) error
	LoadHandForUpdate(context.Context, domain.HandID) (domain.HandSnapshot, error)
	UpdateHand(context.Context, domain.HandSnapshot, uint64, time.Time) error

	InsertContributionRecord(context.Context, ProtectedContributionRecord) error
	AppendOutbox(context.Context, []OutboxEvent) error
}

type Clock interface{ Now() time.Time }
type IDGenerator interface{ NewID() (string, error) }
type HandSetupProvider interface {
	PrepareHand(context.Context, domain.RoomSnapshot, domain.HandID) (HandSetup, error)
	ReleaseHand(context.Context, domain.HandID) error
}
type EnvelopeOpener interface {
	Open(context.Context, SecureEnvelope, []byte, RevealKeyContext) ([]byte, error)
}
type ContributionProtector interface {
	Seal(context.Context, []byte, []byte) (ProtectedPayload, error)
}
type PhraseNormalizer interface{ Normalize(string) (string, error) }
