package randomizedsetup_test

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"testing"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/carddeck"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/randomizedsetup"
	"github.com/oublie6/awesome-zero-platform/server/business/gamecore"
)

type goldenVector struct {
	HandID           string `json:"handId"`
	ServerSeed       string `json:"serverSeed"`
	ServerCommitment string `json:"serverCommitment"`
	Contributions    []struct {
		Seat       uint8  `json:"seat"`
		Digest     string `json:"digest"`
		Commitment string `json:"commitment"`
	} `json:"contributions"`
	BeaconProvider        string     `json:"beaconProvider"`
	BeaconRound           string     `json:"beaconRound"`
	BeaconDigest          string     `json:"beaconDigest"`
	BeaconProofRef        string     `json:"beaconProofRef"`
	RevealKeyID           string     `json:"revealKeyId"`
	RevealPublicKeySHA256 string     `json:"revealPublicKeySha256"`
	ShuffleSeedDigest     string     `json:"shuffleSeedDigest"`
	RandomStreamFirst64   string     `json:"randomStreamFirst64"`
	Deck                  []string   `json:"deck"`
	DeckDigest            string     `json:"deckDigest"`
	Hands                 [][]string `json:"hands"`
	LandlordCards         []string   `json:"landlordCards"`
	DealDigest            string     `json:"dealDigest"`
	TranscriptDigest      string     `json:"transcriptDigest"`
}

func readGolden(t *testing.T) goldenVector {
	t.Helper()
	encoded, err := os.ReadFile("../carddeck/testdata/golden-v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var vector goldenVector
	if err := json.Unmarshal(encoded, &vector); err != nil {
		t.Fatal(err)
	}
	return vector
}

func decodeDigest(t *testing.T, value string) gamecore.Digest {
	t.Helper()
	decoded, err := hex.DecodeString(value)
	if err != nil || len(decoded) != 32 {
		t.Fatalf("invalid digest %q: %v", value, err)
	}
	var result gamecore.Digest
	copy(result[:], decoded)
	return result
}

func materialFromGolden(t *testing.T, vector goldenVector) gamecore.FairnessMaterial {
	t.Helper()
	participants := make([]gamecore.ParticipantFairness, len(vector.Contributions))
	for index, contribution := range vector.Contributions {
		participants[index] = gamecore.ParticipantFairness{
			Position:     contribution.Seat,
			Contribution: decodeDigest(t, contribution.Digest),
			Commitment:   decodeDigest(t, contribution.Commitment),
		}
	}
	return gamecore.FairnessMaterial{
		Descriptor:       randomizedsetup.Descriptor(),
		InstanceID:       gamecore.InstanceID(vector.HandID),
		ServerSeed:       gamecore.Seed(decodeDigest(t, vector.ServerSeed)),
		ServerCommitment: decodeDigest(t, vector.ServerCommitment),
		Participants:     participants,
		Beacon: gamecore.BeaconEvidence{
			Provider: vector.BeaconProvider,
			Round:    vector.BeaconRound,
			Digest:   decodeDigest(t, vector.BeaconDigest),
			ProofRef: vector.BeaconProofRef,
		},
		RevealKey: gamecore.RevealKeyAudit{
			KeyID:           vector.RevealKeyID,
			PublicKeySHA256: decodeDigest(t, vector.RevealPublicKeySHA256),
		},
	}
}

func cardCodes(t *testing.T, cards []carddeck.Card) []string {
	t.Helper()
	result := make([]string, len(cards))
	for index, card := range cards {
		code, err := card.Code()
		if err != nil {
			t.Fatal(err)
		}
		result[index] = code
	}
	return result
}

func TestModulePreservesGoal0023GoldenVector(t *testing.T) {
	vector := readGolden(t)
	material := materialFromGolden(t, vector)
	module := randomizedsetup.NewModule()
	artifact, err := module.GenerateSetup(material)
	if err != nil {
		t.Fatal(err)
	}
	if err := module.VerifySetup(material, artifact); err != nil {
		t.Fatal(err)
	}
	setup, err := randomizedsetup.DecodeArtifact(artifact)
	if err != nil {
		t.Fatal(err)
	}
	if got := hex.EncodeToString(setup.ShuffleSeed[:]); got != vector.ShuffleSeedDigest {
		t.Fatalf("shuffle seed=%s", got)
	}
	if got := hex.EncodeToString(setup.DeckDigest[:]); got != vector.DeckDigest {
		t.Fatalf("deck digest=%s", got)
	}
	if got := hex.EncodeToString(setup.DealDigest[:]); got != vector.DealDigest {
		t.Fatalf("deal digest=%s", got)
	}
	if got := cardCodes(t, setup.Deck.Cards()); !equalStrings(got, vector.Deck) {
		t.Fatalf("deck mismatch\ngot=%v\nwant=%v", got, vector.Deck)
	}
	for seat, hand := range setup.Hands {
		if got := cardCodes(t, hand[:]); !equalStrings(got, vector.Hands[seat]) {
			t.Fatalf("hand %d mismatch\ngot=%v\nwant=%v", seat+1, got, vector.Hands[seat])
		}
	}
	if got := cardCodes(t, setup.LandlordCards[:]); !equalStrings(got, vector.LandlordCards) {
		t.Fatalf("landlord cards mismatch: %v", got)
	}

	input := carddeck.ShuffleInput{
		HandID:         vector.HandID,
		ServerSeed:     carddeck.Seed(material.ServerSeed),
		BeaconProvider: vector.BeaconProvider,
		BeaconRound:    vector.BeaconRound,
		BeaconDigest:   carddeck.BeaconDigest(material.Beacon.Digest),
	}
	for index, participant := range material.Participants {
		input.Contributions[index] = carddeck.ContributionDigest(participant.Contribution)
	}
	seed, err := input.DeriveSeed()
	if err != nil {
		t.Fatal(err)
	}
	stream, err := carddeck.NewStream(seed)
	if err != nil {
		t.Fatal(err)
	}
	first64 := make([]byte, 64)
	if _, err := io.ReadFull(stream, first64); err != nil {
		t.Fatal(err)
	}
	if got := hex.EncodeToString(first64); got != vector.RandomStreamFirst64 {
		t.Fatalf("random stream=%s", got)
	}
}

func TestModuleRejectsTamperingAndIdentityMismatch(t *testing.T) {
	vector := readGolden(t)
	material := materialFromGolden(t, vector)
	module := randomizedsetup.NewModule()
	artifact, err := module.GenerateSetup(material)
	if err != nil {
		t.Fatal(err)
	}

	tamperedMaterial := material.Clone()
	tamperedMaterial.Participants[0].Commitment[0] ^= 1
	if _, err := module.GenerateSetup(tamperedMaterial); !errors.Is(err, gamecore.ErrVerificationFailed) {
		t.Fatalf("tampered commitment: %v", err)
	}

	payload := artifact.Payload()
	payload[len(payload)-1] ^= 1
	tamperedArtifact, err := gamecore.NewSetupArtifact(artifact.Descriptor(), artifact.Version(), payload)
	if err != nil {
		t.Fatal(err)
	}
	if err := module.VerifySetup(material, tamperedArtifact); !errors.Is(err, gamecore.ErrVerificationFailed) {
		t.Fatalf("tampered artifact: %v", err)
	}

	unsupported, err := gamecore.NewSetupArtifact(artifact.Descriptor(), "doudizhu-card-deal-artifact-v2", artifact.Payload())
	if err != nil {
		t.Fatal(err)
	}
	if err := module.VerifySetup(material, unsupported); !errors.Is(err, gamecore.ErrUnsupportedVersion) {
		t.Fatalf("unsupported version: %v", err)
	}

	otherDescriptor, err := gamecore.NewDescriptor("other", randomizedsetup.RulesetVersion, randomizedsetup.ModuleVersion, randomizedsetup.FairnessSuiteID, 3)
	if err != nil {
		t.Fatal(err)
	}
	mismatched := material.Clone()
	mismatched.Descriptor = otherDescriptor
	if _, err := module.GenerateSetup(mismatched); !errors.Is(err, gamecore.ErrInvalidArgument) {
		t.Fatalf("descriptor mismatch: %v", err)
	}

	registry, err := gamecore.NewRegistry(module)
	if err != nil {
		t.Fatal(err)
	}
	registered, err := registry.Lookup(randomizedsetup.Descriptor().Key())
	if err != nil {
		t.Fatal(err)
	}
	registeredArtifact, err := registered.GenerateSetup(material)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(registeredArtifact.Payload(), artifact.Payload()) || registeredArtifact.Digest() != artifact.Digest() {
		t.Fatal("registry wrapper changed artifact")
	}
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
