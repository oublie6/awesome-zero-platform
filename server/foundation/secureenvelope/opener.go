package secureenvelope

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/cloudflare/circl/hpke"
)

const aesGCMTagSize = 16

// Opener validates and decrypts secure-envelope-v1 messages.
type Opener struct {
	keys               PrivateKeyProvider
	maxPlaintextLength int
}

// NewOpener constructs an opener with the default 64 KiB plaintext limit.
func NewOpener(keys PrivateKeyProvider) (*Opener, error) {
	return NewOpenerWithLimit(keys, DefaultMaxPlaintext)
}

// NewOpenerWithLimit constructs an opener with an explicit positive plaintext limit.
func NewOpenerWithLimit(keys PrivateKeyProvider, maxPlaintextLength int) (*Opener, error) {
	if keys == nil || maxPlaintextLength <= 0 {
		return nil, fmt.Errorf("%w: opener configuration", ErrInvalidEnvelope)
	}
	return &Opener{keys: keys, maxPlaintextLength: maxPlaintextLength}, nil
}

// Open authenticates aad and returns the decrypted plaintext. It deliberately
// maps all cryptographic failures to ErrOpenFailed so transport handlers do not
// disclose whether a key, encapsulation, tag, or plaintext was almost valid.
func (o *Opener) Open(ctx context.Context, envelope Envelope, aad []byte) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := validateEnvelopeMetadata(envelope); err != nil {
		return nil, err
	}

	enc, err := decodeField(envelope.EncapsulatedKey, EncapsulatedKeySize, EncapsulatedKeySize)
	if err != nil {
		return nil, ErrInvalidEnvelope
	}
	ciphertext, err := decodeField(
		envelope.Ciphertext,
		aesGCMTagSize,
		o.maxPlaintextLength+aesGCMTagSize,
	)
	if err != nil {
		return nil, ErrInvalidEnvelope
	}

	privateKey, err := o.keys.PrivateKey(ctx, envelope.KeyID)
	if err != nil {
		if errors.Is(err, ErrKeyUnavailable) {
			return nil, ErrKeyUnavailable
		}
		return nil, err
	}
	defer clear(privateKey)
	if len(privateKey) != X25519KeySize {
		return nil, ErrOpenFailed
	}

	suite := hpke.NewSuite(
		hpke.KEM_X25519_HKDF_SHA256,
		hpke.KDF_HKDF_SHA256,
		hpke.AEAD_AES256GCM,
	)
	sk, err := hpke.KEM_X25519_HKDF_SHA256.Scheme().UnmarshalBinaryPrivateKey(privateKey)
	if err != nil {
		return nil, ErrOpenFailed
	}
	receiver, err := suite.NewReceiver(sk, []byte(InfoV1))
	if err != nil {
		return nil, ErrOpenFailed
	}
	contextOpener, err := receiver.Setup(enc)
	if err != nil {
		return nil, ErrOpenFailed
	}
	plaintext, err := contextOpener.Open(ciphertext, aad)
	if err != nil || len(plaintext) > o.maxPlaintextLength {
		clear(plaintext)
		return nil, ErrOpenFailed
	}
	return plaintext, nil
}

func validateEnvelopeMetadata(envelope Envelope) error {
	if envelope.Version != VersionV1 || envelope.Suite != SuiteV1 {
		return ErrInvalidEnvelope
	}
	if err := validateKeyID(envelope.KeyID); err != nil {
		return ErrInvalidEnvelope
	}
	if envelope.EncapsulatedKey == "" || envelope.Ciphertext == "" {
		return ErrInvalidEnvelope
	}
	return nil
}

func decodeField(value string, minLength, maxLength int) ([]byte, error) {
	decoded, err := base64.RawURLEncoding.Strict().DecodeString(value)
	if err != nil || len(decoded) < minLength || len(decoded) > maxLength {
		return nil, ErrInvalidEnvelope
	}
	return decoded, nil
}
