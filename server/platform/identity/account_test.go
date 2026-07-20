package identity

import (
	"strings"
	"testing"
)

func TestIdentityNormalize(t *testing.T) {
	t.Parallel()

	normalized, err := Identity{
		Username: "  Alice.Admin  ",
		Email:    "  Alice.Admin@Example.com  ",
		Phone:    "  +14155550123  ",
	}.Normalize()
	if err != nil {
		t.Fatalf("Normalize() error = %v", err)
	}

	if normalized.Username != "Alice.Admin" || normalized.UsernameKey != "alice.admin" {
		t.Fatalf("username normalization = %#v", normalized)
	}
	if normalized.Email != "Alice.Admin@Example.com" || normalized.EmailKey != "alice.admin@example.com" {
		t.Fatalf("email normalization = %#v", normalized)
	}
	if normalized.Phone != "+14155550123" || normalized.PhoneKey != "+14155550123" {
		t.Fatalf("phone normalization = %#v", normalized)
	}
}

func TestIdentityNormalizeRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		identity Identity
	}{
		{
			name:     "missing all identities",
			identity: Identity{},
		},
		{
			name: "invalid username",
			identity: Identity{
				Username: "bad username",
			},
		},
		{
			name: "invalid email",
			identity: Identity{
				Email: "not-an-email",
			},
		},
		{
			name: "invalid phone",
			identity: Identity{
				Phone: "13800138000",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if _, err := tt.identity.Normalize(); err == nil {
				t.Fatal("expected validation error, got nil")
			}
		})
	}
}

func TestValidatePasswordInput(t *testing.T) {
	t.Parallel()

	if err := validatePasswordInput(""); err == nil {
		t.Fatal("expected empty password error, got nil")
	}
	if err := validatePasswordInput("short"); err == nil {
		t.Fatal("expected too-short password error, got nil")
	}
	if err := validatePasswordInput(strings.Repeat("a", PasswordMaxBytes+1)); err == nil {
		t.Fatal("expected oversized password error, got nil")
	}
	if err := validatePasswordInput("valid-pass-123"); err != nil {
		t.Fatalf("unexpected valid password error: %v", err)
	}
}
