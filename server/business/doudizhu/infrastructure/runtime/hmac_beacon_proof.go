package runtime

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
)

const HMACBeaconProofPrefix = "hmac-sha256:"

type HMACBeaconProofVerifier struct{ secret []byte }

func NewHMACBeaconProofVerifier(secret []byte) (*HMACBeaconProofVerifier, error) {
	if len(secret) < 32 {
		return nil, fmt.Errorf("%w: beacon proof secret must contain at least 32 bytes", domain.ErrInvalidArgument)
	}
	return &HMACBeaconProofVerifier{secret: append([]byte(nil), secret...)}, nil
}

func (v *HMACBeaconProofVerifier) Verify(ctx context.Context, plan domain.BeaconPlan, candidate domain.BeaconValue) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if v == nil || len(v.secret) < 32 || candidate.Provider != plan.Provider || candidate.Round != plan.Round {
		return domain.ErrBeaconMismatch
	}
	proofRef := strings.TrimSpace(candidate.ProofRef)
	if !strings.HasPrefix(proofRef, HMACBeaconProofPrefix) {
		return domain.ErrBeaconMismatch
	}
	encoded := strings.TrimPrefix(proofRef, HMACBeaconProofPrefix)
	proof, err := base64.RawURLEncoding.Strict().DecodeString(encoded)
	if err != nil || len(proof) != sha256.Size {
		return domain.ErrBeaconMismatch
	}
	expected := v.sign(plan, candidate.Digest)
	if !hmac.Equal(proof, expected) {
		return domain.ErrBeaconMismatch
	}
	return nil
}

func (v *HMACBeaconProofVerifier) ProofRef(plan domain.BeaconPlan, digest domain.BeaconDigest) string {
	return HMACBeaconProofPrefix + base64.RawURLEncoding.EncodeToString(v.sign(plan, digest))
}

func (v *HMACBeaconProofVerifier) sign(plan domain.BeaconPlan, digest domain.BeaconDigest) []byte {
	mac := hmac.New(sha256.New, v.secret)
	writeProofField(mac, []byte("fair-doudizhu/beacon-proof/v1"))
	writeProofField(mac, []byte(plan.Provider))
	writeProofField(mac, []byte(plan.Round))
	writeProofField(mac, digest[:])
	return mac.Sum(nil)
}

func writeProofField(dst interface{ Write([]byte) (int, error) }, value []byte) {
	var size [4]byte
	binary.BigEndian.PutUint32(size[:], uint32(len(value)))
	_, _ = dst.Write(size[:])
	_, _ = dst.Write(value)
}
