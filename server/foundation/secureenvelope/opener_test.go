package secureenvelope

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/cloudflare/circl/hpke"
)

func TestOpenerRoundTrip(t *testing.T) {
	t.Parallel()

	envelope, privateKey, aad, plaintext := makeEnvelope(t)
	keyring, err := NewStaticKeyring(map[string][]byte{envelope.KeyID: privateKey})
	if err != nil {
		t.Fatal(err)
	}
	opener, err := NewOpener(keyring)
	if err != nil {
		t.Fatal(err)
	}

	opened, err := opener.Open(context.Background(), envelope, aad)
	if err != nil {
		t.Fatal(err)
	}
	if string(opened) != string(plaintext) {
		t.Fatalf("opened plaintext = %q, want %q", opened, plaintext)
	}
}

func TestOpenerRejectsTampering(t *testing.T) {
	t.Parallel()

	envelope, privateKey, aad, _ := makeEnvelope(t)
	keyring, err := NewStaticKeyring(map[string][]byte{envelope.KeyID: privateKey})
	if err != nil {
		t.Fatal(err)
	}
	opener, err := NewOpener(keyring)
	if err != nil {
		t.Fatal(err)
	}

	cases := map[string]func(Envelope) (Envelope, []byte){
		"aad": func(candidate Envelope) (Envelope, []byte) {
			return candidate, append(append([]byte(nil), aad...), 0x01)
		},
		"ciphertext": func(candidate Envelope) (Envelope, []byte) {
			decoded, decodeErr := base64.RawURLEncoding.DecodeString(candidate.Ciphertext)
			if decodeErr != nil {
				t.Fatal(decodeErr)
			}
			decoded[len(decoded)-1] ^= 0x01
			candidate.Ciphertext = base64.RawURLEncoding.EncodeToString(decoded)
			return candidate, aad
		},
		"encapsulation": func(candidate Envelope) (Envelope, []byte) {
			decoded, decodeErr := base64.RawURLEncoding.DecodeString(candidate.EncapsulatedKey)
			if decodeErr != nil {
				t.Fatal(decodeErr)
			}
			decoded[0] ^= 0x01
			candidate.EncapsulatedKey = base64.RawURLEncoding.EncodeToString(decoded)
			return candidate, aad
		},
	}

	for name, mutate := range cases {
		name, mutate := name, mutate
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			candidate, candidateAAD := mutate(envelope)
			_, openErr := opener.Open(context.Background(), candidate, candidateAAD)
			if !errors.Is(openErr, ErrOpenFailed) {
				t.Fatalf("Open() error = %v, want ErrOpenFailed", openErr)
			}
		})
	}
}

func TestOpenerRejectsInvalidMetadataAndLimits(t *testing.T) {
	t.Parallel()

	envelope, privateKey, aad, _ := makeEnvelope(t)
	keyring, err := NewStaticKeyring(map[string][]byte{envelope.KeyID: privateKey})
	if err != nil {
		t.Fatal(err)
	}
	opener, err := NewOpenerWithLimit(keyring, 4)
	if err != nil {
		t.Fatal(err)
	}

	invalidVersion := envelope
	invalidVersion.Version = "secure-envelope-v2"
	if _, err = opener.Open(context.Background(), invalidVersion, aad); !errors.Is(err, ErrInvalidEnvelope) {
		t.Fatalf("version error = %v, want ErrInvalidEnvelope", err)
	}

	invalidSuite := envelope
	invalidSuite.Suite = "custom"
	if _, err = opener.Open(context.Background(), invalidSuite, aad); !errors.Is(err, ErrInvalidEnvelope) {
		t.Fatalf("suite error = %v, want ErrInvalidEnvelope", err)
	}

	if _, err = opener.Open(context.Background(), envelope, aad); !errors.Is(err, ErrInvalidEnvelope) {
		t.Fatalf("limit error = %v, want ErrInvalidEnvelope", err)
	}

	unknownKey := envelope
	unknownKey.KeyID = "missing"
	if _, err = opener.Open(context.Background(), unknownKey, aad); !errors.Is(err, ErrKeyUnavailable) {
		t.Fatalf("key error = %v, want ErrKeyUnavailable", err)
	}
}

func TestStaticKeyringDefensivelyCopiesKeys(t *testing.T) {
	t.Parallel()

	original := make([]byte, X25519KeySize)
	original[0] = 7
	keyring, err := NewStaticKeyring(map[string][]byte{"key-1": original})
	if err != nil {
		t.Fatal(err)
	}
	original[0] = 9

	first, err := keyring.PrivateKey(context.Background(), "key-1")
	if err != nil {
		t.Fatal(err)
	}
	if first[0] != 7 {
		t.Fatalf("stored key was mutated: %d", first[0])
	}
	first[0] = 11
	second, err := keyring.PrivateKey(context.Background(), "key-1")
	if err != nil {
		t.Fatal(err)
	}
	if second[0] != 7 {
		t.Fatalf("returned key was not copied: %d", second[0])
	}
}

func TestTypeScriptInteropVector(t *testing.T) {
	path := os.Getenv("SECURE_ENVELOPE_INTEROP_VECTOR")
	if path == "" {
		t.Skip("SECURE_ENVELOPE_INTEROP_VECTOR is not set")
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var vector struct {
		RecipientPrivateKey string   `json:"recipientPrivateKey"`
		AAD                 string   `json:"aad"`
		Plaintext           string   `json:"plaintext"`
		Envelope            Envelope `json:"envelope"`
	}
	if err = json.Unmarshal(contents, &vector); err != nil {
		t.Fatal(err)
	}
	privateKey, err := base64.RawURLEncoding.DecodeString(vector.RecipientPrivateKey)
	if err != nil {
		t.Fatal(err)
	}
	aad, err := base64.RawURLEncoding.DecodeString(vector.AAD)
	if err != nil {
		t.Fatal(err)
	}
	plaintext, err := base64.RawURLEncoding.DecodeString(vector.Plaintext)
	if err != nil {
		t.Fatal(err)
	}
	keyring, err := NewStaticKeyring(map[string][]byte{vector.Envelope.KeyID: privateKey})
	if err != nil {
		t.Fatal(err)
	}
	opener, err := NewOpener(keyring)
	if err != nil {
		t.Fatal(err)
	}
	opened, err := opener.Open(context.Background(), vector.Envelope, aad)
	if err != nil {
		t.Fatal(err)
	}
	if string(opened) != string(plaintext) {
		t.Fatalf("interop plaintext = %x, want %x", opened, plaintext)
	}
}

func makeEnvelope(t *testing.T) (Envelope, []byte, []byte, []byte) {
	t.Helper()

	suite := hpke.NewSuite(
		hpke.KEM_X25519_HKDF_SHA256,
		hpke.KDF_HKDF_SHA256,
		hpke.AEAD_AES256GCM,
	)
	publicKey, privateKey, err := hpke.KEM_X25519_HKDF_SHA256.Scheme().GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	privateKeyBytes, err := privateKey.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	sender, err := suite.NewSender(publicKey, []byte(InfoV1))
	if err != nil {
		t.Fatal(err)
	}
	enc, sealer, err := sender.Setup(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	aad := []byte("game=game-1;hand=hand-1;seat=0;command=cmd-1")
	plaintext := []byte(`{"secureRandom":"test-random","phraseRaw":"test phrase"}`)
	ciphertext, err := sealer.Seal(plaintext, aad)
	if err != nil {
		t.Fatal(err)
	}
	return Envelope{
		Version:         VersionV1,
		KeyID:           "test-key-1",
		Suite:           SuiteV1,
		EncapsulatedKey: base64.RawURLEncoding.EncodeToString(enc),
		Ciphertext:      base64.RawURLEncoding.EncodeToString(ciphertext),
	}, privateKeyBytes, aad, plaintext
}
