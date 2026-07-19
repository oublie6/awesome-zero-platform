package httpmiddleware

import (
	"net/http"

	"github.com/oublie6/awesome-zero-platform/server/foundation/requestid"
)

type RequestIDConfig struct {
	HeaderName string
	MaxLength  int
}

func RequestID(cfg RequestIDConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			effective := requestid.Effective(r.Header.Get(cfg.HeaderName), cfg.MaxLength)
			w.Header().Set(cfg.HeaderName, effective)
			r.Header.Set(cfg.HeaderName, effective)

			ctx := requestid.IntoContext(r.Context(), effective)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
