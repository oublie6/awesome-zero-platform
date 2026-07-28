package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
)

type SetReadyInput struct{ Ready bool }
type StartHandInput struct{ HandID domain.HandID }

func (s *Service) CreateRoom(ctx context.Context, actor domain.AccountID, command Command) (CommandResult, error) {
	return s.execute(ctx, actor, command, CommandRoomCreate, domain.AggregateRoom, true, struct{}{},
		func(txCtx context.Context, tx Transaction, now time.Time) (mutationOutcome, error) {
			room, events, err := domain.NewRoom(domain.RoomID(command.AggregateID), actor)
			if err != nil {
				return mutationOutcome{}, err
			}
			snapshot := room.Snapshot()
			if err := tx.InsertRoom(txCtx, snapshot, now); err != nil {
				return mutationOutcome{}, err
			}
			return mutationOutcome{aggregateVersion: snapshot.Version, events: events}, nil
		})
}

func (s *Service) JoinRoom(ctx context.Context, actor domain.AccountID, command Command) (CommandResult, error) {
	return s.mutateRoom(ctx, actor, command, CommandRoomJoin, struct{}{}, func(room *domain.Room) ([]domain.Event, error) {
		return room.Join(actor, command.ExpectedVersion)
	})
}

func (s *Service) LeaveRoom(ctx context.Context, actor domain.AccountID, command Command) (CommandResult, error) {
	return s.mutateRoom(ctx, actor, command, CommandRoomLeave, struct{}{}, func(room *domain.Room) ([]domain.Event, error) {
		return room.Leave(actor, command.ExpectedVersion)
	})
}

func (s *Service) SetRoomReady(ctx context.Context, actor domain.AccountID, command Command, input SetReadyInput) (CommandResult, error) {
	return s.mutateRoom(ctx, actor, command, CommandRoomReadySet, input, func(room *domain.Room) ([]domain.Event, error) {
		return room.SetReady(actor, input.Ready, command.ExpectedVersion)
	})
}

func (s *Service) StartRoomHand(ctx context.Context, actor domain.AccountID, command Command, input StartHandInput) (CommandResult, error) {
	return s.execute(ctx, actor, command, CommandRoomHandStart, domain.AggregateRoom, false, input,
		func(txCtx context.Context, tx Transaction, now time.Time) (mutationOutcome, error) {
			roomSnapshot, err := tx.LoadRoomForUpdate(txCtx, domain.RoomID(command.AggregateID))
			if err != nil {
				return mutationOutcome{}, err
			}
			room, err := domain.RestoreRoom(roomSnapshot)
			if err != nil {
				return mutationOutcome{}, wrapInfrastructure("restore room snapshot", err)
			}
			roomEvents, err := room.StartHand(actor, input.HandID, command.ExpectedVersion)
			if err != nil {
				return mutationOutcome{}, &aggregateMutationError{cause: err, currentVersion: roomSnapshot.Version}
			}
			seats, err := room.HandSeats()
			if err != nil {
				return mutationOutcome{}, wrapInfrastructure("snapshot hand seats", err)
			}
			setup, err := s.setups.PrepareHand(txCtx, roomSnapshot, input.HandID)
			if err != nil {
				return mutationOutcome{}, wrapInfrastructure("prepare hand setup", err)
			}
			release := func() error {
				return s.setups.ReleaseHand(context.Background(), input.HandID)
			}
			failAfterPreparation := func(cause error) (mutationOutcome, error) {
				if releaseErr := release(); releaseErr != nil {
					cause = errors.Join(cause, wrapInfrastructure("release prepared hand setup", releaseErr))
				}
				return mutationOutcome{}, cause
			}
			if setup.HandID != input.HandID || setup.HandID == "" {
				return failAfterPreparation(wrapInfrastructure("prepare hand setup", fmt.Errorf("%w: hand setup ID mismatch", ErrInvalidCommand)))
			}
			hand, handEvents, err := domain.NewHand(
				setup.HandID, roomSnapshot.ID, seats, setup.ServerCommitment, setup.RevealKeyID, setup.BeaconPlan,
				domain.RevealKeyBinding{PublicKeySHA256: setup.RevealPublicKeySHA256, BoundAt: setup.RevealKeyBoundAt},
			)
			if err != nil {
				return failAfterPreparation(wrapInfrastructure("construct prepared hand", err))
			}
			if err := tx.InsertHand(txCtx, hand.Snapshot(), now); err != nil {
				return failAfterPreparation(err)
			}
			if err := tx.UpdateRoom(txCtx, room.Snapshot(), roomSnapshot.Version, now); err != nil {
				return failAfterPreparation(err)
			}
			events := append(append([]domain.Event{}, roomEvents...), handEvents...)
			return mutationOutcome{
				aggregateVersion: room.Snapshot().Version,
				events:           events,
				rollback:         release,
			}, nil
		})
}

func (s *Service) mutateRoom(ctx context.Context, actor domain.AccountID, command Command, name string, payload any, mutation func(*domain.Room) ([]domain.Event, error)) (CommandResult, error) {
	return s.execute(ctx, actor, command, name, domain.AggregateRoom, false, payload,
		func(txCtx context.Context, tx Transaction, now time.Time) (mutationOutcome, error) {
			snapshot, err := tx.LoadRoomForUpdate(txCtx, domain.RoomID(command.AggregateID))
			if err != nil {
				return mutationOutcome{}, err
			}
			room, err := domain.RestoreRoom(snapshot)
			if err != nil {
				return mutationOutcome{}, wrapInfrastructure("restore room snapshot", err)
			}
			events, err := mutation(room)
			if err != nil {
				return mutationOutcome{}, &aggregateMutationError{cause: err, currentVersion: snapshot.Version}
			}
			updated := room.Snapshot()
			if err := tx.UpdateRoom(txCtx, updated, snapshot.Version, now); err != nil {
				return mutationOutcome{}, err
			}
			return mutationOutcome{aggregateVersion: updated.Version, events: events}, nil
		})
}
