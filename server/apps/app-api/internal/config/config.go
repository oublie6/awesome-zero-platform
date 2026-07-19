// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package config

import (
	"time"

	"github.com/oublie6/awesome-zero-platform/server/foundation/cache"
	"github.com/oublie6/awesome-zero-platform/server/foundation/database"
	"github.com/zeromicro/go-zero/rest"
)

type Config struct {
	rest.RestConf
	HTTP      HTTPConfig
	Postgres  database.Config `json:",optional"`
	Redis     cache.Config    `json:",optional"`
	Readiness ReadinessConfig `json:",optional"`
	Startup   StartupConfig   `json:",optional"`
}

type HTTPConfig struct {
	RequestID       RequestIDConfig       `json:",optional"`
	SecurityHeaders SecurityHeadersConfig `json:",optional"`
	CORS            CORSConfig            `json:",optional"`
	MaxBodyBytes    int64                 `json:",default=1048576"`
}

type RequestIDConfig struct {
	HeaderName string `json:",default=X-Request-Id"`
	MaxLength  int    `json:",default=64"`
}

type SecurityHeadersConfig struct {
	ContentTypeOptions string `json:",default=nosniff"`
	FrameOptions       string `json:",default=DENY"`
	ReferrerPolicy     string `json:",default=no-referrer"`
}

type CORSConfig struct {
	Enabled          bool
	AllowedOrigins   []string `json:",optional"`
	AllowedMethods   []string `json:",optional"`
	AllowedHeaders   []string `json:",optional"`
	ExposedHeaders   []string `json:",optional"`
	AllowCredentials bool
}

type ReadinessConfig struct {
	Timeout time.Duration `json:",default=2s"`
}

type StartupConfig struct {
	ConnectivityTimeout time.Duration `json:",default=3s"`
}
