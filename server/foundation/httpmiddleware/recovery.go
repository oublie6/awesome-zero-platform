package httpmiddleware

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/oublie6/awesome-zero-platform/server/foundation/apperrors"
	"github.com/oublie6/awesome-zero-platform/server/foundation/requestid"
	platformresponse "github.com/oublie6/awesome-zero-platform/server/foundation/response"
	"github.com/zeromicro/go-zero/core/logx"
)

func Recovery() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if recovered := recover(); recovered != nil {
					stack := debug.Stack()
					logx.WithContext(r.Context()).Errorf(
						"panic recovered: requestId=%s panic=%v stack=%s",
						requestid.FromContext(r.Context()),
						recovered,
						string(stack),
					)
					platformresponse.WriteError(r.Context(), w, apperrors.Internal(fmt.Errorf("panic: %v", recovered)))
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
