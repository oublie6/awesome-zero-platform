// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package handler

import (
	"net/http"

	"github.com/oublie6/awesome-zero-platform/server/apps/app-api/internal/logic"
	"github.com/oublie6/awesome-zero-platform/server/apps/app-api/internal/svc"
	platformresponse "github.com/oublie6/awesome-zero-platform/server/foundation/response"
)

func readyHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := logic.NewReadyLogic(r.Context(), svcCtx)
		resp, err := l.Ready()
		if err != nil {
			platformresponse.WriteError(r.Context(), w, err)
		} else {
			platformresponse.WriteJSON(r.Context(), w, http.StatusOK, resp)
		}
	}
}
