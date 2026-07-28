package application

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
)

const revealNormalizationNFKCV1 = "NFKC-v1"

type revealPlaintext struct {
	Version       string `json:"v"`
	HandID        string `json:"handId"`
	Seat          uint8  `json:"seat"`
	SecureRandom  string `json:"secureRandom"`
	Phrase        string `json:"phrase"`
	Normalization string `json:"normalization"`
}

type decodedReveal struct {
	SecureRandom [32]byte
	PhraseHash   [32]byte
	Digest       domain.ContributionDigest
}

func BuildRevealAAD(command Command, actor domain.AccountID, seat domain.Seat, commitment domain.Commitment, revealKeyID string) ([]byte, error) {
	if command.Name != CommandHandRevealSubmit || command.AggregateType != domain.AggregateHand || !seat.Valid() {
		return nil, fmt.Errorf("%w: reveal AAD context", ErrRevealInvalid)
	}
	if strings.TrimSpace(revealKeyID) == "" || len(revealKeyID) > 128 {
		return nil, fmt.Errorf("%w: reveal key ID", ErrRevealInvalid)
	}
	buffer := bytes.NewBuffer(make([]byte, 0, 512))
	writeCanonicalString(buffer, RevealAADV1)
	writeCanonicalString(buffer, command.Version)
	writeCanonicalString(buffer, command.Name)
	writeCanonicalString(buffer, revealKeyID)
	writeCanonicalString(buffer, command.CommandID)
	writeCanonicalString(buffer, string(command.AggregateType))
	writeCanonicalString(buffer, command.AggregateID)
	writeCanonicalString(buffer, string(actor))
	buffer.WriteByte(byte(seat))
	writeCanonicalUint64(buffer, command.ClientSeq)
	writeCanonicalUint64(buffer, command.ExpectedVersion)
	buffer.Write(commitment[:])
	writeCanonicalInt64(buffer, command.IssuedAt.UTC().UnixMilli())
	writeCanonicalInt64(buffer, command.ExpiresAt.UTC().UnixMilli())
	return buffer.Bytes(), nil
}

func BuildContributionRecordAAD(recordID string, handID domain.HandID, seat domain.Seat, actor domain.AccountID, commandID string, digest domain.ContributionDigest) ([]byte, error) {
	if strings.TrimSpace(recordID) == "" || strings.TrimSpace(string(handID)) == "" || strings.TrimSpace(string(actor)) == "" || strings.TrimSpace(commandID) == "" || !seat.Valid() {
		return nil, fmt.Errorf("%w: contribution record AAD", ErrProtectionFailed)
	}
	buffer := bytes.NewBuffer(make([]byte, 0, 384))
	writeCanonicalString(buffer, RecordAADV1)
	writeCanonicalString(buffer, recordID)
	writeCanonicalString(buffer, string(handID))
	buffer.WriteByte(byte(seat))
	writeCanonicalString(buffer, string(actor))
	writeCanonicalString(buffer, commandID)
	buffer.Write(digest[:])
	return buffer.Bytes(), nil
}

func decodeRevealPlaintext(raw []byte, expectedHand domain.HandID, expectedSeat domain.Seat, normalizer PhraseNormalizer, maxPhraseBytes int) (decodedReveal, error) {
	var payload revealPlaintext
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		return decodedReveal{}, fmt.Errorf("%w: decode reveal", ErrRevealInvalid)
	}
	if err := ensureJSONEOF(decoder); err != nil {
		return decodedReveal{}, err
	}
	if payload.Version != RevealPlaintextV1 || payload.HandID != string(expectedHand) || payload.Seat != uint8(expectedSeat) || payload.Normalization != revealNormalizationNFKCV1 {
		return decodedReveal{}, fmt.Errorf("%w: reveal context mismatch", ErrRevealInvalid)
	}
	if payload.Phrase == "" || len([]byte(payload.Phrase)) > maxPhraseBytes {
		return decodedReveal{}, fmt.Errorf("%w: reveal phrase size", ErrRevealInvalid)
	}
	randomBytes, err := base64.RawURLEncoding.Strict().DecodeString(payload.SecureRandom)
	if err != nil || len(randomBytes) != 32 {
		clearBytes(randomBytes)
		return decodedReveal{}, fmt.Errorf("%w: secure random", ErrRevealInvalid)
	}
	var secureRandom [32]byte
	copy(secureRandom[:], randomBytes)
	clearBytes(randomBytes)

	normalized, err := normalizer.Normalize(payload.Phrase)
	if err != nil || normalized == "" || len([]byte(normalized)) > maxPhraseBytes*4 {
		clearBytes(secureRandom[:])
		return decodedReveal{}, fmt.Errorf("%w: normalize phrase", ErrRevealInvalid)
	}
	phraseHash := sha256.Sum256([]byte(normalized))
	digest := computeContributionDigest(expectedHand, expectedSeat, secureRandom, phraseHash)
	return decodedReveal{SecureRandom: secureRandom, PhraseHash: phraseHash, Digest: digest}, nil
}

func computeContributionDigest(handID domain.HandID, seat domain.Seat, secureRandom [32]byte, phraseHash [32]byte) domain.ContributionDigest {
	hash := sha256.New()
	hash.Write([]byte(ContributionV1))
	hash.Write([]byte{0})
	writeHashLengthPrefixed(hash, []byte(handID))
	hash.Write([]byte{byte(seat)})
	hash.Write(secureRandom[:])
	hash.Write(phraseHash[:])
	var result domain.ContributionDigest
	copy(result[:], hash.Sum(nil))
	return result
}

func ensureJSONEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return fmt.Errorf("%w: trailing reveal JSON", ErrRevealInvalid)
	}
	return nil
}

func writeCanonicalString(buffer *bytes.Buffer, value string) { writeCanonicalBytes(buffer, []byte(value)) }
func writeCanonicalBytes(buffer *bytes.Buffer, value []byte) { var size [4]byte; binary.BigEndian.PutUint32(size[:], uint32(len(value))); buffer.Write(size[:]); buffer.Write(value) }
func writeCanonicalUint64(buffer *bytes.Buffer, value uint64) { var encoded [8]byte; binary.BigEndian.PutUint64(encoded[:], value); buffer.Write(encoded[:]) }
func writeCanonicalInt64(buffer *bytes.Buffer, value int64) { writeCanonicalUint64(buffer, uint64(value)) }
func writeHashLengthPrefixed(writer interface{ Write([]byte) (int, error) }, value []byte) { var size [4]byte; binary.BigEndian.PutUint32(size[:], uint32(len(value))); _, _ = writer.Write(size[:]); _, _ = writer.Write(value) }
func clearBytes(value []byte) { for index := range value { value[index] = 0 } }
func validMillisecondPrecision(value time.Time) bool { return value.Nanosecond()%int(time.Millisecond) == 0 }
