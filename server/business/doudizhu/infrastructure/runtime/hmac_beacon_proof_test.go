package runtime

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
)

func TestHMACBeaconProofVerifier(t *testing.T) {
	verifier, err := NewHMACBeaconProofVerifier(bytes.Repeat([]byte{7}, 32))
	if err != nil {
		t.Fatal(err)
	}
	plan := domain.BeaconPlan{Provider: "trusted-adapter", Round: "round-1"}
	var digest domain.BeaconDigest
	digest[0] = 1
	candidate := domain.BeaconValue{Provider: plan.Provider, Round: plan.Round, Digest: digest}
	candidate.ProofRef = verifier.ProofRef(plan, digest)
	if err := verifier.Verify(context.Background(), plan, candidate); err != nil {
		t.Fatal(err)
	}
	validProof := candidate.ProofRef
	candidate.Digest[1] = 2
	if err := verifier.Verify(context.Background(), plan, candidate); !errors.Is(err, domain.ErrBeaconMismatch) {
		t.Fatalf("tamper error=%v", err)
	}
	candidate.Digest[1] = 0
	candidate.ProofRef = strings.TrimPrefix(validProof, HMACBeaconProofPrefix)
	if err := verifier.Verify(context.Background(), plan, candidate); !errors.Is(err, domain.ErrBeaconMismatch) {
		t.Fatalf("missing prefix error=%v", err)
	}
}
