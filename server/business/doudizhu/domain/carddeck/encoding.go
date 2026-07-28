package carddeck

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash"
	"strings"
)

const maxTextBytes = 128

func validateText(name, value string, allowEmpty bool) error {
	if !allowEmpty && strings.TrimSpace(value) == "" {
		return fmt.Errorf("%w: %s is empty", ErrInvalidArgument, name)
	}
	if value != strings.TrimSpace(value) {
		return fmt.Errorf("%w: %s has surrounding whitespace", ErrInvalidArgument, name)
	}
	if len(value) > maxTextBytes {
		return fmt.Errorf("%w: %s exceeds %d bytes", ErrInvalidArgument, name, maxTextBytes)
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

func writeU16(dst interface{ Write([]byte) (int, error) }, value int) {
	var encoded [2]byte
	binary.BigEndian.PutUint16(encoded[:], uint16(value))
	_, _ = dst.Write(encoded[:])
}

func writeU64(dst interface{ Write([]byte) (int, error) }, value uint64) {
	var encoded [8]byte
	binary.BigEndian.PutUint64(encoded[:], value)
	_, _ = dst.Write(encoded[:])
}

func allZero(value [32]byte) bool {
	var combined byte
	for _, item := range value {
		combined |= item
	}
	return combined == 0
}

func writeDomainToBuffer(buffer *bytes.Buffer, domain string) {
	_, _ = buffer.WriteString(domain)
	_ = buffer.WriteByte(0)
}
