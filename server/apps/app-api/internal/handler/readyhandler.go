// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package handler

import (
	"net/http"

	healthlogic "github.com/oublie6/awesome-zero-platform/server/apps/app-api/internal/logic/health"
	"github.com/oublie6/awesome-zero-platform/server/apps/app-api/internal/svc"
	platformresponse "github.com/oublie6/awesome-zero-platform/server/foundation/response"
)

func readyHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status, resp := healthlogic.Ready(r.Context(), svcCtx)
		platformresponse.WriteJSON(r.Context(), w, status, resp)
	}
}
