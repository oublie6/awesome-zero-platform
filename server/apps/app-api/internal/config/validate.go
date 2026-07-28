package config

import (
	"fmt"
	"slices"
	"strings"
	"time"
)

func (c *Config) Prepare() {
	c.Middlewares.Log = false
	c.Middlewares.Recover = false
	c.Middlewares.MaxBytes = false

	if c.HTTP.RequestID.HeaderName == "" {
		c.HTTP.RequestID.HeaderName = "X-Request-Id"
	}
	if c.HTTP.RequestID.MaxLength == 0 {
		c.HTTP.RequestID.MaxLength = 64
	}
	if c.HTTP.MaxBodyBytes == 0 {
		c.HTTP.MaxBodyBytes = 1048576
	}
	if c.HTTP.SecurityHeaders.ContentTypeOptions == "" {
		c.HTTP.SecurityHeaders.ContentTypeOptions = "nosniff"
	}
	if c.HTTP.SecurityHeaders.FrameOptions == "" {
		c.HTTP.SecurityHeaders.FrameOptions = "DENY"
	}
	if c.HTTP.SecurityHeaders.ReferrerPolicy == "" {
		c.HTTP.SecurityHeaders.ReferrerPolicy = "no-referrer"
	}
	if c.HTTP.CORS.Enabled {
		if len(c.HTTP.CORS.AllowedMethods) == 0 {
			c.HTTP.CORS.AllowedMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
		}
		if len(c.HTTP.CORS.AllowedHeaders) == 0 {
			c.HTTP.CORS.AllowedHeaders = []string{"Content-Type", "Origin", "Accept", "Authorization", c.HTTP.RequestID.HeaderName}
		}
		if len(c.HTTP.CORS.ExposedHeaders) == 0 {
			c.HTTP.CORS.ExposedHeaders = []string{c.HTTP.RequestID.HeaderName}
		}
	}
	if c.Readiness.Timeout == 0 {
		c.Readiness.Timeout = 2 * time.Second
	}
	if c.Startup.ConnectivityTimeout == 0 {
		c.Startup.ConnectivityTimeout = 3 * time.Second
	}
	if c.Authentication.Enabled {
		if c.Authentication.Issuer == "" {
			c.Authentication.Issuer = c.Name
		}
		if c.Authentication.AccessTTL == 0 {
			c.Authentication.AccessTTL = 15 * time.Minute
		}
		if c.Authentication.RefreshTTL == 0 {
			c.Authentication.RefreshTTL = 30 * 24 * time.Hour
		}
		if c.Authentication.SessionKeyPrefix == "" {
			c.Authentication.SessionKeyPrefix = "authn:session:"
		}
	}
	if c.Authorization.Cluster.Enabled {
		if strings.TrimSpace(c.Authorization.Cluster.Channel) == "" {
			c.Authorization.Cluster.Channel = "awesome-zero-platform:authz:policy-changed"
		}
		if c.Authorization.Cluster.PollInterval == 0 {
			c.Authorization.Cluster.PollInterval = 20 * time.Second
		}
		if c.Authorization.Cluster.PublishTimeout == 0 {
			c.Authorization.Cluster.PublishTimeout = 2 * time.Second
		}
		if c.Authorization.Cluster.ReloadTimeout == 0 {
			c.Authorization.Cluster.ReloadTimeout = 5 * time.Second
		}
	}
	if c.Observability.Metrics.Enabled {
		if c.Observability.Metrics.Path == "" {
			c.Observability.Metrics.Path = "/metrics"
		}
		if c.Observability.Metrics.Namespace == "" {
			c.Observability.Metrics.Namespace = "awesome_zero_platform"
		}
	}
	c.MySQL.Prepare()
	c.Redis.Prepare()
	c.MySQL.StartupTimeout = c.Startup.ConnectivityTimeout
	c.Redis.StartupTimeout = c.Startup.ConnectivityTimeout
	c.MySQL.ReadinessTimeout = c.Readiness.Timeout
	c.Redis.ReadinessTimeout = c.Readiness.Timeout
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.Name) == "" {
		return fmt.Errorf("name must not be empty")
	}
	if strings.TrimSpace(c.Host) == "" {
		return fmt.Errorf("host must not be empty")
	}
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}
	if strings.TrimSpace(c.HTTP.RequestID.HeaderName) == "" {
		return fmt.Errorf("http.requestID.headerName must not be empty")
	}
	if c.HTTP.RequestID.MaxLength < 1 || c.HTTP.RequestID.MaxLength > 256 {
		return fmt.Errorf("http.requestID.maxLength must be between 1 and 256")
	}
	if c.HTTP.MaxBodyBytes < 1 {
		return fmt.Errorf("http.maxBodyBytes must be greater than 0")
	}
	if strings.TrimSpace(c.HTTP.SecurityHeaders.ContentTypeOptions) == "" || strings.TrimSpace(c.HTTP.SecurityHeaders.FrameOptions) == "" || strings.TrimSpace(c.HTTP.SecurityHeaders.ReferrerPolicy) == "" {
		return fmt.Errorf("http.securityHeaders values must not be empty")
	}
	if c.HTTP.CORS.Enabled {
		if len(c.HTTP.CORS.AllowedOrigins) == 0 {
			return fmt.Errorf("http.cors.allowedOrigins must not be empty when cors is enabled")
		}
		if len(c.HTTP.CORS.AllowedMethods) == 0 {
			return fmt.Errorf("http.cors.allowedMethods must not be empty when cors is enabled")
		}
		if len(c.HTTP.CORS.AllowedHeaders) == 0 {
			return fmt.Errorf("http.cors.allowedHeaders must not be empty when cors is enabled")
		}
		if c.HTTP.CORS.AllowCredentials && slices.Contains(c.HTTP.CORS.AllowedOrigins, "*") {
			return fmt.Errorf("http.cors.allowedOrigins must not contain * when credentials are enabled")
		}
	}
	if c.Readiness.Timeout <= 0 {
		return fmt.Errorf("readiness.timeout must be greater than 0")
	}
	if c.Startup.ConnectivityTimeout <= 0 {
		return fmt.Errorf("startup.connectivityTimeout must be greater than 0")
	}
	if c.Authentication.Enabled {
		if strings.TrimSpace(c.Authentication.Issuer) == "" {
			return fmt.Errorf("authentication.issuer must not be empty")
		}
		if len(strings.TrimSpace(c.Authentication.AccessTokenSecret)) < 32 {
			return fmt.Errorf("authentication.accessTokenSecret must contain at least 32 characters")
		}
		if c.Authentication.AccessTTL <= 0 {
			return fmt.Errorf("authentication.accessTTL must be greater than 0")
		}
		if c.Authentication.RefreshTTL <= c.Authentication.AccessTTL {
			return fmt.Errorf("authentication.refreshTTL must be greater than accessTTL")
		}
		if strings.TrimSpace(c.Authentication.SessionKeyPrefix) == "" {
			return fmt.Errorf("authentication.sessionKeyPrefix must not be empty")
		}
	}
	if c.Authorization.Enabled && !c.Authentication.Enabled {
		return fmt.Errorf("authorization requires authentication to be enabled")
	}
	if c.Authorization.Cluster.Enabled {
		if !c.Authorization.Enabled {
			return fmt.Errorf("authorization.cluster requires authorization to be enabled")
		}
		if strings.TrimSpace(c.Authorization.Cluster.Channel) == "" {
			return fmt.Errorf("authorization.cluster.channel must not be empty")
		}
		if c.Authorization.Cluster.PollInterval <= 0 {
			return fmt.Errorf("authorization.cluster.pollInterval must be greater than 0")
		}
		if c.Authorization.Cluster.PublishTimeout <= 0 {
			return fmt.Errorf("authorization.cluster.publishTimeout must be greater than 0")
		}
		if c.Authorization.Cluster.ReloadTimeout <= 0 {
			return fmt.Errorf("authorization.cluster.reloadTimeout must be greater than 0")
		}
	}
	if c.Admin.Enabled && (!c.Authentication.Enabled || !c.Authorization.Enabled) {
		return fmt.Errorf("admin requires authentication and authorization to be enabled")
	}
	if token := strings.TrimSpace(c.Admin.BootstrapToken); token != "" && len(token) < 32 {
		return fmt.Errorf("admin.bootstrapToken must contain at least 32 characters when configured")
	}
	if c.RevealKeys.Enabled && strings.TrimSpace(c.RevealKeys.StaticJSON) == "" {
		return fmt.Errorf("revealKeys.staticJSON must not be empty when reveal key publication is enabled")
	}
	if c.Observability.Metrics.Enabled {
		if !strings.HasPrefix(c.Observability.Metrics.Path, "/") || strings.ContainsAny(c.Observability.Metrics.Path, " \t\r\n") {
			return fmt.Errorf("observability.metrics.path must be an absolute path without whitespace")
		}
		if strings.TrimSpace(c.Observability.Metrics.Namespace) == "" {
			return fmt.Errorf("observability.metrics.namespace must not be empty")
		}
	}
	if err := c.MySQL.Validate(); err != nil {
		return err
	}
	if err := c.Redis.Validate(); err != nil {
		return err
	}
	return nil
}
