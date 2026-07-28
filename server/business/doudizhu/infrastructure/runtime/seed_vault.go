package runtime

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"

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

type SeedVault struct {
	mu      sync.RWMutex
	entropy io.Reader
	seeds   map[domain.HandID]carddeck.Seed
}

func NewSeedVault(entropy io.Reader) (*SeedVault, error) {
	if entropy == nil {
		entropy = rand.Reader
	}
	return &SeedVault{entropy: entropy, seeds: make(map[domain.HandID]carddeck.Seed)}, nil
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
		return carddeck.Seed{}, fmt.Errorf("%w: %s", ErrSeedNotFound, handID)
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
	if !ok {
		return fmt.Errorf("%w: %s", ErrSeedNotFound, handID)
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
	return ok
}

func (v *SeedVault) Count() int {
	if v == nil {
		return 0
	}
	v.mu.RLock()
	defer v.mu.RUnlock()
	return len(v.seeds)
}
