package identity

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"testing"
)

func TestExternalIdentityPendingTokenValidation(t *testing.T) {
	raw := make([]byte, 32)
	token := base64.RawURLEncoding.EncodeToString(raw)
	got, err := pendingDigest(token)
	if err != nil {
		t.Fatal(err)
	}
	if got != sha256.Sum256(raw) {
		t.Fatal("incorrect pending proof hash")
	}
	for _, bad := range []string{"", "short", "some-token-with-padding==", token + "x", token + "="} {
		if _, err := pendingDigest(bad); !errors.Is(err, ErrPendingInvalid) {
			t.Fatalf("invalid %q got %v", bad, err)
		}
	}
}
func TestVerifiedSubjectValidation(t *testing.T) {
	b := &ExternalBinding{key: make([]byte, 32)}
	for _, in := range []struct{ provider, subject string }{
		{"other", "verified-subject"}, {"wechat", ""}, {"wechat", " trailing"}, {"wechat", "a"},
	} {
		if _, err := b.digest(in.provider, in.subject); !errors.Is(err, ErrPendingInvalid) {
			t.Fatalf("invalid provider/subject accepted: %v", err)
		}
	}
	one, err := b.digest("wechat", "openid-one")
	if err != nil {
		t.Fatal(err)
	}
	two, err := b.digest("wechat", "openid-two")
	if err != nil {
		t.Fatal(err)
	}
	if string(one) == string(two) {
		t.Fatal("distinct subjects share HMAC digest")
	}
}
