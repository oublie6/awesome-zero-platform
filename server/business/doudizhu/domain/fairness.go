package domain

import (
	"crypto/sha256"
	"encoding/binary"
)

type Commitment [32]byte
type ContributionDigest [32]byte
type BeaconDigest [32]byte
type ServerCommitment [32]byte

const clientCommitDomain = "fair-doudizhu/client-commit/v1"

func ComputeClientCommitment(handID HandID, seat Seat, contribution ContributionDigest) Commitment {
	h := sha256.New()
	h.Write([]byte(clientCommitDomain))
	h.Write([]byte{0})
	writeLengthPrefixed(h, []byte(handID))
	h.Write([]byte{byte(seat)})
	h.Write(contribution[:])
	var result Commitment
	copy(result[:], h.Sum(nil))
	return result
}

func writeLengthPrefixed(dst interface{ Write([]byte) (int, error) }, value []byte) {
	var size [4]byte
	binary.BigEndian.PutUint32(size[:], uint32(len(value)))
	_, _ = dst.Write(size[:])
	_, _ = dst.Write(value)
}

func allZero32(value [32]byte) bool {
	var combined byte
	for _, item := range value {
		combined |= item
	}
	return combined == 0
}
