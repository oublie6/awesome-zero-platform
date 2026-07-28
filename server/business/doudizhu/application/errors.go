package application

import (
	"errors"
	"fmt"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
)

var (
	ErrNotFound             = errors.New("doudizhu aggregate not found")
	ErrAlreadyExists        = errors.New("doudizhu record already exists")
	ErrOptimisticConflict   = errors.New("doudizhu optimistic persistence conflict")
	ErrSequenceConflict     = errors.New("doudizhu client sequence conflict")
	ErrInvalidCommand       = errors.New("invalid doudizhu command")
	ErrRevealInvalid        = errors.New("invalid fairness reveal")
	ErrProtectionFailed     = errors.New("contribution protection failed")
	ErrRetryableTransaction = errors.New("doudizhu transaction should be retried")
)

const (
	CodeInvalidCommand        = "DDZ_INVALID_COMMAND"
	CodeForbidden             = "DDZ_FORBIDDEN"
	CodeNotFound              = "DDZ_NOT_FOUND"
	CodeNotSeated             = "DDZ_NOT_SEATED"
	CodeRoomFull              = "DDZ_ROOM_FULL"
	CodeRoomNotReady          = "DDZ_ROOM_NOT_READY"
	CodeHandActive            = "DDZ_HAND_ACTIVE"
	CodeWrongPhase            = "DDZ_WRONG_PHASE"
	CodeDuplicateContribution = "DDZ_DUPLICATE_CONTRIBUTION"
	CodeCommitmentMismatch    = "DDZ_COMMITMENT_MISMATCH"
	CodeBeaconMismatch        = "DDZ_BEACON_MISMATCH"
	CodeVersionConflict       = "DDZ_VERSION_CONFLICT"
	CodeSequenceConflict      = "DDZ_SEQUENCE_CONFLICT"
	CodeHandTerminal          = "DDZ_HAND_TERMINAL"
	CodeRevealInvalid         = "DDZ_REVEAL_INVALID"
	CodeConflict              = "DDZ_CONFLICT"
)

type businessRejection struct {
	failure CommandFailure
}

func (e *businessRejection) Error() string { return e.failure.Code + ": " + e.failure.Message }

func reject(code, message string) *businessRejection {
	return &businessRejection{failure: CommandFailure{Code: code, Message: message}}
}

func mapBusinessError(err error) *businessRejection {
	if err == nil {
		return nil
	}
	var versionErr *domain.VersionConflictError
	if errors.As(err, &versionErr) {
		actual := versionErr.Actual
		result := reject(CodeVersionConflict, "aggregate version conflict")
		result.failure.CurrentVersion = &actual
		return result
	}
	switch {
	case errors.Is(err, ErrNotFound):
		return reject(CodeNotFound, "aggregate not found")
	case errors.Is(err, ErrAlreadyExists):
		return reject(CodeConflict, "state conflict")
	case errors.Is(err, ErrSequenceConflict):
		return reject(CodeSequenceConflict, "client sequence conflict")
	case errors.Is(err, ErrInvalidCommand), errors.Is(err, domain.ErrInvalidArgument), errors.Is(err, domain.ErrNoStateChange):
		return reject(CodeInvalidCommand, "invalid command")
	case errors.Is(err, ErrRevealInvalid):
		return reject(CodeRevealInvalid, "invalid fairness reveal")
	case errors.Is(err, domain.ErrForbidden), errors.Is(err, domain.ErrOwnerMustRemain):
		return reject(CodeForbidden, "operation forbidden")
	case errors.Is(err, domain.ErrNotSeated):
		return reject(CodeNotSeated, "account is not seated")
	case errors.Is(err, domain.ErrRoomFull):
		return reject(CodeRoomFull, "room is full")
	case errors.Is(err, domain.ErrRoomNotReady):
		return reject(CodeRoomNotReady, "room is not ready")
	case errors.Is(err, domain.ErrHandActive):
		return reject(CodeHandActive, "hand is active")
	case errors.Is(err, domain.ErrWrongPhase), errors.Is(err, domain.ErrHandNotActive), errors.Is(err, domain.ErrRoomClosed), errors.Is(err, domain.ErrAlreadySeated):
		return reject(CodeWrongPhase, "command is not allowed in the current state")
	case errors.Is(err, domain.ErrDuplicateContribution):
		return reject(CodeDuplicateContribution, "contribution already submitted")
	case errors.Is(err, domain.ErrCommitmentMismatch):
		return reject(CodeCommitmentMismatch, "reveal does not match commitment")
	case errors.Is(err, domain.ErrBeaconMismatch):
		return reject(CodeBeaconMismatch, "beacon does not match locked plan")
	case errors.Is(err, domain.ErrHandTerminal):
		return reject(CodeHandTerminal, "hand is terminal")
	default:
		return nil
	}
}

func wrapInfrastructure(operation string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", operation, err)
}
