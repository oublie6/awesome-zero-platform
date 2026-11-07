package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
)

// RevealBinding exposes only the caller's own committed contribution and the
// immutable hand-key binding. It never exposes another player's contribution,
// protected reveal plaintext, server seed or private hand.
type RevealBinding struct {
	Mode            domain.FairnessMode
	HandID          domain.HandID
	Seat            domain.Seat
	Version         uint64
	Commitment      domain.Commitment
	KeyID           string
	PublicKeySHA256 domain.RevealPublicKeyHash
	BoundAt         time.Time
}

// GetRevealBinding is a read-only authorization gate for short-lived,
// authenticated server-issued AAD tickets.
func (s *Service) GetRevealBinding(ctx context.Context, actor domain.AccountID, handID domain.HandID) (RevealBinding, error) {
	if s == nil || s.store == nil || strings.TrimSpace(string(actor)) == "" ||
		strings.TrimSpace(string(handID)) == "" || len(actor) > 128 || len(handID) > 128 {
		return RevealBinding{}, fmt.Errorf("%w: invalid reveal binding query", ErrInvalidCommand)
	}
	snapshot, err := s.store.LoadHand(ctx, handID)
	if err != nil {
		return RevealBinding{}, err
	}
	seat, contribution, err := resolveHandContribution(snapshot, actor)
	if err != nil {
		return RevealBinding{}, err
	}
	custodied := snapshot.FairnessMode == domain.FairnessModeClientCustodied
	eligible := custodied && snapshot.Phase == domain.HandFairnessCommitting && !contribution.Committed && !contribution.Revealed
	legacy := !custodied && snapshot.Phase == domain.HandFairnessRevealing && contribution.Committed &&
		!contribution.Revealed && contribution.Commitment != (domain.Commitment{})
	if (!eligible && !legacy) || snapshot.RevealKeyID == "" ||
		snapshot.RevealPublicKeySHA256 == (domain.RevealPublicKeyHash{}) ||
		snapshot.RevealKeyBoundAt.IsZero() {
		return RevealBinding{}, fmt.Errorf("%w: reveal not eligible", domain.ErrWrongPhase)
	}
	return RevealBinding{
		Mode:   snapshot.FairnessMode,
		HandID: handID, Seat: seat, Version: snapshot.Version, Commitment: contribution.Commitment,
		KeyID: snapshot.RevealKeyID, PublicKeySHA256: snapshot.RevealPublicKeySHA256, BoundAt: snapshot.RevealKeyBoundAt,
	}, nil
}
