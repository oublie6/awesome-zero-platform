package application

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
)

type Config struct {
	MaxCommandTTL        time.Duration
	MaxClockSkew         time.Duration
	MaxRevealPhraseBytes int
}

func DefaultConfig() Config {
	return Config{
		MaxCommandTTL:        2 * time.Minute,
		MaxClockSkew:         30 * time.Second,
		MaxRevealPhraseBytes: 1024,
	}
}

type Service struct {
	store      Store
	clock      Clock
	ids        IDGenerator
	setups     HandSetupProvider
	opener     EnvelopeOpener
	protector  ContributionProtector
	normalizer PhraseNormalizer
	config     Config
}

func NewService(
	store Store,
	clock Clock,
	ids IDGenerator,
	setups HandSetupProvider,
	opener EnvelopeOpener,
	protector ContributionProtector,
	normalizer PhraseNormalizer,
	config Config,
) (*Service, error) {
	if store == nil || clock == nil || ids == nil || setups == nil || opener == nil || protector == nil || normalizer == nil {
		return nil, fmt.Errorf("%w: application dependencies", ErrInvalidCommand)
	}
	if config.MaxCommandTTL <= 0 || config.MaxClockSkew < 0 || config.MaxRevealPhraseBytes <= 0 {
		return nil, fmt.Errorf("%w: application configuration", ErrInvalidCommand)
	}
	return &Service{
		store: store, clock: clock, ids: ids, setups: setups,
		opener: opener, protector: protector, normalizer: normalizer, config: config,
	}, nil
}

type mutationOutcome struct {
	aggregateVersion uint64
	events           []domain.Event
}

type aggregateMutationError struct {
	cause          error
	currentVersion uint64
}

func (e *aggregateMutationError) Error() string { return e.cause.Error() }
func (e *aggregateMutationError) Unwrap() error { return e.cause }

type mutationHandler func(context.Context, Transaction, time.Time) (mutationOutcome, error)

func (s *Service) execute(
	ctx context.Context,
	actor domain.AccountID,
	command Command,
	expectedName string,
	expectedType domain.AggregateType,
	allowZeroVersion bool,
	payload any,
	handler mutationHandler,
) (CommandResult, error) {
	boundCommand, err := bindCommandPayload(command, payload)
	if err != nil {
		return CommandResult{}, wrapInfrastructure("bind command payload", err)
	}
	command = boundCommand
	if failure := validateCommandIdentity(actor, command, expectedName, expectedType, allowZeroVersion); failure != nil {
		return rejectedResult(command, failure.failure, 0), nil
	}

	var result CommandResult
	err = s.store.WithinCommand(ctx, actor, command.CommandID, func(txCtx context.Context, tx Transaction) error {
		now := s.clock.Now().UTC()
		stored, completed, err := tx.ClaimCommand(txCtx, actor, command, now)
		if err != nil {
			return wrapInfrastructure("claim command", err)
		}
		if !sameCommand(stored.Command, command) {
			currentVersion := uint64(0)
			if completed {
				currentVersion = stored.Result.AggregateVersion
			}
			result = rejectedResult(command, reject(CodeConflict, "command ID is already bound to another request").failure, currentVersion)
			return nil
		}
		if completed {
			result = cloneResult(stored.Result)
			result.Duplicate = true
			return nil
		}

		if failure := s.validateFreshness(command, now); failure != nil {
			result = rejectedResult(command, failure.failure, 0)
			if err := tx.CompleteCommand(txCtx, actor, command.CommandID, result, now); err != nil {
				return wrapInfrastructure("complete rejected command", err)
			}
			return nil
		}

		lastSequence, err := tx.LockClientSequence(txCtx, command.AggregateType, command.AggregateID, actor)
		if err != nil {
			return wrapInfrastructure("lock client sequence", err)
		}
		if command.ClientSeq <= lastSequence {
			result = rejectedResult(command, reject(CodeSequenceConflict, "client sequence must increase").failure, 0)
			if err := tx.CompleteCommand(txCtx, actor, command.CommandID, result, now); err != nil {
				return wrapInfrastructure("complete sequence rejection", err)
			}
			return nil
		}

		outcome, mutationErr := handler(txCtx, tx, now)
		if mutationErr != nil {
			business := mapBusinessError(mutationErr)
			if business == nil {
				return mutationErr
			}
			currentVersion := uint64(0)
			if aggregateErr, ok := mutationErr.(*aggregateMutationError); ok {
				currentVersion = aggregateErr.currentVersion
			}
			if business.failure.CurrentVersion != nil {
				currentVersion = *business.failure.CurrentVersion
			}
			result = rejectedResult(command, business.failure, currentVersion)
			if err := tx.SaveClientSequence(txCtx, command.AggregateType, command.AggregateID, actor, command.ClientSeq, now); err != nil {
				return wrapInfrastructure("save rejected command sequence", err)
			}
			if err := tx.CompleteCommand(txCtx, actor, command.CommandID, result, now); err != nil {
				return wrapInfrastructure("complete rejected command", err)
			}
			return nil
		}

		outbox, refs, err := s.buildOutbox(actor, command, outcome.events, now)
		if err != nil {
			return err
		}
		if err := tx.AppendOutbox(txCtx, outbox); err != nil {
			return wrapInfrastructure("append outbox events", err)
		}
		if err := tx.SaveClientSequence(txCtx, command.AggregateType, command.AggregateID, actor, command.ClientSeq, now); err != nil {
			return wrapInfrastructure("save client sequence", err)
		}
		result = CommandResult{
			Version:          CommandResultV1,
			CommandID:        command.CommandID,
			Accepted:         true,
			Duplicate:        false,
			AggregateType:    command.AggregateType,
			AggregateID:      command.AggregateID,
			AggregateVersion: outcome.aggregateVersion,
			Events:           refs,
		}
		if err := tx.CompleteCommand(txCtx, actor, command.CommandID, result, now); err != nil {
			return wrapInfrastructure("complete command", err)
		}
		return nil
	})
	if err != nil {
		return CommandResult{}, err
	}
	return result, nil
}

func validateCommandIdentity(actor domain.AccountID, command Command, expectedName string, expectedType domain.AggregateType, allowZeroVersion bool) *businessRejection {
	if strings.TrimSpace(string(actor)) == "" || len(actor) > 128 {
		return reject(CodeInvalidCommand, "authenticated actor is invalid")
	}
	if command.Version != CommandProtocolV1 || command.Name != expectedName || command.AggregateType != expectedType {
		return reject(CodeInvalidCommand, "command protocol, name, or aggregate type is invalid")
	}
	if strings.TrimSpace(command.CommandID) == "" || len(command.CommandID) > 128 || strings.TrimSpace(command.AggregateID) == "" || len(command.AggregateID) > 128 {
		return reject(CodeInvalidCommand, "command or aggregate ID is invalid")
	}
	if command.ClientSeq == 0 {
		return reject(CodeInvalidCommand, "client sequence must be positive")
	}
	if !allowZeroVersion && command.ExpectedVersion == 0 {
		return reject(CodeInvalidCommand, "expected version must be positive")
	}
	if allowZeroVersion && command.ExpectedVersion != 0 {
		return reject(CodeInvalidCommand, "creation expected version must be zero")
	}
	return nil
}

func (s *Service) validateFreshness(command Command, now time.Time) *businessRejection {
	if command.IssuedAt.IsZero() || command.ExpiresAt.IsZero() {
		return reject(CodeInvalidCommand, "issued and expiry times are required")
	}
	if !validMillisecondPrecision(command.IssuedAt) || !validMillisecondPrecision(command.ExpiresAt) {
		return reject(CodeInvalidCommand, "command times must use millisecond precision")
	}
	issued := command.IssuedAt.UTC()
	expires := command.ExpiresAt.UTC()
	if !expires.After(issued) || expires.Sub(issued) > s.config.MaxCommandTTL {
		return reject(CodeInvalidCommand, "command expiry policy is invalid")
	}
	if issued.After(now.Add(s.config.MaxClockSkew)) || now.After(expires) {
		return reject(CodeInvalidCommand, "command is expired or issued in the future")
	}
	return nil
}

func (s *Service) buildOutbox(actor domain.AccountID, command Command, events []domain.Event, occurredAt time.Time) ([]OutboxEvent, []EventRef, error) {
	outbox := make([]OutboxEvent, 0, len(events))
	refs := make([]EventRef, 0, len(events))
	for _, event := range events {
		eventID, err := s.ids.NewID()
		if err != nil {
			return nil, nil, wrapInfrastructure("generate event ID", err)
		}
		payload, err := json.Marshal(event.Payload)
		if err != nil {
			return nil, nil, wrapInfrastructure("marshal domain event", err)
		}
		outbox = append(outbox, OutboxEvent{
			EventID: eventID, Protocol: EventProtocolV1, Name: event.Name,
			AggregateType: event.AggregateType, AggregateID: event.AggregateID,
			AggregateVersion: event.Version, OccurredAt: occurredAt,
			CausationCommandID: command.CommandID, ActorAccountID: actor, PayloadJSON: payload,
		})
		refs = append(refs, EventRef{AggregateType: event.AggregateType, AggregateID: event.AggregateID, Name: event.Name, Version: event.Version})
	}
	return outbox, refs, nil
}

func rejectedResult(command Command, failure CommandFailure, aggregateVersion uint64) CommandResult {
	failureCopy := failure
	return CommandResult{
		Version: CommandResultV1, CommandID: command.CommandID, Accepted: false,
		AggregateType: command.AggregateType, AggregateID: command.AggregateID,
		AggregateVersion: aggregateVersion, Events: []EventRef{}, Failure: &failureCopy,
	}
}

func cloneResult(value CommandResult) CommandResult {
	result := value
	result.Events = append([]EventRef(nil), value.Events...)
	if value.Failure != nil {
		copyFailure := *value.Failure
		if value.Failure.CurrentVersion != nil {
			version := *value.Failure.CurrentVersion
			copyFailure.CurrentVersion = &version
		}
		result.Failure = &copyFailure
	}
	return result
}

func sameCommand(left, right Command) bool {
	return left.Version == right.Version && left.Name == right.Name && left.CommandID == right.CommandID &&
		left.AggregateType == right.AggregateType && left.AggregateID == right.AggregateID &&
		left.ClientSeq == right.ClientSeq && left.ExpectedVersion == right.ExpectedVersion &&
		left.IssuedAt.Equal(right.IssuedAt) && left.ExpiresAt.Equal(right.ExpiresAt) &&
		left.PayloadDigest == right.PayloadDigest
}

func bindCommandPayload(command Command, payload any) (Command, error) {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return Command{}, err
	}
	hash := sha256.New()
	hash.Write([]byte("fair-doudizhu/command-payload/v1"))
	hash.Write([]byte{0})
	hash.Write([]byte(command.Name))
	hash.Write([]byte{0})
	hash.Write(encoded)
	copy(command.PayloadDigest[:], hash.Sum(nil))
	return command, nil
}
