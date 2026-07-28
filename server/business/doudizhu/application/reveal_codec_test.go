package application

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
)

func TestRevealAADBindsCommandAndSeatContext(t *testing.T) {
	command := Command{
		Version: CommandProtocolV1, Name: CommandHandRevealSubmit, CommandID: "command-1",
		AggregateType: domain.AggregateHand, AggregateID: "hand-1", ClientSeq: 3, ExpectedVersion: 8,
		IssuedAt: time.UnixMilli(1000).UTC(), ExpiresAt: time.UnixMilli(2000).UTC(),
	}
	commitment := domain.Commitment{1}
	base, err := BuildRevealAAD(command, "account-1", domain.SeatOne, commitment, "key-1")
	if err != nil {
		t.Fatal(err)
	}
	changed := command
	changed.ClientSeq++
	other, err := BuildRevealAAD(changed, "account-1", domain.SeatOne, commitment, "key-1")
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(base, other) {
		t.Fatal("AAD did not bind client sequence")
	}
	other, _ = BuildRevealAAD(command, "account-1", domain.SeatTwo, commitment, "key-1")
	if bytes.Equal(base, other) {
		t.Fatal("AAD did not bind seat")
	}
}

func TestDecodeRevealUsesNormalizedPhraseHash(t *testing.T) {
	random := bytes.Repeat([]byte{0x2a}, 32)
	payload, _ := json.Marshal(map[string]any{
		"v": RevealPlaintextV1, "handId": "hand-1", "seat": 1,
		"secureRandom": base64.RawURLEncoding.EncodeToString(random),
		"phrase":       "fullwidth", "normalization": revealNormalizationNFKCV1,
	})
	normalizer := fixedNormalizer{value: "normalized"}
	decoded, err := decodeRevealPlaintext(payload, "hand-1", domain.SeatOne, normalizer, 128)
	if err != nil {
		t.Fatal(err)
	}
	phraseHash := sha256.Sum256([]byte("normalized"))
	var secure [32]byte
	copy(secure[:], random)
	want := computeContributionDigest("hand-1", domain.SeatOne, secure, phraseHash)
	if decoded.Digest != want || decoded.PhraseHash != phraseHash {
		t.Fatalf("decoded=%#v wantDigest=%x", decoded, want)
	}
}

func TestDecodeRevealRejectsUnknownFieldsAndContextMismatch(t *testing.T) {
	random := base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{1}, 32))
	unknown := []byte(`{"v":"fair-doudizhu-reveal-v1","handId":"hand-1","seat":1,"secureRandom":"` + random + `","phrase":"x","normalization":"NFKC-v1","extra":true}`)
	if _, err := decodeRevealPlaintext(unknown, "hand-1", domain.SeatOne, fixedNormalizer{value: "x"}, 128); err == nil {
		t.Fatal("unknown field was accepted")
	}
	mismatch := []byte(`{"v":"fair-doudizhu-reveal-v1","handId":"other","seat":1,"secureRandom":"` + random + `","phrase":"x","normalization":"NFKC-v1"}`)
	if _, err := decodeRevealPlaintext(mismatch, "hand-1", domain.SeatOne, fixedNormalizer{value: "x"}, 128); err == nil {
		t.Fatal("mismatched hand was accepted")
	}
}

type fixedNormalizer struct{ value string }

func (n fixedNormalizer) Normalize(string) (string, error) { return n.value, nil }
