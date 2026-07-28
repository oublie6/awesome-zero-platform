package gamecore

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
)

const RandomStreamVersion = "gamecore-hmac-counter-v1"

const randomBlockDomain = "gamecore/random-block/v1"

const MaxPermutationItems = 1_000_000

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
	if allZero(Digest(seed)) {
		return nil, fmt.Errorf("%w: random stream seed is zero", ErrInvalidArgument)
	}
	return &Stream{key: seed, offset: sha256.Size}, nil
}

func (s *Stream) fill() {
	mac := hmac.New(sha256.New, s.key[:])
	writeDomain(mac, randomBlockDomain)
	writeString(mac, RandomStreamVersion)
	writeU64(mac, s.counter)
	copy(s.block[:], mac.Sum(nil))
	s.counter++
	s.offset = 0
}

func (s *Stream) Read(target []byte) (int, error) {
	if s == nil {
		return 0, fmt.Errorf("%w: nil random stream", ErrInvalidArgument)
	}
	written := 0
	for written < len(target) {
		if s.offset >= len(s.block) {
			s.fill()
		}
		count := copy(target[written:], s.block[s.offset:])
		written += count
		s.offset += count
	}
	return written, nil
}

func (s *Stream) Uint64() uint64 {
	var encoded [8]byte
	_, _ = io.ReadFull(s, encoded[:])
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

func Permutation(count int, source Uint64Source) ([]uint32, error) {
	if count < 1 || count > MaxPermutationItems {
		return nil, fmt.Errorf("%w: permutation count %d", ErrInvalidArgument, count)
	}
	if source == nil {
		return nil, fmt.Errorf("%w: nil random source", ErrInvalidArgument)
	}
	items := make([]uint32, count)
	for index := range items {
		items[index] = uint32(index)
	}
	for index := len(items) - 1; index > 0; index-- {
		selected, err := Uniform(source, uint64(index+1))
		if err != nil {
			return nil, err
		}
		items[index], items[selected] = items[selected], items[index]
	}
	return items, nil
}
