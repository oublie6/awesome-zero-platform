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
			c.HTTP.CORS.AllowedHeaders = []string{"Content-Type", "Origin", "Accept", c.HTTP.RequestID.HeaderName}
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

	if strings.TrimSpace(c.HTTP.SecurityHeaders.ContentTypeOptions) == "" ||
		strings.TrimSpace(c.HTTP.SecurityHeaders.FrameOptions) == "" ||
		strings.TrimSpace(c.HTTP.SecurityHeaders.ReferrerPolicy) == "" {
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

	if err := c.MySQL.Validate(); err != nil {
		return err
	}

	if err := c.Redis.Validate(); err != nil {
		return err
	}

	return nil
}
