package main

import (
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/oublie6/awesome-zero-platform/server/foundation/revealkeys"
)

func main() {
	now := time.Now().UTC()
	revealPrivate, err := ecdh.X25519().GenerateKey(rand.Reader)
	must(err)
	_, signingPrivate, err := ed25519.GenerateKey(rand.Reader)
	must(err)
	beaconProofSecret := randomBytes(32)
	contributionKey := randomBytes(32)

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
	must(err)

	writeExport("APP_REVEAL_KEYS_ENABLED", "true")
	writeExport("APP_REVEAL_KEYS_STATIC_JSON", string(encoded))
	writeExport("APP_DOUDIZHU_ENABLED", "true")
	writeExport("APP_DOUDIZHU_BEACON_PROVIDER", "local-hmac")
	writeExport("APP_DOUDIZHU_BEACON_ROUND", "local-round-1")
	writeExport("APP_DOUDIZHU_BEACON_PROOF_SECRET", base64.RawURLEncoding.EncodeToString(beaconProofSecret))
	writeExport("APP_DOUDIZHU_CONTRIBUTION_KEY_ID", "local-contribution-v1")
	writeExport("APP_DOUDIZHU_CONTRIBUTION_KEY_HEX", hex.EncodeToString(contributionKey))
}

func randomBytes(size int) []byte {
	value := make([]byte, size)
	_, err := rand.Read(value)
	must(err)
	return value
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
