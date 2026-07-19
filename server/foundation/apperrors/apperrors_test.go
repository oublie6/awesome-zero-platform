package apperrors

import (
	"errors"
	"net/http"
	"testing"
)

func TestInternalWrapsCause(t *testing.T) {
	t.Parallel()

	cause := errors.New("database offline")
	err := Internal(cause)

	if err.Status() != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", err.Status(), http.StatusInternalServerError)
	}
	if err.Code() != CodeInternal {
		t.Fatalf("code = %q, want %q", err.Code(), CodeInternal)
	}
	if !errors.Is(err, cause) {
		t.Fatalf("expected wrapped cause")
	}
}

func TestAsUnknown(t *testing.T) {
	t.Parallel()

	code := StableCode(errors.New("boom"))
	if code != CodeInternal {
		t.Fatalf("code = %q, want %q", code, CodeInternal)
	}
}
