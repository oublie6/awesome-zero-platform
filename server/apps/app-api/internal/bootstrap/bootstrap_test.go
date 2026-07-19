package bootstrap

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewRejectsInvalidConfig(t *testing.T) {
	t.Parallel()

	configPath := writeConfig(t, "Name: main-api\nHost: 127.0.0.1\nPort: 0\n")

	_, err := New(configPath)
	if err == nil {
		t.Fatal("expected invalid config error, got nil")
	}
}

func writeConfig(t *testing.T, contents string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "main-api.yaml")
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	return path
}
