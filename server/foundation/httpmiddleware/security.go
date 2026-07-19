package httpmiddleware

import "net/http"

type SecurityHeadersConfig struct {
	ContentTypeOptions string
	FrameOptions       string
	ReferrerPolicy     string
}

func SecurityHeaders(cfg SecurityHeadersConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Content-Type-Options", cfg.ContentTypeOptions)
			w.Header().Set("X-Frame-Options", cfg.FrameOptions)
			w.Header().Set("Referrer-Policy", cfg.ReferrerPolicy)
			next.ServeHTTP(w, r)
		})
	}
}
