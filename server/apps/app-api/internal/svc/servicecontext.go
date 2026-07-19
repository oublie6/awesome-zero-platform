// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package svc

import (
	"github.com/oublie6/awesome-zero-platform/server/apps/app-api/internal/config"
	"github.com/oublie6/awesome-zero-platform/server/foundation/cache"
	"github.com/oublie6/awesome-zero-platform/server/foundation/database"
	"github.com/oublie6/awesome-zero-platform/server/foundation/readiness"
)

type ServiceContext struct {
	Config    config.Config
	Postgres  database.Handle
	Redis     cache.Handle
	Readiness *readiness.Checker
}

func NewServiceContext(c config.Config, postgres database.Handle, redis cache.Handle, checker *readiness.Checker) *ServiceContext {
	return &ServiceContext{
		Config:    c,
		Postgres:  postgres,
		Redis:     redis,
		Readiness: checker,
	}
}
