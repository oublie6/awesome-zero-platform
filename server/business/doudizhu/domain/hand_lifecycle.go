package domain

import (
	"fmt"
	"strings"
)

func (h *Hand) MarkDealt(expectedVersion uint64) ([]Event, error) {
	return h.transition(expectedVersion, HandDealing, HandBidding)
}

func (h *Hand) StartPlaying(expectedVersion uint64) ([]Event, error) {
	return h.transition(expectedVersion, HandBidding, HandPlaying)
}

func (h *Hand) StartSettling(expectedVersion uint64) ([]Event, error) {
	return h.transition(expectedVersion, HandPlaying, HandSettling)
}

func (h *Hand) Complete(expectedVersion uint64) ([]Event, error) {
	return h.transition(expectedVersion, HandSettling, HandCompleted)
}

func (h *Hand) Cancel(reason string, expectedVersion uint64) ([]Event, error) {
	if h.phase == HandDealing || h.phase == HandBidding || h.phase == HandPlaying || h.phase == HandSettling {
		return nil, ErrWrongPhase
	}
	return h.terminate(expectedVersion, HandCancelled, reason)
}

func (h *Hand) Abort(reason string, expectedVersion uint64) ([]Event, error) {
	return h.terminate(expectedVersion, HandAborted, reason)
}

func (h *Hand) Expire(reason string, expectedVersion uint64) ([]Event, error) {
	return h.terminate(expectedVersion, HandExpired, reason)
}

func (h *Hand) transition(expectedVersion uint64, from, to HandPhase) ([]Event, error) {
	if err := h.prepareMutation(expectedVersion, from); err != nil {
		return nil, err
	}
	return []Event{h.changePhase(to)}, nil
}

func (h *Hand) terminate(expectedVersion uint64, terminal HandPhase, reason string) ([]Event, error) {
	if err := checkExpectedVersion(h.version, expectedVersion); err != nil {
		return nil, err
	}
	if h.phase.Terminal() {
		return nil, ErrHandTerminal
	}
	reason = strings.TrimSpace(reason)
	if reason == "" || len(reason) > 256 {
		return nil, fmt.Errorf("%w: invalid termination reason", ErrInvalidArgument)
	}
	h.terminationReason = reason
	from := h.phase
	h.phase = terminal
	return []Event{
		h.record(EventHandTerminated, HandTerminationPayload{Phase: terminal, Reason: reason}),
		h.record(EventHandPhaseChanged, HandPhasePayload{From: from, To: terminal}),
	}, nil
}

func (h *Hand) prepareMutation(expectedVersion uint64, required HandPhase) error {
	if err := checkExpectedVersion(h.version, expectedVersion); err != nil {
		return err
	}
	if h.phase.Terminal() {
		return ErrHandTerminal
	}
	if h.phase != required {
		return ErrWrongPhase
	}
	return nil
}

func (h *Hand) resolveSeat(actor AccountID) (Seat, error) {
	if err := validateIdentifier("actor", string(actor)); err != nil {
		return 0, err
	}
	for _, current := range h.seats {
		if current.AccountID == actor {
			return current.Seat, nil
		}
	}
	return 0, ErrNotSeated
}

func (h *Hand) changePhase(to HandPhase) Event {
	from := h.phase
	h.phase = to
	return h.record(EventHandPhaseChanged, HandPhasePayload{From: from, To: to})
}

func (h *Hand) record(name string, payload any) Event {
	h.version++
	return Event{
		AggregateType: AggregateHand,
		AggregateID:   string(h.id),
		Name:          name,
		Version:       h.version,
		Payload:       payload,
	}
}

func (phase HandPhase) Terminal() bool {
	switch phase {
	case HandCompleted, HandCancelled, HandAborted, HandExpired:
		return true
	default:
		return false
	}
}
