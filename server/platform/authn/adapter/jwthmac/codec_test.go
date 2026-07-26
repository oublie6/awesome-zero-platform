package jwthmac

import (
	"errors"
	"testing"
	"time"

	"github.com/oublie6/awesome-zero-platform/server/platform/authn"
)

func TestCodecRoundTrip(t *testing.T) {
	codec, err := New("0123456789abcdef0123456789abcdef", "awesome-zero-platform")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	now := time.Now().UTC().Truncate(time.Second)
	input := authn.AccessClaims{
		Subject:   "account-1",
		SessionID: "session-1",
		IssuedAt:  now,
		ExpiresAt: now.Add(15 * time.Minute),
	}
	token, err := codec.Issue(input)
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	claims, err := codec.Parse(token)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if claims.Subject != input.Subject || claims.SessionID != input.SessionID || !claims.IssuedAt.Equal(input.IssuedAt) || !claims.ExpiresAt.Equal(input.ExpiresAt) {
		t.Fatalf("claims = %#v, want %#v", claims, input)
	}
}

func TestCodecRejectsWrongSecretAndExpiredToken(t *testing.T) {
	issuer := "awesome-zero-platform"
	codec, err := New("0123456789abcdef0123456789abcdef", issuer)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	other, err := New("abcdef0123456789abcdef0123456789", issuer)
	if err != nil {
		t.Fatalf("New(other) error = %v", err)
	}

	now := time.Now().UTC()
	valid, err := codec.Issue(authn.AccessClaims{
		Subject:   "account-1",
		SessionID: "session-1",
		IssuedAt:  now,
		ExpiresAt: now.Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("Issue(valid) error = %v", err)
	}
	if _, err := other.Parse(valid); !errors.Is(err, authn.ErrInvalidToken) {
		t.Fatalf("Parse() wrong secret error = %v, want ErrInvalidToken", err)
	}

	expired, err := codec.Issue(authn.AccessClaims{
		Subject:   "account-1",
		SessionID: "session-1",
		IssuedAt:  now.Add(-2 * time.Minute),
		ExpiresAt: now.Add(-time.Minute),
	})
	if err != nil {
		t.Fatalf("Issue(expired) error = %v", err)
	}
	if _, err := codec.Parse(expired); !errors.Is(err, authn.ErrInvalidToken) {
		t.Fatalf("Parse() expired error = %v, want ErrInvalidToken", err)
	}
}

func TestNewRejectsWeakConfiguration(t *testing.T) {
	if _, err := New("too-short", "issuer"); err == nil {
		t.Fatal("New() expected weak secret error")
	}
	if _, err := New("0123456789abcdef0123456789abcdef", ""); err == nil {
		t.Fatal("New() expected empty issuer error")
	}
}
