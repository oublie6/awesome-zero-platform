// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package svc

import (
	"github.com/oublie6/awesome-zero-platform/server/apps/app-api/internal/config"
	"github.com/oublie6/awesome-zero-platform/server/foundation/cache"
	"github.com/oublie6/awesome-zero-platform/server/foundation/database"
	"github.com/oublie6/awesome-zero-platform/server/foundation/observability"
	"github.com/oublie6/awesome-zero-platform/server/foundation/readiness"
	"github.com/oublie6/awesome-zero-platform/server/platform/authn"
	"github.com/oublie6/awesome-zero-platform/server/platform/authz"
	"github.com/oublie6/awesome-zero-platform/server/platform/identity"
)

type ServiceContext struct {
	Config        config.Config
	MySQL         database.Handle
	Redis         cache.Handle
	Readiness     *readiness.Checker
	Identity      *identity.Service
	Authn         *authn.Service
	Authz         *authz.Service
	Authorizer    authz.Authorizer
	Metrics       *observability.Metrics
}

func NewServiceContext(c config.Config, mysql database.Handle, redis cache.Handle, checker *readiness.Checker) *ServiceContext {
	var identityService *identity.Service
	if mysql != nil && mysql.DB() != nil {
		identityService = identity.NewService(mysql)
	}

	return &ServiceContext{
		Config:    c,
		MySQL:     mysql,
		Redis:     redis,
		Readiness: checker,
		Identity:  identityService,
	}
}
