package config

import "testing"

func TestValidate(t *testing.T) {
	t.Parallel()

	valid := Config{}
	valid.Name = "main-api"
	valid.Host = "127.0.0.1"
	valid.Port = 8888

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
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := valid
			if tt.mutate != nil {
				tt.mutate(&cfg)
			}

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
