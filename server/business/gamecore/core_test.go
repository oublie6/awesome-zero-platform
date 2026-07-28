package gamecore_test

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/oublie6/awesome-zero-platform/server/business/gamecore"
)

func testDescriptor(t *testing.T, participants uint8) gamecore.Descriptor {
	t.Helper()
	descriptor, err := gamecore.NewDescriptor("sequence", "sequence-rules-v1", "sequence-module-v1", "sequence-fairness-v1", participants)
	if err != nil {
		t.Fatal(err)
	}
	return descriptor
}

func digest(fill byte) gamecore.Digest {
	var value gamecore.Digest
	for index := range value {
		value[index] = fill + byte(index)
	}
	return value
}

func testMaterial(t *testing.T, descriptor gamecore.Descriptor) gamecore.FairnessMaterial {
	t.Helper()
	participants := make([]gamecore.ParticipantFairness, descriptor.ParticipantCount())
	for index := range participants {
		participants[index] = gamecore.ParticipantFairness{
			Position:     uint8(index + 1),
			Contribution: digest(byte(10 + index)),
			Commitment:   digest(byte(40 + index)),
		}
	}
	material := gamecore.FairnessMaterial{
		Descriptor:       descriptor,
		InstanceID:       "instance-1",
		ServerSeed:       gamecore.Seed(digest(1)),
		ServerCommitment: digest(70),
		Participants:     participants,
		Beacon: gamecore.BeaconEvidence{
			Provider: "test-beacon",
			Round:    "round-1",
			Digest:   digest(100),
			ProofRef: "proof:1",
		},
		RevealKey: gamecore.RevealKeyAudit{
			KeyID:           "reveal-key-1",
			PublicKeySHA256: digest(130),
		},
	}
	if err := material.Validate(); err != nil {
		t.Fatal(err)
	}
	return material
}

func TestDescriptorValidationAndIdentity(t *testing.T) {
	descriptor := testDescriptor(t, 4)
	if descriptor.GameID() != "sequence" || descriptor.RulesetVersion() != "sequence-rules-v1" || descriptor.ModuleVersion() != "sequence-module-v1" || descriptor.FairnessSuiteID() != "sequence-fairness-v1" || descriptor.ParticipantCount() != 4 {
		t.Fatalf("unexpected descriptor: %#v", descriptor)
	}
	if descriptor.Key() != (gamecore.DescriptorKey{GameID: "sequence", RulesetVersion: "sequence-rules-v1", ModuleVersion: "sequence-module-v1"}) {
		t.Fatalf("unexpected key: %#v", descriptor.Key())
	}

	cases := []struct {
		name         string
		gameID       gamecore.GameID
		ruleset      gamecore.RulesetVersion
		module       gamecore.ModuleVersion
		fairness     gamecore.FairnessSuiteID
		participants uint8
	}{
		{name: "empty game", ruleset: "r", module: "m", fairness: "f", participants: 1},
		{name: "whitespace", gameID: " game", ruleset: "r", module: "m", fairness: "f", participants: 1},
		{name: "oversized", gameID: gamecore.GameID(strings.Repeat("a", 129)), ruleset: "r", module: "m", fairness: "f", participants: 1},
		{name: "zero participants", gameID: "g", ruleset: "r", module: "m", fairness: "f", participants: 0},
		{name: "too many participants", gameID: "g", ruleset: "r", module: "m", fairness: "f", participants: gamecore.MaxParticipantCount + 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := gamecore.NewDescriptor(tc.gameID, tc.ruleset, tc.module, tc.fairness, tc.participants); !errors.Is(err, gamecore.ErrInvalidArgument) {
				t.Fatalf("expected invalid argument, got %v", err)
			}
		})
	}
}

func TestFairnessMaterialValidationCloneAndDigest(t *testing.T) {
	descriptor := testDescriptor(t, 4)
	material := testMaterial(t, descriptor)
	clone := material.Clone()
	clone.Participants[0].Position = 9
	if material.Participants[0].Position != 1 {
		t.Fatal("clone shared participant slice")
	}

	first, err := material.CanonicalDigest()
	if err != nil {
		t.Fatal(err)
	}
	seed, err := material.DeriveSeed()
	if err != nil {
		t.Fatal(err)
	}
	if gamecore.Digest(seed) != first {
		t.Fatal("derived seed differs from canonical digest")
	}
	changed := material.Clone()
	changed.Participants[2].Contribution[0] ^= 0xff
	second, err := changed.CanonicalDigest()
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("digest did not bind participant contribution")
	}

	invalid := material.Clone()
	invalid.Participants[1].Position = 1
	if !errors.Is(invalid.Validate(), gamecore.ErrInvalidArgument) {
		t.Fatal("participant order mismatch was accepted")
	}
	invalid = material.Clone()
	invalid.Participants = invalid.Participants[:3]
	if !errors.Is(invalid.Validate(), gamecore.ErrInvalidArgument) {
		t.Fatal("participant count mismatch was accepted")
	}
	invalid = material.Clone()
	invalid.Beacon.Digest = gamecore.Digest{}
	if !errors.Is(invalid.Validate(), gamecore.ErrInvalidArgument) {
		t.Fatal("zero beacon digest was accepted")
	}
}

type scriptedSource struct {
	values []uint64
	calls  int
}

func (s *scriptedSource) Uint64() uint64 {
	value := s.values[s.calls]
	s.calls++
	return value
}

func TestRandomStreamUniformAndPermutation(t *testing.T) {
	seed := gamecore.Seed(digest(1))
	stream, err := gamecore.NewStream(seed)
	if err != nil {
		t.Fatal(err)
	}
	actual := make([]byte, 64)
	if _, err := io.ReadFull(stream, actual); err != nil {
		t.Fatal(err)
	}
	const expectedHex = "61a8612601f4e2541d90785383a8f09b75196b477038e4df3f8f61a9fc6345c28da31484fa8c17a1d3e08b150a87eab26595a633e1c716a8609a290b19fe69d3"
	if got := fmtHex(actual); got != expectedHex {
		t.Fatalf("random vector changed: %s", got)
	}

	source := &scriptedSource{values: []uint64{5, 16}}
	value, err := gamecore.Uniform(source, 10)
	if err != nil {
		t.Fatal(err)
	}
	if value != 6 || source.calls != 2 {
		t.Fatalf("rejection sampling got value=%d calls=%d", value, source.calls)
	}
	if _, err := gamecore.Uniform(source, 0); !errors.Is(err, gamecore.ErrInvalidArgument) {
		t.Fatalf("zero bound: %v", err)
	}

	for _, count := range []int{1, 2, 7, 54, 137} {
		stream, err := gamecore.NewStream(seed)
		if err != nil {
			t.Fatal(err)
		}
		permutation, err := gamecore.Permutation(count, stream)
		if err != nil {
			t.Fatal(err)
		}
		seen := make([]bool, count)
		for _, item := range permutation {
			if int(item) >= count || seen[item] {
				t.Fatalf("invalid permutation for %d: %v", count, permutation)
			}
			seen[item] = true
		}
	}
}

func fmtHex(value []byte) string {
	const digits = "0123456789abcdef"
	result := make([]byte, len(value)*2)
	for index, item := range value {
		result[index*2] = digits[item>>4]
		result[index*2+1] = digits[item&0xf]
	}
	return string(result)
}

type sequenceModule struct {
	descriptor gamecore.Descriptor
	itemCount  int
}

func (m *sequenceModule) Descriptor() gamecore.Descriptor { return m.descriptor }

func (m *sequenceModule) GenerateSetup(material gamecore.FairnessMaterial) (gamecore.SetupArtifact, error) {
	if err := material.Validate(); err != nil {
		return gamecore.SetupArtifact{}, err
	}
	if !material.Descriptor.Equal(m.descriptor) {
		return gamecore.SetupArtifact{}, gamecore.ErrInvalidArgument
	}
	seed, err := material.DeriveSeed()
	if err != nil {
		return gamecore.SetupArtifact{}, err
	}
	stream, err := gamecore.NewStream(seed)
	if err != nil {
		return gamecore.SetupArtifact{}, err
	}
	permutation, err := gamecore.Permutation(m.itemCount, stream)
	if err != nil {
		return gamecore.SetupArtifact{}, err
	}
	payload := make([]byte, len(permutation)*4)
	for index, item := range permutation {
		binary.BigEndian.PutUint32(payload[index*4:], item)
	}
	return gamecore.NewSetupArtifact(m.descriptor, "sequence-artifact-v1", payload)
}

func (m *sequenceModule) VerifySetup(material gamecore.FairnessMaterial, artifact gamecore.SetupArtifact) error {
	expected, err := m.GenerateSetup(material)
	if err != nil {
		return err
	}
	if artifact.Descriptor() != m.descriptor || artifact.Version() != expected.Version() || artifact.Digest() != expected.Digest() || !bytes.Equal(artifact.Payload(), expected.Payload()) {
		return gamecore.ErrVerificationFailed
	}
	return nil
}

func TestSetupArtifactAndRegistry(t *testing.T) {
	descriptor := testDescriptor(t, 4)
	module := &sequenceModule{descriptor: descriptor, itemCount: 11}
	registry, err := gamecore.NewRegistry(module)
	if err != nil {
		t.Fatal(err)
	}
	if registry.Count() != 1 {
		t.Fatalf("count=%d", registry.Count())
	}
	if err := registry.Register(module); !errors.Is(err, gamecore.ErrDuplicateRegistration) {
		t.Fatalf("duplicate registration: %v", err)
	}
	registered, err := registry.Lookup(descriptor.Key())
	if err != nil {
		t.Fatal(err)
	}
	material := testMaterial(t, descriptor)
	artifact, err := registered.GenerateSetup(material)
	if err != nil {
		t.Fatal(err)
	}
	if err := registered.VerifySetup(material, artifact); err != nil {
		t.Fatal(err)
	}

	payload := artifact.Payload()
	payload[0] ^= 0xff
	if bytes.Equal(payload, artifact.Payload()) {
		t.Fatal("artifact payload accessor did not copy")
	}
	if _, err := gamecore.RestoreSetupArtifact(artifact.Descriptor(), artifact.Version(), payload, artifact.Digest()); !errors.Is(err, gamecore.ErrVerificationFailed) {
		t.Fatalf("tampered payload restore: %v", err)
	}

	other, err := gamecore.NewDescriptor("other", "sequence-rules-v1", "sequence-module-v1", "sequence-fairness-v1", 4)
	if err != nil {
		t.Fatal(err)
	}
	mismatched := material.Clone()
	mismatched.Descriptor = other
	if _, err := registered.GenerateSetup(mismatched); !errors.Is(err, gamecore.ErrInvalidArgument) {
		t.Fatalf("descriptor mismatch: %v", err)
	}
	if _, err := registry.Lookup(other.Key()); !errors.Is(err, gamecore.ErrModuleNotFound) {
		t.Fatalf("missing module: %v", err)
	}
}
