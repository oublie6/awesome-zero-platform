package gamecore

import (
	"encoding/binary"
	"fmt"
	"hash"
	"strings"
	"unicode/utf8"
)

const (
	maxIdentifierBytes = 128
	maxPayloadBytes    = 16 << 20
)

func validateIdentifier(name, value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%w: %s is empty", ErrInvalidArgument, name)
	}
	if value != strings.TrimSpace(value) {
		return fmt.Errorf("%w: %s has surrounding whitespace", ErrInvalidArgument, name)
	}
	if !utf8.ValidString(value) {
		return fmt.Errorf("%w: %s is not valid UTF-8", ErrInvalidArgument, name)
	}
	if len(value) > maxIdentifierBytes {
		return fmt.Errorf("%w: %s exceeds %d bytes", ErrInvalidArgument, name, maxIdentifierBytes)
	}
	return nil
}

func validatePayload(name string, payload []byte, allowEmpty bool) error {
	if !allowEmpty && len(payload) == 0 {
		return fmt.Errorf("%w: %s is empty", ErrInvalidArgument, name)
	}
	if len(payload) > maxPayloadBytes {
		return fmt.Errorf("%w: %s exceeds %d bytes", ErrInvalidArgument, name, maxPayloadBytes)
	}
	return nil
}

func writeDomain(dst hash.Hash, domain string) {
	_, _ = dst.Write([]byte(domain))
	_, _ = dst.Write([]byte{0})
}

func writeBytes(dst interface{ Write([]byte) (int, error) }, value []byte) {
	var size [4]byte
	binary.BigEndian.PutUint32(size[:], uint32(len(value)))
	_, _ = dst.Write(size[:])
	_, _ = dst.Write(value)
}

func writeString(dst interface{ Write([]byte) (int, error) }, value string) {
	writeBytes(dst, []byte(value))
}

func writeU64(dst interface{ Write([]byte) (int, error) }, value uint64) {
	var encoded [8]byte
	binary.BigEndian.PutUint64(encoded[:], value)
	_, _ = dst.Write(encoded[:])
}

func allZero(value Digest) bool {
	var combined byte
	for _, item := range value {
		combined |= item
	}
	return combined == 0
}

func cloneBytes(value []byte) []byte {
	if value == nil {
		return nil
	}
	result := make([]byte, len(value))
	copy(result, value)
	return result
}
