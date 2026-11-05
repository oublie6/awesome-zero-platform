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
	actor       domain.AccountID
	phase       phase
	deadline    time.Time
	liveVersion uint64 // last live callback version applied to this timer
	pending     *pendingTermination
}

type pendingTermination struct {
	kind        terminationKind
	reason      string
	liveVersion uint64
	command     application.Command
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
	snapshot, err := s.authorize(ctx, actor, handID)
	if err != nil {
		return err
	}
	// A durable hand.dealt receipt may be replayed after the original
	// transition. Never regress an already-playing hand to bidding.
	if snapshot.Phase != domain.HandBidding {
		return nil
	}
	// Capture the already committed game version when first arming the timer.
	// The live executor can commit/publish before the transport's callback.
	progress, hasProgress, err := s.committedProgress(ctx, actor, handID)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	// The first successful registration owns the original deadline. A
	// verified duplicate may repair a missing entry but must not extend or
	// overwrite an existing bidding/playing timer.
	if _, tracked := s.entries[handID]; !tracked {
		current := entry{actor: actor, phase: phaseBidding, deadline: s.clock.Now().UTC().Add(s.config.BiddingTimeout)}
		if hasProgress {
			current.liveVersion = progress.version
			if progress.phase == phasePlaying {
				current.phase = phasePlaying
				current.deadline = s.clock.Now().UTC().Add(s.config.PlayingTimeout)
			}
		}
		s.entries[handID] = current
	}
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
		s.schedule(handID, actor, phaseBidding, s.config.BiddingTimeout, result.Version)
	case livehand.PhasePlaying:
		if err := s.advanceLiveHand(ctx, handID, false); err != nil {
			return Outcome{}, err
		}
		s.schedule(handID, actor, phasePlaying, s.config.PlayingTimeout, result.Version)
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
		s.schedule(handID, actor, phasePlaying, s.config.PlayingTimeout, result.Version)
	case livehand.PhaseCompleted:
		if err := s.advanceLiveHand(ctx, handID, true); err != nil {
			return err
		}
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
	type dueCandidate struct {
		handID domain.HandID
		state  entry
	}
	due := make([]dueCandidate, 0)
	for handID, current := range s.entries {
		if current.pending == nil && !now.Before(current.deadline) {
			due = append(due, dueCandidate{handID: handID, state: current})
		}
	}
	s.mu.Unlock()
	sort.Slice(due, func(i, j int) bool { return due[i].handID < due[j].handID })

	terminated := make([]domain.HandID, 0, len(due))
	for _, candidate := range due {
		// The committed live game may be newer than a timer whose callback is
		// still blocked in the transport. Never use that stale deadline.
		changed, err := s.refreshCommittedProgress(ctx, candidate.handID, candidate.state)
		if err != nil {
			return terminated, err
		}
		if changed {
			continue
		}
		reason := ReasonPlayingTimeout
		if candidate.state.phase == phaseBidding {
			reason = ReasonBiddingTimeout
		}
		outcome, err := s.terminateWhenDue(ctx, candidate.state.actor, candidate.handID,
			terminationExpire, reason, &candidate.state)
		if err != nil {
			return terminated, err
		}
		if outcome.Triggered {
			terminated = append(terminated, candidate.handID)
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

// schedule applies committed live callbacks in their state-version order, not
// their asynchronous response order. A newer callback is allowed to renew the
// current phase deadline, but an older callback cannot extend or regress it.
// Committed state is a read-only application projection, never client-supplied.
// In production the service delegates to the memory-authoritative runtime after
// MySQL commit and before returning the transport callback.
type committedState struct {
	version uint64
	phase   phase
}

func (s *Supervisor) committedProgress(ctx context.Context, actor domain.AccountID, handID domain.HandID) (committedState, bool, error) {
	reader, ok := s.terminal.(interface {
		GetHandPublicView(context.Context, domain.AccountID, domain.HandID) (application.LiveHandView, error)
	})
	if !ok {
		return committedState{}, false, nil // isolated legacy lifecycle fixtures
	}
	view, err := reader.GetHandPublicView(ctx, actor, handID)
	if err != nil {
		return committedState{}, true, fmt.Errorf("read committed live hand progress: %w", err)
	}
	var payload struct {
		HandID       string `json:"handId"`
		Phase        string `json:"phase"`
		StateVersion uint64 `json:"stateVersion"`
	}
	if err := json.Unmarshal(view.Payload, &payload); err != nil {
		return committedState{}, true, fmt.Errorf("decode committed live progress: %w", err)
	}
	if payload.HandID != string(handID) || payload.StateVersion == 0 || payload.StateVersion != view.Version {
		return committedState{}, true, fmt.Errorf("committed live progress identity mismatch")
	}
	if payload.Phase != livehand.PhaseBidding && payload.Phase != livehand.PhasePlaying {
		return committedState{}, true, fmt.Errorf("committed live phase %q is not active", payload.Phase)
	}
	return committedState{version: view.Version, phase: phase(payload.Phase)}, true, nil
}

// A new durable action may already be published when the prior expiry timer
// fires, but its dispatcher callback might be blocked. Detect that advanced
// version and rearm from the observed committed state, never from the old one.
// The separate MySQL writer watermark guard closes the race after this read.
func (s *Supervisor) refreshCommittedProgress(ctx context.Context, handID domain.HandID, original entry) (bool, error) {
	progress, available, err := s.committedProgress(ctx, original.actor, handID)
	if err != nil {
		return false, err
	}
	if !available {
		return false, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	current, exists := s.entries[handID]
	if !exists || current.pending != nil || current.liveVersion != original.liveVersion ||
		current.phase != original.phase || !current.deadline.Equal(original.deadline) {
		return true, nil
	}
	if progress.version <= current.liveVersion && progress.phase == current.phase {
		return false, nil
	}
	if progress.version < current.liveVersion || (current.phase == phasePlaying && progress.phase == phaseBidding) {
		return false, fmt.Errorf("committed live state regressed")
	}
	current.liveVersion = progress.version
	current.phase = progress.phase
	current.deadline = s.clock.Now().UTC().Add(s.config.BiddingTimeout)
	if progress.phase == phasePlaying {
		current.deadline = s.clock.Now().UTC().Add(s.config.PlayingTimeout)
	}
	s.entries[handID] = current
	return true, nil
}

func (s *Supervisor) schedule(handID domain.HandID, actor domain.AccountID, value phase, timeout time.Duration, version uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	current := s.entries[handID]
	if current.pending != nil || version <= current.liveVersion ||
		(current.phase == phasePlaying && value == phaseBidding) {
		return
	}
	current.actor = actor
	current.phase = value
	current.liveVersion = version
	current.deadline = s.clock.Now().UTC().Add(timeout)
	s.entries[handID] = current
}

// Caller holds s.mu. This comparison is the final linearization check for a
// timeout candidate; it prevents Sweep's stale list from expiring a renewed
// deadline while the operation was authorizing or loading the hand snapshot.
func (s *Supervisor) dueUnchangedLocked(handID domain.HandID, expected entry) bool {
	current, exists := s.entries[handID]
	return exists && current.pending == nil && current.actor == expected.actor &&
		current.phase == expected.phase && current.liveVersion == expected.liveVersion &&
		current.deadline.Equal(expected.deadline) &&
		!s.clock.Now().UTC().Before(current.deadline)
}

func (s *Supervisor) terminate(
	ctx context.Context,
	actor domain.AccountID,
	handID domain.HandID,
	kind terminationKind,
	reason string,
) (Outcome, error) {
	return s.terminateWhenDue(ctx, actor, handID, kind, reason, nil)
}

// Sweep captures the deadline and version at discovery time. Before creating a
// terminal command it must check that a newer accepted action has not renewed
// that deadline. The ordinary cancel/termination path has no expiry guard.
func (s *Supervisor) terminateWhenDue(
	ctx context.Context,
	actor domain.AccountID,
	handID domain.HandID,
	kind terminationKind,
	reason string,
	due *entry,
) (Outcome, error) {
	if _, err := s.authorize(ctx, actor, handID); err != nil {
		return Outcome{}, err
	}
	s.mu.Lock()
	current := s.entries[handID]
	if due != nil && !s.dueUnchangedLocked(handID, *due) {
		s.mu.Unlock()
		return Outcome{}, nil
	}
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
		kind:        kind,
		reason:      reason,
		liveVersion: current.liveVersion,
		command: application.Command{
			Version:         application.CommandProtocolV1,
			Name:            name,
			CommandID:       commandID,
			AggregateType:   domain.AggregateHand,
			AggregateID:     string(handID),
			ClientSeq:       snapshot.Version,
			ExpectedVersion: snapshot.Version,
			IssuedAt:        now,
			ExpiresAt:       now.Add(s.config.CommandTTL),
		},
	}
	s.mu.Lock()
	current = s.entries[handID]
	if due != nil && !s.dueUnchangedLocked(handID, *due) {
		s.mu.Unlock()
		return Outcome{}, nil
	}
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
	if pending.kind == terminationExpire {
		input.ExpectedLiveVersion = pending.liveVersion
	}
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
	if errors.Is(err, application.ErrLiveProgressed) {
		// Guarded MySQL transaction saw a newer committed action and rolled
		// back expiry. Retire the stale pending ID before refreshing the timer.
		s.mu.Lock()
		current := s.entries[handID]
		if current.pending != nil && current.pending.command.CommandID == pending.command.CommandID {
			current.pending = nil
			s.entries[handID] = current
		}
		s.mu.Unlock()
		if _, refreshErr := s.refreshCommittedProgress(ctx, handID, current); refreshErr != nil {
			return Outcome{}, refreshErr
		}
		return Outcome{}, nil
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
