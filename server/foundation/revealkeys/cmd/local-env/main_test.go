package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"strings"
	"testing"
	"time"

	"github.com/oublie6/awesome-zero-platform/server/foundation/revealkeys"
)

func TestGenerateLocalEnvironment(t *testing.T) {
	now := time.Date(2026, 7, 29, 10, 0, 0, 0, time.UTC)
	variables, err := generate(now, bytes.NewReader(bytes.Repeat([]byte{0x42}, 256)))
	if err != nil {
		t.Fatal(err)
	}

	values := make(map[string]string, len(variables))
	for _, current := range variables {
		if current.name == "" || current.value == "" {
			t.Fatalf("empty generated variable: %#v", current)
		}
		if _, duplicate := values[current.name]; duplicate {
			t.Fatalf("duplicate variable %q", current.name)
		}
		values[current.name] = current.value
	}

	if values["APP_REVEAL_KEYS_ENABLED"] != "true" || values["APP_DOUDIZHU_ENABLED"] != "true" {
		t.Fatalf("feature flags=%#v", values)
	}
	registry, err := revealkeys.NewFromJSON(values["APP_REVEAL_KEYS_STATIC_JSON"], func() time.Time { return now })
	if err != nil {
		t.Fatalf("load generated reveal key registry: %v", err)
	}
	manifest, err := registry.Current(context.Background())
	if err != nil {
		t.Fatalf("load generated current manifest: %v", err)
	}
	if manifest.KeyID != "local-reveal-v1" {
		t.Fatalf("manifest key id=%q", manifest.KeyID)
	}

	proof, err := base64.RawURLEncoding.Strict().DecodeString(values["APP_DOUDIZHU_BEACON_PROOF_SECRET"])
	if err != nil || len(proof) != 32 {
		t.Fatalf("beacon proof secret length=%d err=%v", len(proof), err)
	}
	contribution, err := hex.DecodeString(values["APP_DOUDIZHU_CONTRIBUTION_KEY_HEX"])
	if err != nil || len(contribution) != 32 || strings.ToLower(values["APP_DOUDIZHU_CONTRIBUTION_KEY_HEX"]) != values["APP_DOUDIZHU_CONTRIBUTION_KEY_HEX"] {
		t.Fatalf("contribution key length=%d err=%v", len(contribution), err)
	}
	if values["APP_DOUDIZHU_BEACON_PROVIDER"] == "" || values["APP_DOUDIZHU_BEACON_ROUND"] == "" || values["APP_DOUDIZHU_CONTRIBUTION_KEY_ID"] == "" {
		t.Fatalf("missing non-secret identifiers: %#v", values)
	}
}

func TestShellQuote(t *testing.T) {
	if got := shellQuote("a'b"); got != `'a'"'"'b'` {
		t.Fatalf("shellQuote=%q", got)
	}
}
