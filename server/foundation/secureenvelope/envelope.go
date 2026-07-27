// Package secureenvelope opens versioned HPKE envelopes produced by trusted clients.
//
// It provides transport confidentiality only. Authentication, authorization,
// replay protection, command idempotency, and business state validation remain
// the responsibility of the calling module.
package secureenvelope

const (
	// VersionV1 identifies the first secure-envelope wire format.
	VersionV1 = "secure-envelope-v1"
	// SuiteV1 is RFC 9180 Base mode with X25519, HKDF-SHA256, and AES-256-GCM.
	SuiteV1 = "hpke-x25519-hkdf-sha256-aes-256-gcm"
	// InfoV1 is the fixed HPKE application information for protocol domain separation.
	InfoV1 = "awesome-zero-platform/secure-envelope/v1"

	X25519KeySize       = 32
	EncapsulatedKeySize = 32
	DefaultMaxPlaintext = 64 * 1024
	MaxKeyIDLength      = 128
)

// Envelope is the JSON-safe HPKE transport envelope. Binary fields use
// unpadded base64url encoding.
type Envelope struct {
	Version         string `json:"version"`
	KeyID           string `json:"keyId"`
	Suite           string `json:"suite"`
	EncapsulatedKey string `json:"encapsulatedKey"`
	Ciphertext      string `json:"ciphertext"`
}
