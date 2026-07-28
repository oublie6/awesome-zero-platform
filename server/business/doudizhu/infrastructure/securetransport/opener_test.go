package securetransport

import (
	"context"
	"crypto/ed25519"
	"testing"
	"time"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/application"
	"github.com/oublie6/awesome-zero-platform/server/foundation/revealkeys"
	"github.com/oublie6/awesome-zero-platform/server/foundation/secureenvelope"
)

type coreStub struct{ calls int }

func (c *coreStub) Open(context.Context, secureenvelope.Envelope, []byte) ([]byte, error) {
	c.calls++
	return []byte("plaintext"), nil
}

func TestOpenerRejectsRevokedAndMismatchedBoundKeysBeforeHPKE(t *testing.T) {
	now := time.Date(2026, 7, 28, 4, 0, 0, 0, time.UTC)
	registry := lifecycleRegistry(t, now, revealkeys.StatusRevoked)
	core := &coreStub{}
	opener, err := New(core, registry)
	if err != nil {
		t.Fatal(err)
	}
	hash, _ := registry.PublicKeyHash("bound-key")
	keyContext := application.RevealKeyContext{KeyID: "bound-key", PublicKeySHA256: hash, BoundAt: now.Add(-time.Hour), UseAt: now}
	_, err = opener.Open(context.Background(), application.SecureEnvelope{KeyID: "bound-key"}, nil, keyContext)
	if err == nil || core.calls != 0 {
		t.Fatalf("error=%v calls=%d", err, core.calls)
	}

	registry = lifecycleRegistry(t, now, revealkeys.StatusActive)
	opener, _ = New(core, registry)
	keyContext.PublicKeySHA256[0] ^= 0xff
	_, err = opener.Open(context.Background(), application.SecureEnvelope{KeyID: "bound-key"}, nil, keyContext)
	if err == nil || core.calls != 0 {
		t.Fatalf("error=%v calls=%d", err, core.calls)
	}
}

func TestOpenerAllowsPreRetirementHandWithinGrace(t *testing.T) {
	now := time.Date(2026, 7, 28, 4, 0, 0, 0, time.UTC)
	registry := lifecycleRegistry(t, now, revealkeys.StatusRetiring)
	core := &coreStub{}
	opener, _ := New(core, registry)
	hash, _ := registry.PublicKeyHash("bound-key")
	keyContext := application.RevealKeyContext{KeyID: "bound-key", PublicKeySHA256: hash, BoundAt: now.Add(-2 * time.Hour), UseAt: now}
	plaintext, err := opener.Open(context.Background(), application.SecureEnvelope{KeyID: "bound-key"}, nil, keyContext)
	if err != nil || string(plaintext) != "plaintext" || core.calls != 1 {
		t.Fatalf("plaintext=%q error=%v calls=%d", plaintext, err, core.calls)
	}
}

func TestOpenerWithoutLifecycleRegistryFailsClosed(t *testing.T) {
	opener, err := New(&coreStub{})
	if err != nil {
		t.Fatal(err)
	}
	_, err = opener.Open(context.Background(), application.SecureEnvelope{}, nil, application.RevealKeyContext{})
	if err == nil {
		t.Fatal("missing registry was accepted")
	}
}

func lifecycleRegistry(t *testing.T, now time.Time, status revealkeys.Status) *revealkeys.Registry {
	t.Helper()
	seed := make([]byte, ed25519.SeedSize)
	signing := ed25519.NewKeyFromSeed(seed)
	bound := revealkeys.Record{ManifestVersion: 1, KeyID: "bound-key", PublicKey: repeated(1), PrivateKey: repeated(2), NotBefore: now.Add(-24 * time.Hour), ExpiresAt: now.Add(24 * time.Hour), Status: status}
	active := revealkeys.Record{ManifestVersion: 2, KeyID: "active-key", PublicKey: repeated(3), PrivateKey: repeated(4), NotBefore: now.Add(-24 * time.Hour), ExpiresAt: now.Add(24 * time.Hour), Status: revealkeys.StatusActive}
	currentKeyID := "active-key"
	if status == revealkeys.StatusActive {
		bound.ManifestVersion = 2
		active.ManifestVersion = 1
		active.Status = revealkeys.StatusRetired
		active.RetiringAt = now.Add(-2 * time.Hour)
		active.RetireAfter = now.Add(-time.Hour)
		currentKeyID = "bound-key"
	}
	if status == revealkeys.StatusRetiring {
		bound.RetiringAt = now.Add(-time.Hour)
		bound.RetireAfter = now.Add(time.Hour)
	}
	registry, err := revealkeys.New([]revealkeys.Record{bound, active}, currentKeyID, "root", signing, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	return registry
}

func repeated(value byte) []byte {
	result := make([]byte, 32)
	for index := range result {
		result[index] = value
	}
	return result
}
