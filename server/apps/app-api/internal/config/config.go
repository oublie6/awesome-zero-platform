// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package config

import (
	"time"

	"github.com/oublie6/awesome-zero-platform/server/foundation/cache"
	"github.com/oublie6/awesome-zero-platform/server/foundation/database"
	"github.com/oublie6/awesome-zero-platform/server/platform/realtime"
	"github.com/zeromicro/go-zero/rest"
)

type Config struct {
	rest.RestConf
	HTTP           HTTPConfig
	MySQL          database.Config      `json:",optional"`
	Redis          cache.Config         `json:",optional"`
	Readiness      ReadinessConfig      `json:",optional"`
	Startup        StartupConfig        `json:",optional"`
	Authentication AuthenticationConfig `json:",optional"`
	Authorization  AuthorizationConfig  `json:",optional"`
	Realtime       realtime.Config      `json:",optional"`
	Admin          AdminConfig          `json:",optional"`
	Observability  ObservabilityConfig  `json:",optional"`
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

type AuthenticationConfig struct {
	Enabled           bool
	Issuer            string        `json:",optional"`
	AccessTokenSecret string        `json:",optional,env=APP_AUTH_ACCESS_TOKEN_SECRET"`
	AccessTTL         time.Duration `json:",default=15m"`
	RefreshTTL        time.Duration `json:",default=720h"`
	SessionKeyPrefix  string        `json:",default=authn:session:"`
}

type AuthorizationConfig struct {
	Enabled bool
	Cluster AuthorizationClusterConfig `json:",optional"`
}

type AuthorizationClusterConfig struct {
	Enabled        bool
	InstanceID     string        `json:",optional,env=APP_INSTANCE_ID"`
	Channel        string        `json:",default=awesome-zero-platform:authz:policy-changed"`
	PollInterval   time.Duration `json:",default=20s"`
	PublishTimeout time.Duration `json:",default=2s"`
	ReloadTimeout  time.Duration `json:",default=5s"`
}

type AdminConfig struct {
	Enabled        bool
	BootstrapToken string `json:",optional,env=APP_ADMIN_BOOTSTRAP_TOKEN"`
}

type ObservabilityConfig struct {
	Metrics MetricsConfig `json:",optional"`
}

type MetricsConfig struct {
	Enabled   bool
	Path      string `json:",default=/metrics"`
	Namespace string `json:",default=awesome_zero_platform"`
}
