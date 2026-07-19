package requestid

import (
	"context"
	"testing"
)

func TestIsValid(t *testing.T) {
	t.Parallel()

	if !IsValid("abc-123._XYZ", 64) {
		t.Fatal("expected request id to be valid")
	}
	if IsValid("bad value", 64) {
		t.Fatal("expected space-containing request id to be invalid")
	}
	if IsValid("", 64) {
		t.Fatal("expected empty request id to be invalid")
	}
}

func TestEffective(t *testing.T) {
	t.Parallel()

	if got := Effective("caller-id", 64); got != "caller-id" {
		t.Fatalf("got %q, want caller-id", got)
	}
	if got := Effective("bad value", 64); got == "bad value" || got == "" {
		t.Fatalf("expected generated request id, got %q", got)
	}
}

func TestContextRoundTrip(t *testing.T) {
	t.Parallel()

	ctx := IntoContext(context.Background(), "req-1")
	if got := FromContext(ctx); got != "req-1" {
		t.Fatalf("got %q, want req-1", got)
	}
}
