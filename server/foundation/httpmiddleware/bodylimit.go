package httpmiddleware

import (
	"net/http"

	"github.com/oublie6/awesome-zero-platform/server/foundation/apperrors"
	platformresponse "github.com/oublie6/awesome-zero-platform/server/foundation/response"
)

func BodyLimit(limit int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if limit > 0 && r.ContentLength > limit {
				platformresponse.WriteError(r.Context(), w, apperrors.RequestTooLarge())
				return
			}

			if limit > 0 {
				r.Body = http.MaxBytesReader(w, r.Body, limit)
			}

			next.ServeHTTP(w, r)
		})
	}
}
