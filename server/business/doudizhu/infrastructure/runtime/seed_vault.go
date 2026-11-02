package runtime

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/carddeck"
)

var (
	ErrSeedExists   = errors.New("doudizhu runtime: server seed already exists")
	ErrSeedNotFound = errors.New("doudizhu runtime: server seed not found")
)

type PreparedSeed struct {
	HandID     domain.HandID
	Commitment domain.ServerCommitment
}

type SeedPersistence interface {
	Save(context.Context, domain.HandID, carddeck.Seed) error
	Load(context.Context, domain.HandID) (carddeck.Seed, error)
	Delete(context.Context, domain.HandID) error
}

// Persist seeds before the preparation command can be acknowledged; the
// persistence adapter owns encryption and integrity checks.
type SeedVault struct {
	mu         sync.RWMutex
	entropy    io.Reader
	seeds      map[domain.HandID]carddeck.Seed
	persistent SeedPersistence
}

func NewSeedVault(entropy io.Reader) (*SeedVault, error) {
	if entropy == nil {
		entropy = rand.Reader
	}
	return &SeedVault{entropy: entropy, seeds: make(map[domain.HandID]carddeck.Seed)}, nil
}

func NewPersistentSeedVault(entropy io.Reader, store SeedPersistence) (*SeedVault, error) {
	if store == nil {
		return nil, fmt.Errorf("%w: missing seed persistence", domain.ErrInvalidArgument)
	}
	v, err := NewSeedVault(entropy)
	if err != nil {
		return nil, err
	}
	v.persistent = store
	return v, nil
}

func (v *SeedVault) storageContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 5*time.Second)
}

func (v *SeedVault) Prepare(handID domain.HandID) (PreparedSeed, error) {
	if v == nil {
		return PreparedSeed{}, fmt.Errorf("%w: nil seed vault", domain.ErrInvalidArgument)
	}
	if strings.TrimSpace(string(handID)) == "" || len(handID) > 128 {
		return PreparedSeed{}, fmt.Errorf("%w: invalid hand ID", domain.ErrInvalidArgument)
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	if _, exists := v.seeds[handID]; exists {
		return PreparedSeed{}, fmt.Errorf("%w: %s", ErrSeedExists, handID)
	}
	var seed carddeck.Seed
	if _, err := io.ReadFull(v.entropy, seed[:]); err != nil {
		clear(seed[:])
		return PreparedSeed{}, fmt.Errorf("read server-seed entropy: %w", err)
	}
	commitment, err := carddeck.ComputeServerCommitment(string(handID), seed)
	if err != nil {
		clear(seed[:])
		return PreparedSeed{}, err
	}
	if v.persistent != nil {
		ctx, cancel := v.storageContext()
		err := v.persistent.Save(ctx, handID, seed)
		cancel()
		if err != nil {
			clear(seed[:])
			return PreparedSeed{}, fmt.Errorf("persist protected server seed: %w", err)
		}
	}
	v.seeds[handID] = seed
	clear(seed[:])
	return PreparedSeed{HandID: handID, Commitment: domain.ServerCommitment(commitment)}, nil
}

func (v *SeedVault) Read(handID domain.HandID) (carddeck.Seed, error) {
	if v == nil {
		return carddeck.Seed{}, fmt.Errorf("%w: nil seed vault", domain.ErrInvalidArgument)
	}
	v.mu.RLock()
	seed, ok := v.seeds[handID]
	v.mu.RUnlock()
	if !ok {
		if v.persistent == nil {
			return carddeck.Seed{}, fmt.Errorf("%w: %s", ErrSeedNotFound, handID)
		}
		ctx, cancel := v.storageContext()
		defer cancel()
		return v.persistent.Load(ctx, handID)
	}
	return seed, nil
}

func (v *SeedVault) Release(handID domain.HandID) error {
	if v == nil {
		return fmt.Errorf("%w: nil seed vault", domain.ErrInvalidArgument)
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	seed, ok := v.seeds[handID]
	if !ok && v.persistent == nil {
		return fmt.Errorf("%w: %s", ErrSeedNotFound, handID)
	}
	if v.persistent != nil {
		ctx, cancel := v.storageContext()
		err := v.persistent.Delete(ctx, handID)
		cancel()
		if err != nil {
			return fmt.Errorf("remove protected server seed: %w", err)
		}
	}
	clear(seed[:])
	v.seeds[handID] = seed
	delete(v.seeds, handID)
	return nil
}

func (v *SeedVault) Contains(handID domain.HandID) bool {
	if v == nil {
		return false
	}
	v.mu.RLock()
	_, ok := v.seeds[handID]
	v.mu.RUnlock()
	if ok {
		return true
	}
	if v.persistent == nil {
		return false
	}
	_, err := v.Read(handID)
	return err == nil
}

func (v *SeedVault) Count() int {
	if v == nil {
		return 0
	}
	v.mu.RLock()
	defer v.mu.RUnlock()
	return len(v.seeds)
}
