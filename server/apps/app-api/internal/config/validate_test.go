package config

import (
	"testing"
	"time"
)

func TestValidate(t *testing.T) {
	t.Parallel()

	valid := Config{}
	valid.Name = "main-api"
	valid.Host = "127.0.0.1"
	valid.Port = 8888
	valid.MySQL.Addr = "127.0.0.1:3306"
	valid.MySQL.Database = "awesome_zero_platform"
	valid.MySQL.User = "app_local"
	valid.MySQL.Password = "dev-only-password"
	valid.MySQL.ParseTime = true
	valid.Redis.Addr = "127.0.0.1:6379"
	valid.Prepare()

	tests := []struct {
		name    string
		mutate  func(*Config)
		wantErr bool
	}{
		{
			name: "valid config",
		},
		{
			name: "missing name",
			mutate: func(cfg *Config) {
				cfg.Name = ""
			},
			wantErr: true,
		},
		{
			name: "missing host",
			mutate: func(cfg *Config) {
				cfg.Host = ""
			},
			wantErr: true,
		},
		{
			name: "invalid port",
			mutate: func(cfg *Config) {
				cfg.Port = 0
			},
			wantErr: true,
		},
		{
			name: "invalid body limit",
			mutate: func(cfg *Config) {
				cfg.HTTP.MaxBodyBytes = -1
			},
			wantErr: true,
		},
		{
			name: "wildcard cors with credentials",
			mutate: func(cfg *Config) {
				cfg.HTTP.CORS.Enabled = true
				cfg.HTTP.CORS.AllowedOrigins = []string{"*"}
				cfg.HTTP.CORS.AllowedMethods = []string{"GET"}
				cfg.HTTP.CORS.AllowedHeaders = []string{"Content-Type"}
				cfg.HTTP.CORS.AllowCredentials = true
			},
			wantErr: true,
		},
		{
			name: "invalid readiness timeout",
			mutate: func(cfg *Config) {
				cfg.Readiness.Timeout = -1 * time.Second
			},
			wantErr: true,
		},
		{
			name: "mysql parse time explicitly false",
			mutate: func(cfg *Config) {
				cfg.MySQL.ParseTime = false
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := valid
			if tt.mutate != nil {
				tt.mutate(&cfg)
			}
			cfg.Prepare()

			err := cfg.Validate()
			if tt.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}
		})
	}
}
