package lifecycle

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/application"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/livehand"
)

const (
	ReasonNoLandlord                         = "no_landlord"
	ReasonParticipantCancel                  = "participant_cancelled"
	ReasonBiddingTimeout                     = "bidding_timeout"
	ReasonPlayingTimeout                     = "playing_timeout"
	SystemActor             domain.AccountID = "system:doudizhu-lifecycle"
)

type Config struct {
	BiddingTimeout time.Duration
	PlayingTimeout time.Duration
	SweepInterval  time.Duration
	CommandTTL     time.Duration
}

func DefaultConfig() Config {
	return Config{
		BiddingTimeout: 45 * time.Second,
		PlayingTimeout: 60 * time.Second,
		SweepInterval:  time.Second,
		CommandTTL:     time.Minute,
	}
}

type HandReader interface {
	LoadHand(context.Context, domain.HandID) (domain.HandSnapshot, error)
}

type TerminalService interface {
	AbortHand(context.Context, domain.AccountID, application.Command, application.TerminateHandInput) (application.CommandResult, error)
	ExpireHand(context.Context, domain.AccountID, application.Command, application.TerminateHandInput) (application.CommandResult, error)
}

type Supervisor struct {
	mu       sync.Mutex
	hands    HandReader
	terminal TerminalService
	clock    application.Clock
	ids      application.IDGenerator
	config   Config
	entries  map[domain.HandID]entry
}

type phase string

const (
	phaseBidding phase = "BIDDING"
	phasePlaying phase = "PLAYING"
)

type terminationKind string

const (
	terminationAbort  terminationKind = "ABORT"
	terminationExpire terminationKind = "EXPIRE"
)

type entry struct {
	actor    domain.AccountID
	phase    phase
	deadline time.Time
	pending  *pendingTermination
}

type pendingTermination struct {
	kind    terminationKind
	reason  string
	command application.Command
}

type Outcome struct {
	Triggered bool
	Result    application.CommandResult
}

func New(
	hands HandReader,
	terminal TerminalService,
	clock application.Clock,
	ids application.IDGenerator,
	config Config,
) (*Supervisor, error) {
	if hands == nil || terminal == nil || clock == nil || ids == nil {
		return nil, fmt.Errorf("lifecycle supervisor dependencies are invalid")
	}
	if config.BiddingTimeout <= 0 || config.PlayingTimeout <= 0 || config.SweepInterval <= 0 || config.CommandTTL <= 0 {
		return nil, fmt.Errorf("lifecycle supervisor configuration is invalid")
	}
	return &Supervisor{
		hands: hands, terminal: terminal, clock: clock, ids: ids, config: config,
		entries: make(map[domain.HandID]entry),
	}, nil
}

func (s *Supervisor) TrackBidding(ctx context.Context, actor domain.AccountID, handID domain.HandID) error {
	if _, err := s.authorize(ctx, actor, handID); err != nil {
		return err
	}
	s.schedule(handID, actor, phaseBidding, s.config.BiddingTimeout)
	return nil
}

func (s *Supervisor) AfterBid(
	ctx context.Context,
	actor domain.AccountID,
	handID domain.HandID,
	result application.LiveHandCommandResult,
) (Outcome, error) {
	var bid livehand.BidResult
	if err := decodeStrict(result.Payload, &bid); err != nil {
		return Outcome{}, fmt.Errorf("decode lifecycle bid result: %w", err)
	}
	if bid.HandID != string(handID) || bid.StateVersion != result.Version || bid.RequiresTermination != result.RequiresTermination {
		return Outcome{}, fmt.Errorf("lifecycle bid result identity is invalid")
	}
	if bid.RequiresTermination {
		if bid.Phase != livehand.PhaseNoLandlord {
			return Outcome{}, fmt.Errorf("lifecycle terminal bid phase is invalid")
		}
		return s.terminate(ctx, actor, handID, terminationAbort, ReasonNoLandlord)
	}
	switch bid.Phase {
	case livehand.PhaseBidding:
		s.schedule(handID, actor, phaseBidding, s.config.BiddingTimeout)
	case livehand.PhasePlaying:
		s.schedule(handID, actor, phasePlaying, s.config.PlayingTimeout)
	default:
		return Outcome{}, fmt.Errorf("lifecycle bid phase %q is invalid", bid.Phase)
	}
	return Outcome{}, nil
}

func (s *Supervisor) AfterPlay(
	ctx context.Context,
	actor domain.AccountID,
	handID domain.HandID,
	result application.LiveHandCommandResult,
) error {
	var play livehand.PlayResult
	if err := decodeStrict(result.Payload, &play); err != nil {
		return fmt.Errorf("decode lifecycle play result: %w", err)
	}
	if play.HandID != string(handID) || play.StateVersion != result.Version {
		return fmt.Errorf("lifecycle play result identity is invalid")
	}
	switch play.Phase {
	case livehand.PhasePlaying:
		s.schedule(handID, actor, phasePlaying, s.config.PlayingTimeout)
	case livehand.PhaseCompleted:
		s.Forget(handID)
	default:
		return fmt.Errorf("lifecycle play phase %q is invalid", play.Phase)
	}
	return nil
}

func (s *Supervisor) Cancel(ctx context.Context, actor domain.AccountID, handID domain.HandID) (Outcome, error) {
	return s.terminate(ctx, actor, handID, terminationAbort, ReasonParticipantCancel)
}

func (s *Supervisor) Retry(ctx context.Context, handID domain.HandID) (Outcome, error) {
	s.mu.Lock()
	current, ok := s.entries[handID]
	if !ok || current.pending == nil {
		s.mu.Unlock()
		return Outcome{}, fmt.Errorf("no pending termination for hand %s", handID)
	}
	pending := *current.pending
	s.mu.Unlock()
	return s.executePending(ctx, handID, pending)
}

func (s *Supervisor) Sweep(ctx context.Context) ([]domain.HandID, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	now := s.clock.Now().UTC()
	s.mu.Lock()
	due := make([]domain.HandID, 0)
	for handID, current := range s.entries {
		if current.pending == nil && !now.Before(current.deadline) {
			due = append(due, handID)
		}
	}
	s.mu.Unlock()
	sort.Slice(due, func(i, j int) bool { return due[i] < due[j] })

	terminated := make([]domain.HandID, 0, len(due))
	for _, handID := range due {
		s.mu.Lock()
		current, ok := s.entries[handID]
		s.mu.Unlock()
		if !ok {
			continue
		}
		reason := ReasonPlayingTimeout
		if current.phase == phaseBidding {
			reason = ReasonBiddingTimeout
		}
		outcome, err := s.terminate(ctx, current.actor, handID, terminationExpire, reason)
		if err != nil {
			return terminated, err
		}
		if outcome.Triggered {
			terminated = append(terminated, handID)
		}
	}
	return terminated, nil
}

func (s *Supervisor) Run(ctx context.Context) error {
	ticker := time.NewTicker(s.config.SweepInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if _, err := s.Sweep(ctx); err != nil && !errors.Is(err, context.Canceled) {
				return err
			}
		}
	}
}

func (s *Supervisor) Forget(handID domain.HandID) {
	s.mu.Lock()
	delete(s.entries, handID)
	s.mu.Unlock()
}

func (s *Supervisor) Pending(handID domain.HandID) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	current, ok := s.entries[handID]
	return ok && current.pending != nil
}

func (s *Supervisor) schedule(handID domain.HandID, actor domain.AccountID, value phase, timeout time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	current := s.entries[handID]
	if current.pending != nil {
		return
	}
	current.actor = actor
	current.phase = value
	current.deadline = s.clock.Now().UTC().Add(timeout)
	s.entries[handID] = current
}

func (s *Supervisor) terminate(
	ctx context.Context,
	actor domain.AccountID,
	handID domain.HandID,
	kind terminationKind,
	reason string,
) (Outcome, error) {
	if _, err := s.authorize(ctx, actor, handID); err != nil {
		return Outcome{}, err
	}
	s.mu.Lock()
	current := s.entries[handID]
	if current.pending != nil {
		pending := *current.pending
		s.mu.Unlock()
		if pending.kind != kind || pending.reason != reason {
			return Outcome{}, fmt.Errorf("termination already pending for hand %s", handID)
		}
		return s.executePending(ctx, handID, pending)
	}
	s.mu.Unlock()

	snapshot, err := s.hands.LoadHand(ctx, handID)
	if err != nil {
		return Outcome{}, err
	}
	if snapshot.Phase.Terminal() {
		s.Forget(handID)
		return Outcome{}, nil
	}
	commandID, err := s.ids.NewID()
	if err != nil {
		return Outcome{}, fmt.Errorf("generate lifecycle command ID: %w", err)
	}
	now := s.clock.Now().UTC().Truncate(time.Millisecond)
	name := application.CommandHandAbort
	if kind == terminationExpire {
		name = application.CommandHandExpire
	}
	pending := pendingTermination{
		kind:   kind,
		reason: reason,
		command: application.Command{
			Version:         application.CommandProtocolV1,
			Name:            name,
			CommandID:       commandID,
			AggregateType:   domain.AggregateHand,
			AggregateID:     string(handID),
			ClientSeq:       1,
			ExpectedVersion: snapshot.Version,
			IssuedAt:        now,
			ExpiresAt:       now.Add(s.config.CommandTTL),
		},
	}
	s.mu.Lock()
	current = s.entries[handID]
	if current.pending == nil {
		current.actor = actor
		current.pending = &pending
		s.entries[handID] = current
	} else {
		pending = *current.pending
	}
	s.mu.Unlock()
	return s.executePending(ctx, handID, pending)
}

func (s *Supervisor) executePending(ctx context.Context, handID domain.HandID, pending pendingTermination) (Outcome, error) {
	input := application.TerminateHandInput{ReasonCode: pending.reason}
	var result application.CommandResult
	var err error
	switch pending.kind {
	case terminationAbort:
		result, err = s.terminal.AbortHand(ctx, SystemActor, pending.command, input)
	case terminationExpire:
		result, err = s.terminal.ExpireHand(ctx, SystemActor, pending.command, input)
	default:
		return Outcome{}, fmt.Errorf("unsupported termination kind %q", pending.kind)
	}
	if err != nil {
		return Outcome{Triggered: true, Result: result}, err
	}
	if !result.Accepted {
		return Outcome{Triggered: true, Result: result}, fmt.Errorf("lifecycle termination rejected for hand %s", handID)
	}
	s.Forget(handID)
	return Outcome{Triggered: true, Result: result}, nil
}

func (s *Supervisor) authorize(ctx context.Context, actor domain.AccountID, handID domain.HandID) (domain.HandSnapshot, error) {
	if strings.TrimSpace(string(actor)) == "" || strings.TrimSpace(string(handID)) == "" {
		return domain.HandSnapshot{}, fmt.Errorf("actor and hand ID are required")
	}
	snapshot, err := s.hands.LoadHand(ctx, handID)
	if err != nil {
		return domain.HandSnapshot{}, err
	}
	for _, seat := range snapshot.Seats {
		if seat.AccountID == actor {
			return snapshot, nil
		}
	}
	return domain.HandSnapshot{}, domain.ErrNotSeated
}

func decodeStrict(payload []byte, destination any) error {
	if len(payload) == 0 {
		return fmt.Errorf("empty payload")
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return fmt.Errorf("trailing JSON value")
		}
		return err
	}
	return nil
}
