package runtime

import (
	"context"
	"fmt"
	"strings"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
)

type BeaconProofVerifier interface {
	Verify(context.Context, domain.BeaconPlan, domain.BeaconValue) error
}

type BeaconVerifier struct {
	proofs BeaconProofVerifier
}

func NewBeaconVerifier(proofs BeaconProofVerifier) (*BeaconVerifier, error) {
	if proofs == nil {
		return nil, fmt.Errorf("%w: nil beacon proof verifier", domain.ErrInvalidArgument)
	}
	return &BeaconVerifier{proofs: proofs}, nil
}

func (v *BeaconVerifier) Verify(ctx context.Context, plan domain.BeaconPlan, candidate domain.BeaconValue) (domain.BeaconValue, error) {
	if v == nil || v.proofs == nil {
		return domain.BeaconValue{}, fmt.Errorf("%w: nil beacon verifier", domain.ErrInvalidArgument)
	}
	if err := ctx.Err(); err != nil {
		return domain.BeaconValue{}, err
	}
	if candidate.Provider != plan.Provider || candidate.Round != plan.Round {
		return domain.BeaconValue{}, domain.ErrBeaconMismatch
	}
	candidate.ProofRef = strings.TrimSpace(candidate.ProofRef)
	if candidate.ProofRef == "" || len(candidate.ProofRef) > 512 || candidate.Digest == (domain.BeaconDigest{}) {
		return domain.BeaconValue{}, fmt.Errorf("%w: invalid beacon evidence", domain.ErrInvalidArgument)
	}
	if err := v.proofs.Verify(ctx, plan, candidate); err != nil {
		return domain.BeaconValue{}, fmt.Errorf("verify public beacon proof: %w", err)
	}
	return candidate, nil
}
