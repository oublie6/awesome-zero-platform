package revealkeys

import (
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/oublie6/awesome-zero-platform/server/foundation/secureenvelope"
)

const (
	ManifestProtocolV1  = "reveal-key-manifest-v1"
	MaxIdentifierLength = 128
)

// Manifest is the signed public representation of one X25519 HPKE reveal key.
// Binary values use canonical unpadded Base64URL encoding.
type Manifest struct {
	ManifestVersion uint64 `json:"manifestVersion"`
	ProtocolVersion string `json:"protocolVersion"`
	KeyID           string `json:"keyId"`
	Suite           string `json:"suite"`
	PublicKey       string `json:"publicKey"`
	PublicKeySHA256 string `json:"publicKeySha256"`
	NotBefore       string `json:"notBefore"`
	ExpiresAt       string `json:"expiresAt"`
	Status          Status `json:"status"`
	RetiringAt      string `json:"retiringAt,omitempty"`
	RetireAfter     string `json:"retireAfter,omitempty"`
	SignatureKeyID  string `json:"signatureKeyId"`
	Signature       string `json:"signature"`
}

type unsignedManifest struct {
	ManifestVersion uint64 `json:"manifestVersion"`
	ProtocolVersion string `json:"protocolVersion"`
	KeyID           string `json:"keyId"`
	Suite           string `json:"suite"`
	PublicKey       string `json:"publicKey"`
	PublicKeySHA256 string `json:"publicKeySha256"`
	NotBefore       string `json:"notBefore"`
	ExpiresAt       string `json:"expiresAt"`
	Status          Status `json:"status"`
	RetiringAt      string `json:"retiringAt,omitempty"`
	RetireAfter     string `json:"retireAfter,omitempty"`
	SignatureKeyID  string `json:"signatureKeyId"`
}

func (m Manifest) canonicalUnsigned() unsignedManifest {
	return unsignedManifest{
		ManifestVersion: m.ManifestVersion,
		ProtocolVersion: m.ProtocolVersion,
		KeyID:           m.KeyID,
		Suite:           m.Suite,
		PublicKey:       m.PublicKey,
		PublicKeySHA256: m.PublicKeySHA256,
		NotBefore:       m.NotBefore,
		ExpiresAt:       m.ExpiresAt,
		Status:          m.Status,
		RetiringAt:      m.RetiringAt,
		RetireAfter:     m.RetireAfter,
		SignatureKeyID:  m.SignatureKeyID,
	}
}

// CanonicalBytes returns the exact deterministic bytes covered by Ed25519.
func (m Manifest) CanonicalBytes() ([]byte, error) {
	if err := validateManifestFields(m, false); err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(m.canonicalUnsigned())
	if err != nil {
		return nil, fmt.Errorf("%w: canonical json: %v", ErrInvalidManifest, err)
	}
	return encoded, nil
}

func SignManifest(record Record, signatureKeyID string, privateKey ed25519.PrivateKey) (Manifest, error) {
	if len(privateKey) != ed25519.PrivateKeySize || validateIdentifier(signatureKeyID) != nil {
		return Manifest{}, fmt.Errorf("%w: signing key", ErrInvalidConfig)
	}
	manifest, err := manifestFromRecord(record, signatureKeyID)
	if err != nil {
		return Manifest{}, err
	}
	canonical, err := manifest.CanonicalBytes()
	if err != nil {
		return Manifest{}, err
	}
	manifest.Signature = base64.RawURLEncoding.EncodeToString(ed25519.Sign(privateKey, canonical))
	return manifest, nil
}

func VerifyManifest(manifest Manifest, roots map[string]ed25519.PublicKey) error {
	if err := validateManifestFields(manifest, true); err != nil {
		return err
	}
	root, ok := roots[manifest.SignatureKeyID]
	if !ok || len(root) != ed25519.PublicKeySize {
		return ErrSignatureInvalid
	}
	canonical, err := manifest.CanonicalBytes()
	if err != nil {
		return err
	}
	signature, err := decodeCanonical(manifest.Signature, ed25519.SignatureSize)
	if err != nil || !ed25519.Verify(root, canonical, signature) {
		return ErrSignatureInvalid
	}
	return nil
}

func manifestFromRecord(record Record, signatureKeyID string) (Manifest, error) {
	if err := validateRecord(record); err != nil {
		return Manifest{}, err
	}
	hash := sha256.Sum256(record.PublicKey)
	manifest := Manifest{
		ManifestVersion: record.ManifestVersion,
		ProtocolVersion: ManifestProtocolV1,
		KeyID:           record.KeyID,
		Suite:           secureenvelope.SuiteV1,
		PublicKey:       base64.RawURLEncoding.EncodeToString(record.PublicKey),
		PublicKeySHA256: base64.RawURLEncoding.EncodeToString(hash[:]),
		NotBefore:       formatTime(record.NotBefore),
		ExpiresAt:       formatTime(record.ExpiresAt),
		Status:          record.Status,
		SignatureKeyID:  signatureKeyID,
	}
	if !record.RetiringAt.IsZero() {
		manifest.RetiringAt = formatTime(record.RetiringAt)
	}
	if !record.RetireAfter.IsZero() {
		manifest.RetireAfter = formatTime(record.RetireAfter)
	}
	return manifest, nil
}

func validateManifestFields(manifest Manifest, requireSignature bool) error {
	if manifest.ManifestVersion == 0 || manifest.ProtocolVersion != ManifestProtocolV1 || manifest.Suite != secureenvelope.SuiteV1 {
		return ErrInvalidManifest
	}
	if validateIdentifier(manifest.KeyID) != nil || validateIdentifier(manifest.SignatureKeyID) != nil || !manifest.Status.Valid() {
		return ErrInvalidManifest
	}
	publicKey, err := decodeCanonical(manifest.PublicKey, secureenvelope.X25519KeySize)
	if err != nil {
		return ErrInvalidManifest
	}
	hash, err := decodeCanonical(manifest.PublicKeySHA256, sha256.Size)
	if err != nil {
		return ErrInvalidManifest
	}
	computed := sha256.Sum256(publicKey)
	if subtle.ConstantTimeCompare(hash, computed[:]) != 1 {
		return ErrKeyHashMismatch
	}
	notBefore, err := parseTime(manifest.NotBefore)
	if err != nil {
		return ErrInvalidManifest
	}
	expiresAt, err := parseTime(manifest.ExpiresAt)
	if err != nil || !expiresAt.After(notBefore) {
		return ErrInvalidManifest
	}
	retiringAt, retiringErr := parseOptionalTime(manifest.RetiringAt)
	retireAfter, retireErr := parseOptionalTime(manifest.RetireAfter)
	if retiringErr != nil || retireErr != nil {
		return ErrInvalidManifest
	}
	switch manifest.Status {
	case StatusActive:
		if !retiringAt.IsZero() || !retireAfter.IsZero() {
			return ErrInvalidManifest
		}
	case StatusRetiring:
		if retiringAt.IsZero() || retireAfter.IsZero() || retireAfter.Before(retiringAt) || retireAfter.After(expiresAt) {
			return ErrInvalidManifest
		}
	case StatusRetired, StatusRevoked:
		if !retireAfter.IsZero() && retireAfter.After(expiresAt) {
			return ErrInvalidManifest
		}
	}
	if requireSignature {
		if _, err := decodeCanonical(manifest.Signature, ed25519.SignatureSize); err != nil {
			return ErrInvalidManifest
		}
	}
	return nil
}

func formatTime(value time.Time) string { return value.UTC().Format(time.RFC3339Nano) }

func parseTime(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, ErrInvalidManifest
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil || formatTime(parsed) != value {
		return time.Time{}, ErrInvalidManifest
	}
	return parsed, nil
}

func parseOptionalTime(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, nil
	}
	return parseTime(value)
}

func decodeCanonical(value string, expected int) ([]byte, error) {
	decoded, err := base64.RawURLEncoding.Strict().DecodeString(value)
	if err != nil || len(decoded) != expected || base64.RawURLEncoding.EncodeToString(decoded) != value {
		return nil, ErrInvalidManifest
	}
	return decoded, nil
}

func validateIdentifier(value string) error {
	if value == "" || len(value) > MaxIdentifierLength || strings.TrimSpace(value) != value {
		return ErrInvalidConfig
	}
	for _, current := range value {
		if current < 0x21 || current > 0x7e {
			return ErrInvalidConfig
		}
	}
	return nil
}
