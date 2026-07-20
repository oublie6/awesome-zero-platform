package identity

import (
	"strings"
	"testing"
)

func TestArgon2idHasherHashAndVerify(t *testing.T) {
	t.Parallel()

	hasher := NewTestArgon2idHasher()
	password := "correct-horse-battery-staple"

	hash, err := hasher.Hash(password)
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}
	if !strings.HasPrefix(hash, "$argon2id$") {
		t.Fatalf("hash prefix = %q, want argon2id PHC string", hash)
	}
	if strings.Contains(hash, password) {
		t.Fatal("hash unexpectedly contains plaintext password")
	}

	ok, err := hasher.Verify(hash, password)
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if !ok {
		t.Fatal("expected password verification success")
	}

	ok, err = hasher.Verify(hash, "wrong-password")
	if err != nil {
		t.Fatalf("Verify() with wrong password error = %v", err)
	}
	if ok {
		t.Fatal("expected password verification failure")
	}
}

func TestArgon2idHasherRejectsMalformedHash(t *testing.T) {
	t.Parallel()

	hasher := NewTestArgon2idHasher()
	if _, err := hasher.Verify("not-a-valid-hash", "password"); err == nil {
		t.Fatal("expected malformed hash error, got nil")
	}
}
