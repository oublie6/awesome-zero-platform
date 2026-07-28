package revealkeys

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/subtle"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/oublie6/awesome-zero-platform/server/foundation/secureenvelope"
)

type Status string

const (
	StatusActive   Status = "active"
	StatusRetiring Status = "retiring"
	StatusRetired  Status = "retired"
	StatusRevoked  Status = "revoked"
)

func (s Status) Valid() bool {
	switch s {
	case StatusActive, StatusRetiring, StatusRetired, StatusRevoked:
		return true
	default:
		return false
	}
}

type Record struct {
	ManifestVersion uint64
	KeyID           string
	PublicKey       []byte
	PrivateKey      []byte
	NotBefore       time.Time
	ExpiresAt       time.Time
	Status          Status
	RetiringAt      time.Time
	RetireAfter     time.Time
}

type BoundContext struct {
	KeyID           string
	PublicKeySHA256 [sha256.Size]byte
	BoundAt         time.Time
	UseAt           time.Time
}

type Clock func() time.Time

type Registry struct {
	mu           sync.RWMutex
	records      map[string]Record
	manifests    map[string]Manifest
	currentKeyID string
	clock        Clock
}

func New(records []Record, currentKeyID, signatureKeyID string, signingKey ed25519.PrivateKey, clock Clock) (*Registry, error) {
	if len(records) == 0 || validateIdentifier(currentKeyID) != nil || validateIdentifier(signatureKeyID) != nil || len(signingKey) != ed25519.PrivateKeySize {
		return nil, ErrInvalidConfig
	}
	if clock == nil {
		clock = func() time.Time { return time.Now().UTC() }
	}
	copied := make(map[string]Record, len(records))
	manifests := make(map[string]Manifest, len(records))
	versions := make(map[uint64]struct{}, len(records))
	active := 0
	for _, record := range records {
		if err := validateRecord(record); err != nil {
			return nil, err
		}
		if _, exists := copied[record.KeyID]; exists {
			return nil, fmt.Errorf("%w: duplicate key id", ErrInvalidConfig)
		}
		if _, exists := versions[record.ManifestVersion]; exists {
			return nil, fmt.Errorf("%w: duplicate manifest version", ErrInvalidConfig)
		}
		versions[record.ManifestVersion] = struct{}{}
		if record.Status == StatusActive {
			active++
		}
		record.PublicKey = append([]byte(nil), record.PublicKey...)
		record.PrivateKey = append([]byte(nil), record.PrivateKey...)
		manifest, err := SignManifest(record, signatureKeyID, signingKey)
		if err != nil {
			return nil, err
		}
		copied[record.KeyID] = record
		manifests[record.KeyID] = manifest
	}
	if active != 1 {
		return nil, fmt.Errorf("%w: exactly one active key is required", ErrInvalidConfig)
	}
	current, ok := copied[currentKeyID]
	if !ok || current.Status != StatusActive {
		return nil, fmt.Errorf("%w: current key", ErrInvalidConfig)
	}
	for _, record := range copied {
		if record.ManifestVersion > current.ManifestVersion {
			return nil, fmt.Errorf("%w: current manifest version is not the high-water version", ErrInvalidConfig)
		}
	}
	return &Registry{records: copied, manifests: manifests, currentKeyID: currentKeyID, clock: clock}, nil
}

func validateRecord(record Record) error {
	if record.ManifestVersion == 0 || validateIdentifier(record.KeyID) != nil || len(record.PublicKey) != secureenvelope.X25519KeySize {
		return ErrInvalidConfig
	}
	if len(record.PrivateKey) != 0 && len(record.PrivateKey) != secureenvelope.X25519KeySize {
		return ErrInvalidConfig
	}
	if !record.Status.Valid() || record.NotBefore.IsZero() || record.ExpiresAt.IsZero() || !record.ExpiresAt.After(record.NotBefore) {
		return ErrInvalidConfig
	}
	switch record.Status {
	case StatusActive:
		if !record.RetiringAt.IsZero() || !record.RetireAfter.IsZero() {
			return ErrInvalidConfig
		}
	case StatusRetiring:
		if record.RetiringAt.IsZero() || record.RetireAfter.IsZero() || record.RetireAfter.Before(record.RetiringAt) || record.RetireAfter.After(record.ExpiresAt) {
			return ErrInvalidConfig
		}
	case StatusRetired:
		if record.RetiringAt.IsZero() || record.RetireAfter.IsZero() || record.RetireAfter.After(record.ExpiresAt) {
			return ErrInvalidConfig
		}
	case StatusRevoked:
		if !record.RetireAfter.IsZero() && record.RetireAfter.After(record.ExpiresAt) {
			return ErrInvalidConfig
		}
	}
	return nil
}

func (r *Registry) Current(ctx context.Context) (Manifest, error) {
	if err := ctx.Err(); err != nil {
		return Manifest{}, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	record := r.records[r.currentKeyID]
	now := r.clock().UTC()
	if record.Status != StatusActive || now.Before(record.NotBefore) || !now.Before(record.ExpiresAt) {
		return Manifest{}, ErrKeyNotCurrent
	}
	return r.manifests[r.currentKeyID], nil
}

func (r *Registry) Lookup(ctx context.Context, keyID string) (Manifest, error) {
	if err := ctx.Err(); err != nil {
		return Manifest{}, err
	}
	if validateIdentifier(keyID) != nil {
		return Manifest{}, ErrKeyUnavailable
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	record, ok := r.records[keyID]
	if !ok {
		return Manifest{}, ErrKeyUnavailable
	}
	now := r.clock().UTC()
	if now.Before(record.NotBefore) {
		return Manifest{}, ErrKeyUnavailable
	}
	switch record.Status {
	case StatusActive:
		if !now.Before(record.ExpiresAt) {
			return Manifest{}, ErrKeyExpired
		}
	case StatusRetiring:
		if !now.Before(record.RetireAfter) {
			return Manifest{}, ErrKeyExpired
		}
	case StatusRetired, StatusRevoked:
		return Manifest{}, ErrKeyUnavailable
	}
	return r.manifests[keyID], nil
}

func (r *Registry) AuthorizeBound(ctx context.Context, bound BoundContext) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if validateIdentifier(bound.KeyID) != nil || bound.BoundAt.IsZero() || bound.UseAt.IsZero() {
		return ErrKeyUnavailable
	}
	r.mu.RLock()
	record, ok := r.records[bound.KeyID]
	r.mu.RUnlock()
	if !ok {
		return ErrKeyUnavailable
	}
	hash := sha256.Sum256(record.PublicKey)
	if subtle.ConstantTimeCompare(hash[:], bound.PublicKeySHA256[:]) != 1 {
		return ErrKeyHashMismatch
	}
	useAt := bound.UseAt.UTC()
	if bound.BoundAt.UTC().Before(record.NotBefore) || !bound.BoundAt.UTC().Before(record.ExpiresAt) {
		return ErrKeyExpired
	}
	if useAt.Before(record.NotBefore) || !useAt.Before(record.ExpiresAt) {
		return ErrKeyExpired
	}
	switch record.Status {
	case StatusActive:
		return nil
	case StatusRetiring:
		if !bound.BoundAt.UTC().Before(record.RetiringAt) || !useAt.Before(record.RetireAfter) {
			return ErrKeyExpired
		}
		return nil
	case StatusRevoked:
		return ErrKeyRevoked
	default:
		return ErrKeyUnavailable
	}
}

// PrivateKey implements secureenvelope.PrivateKeyProvider. It returns private
// material only while the record can still decrypt a bound active/retiring hand.
func (r *Registry) PrivateKey(ctx context.Context, keyID string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.mu.RLock()
	record, ok := r.records[keyID]
	r.mu.RUnlock()
	if !ok || len(record.PrivateKey) != secureenvelope.X25519KeySize {
		return nil, secureenvelope.ErrKeyUnavailable
	}
	now := r.clock().UTC()
	if now.Before(record.NotBefore) || !now.Before(record.ExpiresAt) {
		return nil, secureenvelope.ErrKeyUnavailable
	}
	if record.Status == StatusRetiring && !now.Before(record.RetireAfter) {
		return nil, secureenvelope.ErrKeyUnavailable
	}
	if record.Status != StatusActive && record.Status != StatusRetiring {
		return nil, secureenvelope.ErrKeyUnavailable
	}
	return append([]byte(nil), record.PrivateKey...), nil
}

func (r *Registry) PublicKeyHash(keyID string) ([sha256.Size]byte, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	record, ok := r.records[keyID]
	if !ok {
		return [sha256.Size]byte{}, ErrKeyUnavailable
	}
	return sha256.Sum256(record.PublicKey), nil
}

func (r *Registry) ManifestVersions() []uint64 {
	r.mu.RLock()
	defer r.mu.RUnlock()
	versions := make([]uint64, 0, len(r.records))
	for _, current := range r.records {
		versions = append(versions, current.ManifestVersion)
	}
	sort.Slice(versions, func(i, j int) bool { return versions[i] < versions[j] })
	return versions
}
