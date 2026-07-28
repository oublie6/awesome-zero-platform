package protection

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/application"
)

var (
	ErrKeyUnavailable = errors.New("contribution protection key unavailable")
	ErrInvalidPayload = errors.New("invalid protected contribution payload")
	ErrOpenFailed     = errors.New("open protected contribution failed")
)

type KeyProvider interface {
	CurrentKey(context.Context) (string, []byte, error)
	Key(context.Context, string) ([]byte, error)
}

type Protector struct { keys KeyProvider; random io.Reader; randomMu sync.Mutex }

func New(keys KeyProvider) (*Protector, error) { return NewWithRandom(keys, rand.Reader) }
func NewWithRandom(keys KeyProvider, random io.Reader) (*Protector, error) {
	if keys == nil || random == nil { return nil, fmt.Errorf("%w: protector configuration", application.ErrProtectionFailed) }
	return &Protector{keys: keys, random: random}, nil
}

func (p *Protector) Seal(ctx context.Context, plaintext, aad []byte) (application.ProtectedPayload, error) {
	if err := ctx.Err(); err != nil { return application.ProtectedPayload{}, err }
	if len(plaintext) == 0 || len(aad) == 0 { return application.ProtectedPayload{}, ErrInvalidPayload }
	keyID, key, err := p.keys.CurrentKey(ctx); if err != nil { return application.ProtectedPayload{}, err }
	defer clearBytes(key)
	if keyID == "" || len(key) != 32 { return application.ProtectedPayload{}, ErrKeyUnavailable }
	block, err := aes.NewCipher(key); if err != nil { return application.ProtectedPayload{}, ErrKeyUnavailable }
	gcm, err := cipher.NewGCM(block); if err != nil { return application.ProtectedPayload{}, fmt.Errorf("create contribution AEAD: %w", err) }
	nonce := make([]byte, gcm.NonceSize())
	p.randomMu.Lock(); _, randomErr := io.ReadFull(p.random, nonce); p.randomMu.Unlock()
	if randomErr != nil { clearBytes(nonce); return application.ProtectedPayload{}, fmt.Errorf("read contribution nonce: %w", randomErr) }
	ciphertext := gcm.Seal(nil, nonce, plaintext, aad)
	return application.ProtectedPayload{KeyID: keyID, Nonce: nonce, Ciphertext: ciphertext, AADDigest: sha256.Sum256(aad)}, nil
}

func (p *Protector) Open(ctx context.Context, payload application.ProtectedPayload, aad []byte) ([]byte, error) {
	if err := ctx.Err(); err != nil { return nil, err }
	if payload.KeyID == "" || len(payload.Nonce) == 0 || len(payload.Ciphertext) == 0 || len(aad) == 0 || payload.AADDigest != sha256.Sum256(aad) { return nil, ErrInvalidPayload }
	key, err := p.keys.Key(ctx, payload.KeyID); if err != nil { return nil, err }
	defer clearBytes(key)
	if len(key) != 32 { return nil, ErrKeyUnavailable }
	block, err := aes.NewCipher(key); if err != nil { return nil, ErrKeyUnavailable }
	gcm, err := cipher.NewGCM(block); if err != nil || len(payload.Nonce) != gcm.NonceSize() { return nil, ErrInvalidPayload }
	plaintext, err := gcm.Open(nil, payload.Nonce, payload.Ciphertext, aad); if err != nil { return nil, ErrOpenFailed }
	return plaintext, nil
}

func clearBytes(value []byte) { for index := range value { value[index] = 0 } }
