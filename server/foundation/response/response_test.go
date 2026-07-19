package response

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/oublie6/awesome-zero-platform/server/foundation/apperrors"
	"github.com/oublie6/awesome-zero-platform/server/foundation/requestid"
)

func TestSuccessEnvelope(t *testing.T) {
	t.Parallel()

	ctx := requestid.IntoContext(context.Background(), "req-123")
	got := Success(ctx, map[string]string{"status": "ok"})

	if got.Code != apperrors.CodeOK {
		t.Fatalf("code = %q, want %q", got.Code, apperrors.CodeOK)
	}
	if got.RequestID != "req-123" {
		t.Fatalf("requestId = %q, want req-123", got.RequestID)
	}
}

func TestErrorEnvelope(t *testing.T) {
	t.Parallel()

	ctx := requestid.IntoContext(context.Background(), "req-123")
	got := Error(ctx, apperrors.InvalidParameter("invalid parameter"))

	if got.Code != apperrors.CodeParamInvalid {
		t.Fatalf("code = %q, want %q", got.Code, apperrors.CodeParamInvalid)
	}
	if got.Data != nil {
		t.Fatalf("data = %#v, want nil", got.Data)
	}
}

func TestNormalizeUnknownError(t *testing.T) {
	t.Parallel()

	err := normalizeError(errors.New("secret"))
	if err.Status() != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", err.Status(), http.StatusInternalServerError)
	}
}
