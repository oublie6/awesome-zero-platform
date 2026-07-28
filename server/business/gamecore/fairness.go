package gamecore

import (
	"crypto/sha256"
	"fmt"
)

const FairnessMaterialVersion = "gamecore-fairness-material-v1"

const fairnessMaterialDomain = "gamecore/fairness-material/v1"

type Digest [sha256.Size]byte
type Seed Digest

type ParticipantFairness struct {
	Position     uint8
	Contribution Digest
	Commitment   Digest
}

type BeaconEvidence struct {
	Provider string
	Round    string
	Digest   Digest
	ProofRef string
}

type RevealKeyAudit struct {
	KeyID           string
	PublicKeySHA256 Digest
}

type FairnessMaterial struct {
	Descriptor       Descriptor
	InstanceID       InstanceID
	ServerSeed       Seed
	ServerCommitment Digest
	Participants     []ParticipantFairness
	Beacon           BeaconEvidence
	RevealKey        RevealKeyAudit
}

func (m FairnessMaterial) Validate() error {
	if err := m.Descriptor.Validate(); err != nil {
		return err
	}
	if err := validateInstanceID(m.InstanceID); err != nil {
		return err
	}
	if allZero(Digest(m.ServerSeed)) {
		return fmt.Errorf("%w: server seed is zero", ErrInvalidArgument)
	}
	if allZero(m.ServerCommitment) {
		return fmt.Errorf("%w: server commitment is zero", ErrInvalidArgument)
	}
	if len(m.Participants) != int(m.Descriptor.ParticipantCount()) {
		return fmt.Errorf("%w: participant material count %d, want %d", ErrInvalidArgument, len(m.Participants), m.Descriptor.ParticipantCount())
	}
	for index, participant := range m.Participants {
		want := uint8(index + 1)
		if participant.Position != want {
			return fmt.Errorf("%w: participant position %d at index %d, want %d", ErrInvalidArgument, participant.Position, index, want)
		}
		if allZero(participant.Contribution) {
			return fmt.Errorf("%w: participant %d contribution is zero", ErrInvalidArgument, participant.Position)
		}
		if allZero(participant.Commitment) {
			return fmt.Errorf("%w: participant %d commitment is zero", ErrInvalidArgument, participant.Position)
		}
	}
	if err := validateIdentifier("beaconProvider", m.Beacon.Provider); err != nil {
		return err
	}
	if err := validateIdentifier("beaconRound", m.Beacon.Round); err != nil {
		return err
	}
	if allZero(m.Beacon.Digest) {
		return fmt.Errorf("%w: beacon digest is zero", ErrInvalidArgument)
	}
	if err := validateIdentifier("beaconProofRef", m.Beacon.ProofRef); err != nil {
		return err
	}
	if err := validateIdentifier("revealKeyId", m.RevealKey.KeyID); err != nil {
		return err
	}
	if allZero(m.RevealKey.PublicKeySHA256) {
		return fmt.Errorf("%w: reveal public-key digest is zero", ErrInvalidArgument)
	}
	return nil
}

func (m FairnessMaterial) Clone() FairnessMaterial {
	clone := m
	clone.Participants = append([]ParticipantFairness(nil), m.Participants...)
	return clone
}

func (m FairnessMaterial) CanonicalDigest() (Digest, error) {
	if err := m.Validate(); err != nil {
		return Digest{}, err
	}
	h := sha256.New()
	writeDomain(h, fairnessMaterialDomain)
	writeString(h, FairnessMaterialVersion)
	writeString(h, string(m.Descriptor.GameID()))
	writeString(h, string(m.Descriptor.RulesetVersion()))
	writeString(h, string(m.Descriptor.ModuleVersion()))
	writeString(h, string(m.Descriptor.FairnessSuiteID()))
	_, _ = h.Write([]byte{m.Descriptor.ParticipantCount()})
	writeString(h, string(m.InstanceID))
	_, _ = h.Write(m.ServerSeed[:])
	_, _ = h.Write(m.ServerCommitment[:])
	_, _ = h.Write([]byte{byte(len(m.Participants))})
	for _, participant := range m.Participants {
		_, _ = h.Write([]byte{participant.Position})
		_, _ = h.Write(participant.Contribution[:])
		_, _ = h.Write(participant.Commitment[:])
	}
	writeString(h, m.Beacon.Provider)
	writeString(h, m.Beacon.Round)
	_, _ = h.Write(m.Beacon.Digest[:])
	writeString(h, m.Beacon.ProofRef)
	writeString(h, m.RevealKey.KeyID)
	_, _ = h.Write(m.RevealKey.PublicKeySHA256[:])
	var digest Digest
	copy(digest[:], h.Sum(nil))
	return digest, nil
}

func (m FairnessMaterial) DeriveSeed() (Seed, error) {
	digest, err := m.CanonicalDigest()
	return Seed(digest), err
}
