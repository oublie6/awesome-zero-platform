// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package config

import "github.com/zeromicro/go-zero/rest"

type Config struct {
	rest.RestConf
	HTTP HTTPConfig
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
