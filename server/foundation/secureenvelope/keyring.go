package secureenvelope

import (
	"context"
	"fmt"
	"strings"
)

// PrivateKeyProvider returns a raw 32-byte X25519 private key for a key ID.
// Implementations must return a caller-owned copy so it can be cleared after use.
type PrivateKeyProvider interface {
	PrivateKey(ctx context.Context, keyID string) ([]byte, error)
}

// StaticKeyring is an immutable in-memory key provider intended for tests,
// bootstrap wiring, and small deployments. Production deployments can provide
// a KMS-backed implementation through PrivateKeyProvider.
type StaticKeyring struct {
	keys map[string][]byte
}

// NewStaticKeyring validates and defensively copies all private keys.
func NewStaticKeyring(keys map[string][]byte) (*StaticKeyring, error) {
	copied := make(map[string][]byte, len(keys))
	for keyID, key := range keys {
		if err := validateKeyID(keyID); err != nil {
			return nil, fmt.Errorf("%w: key id", ErrInvalidEnvelope)
		}
		if len(key) != X25519KeySize {
			return nil, fmt.Errorf("%w: private key length", ErrInvalidEnvelope)
		}
		copied[keyID] = append([]byte(nil), key...)
	}
	return &StaticKeyring{keys: copied}, nil
}

// PrivateKey returns a caller-owned copy of the private key.
func (k *StaticKeyring) PrivateKey(ctx context.Context, keyID string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	key, ok := k.keys[keyID]
	if !ok {
		return nil, ErrKeyUnavailable
	}
	return append([]byte(nil), key...), nil
}

func validateKeyID(keyID string) error {
	if keyID == "" || len(keyID) > MaxKeyIDLength || strings.TrimSpace(keyID) != keyID {
		return ErrInvalidEnvelope
	}
	for _, r := range keyID {
		if r < 0x21 || r > 0x7e {
			return ErrInvalidEnvelope
		}
	}
	return nil
}
