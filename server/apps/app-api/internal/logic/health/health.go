package health

import (
	"context"
	"net/http"

	"github.com/oublie6/awesome-zero-platform/server/apps/app-api/internal/svc"
	"github.com/oublie6/awesome-zero-platform/server/apps/app-api/internal/types"
)

func Live(context.Context, *svc.ServiceContext) (int, *types.LiveReply) {
	return http.StatusOK, &types.LiveReply{
		Status: "ok",
	}
}

func Ready(ctx context.Context, svcCtx *svc.ServiceContext) (int, *types.ReadyReply) {
	if svcCtx != nil && svcCtx.Readiness != nil && svcCtx.Readiness.Check(ctx).Ready {
		return http.StatusOK, &types.ReadyReply{
			Status: "ready",
		}
	}

	return http.StatusServiceUnavailable, &types.ReadyReply{
		Status: "unready",
	}
}
