package domain

import (
	"fmt"
	"strings"
)

// RestoreRoom reconstructs a room from a repository snapshot without emitting
// a creation event. It rejects inconsistent snapshots before exposing an
// aggregate to the application layer.
func RestoreRoom(snapshot RoomSnapshot) (*Room, error) {
	if err := validateIdentifier("roomId", string(snapshot.ID)); err != nil {
		return nil, err
	}
	if err := validateIdentifier("ownerId", string(snapshot.OwnerID)); err != nil {
		return nil, err
	}
	if snapshot.Version == 0 {
		return nil, fmt.Errorf("%w: room version must be positive", ErrInvalidArgument)
	}
	if !validRoomStatus(snapshot.Status) {
		return nil, fmt.Errorf("%w: invalid room status", ErrInvalidArgument)
	}

	room := &Room{
		id:           snapshot.ID,
		ownerID:      snapshot.OwnerID,
		status:       snapshot.Status,
		activeHandID: snapshot.ActiveHandID,
		version:      snapshot.Version,
	}
	seen := make(map[AccountID]struct{}, 3)
	for index, expectedSeat := range fixedSeats {
		current := snapshot.Seats[index]
		if current.Seat != expectedSeat {
			return nil, fmt.Errorf("%w: room snapshot seat order", ErrInvalidArgument)
		}
		if current.AccountID == "" {
			if current.Ready {
				return nil, fmt.Errorf("%w: empty seat cannot be ready", ErrInvalidArgument)
			}
			continue
		}
		if err := validateIdentifier("accountId", string(current.AccountID)); err != nil {
			return nil, err
		}
		if _, exists := seen[current.AccountID]; exists {
			return nil, fmt.Errorf("%w: duplicate room account", ErrInvalidArgument)
		}
		seen[current.AccountID] = struct{}{}
		room.seats[index] = roomSeat{AccountID: current.AccountID, Ready: current.Ready}
	}
	if room.seats[0].AccountID != snapshot.OwnerID {
		return nil, fmt.Errorf("%w: owner must occupy seat 1", ErrInvalidArgument)
	}
	if err := validateRoomSnapshotState(room); err != nil {
		return nil, err
	}
	return room, nil
}

// RestoreHand reconstructs a hand from a repository snapshot without emitting
// a creation event. Sensitive reveal plaintext is intentionally absent.
func RestoreHand(snapshot HandSnapshot) (*Hand, error) {
	if err := validateIdentifier("handId", string(snapshot.ID)); err != nil {
		return nil, err
	}
	if err := validateIdentifier("roomId", string(snapshot.RoomID)); err != nil {
		return nil, err
	}
	if snapshot.Version == 0 {
		return nil, fmt.Errorf("%w: hand version must be positive", ErrInvalidArgument)
	}
	if !validHandPhase(snapshot.Phase) {
		return nil, fmt.Errorf("%w: invalid hand phase", ErrInvalidArgument)
	}
	if err := validateHandSeats(snapshot.Seats); err != nil {
		return nil, err
	}
	if allZero32([32]byte(snapshot.ServerCommitment)) {
		return nil, fmt.Errorf("%w: server commitment is zero", ErrInvalidArgument)
	}
	if err := validateIdentifier("revealKeyId", snapshot.RevealKeyID); err != nil {
		return nil, err
	}
	if snapshot.RevealPublicKeySHA256 == (RevealPublicKeyHash{}) || snapshot.RevealKeyBoundAt.IsZero() {
		return nil, fmt.Errorf("%w: reveal key binding", ErrInvalidArgument)
	}
	if err := validateBeaconPlan(snapshot.BeaconPlan); err != nil {
		return nil, err
	}

	hand := &Hand{
		id:                    snapshot.ID,
		roomID:                snapshot.RoomID,
		phase:                 snapshot.Phase,
		seats:                 snapshot.Seats,
		serverCommitment:      snapshot.ServerCommitment,
		revealKeyID:           snapshot.RevealKeyID,
		revealPublicKeySHA256: snapshot.RevealPublicKeySHA256,
		revealKeyBoundAt:      snapshot.RevealKeyBoundAt.UTC(),
		beaconPlan:            snapshot.BeaconPlan,
		terminationReason:     snapshot.TerminationReason,
		version:               snapshot.Version,
	}
	if snapshot.Beacon != nil {
		if snapshot.Beacon.Provider != snapshot.BeaconPlan.Provider || snapshot.Beacon.Round != snapshot.BeaconPlan.Round {
			return nil, ErrBeaconMismatch
		}
		if allZero32([32]byte(snapshot.Beacon.Digest)) || strings.TrimSpace(snapshot.Beacon.ProofRef) == "" || len(snapshot.Beacon.ProofRef) > 512 {
			return nil, fmt.Errorf("%w: invalid beacon snapshot", ErrInvalidArgument)
		}
		copyValue := *snapshot.Beacon
		hand.beacon = &copyValue
	}

	for index, expectedSeat := range fixedSeats {
		current := snapshot.Contributions[index]
		if current.Seat != expectedSeat {
			return nil, fmt.Errorf("%w: contribution seat order", ErrInvalidArgument)
		}
		committed := current.Commitment != (Commitment{})
		if current.Committed != committed {
			return nil, fmt.Errorf("%w: contribution committed flag mismatch", ErrInvalidArgument)
		}
		if current.Revealed {
			if !committed || allZero32([32]byte(current.Digest)) {
				return nil, fmt.Errorf("%w: invalid revealed contribution", ErrInvalidArgument)
			}
			if strings.TrimSpace(current.RecordRef) == "" || len(current.RecordRef) > 256 {
				return nil, fmt.Errorf("%w: invalid contribution record reference", ErrInvalidArgument)
			}
		} else if current.Digest != (ContributionDigest{}) || current.RecordRef != "" {
			return nil, fmt.Errorf("%w: unrevealed contribution contains reveal data", ErrInvalidArgument)
		}
		hand.contributions[index] = contributionState{
			Commitment: current.Commitment,
			Revealed:   current.Revealed,
			Digest:     current.Digest,
			RecordRef:  current.RecordRef,
		}
	}
	if err := validateHandSnapshotState(hand); err != nil {
		return nil, err
	}
	return hand, nil
}

func validRoomStatus(status RoomStatus) bool {
	switch status {
	case RoomWaitingPlayers, RoomReady, RoomHandActive, RoomClosed:
		return true
	default:
		return false
	}
}

func validateRoomSnapshotState(room *Room) error {
	switch room.status {
	case RoomHandActive:
		if room.activeHandID == "" || room.occupiedCount() != 3 {
			return fmt.Errorf("%w: active room snapshot", ErrInvalidArgument)
		}
		if err := validateIdentifier("activeHandId", string(room.activeHandID)); err != nil {
			return err
		}
	case RoomReady:
		if room.activeHandID != "" || room.occupiedCount() != 3 || !room.allReady() {
			return fmt.Errorf("%w: ready room snapshot", ErrInvalidArgument)
		}
	case RoomWaitingPlayers:
		if room.activeHandID != "" || (room.occupiedCount() == 3 && room.allReady()) {
			return fmt.Errorf("%w: waiting room snapshot", ErrInvalidArgument)
		}
	case RoomClosed:
		if room.activeHandID != "" || room.occupiedCount() != 0 {
			return fmt.Errorf("%w: closed room snapshot", ErrInvalidArgument)
		}
	}
	return nil
}

func validHandPhase(phase HandPhase) bool {
	switch phase {
	case HandFairnessCommitting, HandFairnessRevealing, HandWaitingPublicBeacon,
		HandDealing, HandBidding, HandPlaying, HandSettling,
		HandCompleted, HandCancelled, HandAborted, HandExpired:
		return true
	default:
		return false
	}
}

func validateHandSnapshotState(hand *Hand) error {
	commits := hand.commitCount()
	reveals := hand.revealCount()
	if reveals > commits {
		return fmt.Errorf("%w: reveals exceed commitments", ErrInvalidArgument)
	}
	if hand.phase.Terminal() {
		if hand.phase == HandCompleted {
			if hand.terminationReason != "" {
				return fmt.Errorf("%w: completed hand has termination reason", ErrInvalidArgument)
			}
			if commits != 3 || reveals != 3 || hand.beacon == nil {
				return fmt.Errorf("%w: completed hand snapshot", ErrInvalidArgument)
			}
			return nil
		}
		if strings.TrimSpace(hand.terminationReason) == "" || len(hand.terminationReason) > 256 {
			return fmt.Errorf("%w: terminal hand reason", ErrInvalidArgument)
		}
		return nil
	}
	if hand.terminationReason != "" {
		return fmt.Errorf("%w: active hand has termination reason", ErrInvalidArgument)
	}
	switch hand.phase {
	case HandFairnessCommitting:
		if commits >= 3 || reveals != 0 || hand.beacon != nil {
			return fmt.Errorf("%w: committing hand snapshot", ErrInvalidArgument)
		}
	case HandFairnessRevealing:
		if commits != 3 || reveals >= 3 || hand.beacon != nil {
			return fmt.Errorf("%w: revealing hand snapshot", ErrInvalidArgument)
		}
	case HandWaitingPublicBeacon:
		if commits != 3 || reveals != 3 || hand.beacon != nil {
			return fmt.Errorf("%w: beacon-wait hand snapshot", ErrInvalidArgument)
		}
	case HandDealing, HandBidding, HandPlaying, HandSettling, HandCompleted:
		if commits != 3 || reveals != 3 || hand.beacon == nil {
			return fmt.Errorf("%w: active gameplay hand snapshot", ErrInvalidArgument)
		}
	}
	return nil
}
