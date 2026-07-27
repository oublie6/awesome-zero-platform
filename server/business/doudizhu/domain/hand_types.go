package domain

import "fmt"

type HandPhase string

const (
	HandFairnessCommitting  HandPhase = "FAIRNESS_COMMITTING"
	HandFairnessRevealing   HandPhase = "FAIRNESS_REVEALING"
	HandWaitingPublicBeacon HandPhase = "WAITING_PUBLIC_BEACON"
	HandDealing             HandPhase = "DEALING"
	HandBidding             HandPhase = "BIDDING"
	HandPlaying             HandPhase = "PLAYING"
	HandSettling            HandPhase = "SETTLING"
	HandCompleted           HandPhase = "COMPLETED"
	HandCancelled           HandPhase = "CANCELLED"
	HandAborted             HandPhase = "ABORTED"
	HandExpired             HandPhase = "EXPIRED"
)

const (
	EventHandCreated            = "doudizhu.hand.created.v1"
	EventFairnessCommitAccepted = "doudizhu.hand.fairness-commit-accepted.v1"
	EventFairnessRevealAccepted = "doudizhu.hand.fairness-reveal-accepted.v1"
	EventPublicBeaconLocked     = "doudizhu.hand.public-beacon-locked.v1"
	EventHandPhaseChanged       = "doudizhu.hand.phase-changed.v1"
	EventHandTerminated         = "doudizhu.hand.terminated.v1"
)

type HandSeat struct {
	Seat      Seat
	AccountID AccountID
}

type BeaconPlan struct {
	Provider string
	Round    string
}

type BeaconValue struct {
	Provider string
	Round    string
	Digest   BeaconDigest
	ProofRef string
}

type contributionState struct {
	Commitment Commitment
	Revealed   bool
	Digest     ContributionDigest
	RecordRef  string
}

type ContributionSnapshot struct {
	Seat       Seat
	Committed  bool
	Commitment Commitment
	Revealed   bool
	Digest     ContributionDigest
	RecordRef  string
}

type HandSnapshot struct {
	ID                HandID
	RoomID            RoomID
	Phase             HandPhase
	Seats             [3]HandSeat
	ServerCommitment  ServerCommitment
	RevealKeyID       string
	BeaconPlan        BeaconPlan
	Beacon            *BeaconValue
	Contributions     [3]ContributionSnapshot
	TerminationReason string
	Version           uint64
}

type Hand struct {
	id                HandID
	roomID            RoomID
	phase             HandPhase
	seats             [3]HandSeat
	serverCommitment  ServerCommitment
	revealKeyID       string
	beaconPlan        BeaconPlan
	beacon            *BeaconValue
	contributions     [3]contributionState
	terminationReason string
	version           uint64
}

type HandCreatedPayload struct {
	RoomID      RoomID
	Seats       [3]HandSeat
	RevealKeyID string
	BeaconPlan  BeaconPlan
}

type FairnessSeatPayload struct {
	Seat Seat
}

type PublicBeaconPayload struct {
	Provider string
	Round    string
	ProofRef string
}

type HandPhasePayload struct {
	From HandPhase
	To   HandPhase
}

type HandTerminationPayload struct {
	Phase  HandPhase
	Reason string
}

func NewHand(
	id HandID,
	roomID RoomID,
	seats [3]HandSeat,
	serverCommitment ServerCommitment,
	revealKeyID string,
	beaconPlan BeaconPlan,
) (*Hand, []Event, error) {
	if err := validateIdentifier("handId", string(id)); err != nil {
		return nil, nil, err
	}
	if err := validateIdentifier("roomId", string(roomID)); err != nil {
		return nil, nil, err
	}
	if allZero32([32]byte(serverCommitment)) {
		return nil, nil, fmt.Errorf("%w: server commitment is zero", ErrInvalidArgument)
	}
	if err := validateIdentifier("revealKeyId", revealKeyID); err != nil {
		return nil, nil, err
	}
	if err := validateBeaconPlan(beaconPlan); err != nil {
		return nil, nil, err
	}
	if err := validateHandSeats(seats); err != nil {
		return nil, nil, err
	}

	hand := &Hand{
		id:               id,
		roomID:           roomID,
		phase:            HandFairnessCommitting,
		seats:            seats,
		serverCommitment: serverCommitment,
		revealKeyID:      revealKeyID,
		beaconPlan:       beaconPlan,
	}
	event := hand.record(EventHandCreated, HandCreatedPayload{
		RoomID:      roomID,
		Seats:       seats,
		RevealKeyID: revealKeyID,
		BeaconPlan:  beaconPlan,
	})
	return hand, []Event{event}, nil
}

func (h *Hand) Snapshot() HandSnapshot {
	var contributions [3]ContributionSnapshot
	for index, seat := range fixedSeats {
		current := h.contributions[index]
		contributions[index] = ContributionSnapshot{
			Seat:       seat,
			Committed:  current.Commitment != (Commitment{}),
			Commitment: current.Commitment,
			Revealed:   current.Revealed,
			Digest:     current.Digest,
			RecordRef:  current.RecordRef,
		}
	}
	var beacon *BeaconValue
	if h.beacon != nil {
		copyValue := *h.beacon
		beacon = &copyValue
	}
	return HandSnapshot{
		ID:                h.id,
		RoomID:            h.roomID,
		Phase:             h.phase,
		Seats:             h.seats,
		ServerCommitment:  h.serverCommitment,
		RevealKeyID:       h.revealKeyID,
		BeaconPlan:        h.beaconPlan,
		Beacon:            beacon,
		Contributions:     contributions,
		TerminationReason: h.terminationReason,
		Version:           h.version,
	}
}
