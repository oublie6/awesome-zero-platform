package config

import (
	"strings"
	"testing"
)

func TestObservabilityEnvironmentOptIn(t *testing.T) {
	var defaults Config
	defaults.Name = "app-api"
	defaults.Host = "127.0.0.1"
	defaults.Port = 8888
	if err := defaults.ApplyEnvironment(); err != nil {
		t.Fatalf("default environment: %v", err)
	}
	if defaults.Log.Mode == "file" || defaults.Telemetry.Endpoint != "" {
		t.Fatalf("unexpected observability default: log=%q trace=%q", defaults.Log.Mode, defaults.Telemetry.Endpoint)
	}

	t.Run("file and OTLP", func(t *testing.T) {
		t.Setenv("APP_LOG_MODE", "file")
		t.Setenv("APP_LOG_PATH", "/app/logs")
		t.Setenv("APP_TELEMETRY_ENDPOINT", "jaeger:4317")
		cfg := Config{}
		if err := cfg.ApplyEnvironment(); err != nil {
			t.Fatal(err)
		}
		if cfg.Log.Mode != "file" || cfg.Log.Encoding != "json" || cfg.Log.Path != "/app/logs" {
			t.Fatalf("invalid file log config: %#v", cfg.Log)
		}
		if cfg.Telemetry.Endpoint != "jaeger:4317" || cfg.Telemetry.Batcher != "otlpgrpc" ||
			cfg.Telemetry.Name != "app-api" || cfg.Telemetry.Sampler != 1 {
			t.Fatalf("invalid trace config: %#v", cfg.Telemetry)
		}
	})

	t.Run("missing file path", func(t *testing.T) {
		t.Setenv("APP_LOG_MODE", "file")
		t.Setenv("APP_LOG_PATH", "")
		cfg := Config{}
		if err := cfg.ApplyEnvironment(); err == nil || !strings.Contains(err.Error(), "APP_LOG_PATH") {
			t.Fatalf("expected file path error, got %v", err)
		}
	})

	t.Run("invalid mode", func(t *testing.T) {
		t.Setenv("APP_LOG_MODE", "stdout+file")
		cfg := Config{}
		if err := cfg.ApplyEnvironment(); err == nil {
			t.Fatal("expected invalid mode rejection")
		}
	})
}
