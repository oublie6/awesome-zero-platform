package architecture

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Static REST registrations must stay synchronized with the go-zero API
// inventory. Dynamic /ws and /metrics are configured by runtime settings.
func TestRegisteredRoutesMatchGoZeroAPIInventory(t *testing.T) {
	const prefix = "../apps/app-api/"
	contract, err := os.ReadFile(filepath.Join(prefix, "contracts", "routes.api"))
	if err != nil {
		t.Fatal(err)
	}
	apiRx := regexp.MustCompile(`(?m)^\s*(get|post|put|patch|delete)\s+(/[^\s]+)`)
	declared := make(map[string]bool)
	for _, match := range apiRx.FindAllStringSubmatch(string(contract), -1) {
		key := strings.ToUpper(match[1]) + " " + match[2]
		if declared[key] {
			t.Errorf("duplicate API contract route %s", key)
		}
		declared[key] = true
	}
	if len(declared) < 3 {
		t.Fatalf("incomplete route inventory: %d entries", len(declared))
	}
	registered := make(map[string]bool)
	routeRx := regexp.MustCompile(`Method:\s*http\.Method([A-Z][A-Za-z]+)\s*,\s*Path:\s*(?:"([^"]+)"|keyPathPrefix\s*\+\s*"([^"]+)")`)
	var sources = []string{
		"internal/handler/routes.go",
		"internal/securityhttp/security.go",
		"internal/adminhttp/admin.go",
		"internal/cryptohttp/http.go",
		"internal/doudizhuapi/http.go",
	}
	for _, source := range sources {
		b, err := os.ReadFile(filepath.Join(prefix, source))
		if err != nil {
			t.Fatal(err)
		}
		raw := string(b)
		keyPrefix := ""
		if strings.Contains(source, "cryptohttp/") {
			keyPrefix = "/api/v1/crypto/reveal-keys/"
		}
		for _, match := range routeRx.FindAllStringSubmatch(raw, -1) {
			path := match[2]
			if path == "" {
				path = keyPrefix + match[3]
			}
			key := strings.ToUpper(match[1]) + " " + path
			if registered[key] {
				t.Errorf("duplicate registered route %s", key)
			}
			registered[key] = true
		}
	}
	for key := range registered {
		if !declared[key] {
			t.Errorf("registered route absent from API contract: %s", key)
		}
	}
	for key := range declared {
		if !registered[key] {
			t.Errorf("API contract route absent from registration: %s", key)
		}
	}
	if t.Failed() {
		t.Log(fmt.Sprintf("declared=%d registered=%d", len(declared), len(registered)))
	}
}
