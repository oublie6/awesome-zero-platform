package httpmiddleware

import (
	"net/http"
	"time"

	"github.com/oublie6/awesome-zero-platform/server/foundation/requestid"
	"github.com/zeromicro/go-zero/core/logx"
)

type AccessLogRecord struct {
	RequestID     string        `json:"requestId"`
	Method        string        `json:"method"`
	Path          string        `json:"path"`
	StatusCode    int           `json:"statusCode"`
	Elapsed       time.Duration `json:"elapsed"`
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

func AccessLog() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			started := time.Now()
			writer := &statusWriter{ResponseWriter: w}

			next.ServeHTTP(writer, r)

			if writer.statusCode == 0 {
				writer.statusCode = http.StatusOK
			}

			record := AccessLogRecord{
				RequestID:     requestid.FromContext(r.Context()),
				Method:        r.Method,
				Path:          r.URL.Path,
				StatusCode:    writer.statusCode,
				Elapsed:       time.Since(started),
				ClientAddress: clientAddress(r),
			}

			logx.WithContext(r.Context()).Infow("http access",
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
