package revealkeys

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestRegistrySignsAndSelectsCurrentManifest(t *testing.T) {
	now := time.Date(2026, 7, 28, 4, 0, 0, 0, time.UTC)
	registry, root := testRegistry(t, now)
	manifest, err := registry.Current(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if manifest.KeyID != "reveal-active" || manifest.ManifestVersion != 2 {
		t.Fatalf("manifest=%#v", manifest)
	}
	if err := VerifyManifest(manifest, map[string]ed25519.PublicKey{"root-2026": root}); err != nil {
		t.Fatal(err)
	}
	publicKey, _ := base64.RawURLEncoding.DecodeString(manifest.PublicKey)
	hash := sha256.Sum256(publicKey)
	if manifest.PublicKeySHA256 != base64.RawURLEncoding.EncodeToString(hash[:]) {
		t.Fatal("public key hash mismatch")
	}
	if got := registry.ManifestVersions(); len(got) != 2 || got[0] != 1 || got[1] != 2 {
		t.Fatalf("versions=%v", got)
	}
}

func TestManifestTamperingAndReplacementAreRejected(t *testing.T) {
	now := time.Date(2026, 7, 28, 4, 0, 0, 0, time.UTC)
	registry, root := testRegistry(t, now)
	manifest, _ := registry.Current(context.Background())
	manifest.KeyID = "attacker-key"
	if err := VerifyManifest(manifest, map[string]ed25519.PublicKey{"root-2026": root}); !errors.Is(err, ErrSignatureInvalid) {
		t.Fatalf("error=%v", err)
	}

	manifest, _ = registry.Current(context.Background())
	manifest.PublicKey = base64.RawURLEncoding.EncodeToString(make([]byte, 32))
	if err := VerifyManifest(manifest, map[string]ed25519.PublicKey{"root-2026": root}); !errors.Is(err, ErrKeyHashMismatch) {
		t.Fatalf("error=%v", err)
	}

	manifest, _ = registry.Current(context.Background())
	if err := VerifyManifest(manifest, map[string]ed25519.PublicKey{"other-root": root}); !errors.Is(err, ErrSignatureInvalid) {
		t.Fatalf("error=%v", err)
	}
}

func TestRetiringKeyAllowsOnlyPreRetirementHandsWithinGrace(t *testing.T) {
	now := time.Date(2026, 7, 28, 4, 0, 0, 0, time.UTC)
	registry, _ := testRegistry(t, now)
	hash, _ := registry.PublicKeyHash("reveal-old")
	bound := BoundContext{KeyID: "reveal-old", PublicKeySHA256: hash, BoundAt: now.Add(-2 * time.Hour), UseAt: now}
	if err := registry.AuthorizeBound(context.Background(), bound); err != nil {
		t.Fatal(err)
	}
	bound.BoundAt = now.Add(-30 * time.Minute)
	if err := registry.AuthorizeBound(context.Background(), bound); !errors.Is(err, ErrKeyExpired) {
		t.Fatalf("post-retirement error=%v", err)
	}
	bound.BoundAt = now.Add(-2 * time.Hour)
	bound.UseAt = now.Add(3 * time.Hour)
	if err := registry.AuthorizeBound(context.Background(), bound); !errors.Is(err, ErrKeyExpired) {
		t.Fatalf("after grace error=%v", err)
	}
}

func TestRevokedAndHashMismatchedKeysFailClosed(t *testing.T) {
	now := time.Date(2026, 7, 28, 4, 0, 0, 0, time.UTC)
	seed := make([]byte, ed25519.SeedSize)
	for i := range seed {
		seed[i] = byte(i + 1)
	}
	signing := ed25519.NewKeyFromSeed(seed)
	record := testRecord("revoked", 1, StatusRevoked, now)
	registry, err := New([]Record{record, testRecord("active", 2, StatusActive, now)}, "active", "root", signing, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	hash, _ := registry.PublicKeyHash("revoked")
	bound := BoundContext{KeyID: "revoked", PublicKeySHA256: hash, BoundAt: now.Add(-time.Hour), UseAt: now}
	if err := registry.AuthorizeBound(context.Background(), bound); !errors.Is(err, ErrKeyRevoked) {
		t.Fatalf("error=%v", err)
	}
	activeHash, _ := registry.PublicKeyHash("active")
	activeHash[0] ^= 0xff
	bound = BoundContext{KeyID: "active", PublicKeySHA256: activeHash, BoundAt: now, UseAt: now}
	if err := registry.AuthorizeBound(context.Background(), bound); !errors.Is(err, ErrKeyHashMismatch) {
		t.Fatalf("error=%v", err)
	}
}

func TestRegistryRejectsInvalidLifecycleAndVersions(t *testing.T) {
	now := time.Date(2026, 7, 28, 4, 0, 0, 0, time.UTC)
	seed := make([]byte, ed25519.SeedSize)
	signing := ed25519.NewKeyFromSeed(seed)
	cases := [][]Record{
		{testRecord("a", 1, StatusActive, now), testRecord("b", 2, StatusActive, now)},
		{testRecord("a", 1, StatusActive, now), testRecord("b", 1, StatusRetired, now)},
		{{ManifestVersion: 1, KeyID: "bad", PublicKey: []byte{1}, NotBefore: now, ExpiresAt: now.Add(time.Hour), Status: StatusActive}},
	}
	for index, records := range cases {
		if _, err := New(records, "a", "root", signing, func() time.Time { return now }); !errors.Is(err, ErrInvalidConfig) {
			t.Fatalf("case %d error=%v", index, err)
		}
	}
}

func TestStaticJSONConfigRejectsUnknownFieldsAndLoadsKeys(t *testing.T) {
	now := time.Date(2026, 7, 28, 4, 0, 0, 0, time.UTC)
	seed := make([]byte, ed25519.SeedSize)
	for i := range seed {
		seed[i] = byte(i)
	}
	signing := ed25519.NewKeyFromSeed(seed)
	record := testRecord("active", 1, StatusActive, now)
	config := StaticConfig{CurrentKeyID: "active", SignatureKeyID: "root", SigningPrivateKey: base64.RawURLEncoding.EncodeToString(signing), Keys: []StaticKeyConfig{{ManifestVersion: 1, KeyID: record.KeyID, PublicKey: base64.RawURLEncoding.EncodeToString(record.PublicKey), PrivateKey: base64.RawURLEncoding.EncodeToString(record.PrivateKey), NotBefore: formatTime(record.NotBefore), ExpiresAt: formatTime(record.ExpiresAt), Status: record.Status}}}
	encoded, _ := json.Marshal(config)
	registry, err := NewFromJSON(string(encoded), func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	if _, err := registry.Current(context.Background()); err != nil {
		t.Fatal(err)
	}
	withUnknown := string(encoded[:len(encoded)-1]) + `,"unexpected":true}`
	if _, err := NewFromJSON(withUnknown, nil); !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("error=%v", err)
	}
}

func testRegistry(t *testing.T, now time.Time) (*Registry, ed25519.PublicKey) {
	t.Helper()
	seed := make([]byte, ed25519.SeedSize)
	for i := range seed {
		seed[i] = byte(i + 1)
	}
	signing := ed25519.NewKeyFromSeed(seed)
	old := testRecord("reveal-old", 1, StatusRetiring, now)
	active := testRecord("reveal-active", 2, StatusActive, now)
	registry, err := New([]Record{old, active}, active.KeyID, "root-2026", signing, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	return registry, signing.Public().(ed25519.PublicKey)
}

func testRecord(id string, version uint64, status Status, now time.Time) Record {
	publicKey := make([]byte, 32)
	privateKey := make([]byte, 32)
	for i := range publicKey {
		publicKey[i] = byte(int(version) + i + 1)
		privateKey[i] = byte(int(version) + i + 33)
	}
	record := Record{ManifestVersion: version, KeyID: id, PublicKey: publicKey, PrivateKey: privateKey, NotBefore: now.Add(-24 * time.Hour), ExpiresAt: now.Add(24 * time.Hour), Status: status}
	switch status {
	case StatusRetiring:
		record.RetiringAt = now.Add(-time.Hour)
		record.RetireAfter = now.Add(2 * time.Hour)
	case StatusRetired:
		record.RetiringAt = now.Add(-3 * time.Hour)
		record.RetireAfter = now.Add(-time.Hour)
	}
	return record
}
