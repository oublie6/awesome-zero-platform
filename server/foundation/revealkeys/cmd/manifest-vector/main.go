package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/oublie6/awesome-zero-platform/server/foundation/revealkeys"
)

type vector struct {
	RootKeyID     string              `json:"rootKeyId"`
	RootPublicKey string              `json:"rootPublicKey"`
	Now           string              `json:"now"`
	Manifest      revealkeys.Manifest `json:"manifest"`
}

func main() {
	if len(os.Args) != 2 {
		panic("usage: manifest-vector OUTPUT")
	}
	seed := make([]byte, ed25519.SeedSize)
	for index := range seed {
		seed[index] = byte(index + 1)
	}
	privateKey := ed25519.NewKeyFromSeed(seed)
	now := time.Date(2026, 7, 28, 4, 0, 0, 0, time.UTC)
	publicKey := make([]byte, 32)
	for index := range publicKey {
		publicKey[index] = byte(index + 40)
	}
	manifest, err := revealkeys.SignManifest(revealkeys.Record{ManifestVersion: 7, KeyID: "reveal-interop-2026-07", PublicKey: publicKey, NotBefore: now.Add(-24 * time.Hour), ExpiresAt: now.Add(30 * 24 * time.Hour), Status: revealkeys.StatusActive}, "root-interop-2026", privateKey)
	if err != nil {
		panic(err)
	}
	encoded, err := json.MarshalIndent(vector{RootKeyID: "root-interop-2026", RootPublicKey: base64.RawURLEncoding.EncodeToString(privateKey.Public().(ed25519.PublicKey)), Now: now.Format(time.RFC3339Nano), Manifest: manifest}, "", "  ")
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(os.Args[1], append(encoded, '\n'), 0o600); err != nil {
		panic(err)
	}
	fmt.Println(os.Args[1])
}
