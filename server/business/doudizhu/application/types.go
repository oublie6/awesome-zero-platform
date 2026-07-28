package application

import (
	"time"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
)

const (
	CommandProtocolV1 = "fair-doudizhu-command-v1"
	CommandResultV1   = "fair-doudizhu-command-result-v1"
	EventProtocolV1   = "fair-doudizhu-event-v1"
	RevealPlaintextV1 = "fair-doudizhu-reveal-v1"
	RevealAADV1       = "fair-doudizhu/reveal-aad/v1"
	ContributionV1    = "fair-doudizhu/contribution/v1"
	RecordAADV1       = "fair-doudizhu/contribution-record-aad/v1"
)

const (
	CommandRoomCreate       = "doudizhu.room.create.v1"
	CommandRoomJoin         = "doudizhu.room.join.v1"
	CommandRoomLeave        = "doudizhu.room.leave.v1"
	CommandRoomReadySet     = "doudizhu.room.ready.set.v1"
	CommandRoomHandStart    = "doudizhu.room.hand.start.v1"
	CommandHandCommitSubmit = "doudizhu.hand.fairness.commit.submit.v1"
	CommandHandRevealSubmit = "doudizhu.hand.fairness.reveal.submit.v1"
	CommandHandBeaconLock   = "doudizhu.hand.beacon.lock.v1"
	CommandHandDealt        = "doudizhu.hand.dealt.v1"
	CommandHandPlayStart    = "doudizhu.hand.play.start.v1"
	CommandHandSettlement   = "doudizhu.hand.settlement.start.v1"
	CommandHandComplete     = "doudizhu.hand.complete.v1"
	CommandHandCancel       = "doudizhu.hand.cancel.v1"
	CommandHandAbort        = "doudizhu.hand.abort.v1"
	CommandHandExpire       = "doudizhu.hand.expire.v1"
)

type Command struct {
	Version         string
	Name            string
	CommandID       string
	AggregateType   domain.AggregateType
	AggregateID     string
	ClientSeq       uint64
	ExpectedVersion uint64
	IssuedAt        time.Time
	ExpiresAt       time.Time
	PayloadDigest   [32]byte
}

type StoredCommandResult struct {
	Command Command
	Result  CommandResult
}

type EventRef struct {
	AggregateType domain.AggregateType `json:"aggregateType"`
	AggregateID   string               `json:"aggregateId"`
	Name          string               `json:"name"`
	Version       uint64               `json:"version"`
}

type CommandFailure struct {
	Code           string  `json:"code"`
	Message        string  `json:"message"`
	CurrentVersion *uint64 `json:"currentVersion,omitempty"`
}

type CommandResult struct {
	Version          string               `json:"v"`
	CommandID        string               `json:"commandId"`
	Accepted         bool                 `json:"accepted"`
	Duplicate        bool                 `json:"duplicate"`
	AggregateType    domain.AggregateType `json:"aggregateType"`
	AggregateID      string               `json:"aggregateId"`
	AggregateVersion uint64               `json:"aggregateVersion"`
	Events           []EventRef           `json:"events"`
	Failure          *CommandFailure      `json:"failure,omitempty"`
}

type OutboxEvent struct {
	EventID            string
	Protocol           string
	Name               string
	AggregateType      domain.AggregateType
	AggregateID        string
	AggregateVersion   uint64
	OccurredAt         time.Time
	CausationCommandID string
	ActorAccountID     domain.AccountID
	PayloadJSON        []byte
}

type ProtectedContributionRecord struct {
	RecordID           string
	HandID             domain.HandID
	Seat               domain.Seat
	ActorAccountID     domain.AccountID
	CommandID          string
	ContributionDigest domain.ContributionDigest
	ProtectionKeyID    string
	Nonce              []byte
	Ciphertext         []byte
	AADDigest          [32]byte
	CreatedAt          time.Time
}

type SecureEnvelope struct {
	Version         string
	KeyID           string
	Suite           string
	EncapsulatedKey string
	Ciphertext      string
}

type ProtectedPayload struct {
	KeyID      string
	Nonce      []byte
	Ciphertext []byte
	AADDigest  [32]byte
}

type HandSetup struct {
	HandID                domain.HandID
	ServerCommitment      domain.ServerCommitment
	RevealKeyID           string
	RevealPublicKeySHA256 domain.RevealPublicKeyHash
	RevealKeyBoundAt      time.Time
	BeaconPlan            domain.BeaconPlan
}

type RevealKeyContext struct {
	KeyID           string
	PublicKeySHA256 [32]byte
	BoundAt         time.Time
	UseAt           time.Time
}
