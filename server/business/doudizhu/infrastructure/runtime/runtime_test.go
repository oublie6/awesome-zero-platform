package runtime

import (
	"strings"
	"testing"
)

func TestUUIDv7Generator(t *testing.T) {
	value, err := (UUIDv7Generator{}).NewID()
	if err != nil {
		t.Fatal(err)
	}
	if len(value) != 36 || value[14] != '7' || !strings.Contains("89ab", strings.ToLower(string(value[19]))) {
		t.Fatalf("UUIDv7=%q", value)
	}
}
