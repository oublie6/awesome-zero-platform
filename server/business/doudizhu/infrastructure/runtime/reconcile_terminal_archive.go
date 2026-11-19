package runtime

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/livehand"
	"github.com/oublie6/awesome-zero-platform/server/business/gamecore"
)

// ReconcileTerminalArchive resolves any unknown MySQL COMMIT result from the
// writer's authenticated log, NEVER from a quarantined in-memory game.
// This is an internal lifecycle call after a committed terminal live turn or
// an accepted domain cancel. It is not an unauthenticated public game API.
// A terminal cursor can be retired only after this AND the domain transition.
func (c *LiveHandCoordinator) ReconcileTerminalArchive(
	ctx context.Context, handID domain.HandID, phase, reason string,
) error {
	if c == nil || c.journal == nil || handID == "" {
		return fmt.Errorf("terminal replay requires durable live journal")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	// Serialize against local commands. Domain terminal transitions already
	// fenced the writer under their own domain+writer transaction: re-acquiring
	// there would unnecessarily increment epoch twice (and break retry
	// semantics). Before domain completion, a committed terminal live action
	// still needs a fresh epoch fence for unacknowledged old executors.
	stripe := c.commandStripe(handID)
	stripe.Lock()
	defer stripe.Unlock()
	snapshot, genesis, actions, err := c.journal.ReadRecoveryHistory(ctx, handID)
	if err != nil {
		return fmt.Errorf("read terminal recovery history: %w", err)
	}
	if !snapshot.Phase.Terminal() {
		if _, _, _, err := c.journal.Acquire(ctx, handID); err != nil {
			return fmt.Errorf("fence active terminal live journal: %w", err)
		}
		snapshot, genesis, actions, err = c.journal.ReadRecoveryHistory(ctx, handID)
		if err != nil {
			return fmt.Errorf("reread fenced terminal history: %w", err)
		}
	}
	if !sameImmutableRecoveryHand(snapshot, genesis) {
		return gamecore.ErrVerificationFailed
	}
	material, err := c.material(genesis)
	if err != nil {
		return err
	}
	artifact, err := c.module.GenerateSetup(material)
	if err != nil {
		return err
	}
	if err := c.module.VerifySetup(material, artifact); err != nil {
		return err
	}
	game, err := livehand.New(genesis, material, artifact)
	if err != nil {
		return err
	}
	var final gamecore.CommandOutcome
	for _, item := range actions {
		position, err := game.PositionForAccount(item.Actor)
		if err != nil || position != item.ActorPosition {
			return fmt.Errorf("%w: terminal actor/seat mismatch", gamecore.ErrVerificationFailed)
		}
		outcome, err := game.Apply(gamecore.Command{
			ActorPosition: item.ActorPosition,
			ExpectedVersion: item.ExpectedVersion,
			Payload: append([]byte(nil), item.Payload...),
		})
		if err != nil {
			return fmt.Errorf("replay terminal command v%d: %w", item.Version, err)
		}
		raw, err := json.Marshal(outcome)
		if err != nil {
			return err
		}
		if outcome.Version != item.Version || sha256.Sum256(raw) != item.ResultDigest {
			return fmt.Errorf("%w: terminal journal result mismatch at v%d", gamecore.ErrVerificationFailed, item.Version)
		}
		if outcome.Terminal {
			final = outcome
		}
	}
	var record gamecore.FinalRecord
	switch phase {
	case livehand.PhaseCompleted:
		if !final.Terminal || game.Phase() != livehand.PhaseCompleted ||
			len(final.FinalPayload) == 0 || (snapshot.Phase.Terminal() && snapshot.Phase != domain.HandCompleted) {
			return fmt.Errorf("%w: inconsistent completed lifecycle", gamecore.ErrVerificationFailed)
		}
		record, err = gamecore.NewFinalRecord(gamecore.InstanceID(handID), game.Descriptor(),
			gamecore.FinalStatusCompleted, final.Version, final.FinalPayload)
	case livehand.PhaseNoLandlord:
		if final.Terminal || game.Phase() != livehand.PhaseNoLandlord ||
			reason != "no_landlord" ||
			(snapshot.Phase.Terminal() && snapshot.TerminationReason != reason) {
			return fmt.Errorf("%w: inconsistent no-landlord lifecycle", gamecore.ErrVerificationFailed)
		}
		record, err = terminalAbortRecord(game, reason)
	case "TERMINATED", livehand.PhaseBidding, livehand.PhasePlaying:
		if !snapshot.Phase.Terminal() || snapshot.Phase == domain.HandCompleted ||
			snapshot.TerminationReason == "" || reason != snapshot.TerminationReason ||
			final.Terminal || (phase != "TERMINATED" && phase != game.Phase()) {
			return fmt.Errorf("%w: inconsistent cancelled/expired lifecycle", gamecore.ErrVerificationFailed)
		}
		record, err = terminalAbortRecord(game, reason)
	default:
		return fmt.Errorf("invalid terminal repair phase %q", phase)
	}
	if err != nil {
		return err
	}
	if err := c.directory.ArchiveVerifiedRecord(record); err != nil {
		return err
	}
	c.finish(handID, record)
	return nil
}

func terminalAbortRecord(game *livehand.Game, reason string) (gamecore.FinalRecord, error) {
	aborted, err := game.Abort(reason)
	if err != nil {
		return gamecore.FinalRecord{}, err
	}
	return gamecore.NewFinalRecord(game.InstanceID(), game.Descriptor(),
		gamecore.FinalStatusAborted, aborted.Version, aborted.FinalPayload)
}
