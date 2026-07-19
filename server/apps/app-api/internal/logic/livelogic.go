// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package logic

import (
	"context"

	"github.com/oublie6/awesome-zero-platform/server/apps/app-api/internal/svc"
	"github.com/oublie6/awesome-zero-platform/server/apps/app-api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type LiveLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewLiveLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LiveLogic {
	return &LiveLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *LiveLogic) Live() (resp *types.LiveReply, err error) {
	return &types.LiveReply{
		Status: "ok",
	}, nil
}
