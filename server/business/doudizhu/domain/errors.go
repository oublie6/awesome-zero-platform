package domain

import (
	"errors"
	"fmt"
)

var (
	ErrInvalidArgument       = errors.New("invalid argument")
	ErrVersionConflict       = errors.New("aggregate version conflict")
	ErrNoStateChange         = errors.New("command would not change state")
	ErrForbidden             = errors.New("operation forbidden")
	ErrRoomClosed            = errors.New("room is closed")
	ErrRoomFull              = errors.New("room is full")
	ErrAlreadySeated         = errors.New("account is already seated")
	ErrNotSeated             = errors.New("account is not seated")
	ErrOwnerMustRemain       = errors.New("room owner must remain while guests are seated")
	ErrHandActive            = errors.New("room already has an active hand")
	ErrHandNotActive         = errors.New("room does not have the requested active hand")
	ErrRoomNotReady          = errors.New("room is not ready to start")
	ErrWrongPhase            = errors.New("command is not allowed in the current hand phase")
	ErrDuplicateContribution = errors.New("seat has already contributed in this phase")
	ErrCommitmentMismatch    = errors.New("reveal does not match the committed contribution")
	ErrBeaconMismatch        = errors.New("beacon value does not match the locked plan")
	ErrHandTerminal          = errors.New("hand is already terminal")
)

type VersionConflictError struct {
	Expected uint64
	Actual   uint64
}

func (e *VersionConflictError) Error() string {
	return fmt.Sprintf("%v: expected %d, actual %d", ErrVersionConflict, e.Expected, e.Actual)
}

func (e *VersionConflictError) Unwrap() error {
	return ErrVersionConflict
}

func checkExpectedVersion(actual, expected uint64) error {
	if actual != expected {
		return &VersionConflictError{Expected: expected, Actual: actual}
	}
	return nil
}
