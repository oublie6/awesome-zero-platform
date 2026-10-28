package httpmiddleware

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zeromicro/go-zero/core/logx"
)

func TestAccessLogEmitsSafeStructuredFields(t *testing.T) {
	var lines bytes.Buffer
	previous := logx.Reset()
	logx.SetWriter(logx.NewWriter(&lines))
	defer func() {
		logx.Reset()
		if previous != nil {
			logx.SetWriter(previous)
		}
	}()

	wrapped := RequestID(RequestIDConfig{HeaderName: "X-Request-Id", MaxLength: 64})(
		AccessLog()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusCreated)
		})),
	)
	req := httptest.NewRequest(http.MethodPost, "/doudizhu/rooms?access_token=must-not-log", nil)
	req.Header.Set("X-Request-Id", "log-observability-test")
	req.Header.Set("Authorization", "Bearer secret-must-not-log")
	wrapped.ServeHTTP(httptest.NewRecorder(), req)

	if bytes.Contains(lines.Bytes(), []byte("must-not-log")) {
		t.Fatalf("sensitive query/header included in log: %s", lines.String())
	}
	var got map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(lines.Bytes()), &got); err != nil {
		t.Fatalf("log json: %v; data=%s", err, lines.String())
	}
	for name, want := range map[string]any{
		"content":     "http access",
		"eventAction": "http.access",
		"module":      "doudizhu",
		"requestId":   "log-observability-test",
		"statusCode":  float64(201),
	} {
		if got[name] != want {
			t.Fatalf("%s = %#v, want %#v", name, got[name], want)
		}
	}
	if _, ok := got["durationMs"]; !ok {
		t.Fatalf("durationMs missing: %s", lines.String())
	}
}
