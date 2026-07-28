package protection

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/application"
)

func TestProtectorRoundTripAndTamperRejection(t *testing.T) {
	keyring, err := NewStaticKeyring("key-1", map[string][]byte{"key-1": bytes.Repeat([]byte{7}, 32)})
	if err != nil { t.Fatal(err) }
	protector, err := NewWithRandom(keyring, bytes.NewReader(bytes.Repeat([]byte{9}, 64)))
	if err != nil { t.Fatal(err) }
	plaintext := []byte(`{"phrase":"原始短语","secureRandom":"secret"}`); aad := []byte("bound-record-context")
	payload, err := protector.Seal(context.Background(), plaintext, aad); if err != nil { t.Fatal(err) }
	if payload.KeyID != "key-1" || len(payload.Nonce) != 12 || bytes.Contains(payload.Ciphertext, plaintext) { t.Fatalf("unexpected payload: %#v", payload) }
	opened, err := protector.Open(context.Background(), payload, aad); if err != nil || !bytes.Equal(opened, plaintext) { t.Fatalf("opened=%q err=%v", opened, err) }
	tampered := payload; tampered.Ciphertext = append([]byte(nil), payload.Ciphertext...); tampered.Ciphertext[0] ^= 1
	if _, err := protector.Open(context.Background(), tampered, aad); !errors.Is(err, ErrOpenFailed) { t.Fatalf("tampered ciphertext error=%v", err) }
	if _, err := protector.Open(context.Background(), payload, []byte("other-aad")); !errors.Is(err, ErrInvalidPayload) { t.Fatalf("tampered AAD error=%v", err) }
}

func TestProtectorRejectsInvalidKeysAndInputs(t *testing.T) {
	if _, err := NewStaticKeyring("bad", map[string][]byte{"bad": bytes.Repeat([]byte{1}, 31)}); !errors.Is(err, ErrKeyUnavailable) { t.Fatalf("keyring error=%v", err) }
	keyring, _ := NewStaticKeyring("key-1", map[string][]byte{"key-1": bytes.Repeat([]byte{1}, 32)})
	protector, _ := NewWithRandom(keyring, bytes.NewReader(bytes.Repeat([]byte{2}, 32)))
	if _, err := protector.Seal(context.Background(), nil, []byte("aad")); !errors.Is(err, ErrInvalidPayload) { t.Fatalf("empty plaintext error=%v", err) }
	if _, err := protector.Seal(context.Background(), []byte("value"), nil); !errors.Is(err, ErrInvalidPayload) { t.Fatalf("empty AAD error=%v", err) }
	if _, err := protector.Open(context.Background(), application.ProtectedPayload{}, []byte("aad")); !errors.Is(err, ErrInvalidPayload) { t.Fatalf("empty payload error=%v", err) }
}

func TestProtectorSerializesInjectedRandomReader(t *testing.T) {
	keyring, err := NewStaticKeyring("key-1", map[string][]byte{"key-1": bytes.Repeat([]byte{7}, 32)}); if err != nil { t.Fatal(err) }
	reader := &concurrencyDetectingReader{}; protector, err := NewWithRandom(keyring, reader); if err != nil { t.Fatal(err) }
	const workers = 32
	var wg sync.WaitGroup; errorsSeen := make(chan error, workers); wg.Add(workers)
	for index := 0; index < workers; index++ { go func(index int) { defer wg.Done(); _, err := protector.Seal(context.Background(), []byte("value"), []byte(fmt.Sprintf("aad-%d", index))); errorsSeen <- err }(index) }
	wg.Wait(); close(errorsSeen)
	for err := range errorsSeen { if err != nil { t.Fatal(err) } }
	if reader.concurrent.Load() { t.Fatal("injected random reader was used concurrently") }
}

type concurrencyDetectingReader struct { active atomic.Int32; counter atomic.Uint32; concurrent atomic.Bool }
func (r *concurrencyDetectingReader) Read(value []byte) (int, error) { if r.active.Add(1) != 1 { r.concurrent.Store(true) }; defer r.active.Add(-1); seed := byte(r.counter.Add(1)); for index := range value { value[index] = seed + byte(index) }; time.Sleep(time.Millisecond); return len(value), nil }
