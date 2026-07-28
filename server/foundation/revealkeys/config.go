package revealkeys

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
)

type StaticConfig struct {
	CurrentKeyID      string            `json:"currentKeyId"`
	SignatureKeyID    string            `json:"signatureKeyId"`
	SigningPrivateKey string            `json:"signingPrivateKey"`
	Keys              []StaticKeyConfig `json:"keys"`
}

type StaticKeyConfig struct {
	ManifestVersion uint64 `json:"manifestVersion"`
	KeyID           string `json:"keyId"`
	PublicKey       string `json:"publicKey"`
	PrivateKey      string `json:"privateKey"`
	NotBefore       string `json:"notBefore"`
	ExpiresAt       string `json:"expiresAt"`
	Status          Status `json:"status"`
	RetiringAt      string `json:"retiringAt,omitempty"`
	RetireAfter     string `json:"retireAfter,omitempty"`
}

func NewFromJSON(encoded string, clock Clock) (*Registry, error) {
	var config StaticConfig
	decoder := json.NewDecoder(strings.NewReader(encoded))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("%w: decode json: %v", ErrInvalidConfig, err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return nil, ErrInvalidConfig
	}
	signingKey, err := base64.RawURLEncoding.Strict().DecodeString(config.SigningPrivateKey)
	if err != nil || len(signingKey) != ed25519.PrivateKeySize {
		return nil, ErrInvalidConfig
	}
	records := make([]Record, 0, len(config.Keys))
	for _, current := range config.Keys {
		publicKey, err := base64.RawURLEncoding.Strict().DecodeString(current.PublicKey)
		if err != nil {
			return nil, ErrInvalidConfig
		}
		privateKey, err := base64.RawURLEncoding.Strict().DecodeString(current.PrivateKey)
		if err != nil {
			return nil, ErrInvalidConfig
		}
		notBefore, err := time.Parse(time.RFC3339Nano, current.NotBefore)
		if err != nil {
			return nil, ErrInvalidConfig
		}
		expiresAt, err := time.Parse(time.RFC3339Nano, current.ExpiresAt)
		if err != nil {
			return nil, ErrInvalidConfig
		}
		retiringAt, err := parseConfigOptionalTime(current.RetiringAt)
		if err != nil {
			return nil, ErrInvalidConfig
		}
		retireAfter, err := parseConfigOptionalTime(current.RetireAfter)
		if err != nil {
			return nil, ErrInvalidConfig
		}
		records = append(records, Record{ManifestVersion: current.ManifestVersion, KeyID: current.KeyID, PublicKey: publicKey, PrivateKey: privateKey, NotBefore: notBefore, ExpiresAt: expiresAt, Status: current.Status, RetiringAt: retiringAt, RetireAfter: retireAfter})
	}
	return New(records, config.CurrentKeyID, config.SignatureKeyID, ed25519.PrivateKey(signingKey), clock)
}

func parseConfigOptionalTime(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339Nano, value)
}
