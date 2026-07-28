package protection

import (
	"context"
	"fmt"
	"sync"
)

// StaticKeyring is intended for tests and explicitly configured development.
// Production composition should replace it with a KMS-backed provider.
type StaticKeyring struct {
	mu sync.RWMutex
	current string
	keys map[string][]byte
}

func NewStaticKeyring(current string, keys map[string][]byte) (*StaticKeyring, error) {
	if current == "" || len(keys) == 0 { return nil, fmt.Errorf("%w: static keyring configuration", ErrKeyUnavailable) }
	copied := make(map[string][]byte, len(keys))
	for id, key := range keys {
		if id == "" || len(key) != 32 { return nil, fmt.Errorf("%w: key %q", ErrKeyUnavailable, id) }
		copied[id] = append([]byte(nil), key...)
	}
	if _, ok := copied[current]; !ok { return nil, fmt.Errorf("%w: current key", ErrKeyUnavailable) }
	return &StaticKeyring{current: current, keys: copied}, nil
}

func (k *StaticKeyring) CurrentKey(ctx context.Context) (string, []byte, error) {
	if err := ctx.Err(); err != nil { return "", nil, err }
	k.mu.RLock(); defer k.mu.RUnlock()
	key, ok := k.keys[k.current]; if !ok { return "", nil, ErrKeyUnavailable }
	return k.current, append([]byte(nil), key...), nil
}
func (k *StaticKeyring) Key(ctx context.Context, id string) ([]byte, error) {
	if err := ctx.Err(); err != nil { return nil, err }
	k.mu.RLock(); defer k.mu.RUnlock()
	key, ok := k.keys[id]; if !ok { return nil, ErrKeyUnavailable }
	return append([]byte(nil), key...), nil
}
