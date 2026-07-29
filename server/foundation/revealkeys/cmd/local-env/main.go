package main

import (
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/oublie6/awesome-zero-platform/server/foundation/revealkeys"
)

type variable struct {
	name  string
	value string
}

func main() {
	variables, err := generate(time.Now().UTC(), rand.Reader)
	must(err)
	for _, current := range variables {
		writeExport(current.name, current.value)
	}
}

func generate(now time.Time, random io.Reader) ([]variable, error) {
	revealPrivate, err := ecdh.X25519().GenerateKey(random)
	if err != nil {
		return nil, err
	}
	_, signingPrivate, err := ed25519.GenerateKey(random)
	if err != nil {
		return nil, err
	}
	beaconProofSecret, err := randomBytes(random, 32)
	if err != nil {
		return nil, err
	}
	contributionKey, err := randomBytes(random, 32)
	if err != nil {
		return nil, err
	}

	config := revealkeys.StaticConfig{
		CurrentKeyID:      "local-reveal-v1",
		SignatureKeyID:    "local-signing-v1",
		SigningPrivateKey: base64.RawURLEncoding.EncodeToString(signingPrivate),
		Keys: []revealkeys.StaticKeyConfig{{
			ManifestVersion: 1,
			KeyID:           "local-reveal-v1",
			PublicKey:       base64.RawURLEncoding.EncodeToString(revealPrivate.PublicKey().Bytes()),
			PrivateKey:      base64.RawURLEncoding.EncodeToString(revealPrivate.Bytes()),
			NotBefore:       now.Add(-time.Minute).Format(time.RFC3339Nano),
			ExpiresAt:       now.Add(24 * time.Hour).Format(time.RFC3339Nano),
			Status:          revealkeys.StatusActive,
		}},
	}
	encoded, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	return []variable{
		{name: "APP_REVEAL_KEYS_ENABLED", value: "true"},
		{name: "APP_REVEAL_KEYS_STATIC_JSON", value: string(encoded)},
		{name: "APP_DOUDIZHU_ENABLED", value: "true"},
		{name: "APP_DOUDIZHU_BEACON_PROVIDER", value: "local-hmac"},
		{name: "APP_DOUDIZHU_BEACON_ROUND", value: "local-round-1"},
		{name: "APP_DOUDIZHU_BEACON_PROOF_SECRET", value: base64.RawURLEncoding.EncodeToString(beaconProofSecret)},
		{name: "APP_DOUDIZHU_CONTRIBUTION_KEY_ID", value: "local-contribution-v1"},
		{name: "APP_DOUDIZHU_CONTRIBUTION_KEY_HEX", value: hex.EncodeToString(contributionKey)},
	}, nil
}

func randomBytes(random io.Reader, size int) ([]byte, error) {
	value := make([]byte, size)
	if _, err := io.ReadFull(random, value); err != nil {
		return nil, err
	}
	return value, nil
}

func writeExport(name, value string) {
	fmt.Printf("export %s=%s\n", name, shellQuote(value))
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func must(err error) {
	if err == nil {
		return
	}
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
