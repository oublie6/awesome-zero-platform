package bootstrap

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/oublie6/awesome-zero-platform/server/foundation/apperrors"
	"github.com/oublie6/awesome-zero-platform/server/foundation/cache"
	"github.com/oublie6/awesome-zero-platform/server/foundation/database"
	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func TestNewRejectsInvalidConfig(t *testing.T) {
	configPath := writeConfig(t, invalidPortConfig("0"))

	_, err := New(configPath)
	if err == nil {
		t.Fatal("expected invalid config error, got nil")
	}
}

func TestNewRejectsContradictoryCORS(t *testing.T) {
	configPath := writeConfig(t, contradictoryCORSConfig())

	_, err := New(configPath)
	if err == nil {
		t.Fatal("expected CORS validation error, got nil")
	}
}

func TestAppHealthHeaders(t *testing.T) {
	restore := stubDependencies(
		t,
		&fakeMySQL{pingErr: nil},
		&fakeRedis{pingErr: nil},
	)
	defer restore()

	port := reservePort(t)
	configPath := writeConfig(t, runtimeConfig(port, false, false))

	app, err := New(configPath)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer app.Stop()

	go app.Start()
	waitForHealthy(t, "http://127.0.0.1:"+strconv.Itoa(port)+"/health/live")

	req, err := http.NewRequest(http.MethodGet, "http://127.0.0.1:"+strconv.Itoa(port)+"/health/live", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("X-Request-Id", "caller-health-id")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	defer resp.Body.Close()

	if got := resp.Header.Get("X-Request-Id"); got != "caller-health-id" {
		t.Fatalf("request id header = %q, want caller-health-id", got)
	}
	if got := resp.Header.Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("X-Content-Type-Options = %q, want nosniff", got)
	}
}

func TestFoundationHTTPBehavior(t *testing.T) {
	mysql := &fakeMySQL{}
	redis := &fakeRedis{}
	restore := stubDependencies(t, mysql, redis)
	defer restore()

	var logBuffer bytes.Buffer
	previousWriter := logx.Reset()
	logx.SetWriter(logx.NewWriter(&logBuffer))
	defer func() {
		logx.Reset()
		if previousWriter != nil {
			logx.SetWriter(previousWriter)
		}
	}()

	port := reservePort(t)
	configPath := writeConfig(t, runtimeConfig(port, true, false))

	app, err := New(configPath)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	registerTestRoutes(app)
	defer app.Stop()

	done := make(chan struct{})
	go func() {
		defer close(done)
		app.Start()
	}()

	baseURL := "http://127.0.0.1:" + strconv.Itoa(port)
	waitForHealthy(t, baseURL+"/health/live")

	success := doRequest(t, http.MethodGet, baseURL+"/_test/success", "", map[string]string{
		"X-Request-Id": "caller-id",
	})
	assertEnvelope(t, success, http.StatusOK, apperrors.CodeOK, "success", "caller-id")

	invalid := doRequest(t, http.MethodGet, baseURL+"/_test/success", "", map[string]string{
		"X-Request-Id": "bad value with spaces",
	})
	var invalidEnvelope struct {
		RequestID string `json:"requestId"`
	}
	if err := json.Unmarshal(invalid.body, &invalidEnvelope); err != nil {
		t.Fatalf("decode invalid request id response: %v", err)
	}
	if invalidEnvelope.RequestID == "" || invalidEnvelope.RequestID == "bad value with spaces" {
		t.Fatalf("expected generated request id, got %q", invalidEnvelope.RequestID)
	}

	appErrResp := doRequest(t, http.MethodGet, baseURL+"/_test/app-error", "", nil)
	assertEnvelope(t, appErrResp, http.StatusBadRequest, apperrors.CodeParamInvalid, "invalid parameter", appErrResp.header.Get("X-Request-Id"))

	internalResp := doRequest(t, http.MethodGet, baseURL+"/_test/internal-error", "", nil)
	assertEnvelope(t, internalResp, http.StatusInternalServerError, apperrors.CodeInternal, "internal server error", internalResp.header.Get("X-Request-Id"))
	if string(internalResp.body) == "" || contains(string(internalResp.body), "database offline") {
		t.Fatalf("internal response leaked details: %s", string(internalResp.body))
	}

	panicResp := doRequest(t, http.MethodGet, baseURL+"/_test/panic", "", nil)
	assertEnvelope(t, panicResp, http.StatusInternalServerError, apperrors.CodeInternal, "internal server error", panicResp.header.Get("X-Request-Id"))

	recovered := doRequest(t, http.MethodGet, baseURL+"/_test/success", "", nil)
	assertEnvelope(t, recovered, http.StatusOK, apperrors.CodeOK, "success", recovered.header.Get("X-Request-Id"))

	tooLarge := doRequest(t, http.MethodPost, baseURL+"/_test/echo", "0123456789", map[string]string{
		"Content-Type": "text/plain",
	})
	assertEnvelope(t, tooLarge, http.StatusRequestEntityTooLarge, apperrors.CodeRequestTooLarge, "request body too large", tooLarge.header.Get("X-Request-Id"))

	preflight := doRequest(t, http.MethodOptions, baseURL+"/_test/success", "", map[string]string{
		"Origin":                         "https://allowed.example",
		"Access-Control-Request-Method":  "GET",
		"Access-Control-Request-Headers": "X-Request-Id",
	})
	if preflight.status != http.StatusNoContent {
		t.Fatalf("preflight status = %d, want %d", preflight.status, http.StatusNoContent)
	}
	if got := preflight.header.Get("Access-Control-Allow-Origin"); got != "https://allowed.example" {
		t.Fatalf("allowed origin header = %q, want https://allowed.example", got)
	}

	denied := doRequest(t, http.MethodGet, baseURL+"/_test/success", "", map[string]string{
		"Origin": "https://denied.example",
	})
	if got := denied.header.Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("denied origin unexpectedly allowed: %q", got)
	}

	mysql.pingErr = errors.New("mysql unavailable")
	notReadyMySQL := doRequest(t, http.MethodGet, baseURL+"/health/ready", "", nil)
	assertHealthStatus(t, notReadyMySQL, http.StatusServiceUnavailable, "unready")

	mysql.pingErr = nil
	redis.pingErr = errors.New("redis unavailable")
	notReadyRedis := doRequest(t, http.MethodGet, baseURL+"/health/ready", "", nil)
	assertHealthStatus(t, notReadyRedis, http.StatusServiceUnavailable, "unready")

	redis.pingErr = nil
	assertAccessLog(t, logBuffer.String(), "/_test/success", http.StatusOK, "caller-id")

	select {
	case <-done:
		t.Fatal("server exited unexpectedly")
	case <-time.After(100 * time.Millisecond):
	}
}

func TestPartialStartupFailureClosesMySQL(t *testing.T) {
	mysql := &fakeMySQL{}
	originalMySQLOpener := openMySQL
	originalRedisOpener := openRedis
	openMySQL = func(context.Context, database.Config) (database.Handle, error) {
		return mysql, nil
	}
	openRedis = func(context.Context, cache.Config) (cache.Handle, error) {
		return nil, errors.New("redis open failed")
	}
	defer func() {
		openMySQL = originalMySQLOpener
		openRedis = originalRedisOpener
	}()

	configPath := writeConfig(t, runtimeConfig(reservePort(t), false, false))
	_, err := New(configPath)
	if err == nil {
		t.Fatal("expected startup error, got nil")
	}
	if !mysql.closed {
		t.Fatal("expected mysql to be closed after redis startup failure")
	}
}

func TestAppStopIsIdempotent(t *testing.T) {
	mysql := &fakeMySQL{}
	redis := &fakeRedis{}
	restore := stubDependencies(t, mysql, redis)
	defer restore()

	configPath := writeConfig(t, runtimeConfig(reservePort(t), false, false))
	app, err := New(configPath)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	app.Stop()
	app.Stop()

	if mysql.closeCalls != 1 {
		t.Fatalf("mysql close calls = %d, want 1", mysql.closeCalls)
	}
	if redis.closeCalls != 1 {
		t.Fatalf("redis close calls = %d, want 1", redis.closeCalls)
	}
}

func writeConfig(t *testing.T, contents string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "main-api.yaml")
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	return path
}

type httpResponse struct {
	status int
	header http.Header
	body   []byte
}

func doRequest(t *testing.T, method, url, body string, headers map[string]string) httpResponse {
	t.Helper()

	req, err := http.NewRequest(method, url, strings.NewReader(body))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	defer resp.Body.Close()

	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	return httpResponse{
		status: resp.StatusCode,
		header: resp.Header.Clone(),
		body:   payload,
	}
}

func assertEnvelope(t *testing.T, resp httpResponse, wantStatus int, wantCode, wantMessage, wantRequestID string) {
	t.Helper()

	if resp.status != wantStatus {
		t.Fatalf("status = %d, want %d, body=%s", resp.status, wantStatus, string(resp.body))
	}

	var envelope struct {
		Code      string          `json:"code"`
		Message   string          `json:"message"`
		RequestID string          `json:"requestId"`
		Data      json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(resp.body, &envelope); err != nil {
		t.Fatalf("decode envelope: %v, body=%s", err, string(resp.body))
	}

	if envelope.Code != wantCode {
		t.Fatalf("code = %q, want %q", envelope.Code, wantCode)
	}
	if envelope.Message != wantMessage {
		t.Fatalf("message = %q, want %q", envelope.Message, wantMessage)
	}
	if envelope.RequestID != wantRequestID {
		t.Fatalf("requestId = %q, want %q", envelope.RequestID, wantRequestID)
	}
	if got := resp.header.Get("X-Request-Id"); got != wantRequestID {
		t.Fatalf("response header request id = %q, want %q", got, wantRequestID)
	}
}

func assertHealthStatus(t *testing.T, resp httpResponse, wantStatus int, wantValue string) {
	t.Helper()

	if resp.status != wantStatus {
		t.Fatalf("status = %d, want %d, body=%s", resp.status, wantStatus, string(resp.body))
	}

	var payload struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(resp.body, &payload); err != nil {
		t.Fatalf("decode health response: %v", err)
	}
	if payload.Status != wantValue {
		t.Fatalf("status payload = %q, want %q", payload.Status, wantValue)
	}
	if strings.Contains(string(resp.body), "127.0.0.1") || strings.Contains(string(resp.body), "password") {
		t.Fatalf("health response leaked internal detail: %s", string(resp.body))
	}
}

func contains(haystack, needle string) bool {
	return strings.Contains(haystack, needle)
}

func assertAccessLog(t *testing.T, logs, path string, wantStatus int, wantRequestID string) {
	t.Helper()

	for _, line := range strings.Split(strings.TrimSpace(logs), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}

		var payload map[string]any
		if err := json.Unmarshal([]byte(line), &payload); err != nil {
			continue
		}

		if payload["content"] != "http access" || payload["path"] != path {
			continue
		}

		if payload["requestId"] != wantRequestID {
			t.Fatalf("requestId in log = %#v, want %q", payload["requestId"], wantRequestID)
		}
		if payload["method"] != "GET" {
			t.Fatalf("method in log = %#v, want GET", payload["method"])
		}
		if int(payload["statusCode"].(float64)) != wantStatus {
			t.Fatalf("statusCode in log = %#v, want %d", payload["statusCode"], wantStatus)
		}
		if payload["elapsed"] == "" {
			t.Fatal("elapsed missing from access log")
		}
		if payload["clientAddress"] == "" {
			t.Fatal("clientAddress missing from access log")
		}
		return
	}

	t.Fatalf("no matching access log found for path %s in logs: %s", path, logs)
}

func registerTestRoutes(app *App) {
	app.server.AddRoutes([]rest.Route{
		{
			Method: http.MethodGet,
			Path:   "/_test/success",
			Handler: func(w http.ResponseWriter, r *http.Request) {
				httpx.OkJsonCtx(r.Context(), w, map[string]string{"status": "ok"})
			},
		},
		{
			Method: http.MethodGet,
			Path:   "/_test/app-error",
			Handler: func(w http.ResponseWriter, r *http.Request) {
				httpx.ErrorCtx(r.Context(), w, apperrors.InvalidParameter("invalid parameter"))
			},
		},
		{
			Method: http.MethodGet,
			Path:   "/_test/internal-error",
			Handler: func(w http.ResponseWriter, r *http.Request) {
				httpx.ErrorCtx(r.Context(), w, errors.New("database offline"))
			},
		},
		{
			Method: http.MethodGet,
			Path:   "/_test/panic",
			Handler: func(http.ResponseWriter, *http.Request) {
				panic("panic secret")
			},
		},
		{
			Method: http.MethodPost,
			Path:   "/_test/echo",
			Handler: func(w http.ResponseWriter, r *http.Request) {
				body, err := io.ReadAll(r.Body)
				if err != nil {
					httpx.ErrorCtx(r.Context(), w, err)
					return
				}
				httpx.OkJsonCtx(r.Context(), w, map[string]int{"length": len(body)})
			},
		},
	})
}

func stubDependencies(t *testing.T, mysql database.Handle, redis cache.Handle) func() {
	t.Helper()

	originalMySQLOpener := openMySQL
	originalRedisOpener := openRedis
	openMySQL = func(context.Context, database.Config) (database.Handle, error) {
		return mysql, nil
	}
	openRedis = func(context.Context, cache.Config) (cache.Handle, error) {
		return redis, nil
	}

	return func() {
		openMySQL = originalMySQLOpener
		openRedis = originalRedisOpener
	}
}

func invalidPortConfig(port string) string {
	return baseConfig() + "Port: " + port + "\n"
}

func contradictoryCORSConfig() string {
	return baseConfig() + "Port: 8888\nHTTP:\n  MaxBodyBytes: 1024\n  RequestID:\n    HeaderName: X-Request-Id\n    MaxLength: 64\n  SecurityHeaders:\n    ContentTypeOptions: nosniff\n    FrameOptions: DENY\n    ReferrerPolicy: no-referrer\n  CORS:\n    Enabled: true\n    AllowedOrigins:\n      - \"*\"\n    AllowedMethods:\n      - GET\n    AllowedHeaders:\n      - Content-Type\n    AllowCredentials: true\n" + dependencyConfig()
}

func runtimeConfig(port int, corsEnabled bool, minimal bool) string {
	httpBlock := "HTTP:\n  MaxBodyBytes: 8\n  RequestID:\n    HeaderName: X-Request-Id\n    MaxLength: 16\n  SecurityHeaders:\n    ContentTypeOptions: nosniff\n    FrameOptions: DENY\n    ReferrerPolicy: no-referrer\n"
	if minimal {
		httpBlock = "HTTP:\n  MaxBodyBytes: 1024\n  RequestID:\n    HeaderName: X-Request-Id\n    MaxLength: 64\n  SecurityHeaders:\n    ContentTypeOptions: nosniff\n    FrameOptions: DENY\n    ReferrerPolicy: no-referrer\n"
	}
	if corsEnabled {
		httpBlock += "  CORS:\n    Enabled: true\n    AllowedOrigins:\n      - https://allowed.example\n    AllowedMethods:\n      - GET\n      - POST\n      - OPTIONS\n    AllowedHeaders:\n      - Content-Type\n      - X-Request-Id\n    ExposedHeaders:\n      - X-Request-Id\n    AllowCredentials: false\n"
	}

	return baseConfig() + "Port: " + strconv.Itoa(port) + "\n" + httpBlock + dependencyConfig()
}

func baseConfig() string {
	return "Name: main-api\nHost: 127.0.0.1\n"
}

func dependencyConfig() string {
	return "MySQL:\n  Addr: 127.0.0.1:3306\n  Database: awesome_zero_platform\n  User: app_local\n  Password: local-dev-only-mysql-password\n  Charset: utf8mb4\n  ParseTime: true\n  Location: UTC\n  TimeZone: +00:00\n  Timeout: 3s\n  MaxOpenConns: 4\n  MaxIdleConns: 2\n  ConnMaxLifetime: 5m\n  StartupTimeout: 3s\n  ReadinessTimeout: 2s\nRedis:\n  Addr: 127.0.0.1:6379\n  Password: local-dev-only-redis-password\n  DB: 0\n  PoolSize: 10\n  DialTimeout: 3s\n  ReadTimeout: 3s\n  WriteTimeout: 3s\n  StartupTimeout: 3s\n  ReadinessTimeout: 2s\nReadiness:\n  Timeout: 2s\nStartup:\n  ConnectivityTimeout: 3s\n"
}

type fakeMySQL struct {
	pingErr    error
	closed     bool
	closeCalls int
}

func (f *fakeMySQL) DB() *sql.DB                { return nil }
func (f *fakeMySQL) Ping(context.Context) error { return f.pingErr }
func (f *fakeMySQL) Close() error {
	f.closed = true
	f.closeCalls++
	return nil
}
func (f *fakeMySQL) WithinTransaction(context.Context, func(context.Context, *sql.Tx) error) error {
	return nil
}

type fakeRedis struct {
	pingErr    error
	closed     bool
	closeCalls int
}

func (f *fakeRedis) Client() *redis.Client      { return nil }
func (f *fakeRedis) Ping(context.Context) error { return f.pingErr }
func (f *fakeRedis) Close() error {
	f.closed = true
	f.closeCalls++
	return nil
}
