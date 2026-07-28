package runtime

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/application"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/livehand"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/randomizedsetup"
	"github.com/oublie6/awesome-zero-platform/server/business/gamecore"
)

type LiveHandCoordinator struct {
	mu        sync.RWMutex
	seeds     *SeedVault
	directory *gamecore.LiveDirectory
	module    randomizedsetup.Module
	games     map[domain.HandID]*livehand.Game
	finalized map[domain.HandID]gamecore.FinalRecord
}

func NewLiveHandCoordinator(seeds *SeedVault, directory *gamecore.LiveDirectory) (*LiveHandCoordinator, error) {
	if seeds == nil || directory == nil {
		return nil, fmt.Errorf("%w: live-hand coordinator dependencies", domain.ErrInvalidArgument)
	}
	return &LiveHandCoordinator{
		seeds: seeds, directory: directory, module: randomizedsetup.NewModule(),
		games: make(map[domain.HandID]*livehand.Game), finalized: make(map[domain.HandID]gamecore.FinalRecord),
	}, nil
}

func (c *LiveHandCoordinator) Start(ctx context.Context, snapshot domain.HandSnapshot) error {
	if c == nil {
		return fmt.Errorf("%w: nil live-hand coordinator", domain.ErrInvalidArgument)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	material, err := c.material(snapshot)
	if err != nil {
		return err
	}
	artifact, err := c.module.GenerateSetup(material)
	if err != nil {
		return fmt.Errorf("generate Doudizhu setup: %w", err)
	}
	if err := c.module.VerifySetup(material, artifact); err != nil {
		return fmt.Errorf("verify Doudizhu setup: %w", err)
	}
	game, err := livehand.New(snapshot, material, artifact)
	if err != nil {
		return fmt.Errorf("construct Doudizhu live hand: %w", err)
	}
	if err := c.directory.Add(randomizedsetup.Descriptor(), game); err != nil {
		return err
	}
	c.mu.Lock()
	if _, exists := c.games[snapshot.ID]; exists {
		c.mu.Unlock()
		_ = c.directory.RemoveForCompensation(gamecore.InstanceID(snapshot.ID))
		return fmt.Errorf("%w: duplicate Doudizhu live hand %s", gamecore.ErrInstanceExists, snapshot.ID)
	}
	c.games[snapshot.ID] = game
	c.mu.Unlock()
	return nil
}

func (c *LiveHandCoordinator) RollbackStart(ctx context.Context, handID domain.HandID) error {
	if c == nil {
		return fmt.Errorf("%w: nil live-hand coordinator", domain.ErrInvalidArgument)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	err := c.directory.RemoveForCompensation(gamecore.InstanceID(handID))
	c.mu.Lock()
	delete(c.games, handID)
	c.mu.Unlock()
	return err
}

func (c *LiveHandCoordinator) ReleasePrepared(ctx context.Context, handID domain.HandID) error {
	if c == nil {
		return fmt.Errorf("%w: nil live-hand coordinator", domain.ErrInvalidArgument)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if c.Contains(handID) {
		return fmt.Errorf("%w: live hand %s still active", domain.ErrHandActive, handID)
	}
	if !c.seeds.Contains(handID) {
		return nil
	}
	return c.seeds.Release(handID)
}

func (c *LiveHandCoordinator) PublicView(ctx context.Context, handID domain.HandID, actor domain.AccountID) (application.LiveHandView, error) {
	game, err := c.game(ctx, handID)
	if err != nil {
		return application.LiveHandView{}, err
	}
	if _, err := game.PositionForAccount(actor); err != nil {
		return application.LiveHandView{}, err
	}
	view, err := c.directory.View(gamecore.InstanceID(handID), gamecore.ViewRequest{PublicOnly: true})
	if err != nil {
		return application.LiveHandView{}, err
	}
	return application.LiveHandView{Version: view.Version, Payload: append([]byte(nil), view.Payload...)}, nil
}

func (c *LiveHandCoordinator) PrivateView(ctx context.Context, handID domain.HandID, actor domain.AccountID) (application.LiveHandView, error) {
	game, err := c.game(ctx, handID)
	if err != nil {
		return application.LiveHandView{}, err
	}
	position, err := game.PositionForAccount(actor)
	if err != nil {
		return application.LiveHandView{}, err
	}
	view, err := c.directory.View(gamecore.InstanceID(handID), gamecore.ViewRequest{ViewerPosition: position})
	if err != nil {
		return application.LiveHandView{}, err
	}
	return application.LiveHandView{Version: view.Version, Payload: append([]byte(nil), view.Payload...)}, nil
}

func (c *LiveHandCoordinator) Abort(ctx context.Context, handID domain.HandID, reason string) (gamecore.FinalRecord, error) {
	if c == nil {
		return gamecore.FinalRecord{}, fmt.Errorf("%w: nil live-hand coordinator", domain.ErrInvalidArgument)
	}
	if err := ctx.Err(); err != nil {
		return gamecore.FinalRecord{}, err
	}
	c.mu.RLock()
	if record, ok := c.finalized[handID]; ok {
		c.mu.RUnlock()
		return record, nil
	}
	c.mu.RUnlock()
	pending, ok, pendingErr := c.directory.PendingFinalRecord(gamecore.InstanceID(handID))
	if pendingErr != nil && !errors.Is(pendingErr, gamecore.ErrInstanceNotFound) {
		return gamecore.FinalRecord{}, pendingErr
	}
	if ok {
		record, retryErr := c.directory.RetryArchive(gamecore.InstanceID(handID))
		if retryErr != nil {
			return pending, retryErr
		}
		c.finish(handID, record)
		return record, nil
	}
	record, err := c.directory.Abort(gamecore.InstanceID(handID), reason)
	if err != nil {
		return record, err
	}
	c.finish(handID, record)
	return record, nil
}

func (c *LiveHandCoordinator) RetryArchive(ctx context.Context, handID domain.HandID) (gamecore.FinalRecord, error) {
	if c == nil {
		return gamecore.FinalRecord{}, fmt.Errorf("%w: nil live-hand coordinator", domain.ErrInvalidArgument)
	}
	if err := ctx.Err(); err != nil {
		return gamecore.FinalRecord{}, err
	}
	c.mu.RLock()
	if record, ok := c.finalized[handID]; ok {
		c.mu.RUnlock()
		return record, nil
	}
	c.mu.RUnlock()
	record, err := c.directory.RetryArchive(gamecore.InstanceID(handID))
	if err != nil {
		return record, err
	}
	c.finish(handID, record)
	return record, nil
}

func (c *LiveHandCoordinator) Contains(handID domain.HandID) bool {
	if c == nil {
		return false
	}
	c.mu.RLock()
	_, ok := c.games[handID]
	c.mu.RUnlock()
	return ok
}

func (c *LiveHandCoordinator) material(snapshot domain.HandSnapshot) (gamecore.FairnessMaterial, error) {
	if snapshot.Phase != domain.HandDealing {
		return gamecore.FairnessMaterial{}, domain.ErrWrongPhase
	}
	if snapshot.Beacon == nil {
		return gamecore.FairnessMaterial{}, fmt.Errorf("%w: public beacon is not locked", domain.ErrBeaconMismatch)
	}
	seed, err := c.seeds.Read(snapshot.ID)
	if err != nil {
		return gamecore.FairnessMaterial{}, err
	}
	material := gamecore.FairnessMaterial{
		Descriptor:       randomizedsetup.Descriptor(),
		InstanceID:       gamecore.InstanceID(snapshot.ID),
		ServerSeed:       gamecore.Seed(seed),
		ServerCommitment: gamecore.Digest(snapshot.ServerCommitment),
		Participants:     make([]gamecore.ParticipantFairness, len(snapshot.Contributions)),
		Beacon: gamecore.BeaconEvidence{
			Provider: snapshot.Beacon.Provider,
			Round:    snapshot.Beacon.Round,
			Digest:   gamecore.Digest(snapshot.Beacon.Digest),
			ProofRef: snapshot.Beacon.ProofRef,
		},
		RevealKey: gamecore.RevealKeyAudit{
			KeyID:           snapshot.RevealKeyID,
			PublicKeySHA256: gamecore.Digest(snapshot.RevealPublicKeySHA256),
		},
	}
	for index, contribution := range snapshot.Contributions {
		if contribution.Seat != domain.Seat(index+1) || !contribution.Committed || !contribution.Revealed || contribution.RecordRef == "" {
			return gamecore.FairnessMaterial{}, fmt.Errorf("%w: incomplete contribution for seat %d", domain.ErrWrongPhase, index+1)
		}
		material.Participants[index] = gamecore.ParticipantFairness{
			Position:     uint8(contribution.Seat),
			Contribution: gamecore.Digest(contribution.Digest),
			Commitment:   gamecore.Digest(contribution.Commitment),
		}
	}
	if err := material.Validate(); err != nil {
		return gamecore.FairnessMaterial{}, err
	}
	return material, nil
}

func (c *LiveHandCoordinator) game(ctx context.Context, handID domain.HandID) (*livehand.Game, error) {
	if c == nil {
		return nil, fmt.Errorf("%w: nil live-hand coordinator", domain.ErrInvalidArgument)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	c.mu.RLock()
	game := c.games[handID]
	c.mu.RUnlock()
	if game == nil {
		return nil, fmt.Errorf("%w: %s", gamecore.ErrInstanceNotFound, handID)
	}
	return game, nil
}

func (c *LiveHandCoordinator) finish(handID domain.HandID, record gamecore.FinalRecord) {
	c.mu.Lock()
	delete(c.games, handID)
	c.finalized[handID] = record
	c.mu.Unlock()
	if err := c.seeds.Release(handID); err != nil && !errors.Is(err, ErrSeedNotFound) {
		return
	}
}
