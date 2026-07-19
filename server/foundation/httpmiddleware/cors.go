package httpmiddleware

import (
	"net/http"
	"slices"
	"strings"

	"github.com/zeromicro/go-zero/rest/httpx"
)

type CORSConfig struct {
	Enabled          bool
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	ExposedHeaders   []string
	AllowCredentials bool
}

type WrappedRouter struct {
	httpx.Router
	middleware func(http.Handler) http.Handler
}

func WrapRouter(router httpx.Router, chain ...func(http.Handler) http.Handler) httpx.Router {
	handler := http.Handler(router)
	for i := len(chain) - 1; i >= 0; i-- {
		handler = chain[i](handler)
	}

	return &WrappedRouter{
		Router: router,
		middleware: func(_ http.Handler) http.Handler {
			return handler
		},
	}
}

func (w *WrappedRouter) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	w.middleware(w.Router).ServeHTTP(writer, request)
}

func CORS(cfg CORSConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !cfg.Enabled {
				next.ServeHTTP(w, r)
				return
			}

			origin := r.Header.Get("Origin")
			if origin == "" {
				next.ServeHTTP(w, r)
				return
			}

			if !originAllowed(cfg.AllowedOrigins, origin) {
				if r.Method == http.MethodOptions {
					w.WriteHeader(http.StatusNoContent)
					return
				}

				next.ServeHTTP(w, r)
				return
			}

			headers := w.Header()
			headers.Set("Access-Control-Allow-Origin", allowedOriginValue(cfg, origin))
			headers.Set("Access-Control-Allow-Methods", strings.Join(cfg.AllowedMethods, ", "))
			headers.Set("Access-Control-Allow-Headers", strings.Join(cfg.AllowedHeaders, ", "))
			if len(cfg.ExposedHeaders) > 0 {
				headers.Set("Access-Control-Expose-Headers", strings.Join(cfg.ExposedHeaders, ", "))
			}
			headers.Add("Vary", "Origin")
			headers.Add("Vary", "Access-Control-Request-Method")
			headers.Add("Vary", "Access-Control-Request-Headers")
			if cfg.AllowCredentials {
				headers.Set("Access-Control-Allow-Credentials", "true")
			}

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func originAllowed(origins []string, origin string) bool {
	if slices.Contains(origins, "*") {
		return true
	}

	for _, allowed := range origins {
		if strings.EqualFold(strings.TrimSpace(allowed), origin) {
			return true
		}
	}

	return false
}

func allowedOriginValue(cfg CORSConfig, origin string) string {
	if slices.Contains(cfg.AllowedOrigins, "*") && !cfg.AllowCredentials {
		return "*"
	}

	return origin
}
