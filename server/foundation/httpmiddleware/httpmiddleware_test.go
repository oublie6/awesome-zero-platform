package httpmiddleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/oublie6/awesome-zero-platform/server/foundation/requestid"
)

func TestRequestIDMiddleware(t *testing.T) {
	t.Parallel()

	handler := RequestID(RequestIDConfig{
		HeaderName: "X-Request-Id",
		MaxLength:  64,
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := requestid.FromContext(r.Context()); got != "caller-id" {
			t.Fatalf("request id in context = %q, want caller-id", got)
		}
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-Id", "caller-id")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("X-Request-Id"); got != "caller-id" {
		t.Fatalf("header = %q, want caller-id", got)
	}
}

func TestSecurityHeaders(t *testing.T) {
	t.Parallel()

	handler := SecurityHeaders(SecurityHeadersConfig{
		ContentTypeOptions: "nosniff",
		FrameOptions:       "DENY",
		ReferrerPolicy:     "no-referrer",
	})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("got %q, want nosniff", got)
	}
}
