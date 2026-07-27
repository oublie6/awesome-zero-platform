package domain

import (
	"crypto/subtle"
	"fmt"
	"strings"
)

func (h *Hand) SubmitCommit(actor AccountID, commitment Commitment, expectedVersion uint64) ([]Event, error) {
	if err := h.prepareMutation(expectedVersion, HandFairnessCommitting); err != nil {
		return nil, err
	}
	seat, err := h.resolveSeat(actor)
	if err != nil {
		return nil, err
	}
	if allZero32([32]byte(commitment)) {
		return nil, fmt.Errorf("%w: commitment is zero", ErrInvalidArgument)
	}
	current := &h.contributions[seat-1]
	if current.Commitment != (Commitment{}) {
		return nil, ErrDuplicateContribution
	}
	current.Commitment = commitment
	events := []Event{h.record(EventFairnessCommitAccepted, FairnessSeatPayload{Seat: seat})}
	if h.commitCount() == 3 {
		events = append(events, h.changePhase(HandFairnessRevealing))
	}
	return events, nil
}

func (h *Hand) SubmitReveal(actor AccountID, digest ContributionDigest, recordRef string, expectedVersion uint64) ([]Event, error) {
	if err := h.prepareMutation(expectedVersion, HandFairnessRevealing); err != nil {
		return nil, err
	}
	seat, err := h.resolveSeat(actor)
	if err != nil {
		return nil, err
	}
	if allZero32([32]byte(digest)) {
		return nil, fmt.Errorf("%w: contribution digest is zero", ErrInvalidArgument)
	}
	recordRef = strings.TrimSpace(recordRef)
	if recordRef == "" || len(recordRef) > 256 {
		return nil, fmt.Errorf("%w: invalid contribution record reference", ErrInvalidArgument)
	}
	current := &h.contributions[seat-1]
	if current.Commitment == (Commitment{}) {
		return nil, ErrWrongPhase
	}
	if current.Revealed {
		return nil, ErrDuplicateContribution
	}
	expectedCommitment := ComputeClientCommitment(h.id, seat, digest)
	if subtle.ConstantTimeCompare(expectedCommitment[:], current.Commitment[:]) != 1 {
		return nil, ErrCommitmentMismatch
	}
	current.Revealed = true
	current.Digest = digest
	current.RecordRef = recordRef
	events := []Event{h.record(EventFairnessRevealAccepted, FairnessSeatPayload{Seat: seat})}
	if h.revealCount() == 3 {
		events = append(events, h.changePhase(HandWaitingPublicBeacon))
	}
	return events, nil
}

func (h *Hand) LockPublicBeacon(value BeaconValue, expectedVersion uint64) ([]Event, error) {
	if err := h.prepareMutation(expectedVersion, HandWaitingPublicBeacon); err != nil {
		return nil, err
	}
	if value.Provider != h.beaconPlan.Provider || value.Round != h.beaconPlan.Round {
		return nil, ErrBeaconMismatch
	}
	if allZero32([32]byte(value.Digest)) {
		return nil, fmt.Errorf("%w: beacon digest is zero", ErrInvalidArgument)
	}
	value.ProofRef = strings.TrimSpace(value.ProofRef)
	if value.ProofRef == "" || len(value.ProofRef) > 512 {
		return nil, fmt.Errorf("%w: invalid beacon proof reference", ErrInvalidArgument)
	}
	copyValue := value
	h.beacon = &copyValue
	events := []Event{h.record(EventPublicBeaconLocked, PublicBeaconPayload{
		Provider: value.Provider,
		Round:    value.Round,
		ProofRef: value.ProofRef,
	})}
	events = append(events, h.changePhase(HandDealing))
	return events, nil
}

func (h *Hand) commitCount() int {
	count := 0
	for _, current := range h.contributions {
		if current.Commitment != (Commitment{}) {
			count++
		}
	}
	return count
}

func (h *Hand) revealCount() int {
	count := 0
	for _, current := range h.contributions {
		if current.Revealed {
			count++
		}
	}
	return count
}
