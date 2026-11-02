package lifecycle

import (
	"context"
	"fmt"
	"sort"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/livehand"
)

// ReconcileRecovered restores deadline supervision *after* MySQL has fenced
// previous writers and verified each acknowledged action's result digest.
// It also durably finishes a completion whose final action committed just
// before the server crashed. Fail startup rather than serving inconsistent
// partially-recovered games.
func (s *Supervisor) ReconcileRecovered(ctx context.Context, states map[domain.HandID]string) error {
	if s == nil {
		return fmt.Errorf("missing lifecycle supervisor")
	}
	ids := make([]string, 0, len(states))
	for id := range states {
		ids = append(ids, string(id))
	}
	sort.Strings(ids)
	for _, value := range ids {
		if err := ctx.Err(); err != nil {
			return err
		}
		id := domain.HandID(value)
		snapshot, err := s.hands.LoadHand(ctx, id)
		if err != nil {
			return fmt.Errorf("read recovered hand %s: %w", id, err)
		}
		if snapshot.Phase.Terminal() {
			continue
		}
		if len(snapshot.Seats) < 1 || snapshot.Seats[0].AccountID == "" {
			return fmt.Errorf("recovered hand %s lacks its authorized seat", id)
		}
		actor := snapshot.Seats[0].AccountID
		switch states[id] {
		case livehand.PhaseBidding:
			if snapshot.Phase != domain.HandBidding {
				return fmt.Errorf("recovered bidding phase mismatch %s", id)
			}
			s.schedule(id, actor, phaseBidding, s.config.BiddingTimeout)
		case livehand.PhasePlaying:
			if err := s.advanceLiveHand(ctx, id, false); err != nil {
				return fmt.Errorf("reconcile playing %s: %w", id, err)
			}
			s.schedule(id, actor, phasePlaying, s.config.PlayingTimeout)
		case livehand.PhaseCompleted:
			if err := s.advanceLiveHand(ctx, id, true); err != nil {
				return fmt.Errorf("reconcile completed %s: %w", id, err)
			}
			s.Forget(id)
		case livehand.PhaseNoLandlord:
			if _, err := s.terminate(ctx, actor, id, terminationAbort, ReasonNoLandlord); err != nil {
				return fmt.Errorf("reconcile no-landlord %s: %w", id, err)
			}
		default:
			return fmt.Errorf("unsupported recovered game phase %q for %s", states[id], id)
		}
	}
	return nil
}
