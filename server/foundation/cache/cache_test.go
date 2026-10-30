package cache

import (
	"fmt"
	"testing"
	"time"
)

func TestConfigValidate(t *testing.T) {
	t.Parallel()

	cfg := Config{
		Addr:             "127.0.0.1:6379",
		DB:               0,
		PoolSize:         4,
		DialTimeout:      time.Second,
		ReadTimeout:      time.Second,
		WriteTimeout:     time.Second,
		StartupTimeout:   time.Second,
		ReadinessTimeout: time.Second,
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestCloseIdempotent(t *testing.T) {
	t.Parallel()

	calls := 0
	resource := &Resource{
		closeFn: func() error {
			calls++
			return nil
		},
	}

	if err := resource.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if err := resource.Close(); err != nil {
		t.Fatalf("second Close() error = %v", err)
	}
	if calls != 1 {
		t.Fatalf("close calls = %d, want 1", calls)
	}
}

func TestModelCacheRejectsNonZeroRedisDB(t *testing.T) {
	cfg := Config{
		Addr:             "127.0.0.1:6379",
		DB:               1,
		PoolSize:         4,
		DialTimeout:      time.Second,
		ReadTimeout:      time.Second,
		WriteTimeout:     time.Second,
		StartupTimeout:   time.Second,
		ReadinessTimeout: time.Second,
		Model: ModelConfig{
			Enabled:     true,
			KeyPrefix:   "test:model",
			TTL:         time.Minute,
			NotFoundTTL: time.Second,
		},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected config validation to reject model cache on Redis DB 1")
	}
	if _, err := NewModelCache(cfg, "identity", fmt.Errorf("missing")); err == nil {
		t.Fatal("expected model cache constructor to reject Redis DB 1")
	}
	cfg.Model.Enabled = false
	if err := cfg.Validate(); err != nil {
		t.Fatalf("native Redis DB 1 must remain valid without model cache: %v", err)
	}
}
