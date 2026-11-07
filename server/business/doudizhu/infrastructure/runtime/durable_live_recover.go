package runtime

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/application"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/livehand"
	"github.com/oublie6/awesome-zero-platform/server/business/gamecore"
)

// RecoverActive fences previous writers in MySQL before replay and accepting
// requests. It refuses any missing/corrupt action rather than regenerating a
// different deal or silently skipping acknowledged live operations.
// Call before registering the realtime/HTTP handlers or launching sweeps.
func (c *LiveHandCoordinator) RecoverActive(ctx context.Context,
	loadHand func(context.Context, domain.HandID) (domain.HandSnapshot, error),
) error {
	if c == nil || c.journal == nil || loadHand == nil {
		return fmt.Errorf("durable recovery dependencies unavailable")
	}
	ids, err := c.journal.List(ctx)
	if err != nil {
		return fmt.Errorf("list live recovery: %w", err)
	}
	for _, id := range ids {
		snapshot, err := loadHand(ctx, id)
		if errors.Is(err, application.ErrNotFound) {
			continue
		} // orphan from rolled-back genesis transaction
		if err != nil {
			return fmt.Errorf("load authoritative hand %s: %w", id, err)
		}
		if !recoveryPhaseNeedsReplay(snapshot.Phase) && !snapshot.Phase.Terminal() {
			// Terminal hands are owned by durable lifecycle records; never
			// resurrect a finished game as an active executor.
			continue
		}
		epoch, watermark, genesis, err := c.journal.Acquire(ctx, id)
		if err != nil {
			return fmt.Errorf("take fenced live ownership %s: %w", id, err)
		}
		// The domain hand is authoritative for immutable actor/seat and fair-deal
		// identity. A valid-looking but tampered genesis cannot reassign
		// private cards while retaining the same journal/seed.
		if !sameImmutableRecoveryHand(snapshot, genesis) {
			return fmt.Errorf("%w: genesis/authoritative hand identity mismatch for %s", gamecore.ErrVerificationFailed, id)
		}
		material, err := c.material(genesis)
		if err != nil {
			return fmt.Errorf("recover protected fairness material %s: %w", id, err)
		}
		artifact, err := c.module.GenerateSetup(material)
		if err != nil {
			return fmt.Errorf("regenerate committed deal %s: %w", id, err)
		}
		if err := c.module.VerifySetup(material, artifact); err != nil {
			return fmt.Errorf("verify recovered setup %s: %w", id, err)
		}
		game, err := livehand.New(genesis, material, artifact)
		if err != nil {
			return fmt.Errorf("restore live genesis %s: %w", id, err)
		}
		actions, err := c.journal.Read(ctx, id)
		if err != nil {
			return fmt.Errorf("load protected live log %s: %w", id, err)
		}
		if watermark != uint64(len(actions))+1 {
			return fmt.Errorf("%w: live watermark disagrees with replay log for %s", gamecore.ErrVerificationFailed, id)
		}
		// Verify the authenticated checkpoint and every matching database
		// action before restoring its result index. All later actions are then
		// replayed from the same committed version. Missing checkpoint falls
		// back to genesis replay; corrupt checkpoints fail closed.
		checkpoint, err := c.journal.LoadCheckpoint(ctx, id)
		if err != nil {
			return fmt.Errorf("load protected live checkpoint %s: %w", id, err)
		}
		if checkpoint.Version > watermark {
			return fmt.Errorf("%w: checkpoint exceeds watermark for %s", gamecore.ErrVerificationFailed, id)
		}
		var finalOutcome gamecore.CommandOutcome
		// Publish receipts only after the entire recovered hand validates.
		recoveredReceipts := make(map[string]liveReceipt, len(actions))
		if checkpoint.Version != 0 {
			recoveredReceipts, err = restoreRecoveryCheckpoint(game, checkpoint, actions)
			if err != nil {
				return fmt.Errorf("restore protected checkpoint %s: %w", id, err)
			}
		}
		for _, item := range actions[int(checkpoint.Version)-checkpointStartOffset(checkpoint.Version):] {
			position, err := game.PositionForAccount(item.Actor)
			if err != nil || position != item.ActorPosition {
				return fmt.Errorf("%w: forged or changed live actor for %s", gamecore.ErrVerificationFailed, id)
			}
			outcome, err := game.Apply(gamecore.Command{ActorPosition: item.ActorPosition,
				ExpectedVersion: item.ExpectedVersion, Payload: append([]byte(nil), item.Payload...)})
			if err != nil {
				return fmt.Errorf("replay live action %s/%d: %w", id, item.Version, err)
			}
			raw, err := json.Marshal(outcome)
			if err != nil {
				return err
			}
			digest := sha256.Sum256(raw)
			if outcome.Version != item.Version || digest != item.ResultDigest {
				return fmt.Errorf("%w: divergent live result for %s/%d", gamecore.ErrVerificationFailed, id, item.Version)
			}
			if _, exists := recoveredReceipts[item.CommandID]; exists {
				return fmt.Errorf("%w: duplicate live command identity for %s", gamecore.ErrVerificationFailed, id)
			}
			recoveredReceipts[item.CommandID] = liveReceipt{actor: item.Actor, clientSeq: item.ClientSeq,
				command: gamecore.Command{ActorPosition: item.ActorPosition, ExpectedVersion: item.ExpectedVersion,
					Payload: append([]byte(nil), item.Payload...)},
				outcome: cloneLiveOutcome(outcome)}
			if outcome.Terminal {
				finalOutcome = outcome
			}
		}
		if err := validateReplayedPhase(snapshot.Phase, game.Phase(), finalOutcome.Terminal); err != nil {
			return fmt.Errorf("replay phase mismatch for %s: %w", id, err)
		}
		// EXPIRED/ABORTED/CANCELLED may have accepted live actions but no
		// terminal live action. A crash between the domain COMMIT and the
		// runtime archive must be compensated before serving clients.
		if snapshot.Phase.Terminal() && snapshot.Phase != domain.HandCompleted {
			if err := c.directory.Add(game.Descriptor(), game); err != nil {
				return fmt.Errorf("register terminal recovery %s: %w", id, err)
			}
			record, err := c.directory.Abort(gamecore.InstanceID(id), snapshot.TerminationReason)
			if err != nil {
				return fmt.Errorf("reconcile terminated archive %s: %w", id, err)
			}
			c.finish(id, record)
			c.mu.Lock()
			c.receipts[id] = recoveredReceipts
			c.mu.Unlock()
			continue
		}
		if finalOutcome.Terminal {
			record, err := c.directory.ArchiveRecoveredFinal(game, finalOutcome)
			if err != nil {
				return fmt.Errorf("archive recovered game completion %s: %w", id, err)
			}
			c.finish(id, record)
			c.mu.Lock()
			c.receipts[id] = recoveredReceipts
			c.mu.Unlock()
			continue
		}
		if err := c.directory.Add(game.Descriptor(), game); err != nil {
			return err
		}
		c.mu.Lock()
		c.games[id] = game
		c.epochs[id] = epoch
		c.receipts[id] = recoveredReceipts
		c.mu.Unlock()
	}
	return nil
}

// All immutable deal inputs and each of the three seat/account bindings must
// match the durable domain snapshot. Phase, version and termination status are
// intentionally excluded because lifecycle advances them after the genesis.
func sameImmutableRecoveryHand(authoritative, genesis domain.HandSnapshot) bool {
	return authoritative.ID == genesis.ID &&
        authoritative.FairnessMode == genesis.FairnessMode &&
		authoritative.RoomID == genesis.RoomID &&
		reflect.DeepEqual(authoritative.Seats, genesis.Seats) &&
		authoritative.ServerCommitment == genesis.ServerCommitment &&
		authoritative.RevealKeyID == genesis.RevealKeyID &&
		authoritative.RevealPublicKeySHA256 == genesis.RevealPublicKeySHA256 &&
		authoritative.RevealKeyBoundAt.Equal(genesis.RevealKeyBoundAt) &&
		reflect.DeepEqual(authoritative.BeaconPlan, genesis.BeaconPlan) &&
		reflect.DeepEqual(authoritative.Beacon, genesis.Beacon) &&
		reflect.DeepEqual(authoritative.Contributions, genesis.Contributions)
}

// Bidding, playing and settling are all crash-recoverable stages. Excluding
// PLAYING used to silently abandon already acknowledged card actions.
func recoveryPhaseNeedsReplay(phase domain.HandPhase) bool {
	switch phase {
	case domain.HandBidding, domain.HandPlaying, domain.HandSettling:
		return true
	default:
		return false
	}
}

func validateReplayedPhase(persisted domain.HandPhase, replayed string, terminal bool) error {
	if persisted.Terminal() && persisted != domain.HandCompleted {
		if !terminal && (replayed == livehand.PhaseBidding ||
			replayed == livehand.PhasePlaying || replayed == livehand.PhaseNoLandlord) {
			return nil
		}
		return gamecore.ErrVerificationFailed
	}
	if terminal {
		if replayed != livehand.PhaseCompleted || (!recoveryPhaseNeedsReplay(persisted) && persisted != domain.HandCompleted) {
			return gamecore.ErrVerificationFailed
		}
		return nil
	}
	switch persisted {
	case domain.HandBidding:
		// Domain phase transitions are distinct from live actions; a committed
		// last bid can put the replayed hand into PLAYING before the outbox's
		// matching lifecycle command has advanced the domain snapshot.
		if replayed == livehand.PhaseBidding || replayed == livehand.PhasePlaying || replayed == livehand.PhaseNoLandlord {
			return nil
		}
	case domain.HandPlaying:
		if replayed == livehand.PhasePlaying {
			return nil
		}
	}
	// An unfinished live hand must never be exposed under SETTLING or an
	// incompatible domain phase: a repair/reconciliation is needed instead.
	return gamecore.ErrVerificationFailed
}

// RecoveredPhases is an immutable phase index for the lifecycle supervisor.
// All entries here have passed journal consistency and owner fencing.
func (c *LiveHandCoordinator) RecoveredPhases() map[domain.HandID]string {
	out := make(map[domain.HandID]string)
	if c == nil {
		return out
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	for id, game := range c.games {
		out[id] = game.Phase()
	}
	for id, record := range c.finalized {
		if record.Status() == gamecore.FinalStatusCompleted {
			out[id] = livehand.PhaseCompleted
		}
	}
	return out
}

func checkpointStartOffset(version uint64) int {
	if version == 0 {
		return 0
	}
	return 1
}
