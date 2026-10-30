package httpmiddleware

import (
	"bufio"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/oublie6/awesome-zero-platform/server/foundation/requestid"
	"github.com/zeromicro/go-zero/core/logx"
	"go.opentelemetry.io/otel/propagation"
)

type AccessLogRecord struct {
	RequestID     string        `json:"requestId"`
	Method        string        `json:"method"`
	Path          string        `json:"path"`
	StatusCode    int           `json:"statusCode"`
	Elapsed       time.Duration `json:"elapsed"`
	DurationMs    int64         `json:"durationMs"`
	Module        string        `json:"module"`
	ClientAddress string        `json:"clientAddress"`
}

type statusWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *statusWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Write(body []byte) (int, error) {
	if w.statusCode == 0 {
		w.statusCode = http.StatusOK
	}

	return w.ResponseWriter.Write(body)
}

func (w *statusWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func (w *statusWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, http.ErrNotSupported
	}
	connection, buffered, err := hijacker.Hijack()
	if err == nil {
		w.statusCode = http.StatusSwitchingProtocols
	}
	return connection, buffered, err
}

func (w *statusWriter) Flush() {
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func AccessLog() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			started := time.Now()
			writer := &statusWriter{ResponseWriter: w}

			next.ServeHTTP(writer, r)

			if writer.statusCode == 0 {
				writer.statusCode = http.StatusOK
			}

			elapsed := time.Since(started)
			record := AccessLogRecord{
				RequestID:     requestid.FromContext(r.Context()),
				Method:        r.Method,
				Path:          r.URL.Path,
				StatusCode:    writer.statusCode,
				Elapsed:       elapsed,
				DurationMs:    elapsed.Milliseconds(),
				Module:        accessLogModule(r.URL.Path),
				ClientAddress: clientAddress(r),
			}

			// go-zero's route TraceHandler receives a request copy, so its
			// span context does not reach this outer middleware. It injects
			// that context into response headers; use it for log correlation
			// without creating another span or changing routing/404 behavior.
			logCtx := (propagation.TraceContext{}).Extract(r.Context(), propagation.HeaderCarrier(w.Header()))
			logx.WithContext(logCtx).Infow("http access",
				logx.Field("eventAction", "http.access"),
				logx.Field("module", record.Module),
				logx.Field("durationMs", record.DurationMs),
				logx.Field("requestId", record.RequestID),
				logx.Field("method", record.Method),
				logx.Field("path", record.Path),
				logx.Field("statusCode", record.StatusCode),
				logx.Field("elapsed", record.Elapsed.String()),
				logx.Field("clientAddress", record.ClientAddress),
			)
		})
	}
}

func clientAddress(r *http.Request) string {
	host := r.RemoteAddr
	if host == "" {
		return ""
	}

	return host
}

func accessLogModule(path string) string {
	if path == "/doudizhu" || strings.HasPrefix(path, "/doudizhu/") ||
		path == "/v1/doudizhu" || strings.HasPrefix(path, "/v1/doudizhu/") {
		return "doudizhu"
	}
	return "platform"
}
