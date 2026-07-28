package carddeck

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
)

const (
	SeedVersion            = "fair-doudizhu-shuffle-seed-v1"
	RandomStreamVersion    = "fair-doudizhu-hmac-counter-v1"
	seedDomain             = "fair-doudizhu/shuffle-seed/v1"
	serverCommitDomain     = "fair-doudizhu/server-commit/v1"
	clientCommitDomain     = "fair-doudizhu/client-commit/v1"
	randomBlockDomain      = "fair-doudizhu/random-block/v1"
	deckDigestDomain       = "fair-doudizhu/deck-digest/v1"
	dealDigestDomain       = "fair-doudizhu/deal-digest/v1"
	transcriptDomain       = "fair-doudizhu/transcript-canonical/v1"
	transcriptDigestDomain = "fair-doudizhu/transcript-digest/v1"
)

type Seed [32]byte
type Commitment [32]byte
type ContributionDigest [32]byte
type BeaconDigest [32]byte

type ShuffleInput struct {
	HandID         string
	ServerSeed     Seed
	Contributions  [3]ContributionDigest
	BeaconProvider string
	BeaconRound    string
	BeaconDigest   BeaconDigest
}

func (input ShuffleInput) Validate() error {
	if err := validateText("handId", input.HandID, false); err != nil {
		return err
	}
	if allZero([32]byte(input.ServerSeed)) {
		return fmt.Errorf("%w: server seed is zero", ErrInvalidArgument)
	}
	for index, contribution := range input.Contributions {
		if allZero([32]byte(contribution)) {
			return fmt.Errorf("%w: contribution for seat %d is zero", ErrInvalidArgument, index+1)
		}
	}
	if err := validateText("beaconProvider", input.BeaconProvider, false); err != nil {
		return err
	}
	if err := validateText("beaconRound", input.BeaconRound, false); err != nil {
		return err
	}
	if allZero([32]byte(input.BeaconDigest)) {
		return fmt.Errorf("%w: beacon digest is zero", ErrInvalidArgument)
	}
	return nil
}

func (input ShuffleInput) CanonicalBytes() ([]byte, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}
	var buffer bytes.Buffer
	writeDomainToBuffer(&buffer, seedDomain)
	writeString(&buffer, SeedVersion)
	writeString(&buffer, input.HandID)
	_, _ = buffer.Write(input.ServerSeed[:])
	for index, contribution := range input.Contributions {
		_ = buffer.WriteByte(byte(index + 1))
		_, _ = buffer.Write(contribution[:])
	}
	writeString(&buffer, input.BeaconProvider)
	writeString(&buffer, input.BeaconRound)
	_, _ = buffer.Write(input.BeaconDigest[:])
	return buffer.Bytes(), nil
}

func (input ShuffleInput) DeriveSeed() (Seed, error) {
	encoded, err := input.CanonicalBytes()
	if err != nil {
		return Seed{}, err
	}
	return Seed(sha256.Sum256(encoded)), nil
}

func ComputeServerCommitment(handID string, seed Seed) (Commitment, error) {
	if err := validateText("handId", handID, false); err != nil {
		return Commitment{}, err
	}
	if allZero([32]byte(seed)) {
		return Commitment{}, fmt.Errorf("%w: server seed is zero", ErrInvalidArgument)
	}
	h := sha256.New()
	writeDomain(h, serverCommitDomain)
	writeString(h, handID)
	_, _ = h.Write(seed[:])
	var result Commitment
	copy(result[:], h.Sum(nil))
	return result, nil
}

func ComputeClientCommitment(handID string, seat uint8, contribution ContributionDigest) (Commitment, error) {
	if err := validateText("handId", handID, false); err != nil {
		return Commitment{}, err
	}
	if seat < 1 || seat > 3 {
		return Commitment{}, fmt.Errorf("%w: seat %d", ErrInvalidArgument, seat)
	}
	if allZero([32]byte(contribution)) {
		return Commitment{}, fmt.Errorf("%w: contribution is zero", ErrInvalidArgument)
	}
	h := sha256.New()
	writeDomain(h, clientCommitDomain)
	writeString(h, handID)
	_, _ = h.Write([]byte{seat})
	_, _ = h.Write(contribution[:])
	var result Commitment
	copy(result[:], h.Sum(nil))
	return result, nil
}

type Uint64Source interface {
	Uint64() uint64
}

type Stream struct {
	key     Seed
	counter uint64
	block   [sha256.Size]byte
	offset  int
}

func NewStream(seed Seed) (*Stream, error) {
	if allZero([32]byte(seed)) {
		return nil, fmt.Errorf("%w: random stream seed is zero", ErrInvalidArgument)
	}
	return &Stream{key: seed, offset: len([sha256.Size]byte{})}, nil
}

func (stream *Stream) fill() {
	mac := hmac.New(sha256.New, stream.key[:])
	writeDomain(mac, randomBlockDomain)
	writeString(mac, RandomStreamVersion)
	writeU64(mac, stream.counter)
	copy(stream.block[:], mac.Sum(nil))
	stream.counter++
	stream.offset = 0
}

func (stream *Stream) Read(target []byte) (int, error) {
	if stream == nil {
		return 0, fmt.Errorf("%w: nil random stream", ErrInvalidArgument)
	}
	written := 0
	for written < len(target) {
		if stream.offset >= len(stream.block) {
			stream.fill()
		}
		count := copy(target[written:], stream.block[stream.offset:])
		written += count
		stream.offset += count
	}
	return written, nil
}

func (stream *Stream) Uint64() uint64 {
	var encoded [8]byte
	_, _ = io.ReadFull(stream, encoded[:])
	return binary.BigEndian.Uint64(encoded[:])
}

func Uniform(source Uint64Source, bound uint64) (uint64, error) {
	if source == nil {
		return 0, fmt.Errorf("%w: nil random source", ErrInvalidArgument)
	}
	if bound == 0 {
		return 0, fmt.Errorf("%w: zero bound", ErrInvalidArgument)
	}
	threshold := (uint64(0) - bound) % bound
	for {
		candidate := source.Uint64()
		if candidate >= threshold {
			return candidate % bound, nil
		}
	}
}
