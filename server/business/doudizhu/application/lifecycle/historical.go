package lifecycle

import (
	"context"
	"fmt"
	"time"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/application"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/livehand"
)

// AfterHistoricalBid reconciles a *missing* durable lifecycle transition
// without re-running a historic bid's timer-resetting AfterBid callback.
// The receipt has already been validated against the authenticated actor,
// request identity, original version and payload by the live hand coordinator.
func (s *Supervisor) AfterHistoricalBid(ctx context.Context, actor domain.AccountID,
	id domain.HandID, result application.LiveHandCommandResult,
) error {
	if !result.Historical {
		return fmt.Errorf("historical bid handler requires a verified receipt")
	}
	var bid livehand.BidResult
	if err := decodeStrict(result.Payload, &bid); err != nil {
		return fmt.Errorf("decode historical bid: %w", err)
	}
	if bid.HandID != string(id) || bid.StateVersion != result.Version ||
		bid.RequiresTermination != result.RequiresTermination {
		return fmt.Errorf("historical bid receipt identity mismatch")
	}
	snapshot, err := s.authorize(ctx, actor, id)
	if err != nil {
		return err
	}
	if snapshot.Phase.Terminal() {
		return nil
	}
	switch bid.Phase {
	case livehand.PhaseBidding:
		// A low bid is already acknowledged. A previous timer may be running,
		// and must never be extended by historical reads.
		return nil
	case livehand.PhasePlaying:
		if snapshot.Phase != domain.HandBidding {
			return nil // already advanced; do not rearm or downgrade timeout
		}
		if err := s.advanceLiveHand(ctx, id, false); err != nil {
			return err
		}
		// Re-read authority after advancement. A competing completion may have
		// happened before we got here; do not arm a stale bidding timeout.
		current, err := s.hands.LoadHand(ctx, id)
		if err != nil {
			return err
		}
		if current.Phase == domain.HandPlaying {
			s.scheduleIfPhaseMissing(id, actor, phasePlaying, s.config.PlayingTimeout, result.Version)
		}
		return nil
	case livehand.PhaseNoLandlord:
		if !bid.RequiresTermination || snapshot.Phase != domain.HandBidding {
			return nil
		}
		_, err := s.terminate(ctx, actor, id, terminationAbort, ReasonNoLandlord)
		return err
	default:
		return fmt.Errorf("invalid historical bid phase %q", bid.Phase)
	}
}

// AfterHistoricalPlay runs only the missing domain completion transition.
// Previous passing/playing actions must not update the playing deadline.
func (s *Supervisor) AfterHistoricalPlay(ctx context.Context, actor domain.AccountID,
	id domain.HandID, result application.LiveHandCommandResult,
) error {
	if !result.Historical {
		return fmt.Errorf("historical play handler requires a verified receipt")
	}
	var play livehand.PlayResult
	if err := decodeStrict(result.Payload, &play); err != nil {
		return fmt.Errorf("decode historical play: %w", err)
	}
	if play.HandID != string(id) || play.StateVersion != result.Version {
		return fmt.Errorf("historical play receipt identity mismatch")
	}
	snapshot, err := s.authorize(ctx, actor, id)
	if err != nil {
		return err
	}
	if snapshot.Phase.Terminal() {
		return nil
	}
	switch play.Phase {
	case livehand.PhasePlaying:
		if snapshot.Phase == domain.HandBidding {
			if err := s.advanceLiveHand(ctx, id, false); err != nil {
				return err
			}
			current, err := s.hands.LoadHand(ctx, id)
			if err != nil {
				return err
			}
			if current.Phase == domain.HandPlaying {
				s.scheduleIfPhaseMissing(id, actor, phasePlaying, s.config.PlayingTimeout, result.Version)
			}
		}
		return nil
	case livehand.PhaseCompleted:
		if err := s.advanceLiveHand(ctx, id, true); err != nil {
			return err
		}
		s.Forget(id)
		return nil
	default:
		return fmt.Errorf("invalid historical play phase %q", play.Phase)
	}
}

// Never extend an existing countdown, and never replace a playing countdown
// with an older bidding state. Only fill a genuinely missing phase transition.
func (s *Supervisor) scheduleIfPhaseMissing(id domain.HandID, actor domain.AccountID,
	desired phase, timeout time.Duration, version uint64,
) {
	s.mu.Lock()
	defer s.mu.Unlock()
	current, exists := s.entries[id]
	if current.pending != nil || (exists && (current.liveVersion > version ||
		current.phase == desired || (current.phase == phasePlaying && desired == phaseBidding))) {
		return
	}
	current.actor = actor
	current.phase = desired
	current.liveVersion = version
	current.deadline = s.clock.Now().UTC().Add(timeout)
	s.entries[id] = current
}
