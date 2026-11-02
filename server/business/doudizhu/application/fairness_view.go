package application

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
)

type FairnessView struct {
	HandID                string           `json:"handId"`
	Phase                 domain.HandPhase `json:"phase"`
	Version               uint64           `json:"version"`
	Seat                  uint8            `json:"seat"`
	Committed             bool             `json:"committed"`
	Revealed              bool             `json:"revealed"`
	Commitment            string           `json:"commitment,omitempty"`
	ServerCommitment      string           `json:"serverCommitment"`
	RevealKeyID           string           `json:"revealKeyId"`
	RevealPublicKeySHA256 string           `json:"revealPublicKeySha256"`
	RevealKeyBoundAt      time.Time        `json:"revealKeyBoundAt"`
	LastClientSequence    uint64           `json:"lastClientSequence"`
	BeaconProvider        string           `json:"beaconProvider"`
	BeaconRound           string           `json:"beaconRound"`
	BeaconReleaseAt       *time.Time       `json:"beaconReleaseAt,omitempty"`
}

// GetFairnessView gives each player only their own contribution status. Do
// not expose any other player's secret, contribution digest, or plaintext.
func (s *Service) GetFairnessView(ctx context.Context, actor domain.AccountID, id domain.HandID) (FairnessView, error) {
	if s == nil || s.store == nil || strings.TrimSpace(string(actor)) == "" || actor != domain.AccountID(strings.TrimSpace(string(actor))) ||
		id == "" || id != domain.HandID(strings.TrimSpace(string(id))) || len(actor) > 128 || len(id) > 128 {
		return FairnessView{}, fmt.Errorf("%w: invalid fairness view", ErrInvalidCommand)
	}
	snapshot, err := s.store.LoadHand(ctx, id)
	if err != nil {
		return FairnessView{}, err
	}
	seat, own, err := resolveHandContribution(snapshot, actor)
	if err != nil {
		return FairnessView{}, err
	}
	seqStore, ok := s.store.(interface {
		HandClientSequence(context.Context, domain.HandID, domain.AccountID) (uint64, error)
	})
	if !ok {
		return FairnessView{}, fmt.Errorf("%w: missing hand sequence store", ErrInvalidCommand)
	}
	last, err := seqStore.HandClientSequence(ctx, id, actor)
	if err != nil {
		return FairnessView{}, err
	}
	view := FairnessView{HandID: string(id), Phase: snapshot.Phase, Version: snapshot.Version, Seat: uint8(seat),
		Committed: own.Committed, Revealed: own.Revealed, LastClientSequence: last,
		BeaconProvider:   snapshot.BeaconPlan.Provider,
		BeaconRound:      snapshot.BeaconPlan.Round,
		ServerCommitment: hex.EncodeToString(snapshot.ServerCommitment[:]),
		RevealKeyID:      snapshot.RevealKeyID, RevealPublicKeySHA256: hex.EncodeToString(snapshot.RevealPublicKeySHA256[:]),
		RevealKeyBoundAt: snapshot.RevealKeyBoundAt}
	if snapshot.BeaconPlan.Provider == "drand-quicknet" {
		release, err := QuicknetRoundPublication(snapshot.BeaconPlan)
		if err != nil {
			return FairnessView{}, err
		}
		view.BeaconReleaseAt = &release
	}
	if own.Committed {
		view.Commitment = hex.EncodeToString(own.Commitment[:])
	}
	return view, nil
}
