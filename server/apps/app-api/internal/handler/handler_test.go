package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/oublie6/awesome-zero-platform/server/apps/app-api/internal/config"
	"github.com/oublie6/awesome-zero-platform/server/apps/app-api/internal/svc"
)

func TestLiveHandler(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/health/live", nil)

	liveHandler(testServiceContext()).ServeHTTP(recorder, request)

	assertHealthResponse(t, recorder, "ok")
}

func TestReadyHandler(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/health/ready", nil)

	readyHandler(testServiceContext()).ServeHTTP(recorder, request)

	assertHealthResponse(t, recorder, "ready")
}

func assertHealthResponse(t *testing.T, recorder *httptest.ResponseRecorder, wantStatus string) {
	t.Helper()

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}

	var payload struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if payload.Status != wantStatus {
		t.Fatalf("status field = %q, want %q", payload.Status, wantStatus)
	}
}

func testServiceContext() *svc.ServiceContext {
	cfg := config.Config{}
	cfg.Name = "main-api"
	cfg.Host = "127.0.0.1"
	cfg.Port = 8888

	return svc.NewServiceContext(cfg)
}
