package runtime

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/application"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
	"github.com/oublie6/awesome-zero-platform/server/foundation/revealkeys"
)

type RevealKeySource interface {
	Current(context.Context) (revealkeys.Manifest, error)
}

type SetupClock interface {
	Now() time.Time
}

type SeededHandSetupProvider struct {
	seeds      *SeedVault
	revealKeys RevealKeySource
	clock      SetupClock
	beaconPlan domain.BeaconPlan
}

func NewSeededHandSetupProvider(seeds *SeedVault, revealKeys RevealKeySource, clock SetupClock, beaconPlan domain.BeaconPlan) (*SeededHandSetupProvider, error) {
	if seeds == nil || revealKeys == nil || clock == nil {
		return nil, fmt.Errorf("%w: hand setup dependencies", domain.ErrInvalidArgument)
	}
	if strings.TrimSpace(beaconPlan.Provider) == "" || len(beaconPlan.Provider) > 128 || strings.TrimSpace(beaconPlan.Round) == "" || len(beaconPlan.Round) > 128 {
		return nil, fmt.Errorf("%w: beacon plan", domain.ErrInvalidArgument)
	}
	return &SeededHandSetupProvider{seeds: seeds, revealKeys: revealKeys, clock: clock, beaconPlan: beaconPlan}, nil
}

func (p *SeededHandSetupProvider) PrepareHand(ctx context.Context, room domain.RoomSnapshot, handID domain.HandID) (application.HandSetup, error) {
	if p == nil {
		return application.HandSetup{}, fmt.Errorf("%w: nil hand setup provider", domain.ErrInvalidArgument)
	}
	if err := ctx.Err(); err != nil {
		return application.HandSetup{}, err
	}
	if room.ID == "" || handID == "" {
		return application.HandSetup{}, fmt.Errorf("%w: room or hand ID", domain.ErrInvalidArgument)
	}
	manifest, err := p.revealKeys.Current(ctx)
	if err != nil {
		return application.HandSetup{}, fmt.Errorf("load current reveal key: %w", err)
	}
	publicKeyDigest, err := base64.RawURLEncoding.DecodeString(manifest.PublicKeySHA256)
	if err != nil || len(publicKeyDigest) != 32 {
		return application.HandSetup{}, fmt.Errorf("%w: reveal public-key digest", domain.ErrInvalidArgument)
	}
	prepared, err := p.seeds.Prepare(handID)
	if err != nil {
		return application.HandSetup{}, err
	}
	var hash domain.RevealPublicKeyHash
	copy(hash[:], publicKeyDigest)
	return application.HandSetup{
		HandID:                handID,
		ServerCommitment:      prepared.Commitment,
		RevealKeyID:           manifest.KeyID,
		RevealPublicKeySHA256: hash,
		RevealKeyBoundAt:      p.clock.Now().UTC(),
		BeaconPlan:            p.beaconPlan,
	}, nil
}

func (p *SeededHandSetupProvider) ReleaseHand(ctx context.Context, handID domain.HandID) error {
	if p == nil {
		return fmt.Errorf("%w: nil hand setup provider", domain.ErrInvalidArgument)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return p.seeds.Release(handID)
}
