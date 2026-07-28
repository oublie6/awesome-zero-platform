package application

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
)

type SubmitCommitInput struct{ Commitment domain.Commitment }
type SubmitRevealInput struct{ Envelope SecureEnvelope }
type LockBeaconInput struct{ Value domain.BeaconValue }
type TerminateHandInput struct{ ReasonCode string }

func (s *Service) SubmitHandCommit(ctx context.Context, actor domain.AccountID, command Command, input SubmitCommitInput) (CommandResult, error) {
	return s.mutateHand(ctx, actor, command, CommandHandCommitSubmit, input, func(hand *domain.Hand) ([]domain.Event, error) {
		return hand.SubmitCommit(actor, input.Commitment, command.ExpectedVersion)
	})
}

func (s *Service) SubmitHandReveal(ctx context.Context, actor domain.AccountID, command Command, input SubmitRevealInput) (CommandResult, error) {
	return s.execute(ctx, actor, command, CommandHandRevealSubmit, domain.AggregateHand, false, input,
		func(txCtx context.Context, tx Transaction, now time.Time) (mutationOutcome, error) {
			snapshot, err := tx.LoadHandForUpdate(txCtx, domain.HandID(command.AggregateID))
			if err != nil {
				return mutationOutcome{}, err
			}
			hand, err := domain.RestoreHand(snapshot)
			if err != nil {
				return mutationOutcome{}, wrapInfrastructure("restore hand snapshot", err)
			}
			seat, contribution, err := resolveHandContribution(snapshot, actor)
			if err != nil {
				return mutationOutcome{}, &aggregateMutationError{cause: err, currentVersion: snapshot.Version}
			}
			if input.Envelope.KeyID != snapshot.RevealKeyID {
				return mutationOutcome{}, &aggregateMutationError{cause: ErrRevealInvalid, currentVersion: snapshot.Version}
			}
			aad, err := BuildRevealAAD(command, actor, seat, contribution.Commitment, snapshot.RevealKeyID)
			if err != nil {
				return mutationOutcome{}, &aggregateMutationError{cause: err, currentVersion: snapshot.Version}
			}
			plaintext, err := s.opener.Open(txCtx, input.Envelope, aad, RevealKeyContext{
				KeyID:           snapshot.RevealKeyID,
				PublicKeySHA256: [32]byte(snapshot.RevealPublicKeySHA256),
				BoundAt:         snapshot.RevealKeyBoundAt,
				UseAt:           now,
			})
			if err != nil {
				return mutationOutcome{}, &aggregateMutationError{cause: ErrRevealInvalid, currentVersion: snapshot.Version}
			}
			defer clearBytes(plaintext)

			decoded, err := decodeRevealPlaintext(plaintext, snapshot.ID, seat, s.normalizer, s.config.MaxRevealPhraseBytes)
			if err != nil {
				return mutationOutcome{}, &aggregateMutationError{cause: err, currentVersion: snapshot.Version}
			}
			defer clearBytes(decoded.SecureRandom[:])

			recordID, err := s.ids.NewID()
			if err != nil {
				return mutationOutcome{}, wrapInfrastructure("generate contribution record ID", err)
			}
			events, err := hand.SubmitReveal(actor, decoded.Digest, recordID, command.ExpectedVersion)
			if err != nil {
				return mutationOutcome{}, &aggregateMutationError{cause: err, currentVersion: snapshot.Version}
			}
			recordAAD, err := BuildContributionRecordAAD(recordID, snapshot.ID, seat, actor, command.CommandID, decoded.Digest)
			if err != nil {
				return mutationOutcome{}, err
			}
			protected, err := s.protector.Seal(txCtx, plaintext, recordAAD)
			if err != nil {
				return mutationOutcome{}, fmt.Errorf("%w: %v", ErrProtectionFailed, err)
			}
			if protected.KeyID == "" || len(protected.Nonce) == 0 || len(protected.Ciphertext) == 0 || protected.AADDigest != sha256.Sum256(recordAAD) {
				return mutationOutcome{}, fmt.Errorf("%w: invalid protected payload", ErrProtectionFailed)
			}
			record := ProtectedContributionRecord{
				RecordID: recordID, HandID: snapshot.ID, Seat: seat, ActorAccountID: actor,
				CommandID: command.CommandID, ContributionDigest: decoded.Digest,
				ProtectionKeyID: protected.KeyID, Nonce: append([]byte(nil), protected.Nonce...),
				Ciphertext: append([]byte(nil), protected.Ciphertext...), AADDigest: protected.AADDigest, CreatedAt: now,
			}
			if err := tx.InsertContributionRecord(txCtx, record); err != nil {
				return mutationOutcome{}, err
			}
			updated := hand.Snapshot()
			if err := tx.UpdateHand(txCtx, updated, snapshot.Version, now); err != nil {
				return mutationOutcome{}, err
			}
			return mutationOutcome{aggregateVersion: updated.Version, events: events}, nil
		})
}

func (s *Service) LockHandBeacon(ctx context.Context, actor domain.AccountID, command Command, input LockBeaconInput) (CommandResult, error) {
	return s.mutateHand(ctx, actor, command, CommandHandBeaconLock, input, func(hand *domain.Hand) ([]domain.Event, error) {
		return hand.LockPublicBeacon(input.Value, command.ExpectedVersion)
	})
}

func (s *Service) MarkHandDealt(ctx context.Context, actor domain.AccountID, command Command) (CommandResult, error) {
	return s.mutateHand(ctx, actor, command, CommandHandDealt, struct{}{}, func(hand *domain.Hand) ([]domain.Event, error) {
		return hand.MarkDealt(command.ExpectedVersion)
	})
}

func (s *Service) StartHandPlaying(ctx context.Context, actor domain.AccountID, command Command) (CommandResult, error) {
	return s.mutateHand(ctx, actor, command, CommandHandPlayStart, struct{}{}, func(hand *domain.Hand) ([]domain.Event, error) {
		return hand.StartPlaying(command.ExpectedVersion)
	})
}

func (s *Service) StartHandSettlement(ctx context.Context, actor domain.AccountID, command Command) (CommandResult, error) {
	return s.mutateHand(ctx, actor, command, CommandHandSettlement, struct{}{}, func(hand *domain.Hand) ([]domain.Event, error) {
		return hand.StartSettling(command.ExpectedVersion)
	})
}

func (s *Service) CompleteHand(ctx context.Context, actor domain.AccountID, command Command) (CommandResult, error) {
	return s.terminateHand(ctx, actor, command, CommandHandComplete, struct{}{}, func(hand *domain.Hand) ([]domain.Event, error) {
		return hand.Complete(command.ExpectedVersion)
	})
}

func (s *Service) CancelHand(ctx context.Context, actor domain.AccountID, command Command, input TerminateHandInput) (CommandResult, error) {
	return s.terminateHand(ctx, actor, command, CommandHandCancel, input, func(hand *domain.Hand) ([]domain.Event, error) {
		return hand.Cancel(input.ReasonCode, command.ExpectedVersion)
	})
}

func (s *Service) AbortHand(ctx context.Context, actor domain.AccountID, command Command, input TerminateHandInput) (CommandResult, error) {
	return s.terminateHand(ctx, actor, command, CommandHandAbort, input, func(hand *domain.Hand) ([]domain.Event, error) {
		return hand.Abort(input.ReasonCode, command.ExpectedVersion)
	})
}

func (s *Service) ExpireHand(ctx context.Context, actor domain.AccountID, command Command, input TerminateHandInput) (CommandResult, error) {
	return s.terminateHand(ctx, actor, command, CommandHandExpire, input, func(hand *domain.Hand) ([]domain.Event, error) {
		return hand.Expire(input.ReasonCode, command.ExpectedVersion)
	})
}

func (s *Service) mutateHand(ctx context.Context, actor domain.AccountID, command Command, name string, payload any, mutation func(*domain.Hand) ([]domain.Event, error)) (CommandResult, error) {
	return s.execute(ctx, actor, command, name, domain.AggregateHand, false, payload,
		func(txCtx context.Context, tx Transaction, now time.Time) (mutationOutcome, error) {
			snapshot, err := tx.LoadHandForUpdate(txCtx, domain.HandID(command.AggregateID))
			if err != nil {
				return mutationOutcome{}, err
			}
			hand, err := domain.RestoreHand(snapshot)
			if err != nil {
				return mutationOutcome{}, wrapInfrastructure("restore hand snapshot", err)
			}
			events, err := mutation(hand)
			if err != nil {
				return mutationOutcome{}, &aggregateMutationError{cause: err, currentVersion: snapshot.Version}
			}
			updated := hand.Snapshot()
			if err := tx.UpdateHand(txCtx, updated, snapshot.Version, now); err != nil {
				return mutationOutcome{}, err
			}
			return mutationOutcome{aggregateVersion: updated.Version, events: events}, nil
		})
}

func (s *Service) terminateHand(ctx context.Context, actor domain.AccountID, command Command, name string, payload any, mutation func(*domain.Hand) ([]domain.Event, error)) (CommandResult, error) {
	return s.execute(ctx, actor, command, name, domain.AggregateHand, false, payload,
		func(txCtx context.Context, tx Transaction, now time.Time) (mutationOutcome, error) {
			handSnapshot, err := tx.LoadHandForUpdate(txCtx, domain.HandID(command.AggregateID))
			if err != nil {
				return mutationOutcome{}, err
			}
			hand, err := domain.RestoreHand(handSnapshot)
			if err != nil {
				return mutationOutcome{}, wrapInfrastructure("restore hand snapshot", err)
			}
			handEvents, err := mutation(hand)
			if err != nil {
				return mutationOutcome{}, &aggregateMutationError{cause: err, currentVersion: handSnapshot.Version}
			}
			updatedHand := hand.Snapshot()
			if !updatedHand.Phase.Terminal() {
				return mutationOutcome{}, wrapInfrastructure("terminate hand", fmt.Errorf("non-terminal result"))
			}

			roomSnapshot, err := tx.LoadRoomForUpdate(txCtx, handSnapshot.RoomID)
			if err != nil {
				return mutationOutcome{}, err
			}
			room, err := domain.RestoreRoom(roomSnapshot)
			if err != nil {
				return mutationOutcome{}, wrapInfrastructure("restore room snapshot", err)
			}
			roomEvents, err := room.FinishHand(handSnapshot.ID, roomSnapshot.Version)
			if err != nil {
				return mutationOutcome{}, wrapInfrastructure("release terminal hand room", err)
			}

			if err := tx.UpdateHand(txCtx, updatedHand, handSnapshot.Version, now); err != nil {
				return mutationOutcome{}, err
			}
			if err := tx.UpdateRoom(txCtx, room.Snapshot(), roomSnapshot.Version, now); err != nil {
				return mutationOutcome{}, err
			}
			events := append(append([]domain.Event{}, handEvents...), roomEvents...)
			return mutationOutcome{aggregateVersion: updatedHand.Version, events: events}, nil
		})
}

func resolveHandContribution(snapshot domain.HandSnapshot, actor domain.AccountID) (domain.Seat, domain.ContributionSnapshot, error) {
	for _, seat := range snapshot.Seats {
		if seat.AccountID == actor {
			for _, contribution := range snapshot.Contributions {
				if contribution.Seat == seat.Seat {
					return seat.Seat, contribution, nil
				}
			}
			return 0, domain.ContributionSnapshot{}, fmt.Errorf("%w: missing contribution seat", ErrRevealInvalid)
		}
	}
	return 0, domain.ContributionSnapshot{}, domain.ErrNotSeated
}
