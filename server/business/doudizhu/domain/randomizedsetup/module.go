package randomizedsetup

import (
	"bytes"
	"crypto/subtle"
	"encoding/binary"
	"fmt"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/carddeck"
	"github.com/oublie6/awesome-zero-platform/server/business/gamecore"
)

const (
	GameID           gamecore.GameID          = "doudizhu"
	RulesetVersion   gamecore.RulesetVersion  = "doudizhu-standard-v1"
	ModuleVersion    gamecore.ModuleVersion   = "doudizhu-randomized-setup-v1"
	FairnessSuiteID  gamecore.FairnessSuiteID = "fair-doudizhu-carddeck-v1"
	ArtifactVersion  gamecore.ArtifactVersion = "doudizhu-card-deal-artifact-v1"
	participantCount uint8                    = 3
	payloadMagic                              = "DDZSETUP1"
)

var descriptor = mustDescriptor()

func mustDescriptor() gamecore.Descriptor {
	value, err := gamecore.NewDescriptor(GameID, RulesetVersion, ModuleVersion, FairnessSuiteID, participantCount)
	if err != nil {
		panic(err)
	}
	return value
}

func Descriptor() gamecore.Descriptor { return descriptor }

type Module struct{}

func NewModule() Module                        { return Module{} }
func (Module) Descriptor() gamecore.Descriptor { return descriptor }

type Setup struct {
	ShuffleSeed   carddeck.Seed
	Deck          carddeck.Deck
	DeckDigest    carddeck.DeckDigest
	Hands         [3][carddeck.CardsPerSeat]carddeck.Card
	LandlordCards [carddeck.LandlordCardCount]carddeck.Card
	DealDigest    carddeck.DealDigest
}

func (Module) GenerateSetup(material gamecore.FairnessMaterial) (gamecore.SetupArtifact, error) {
	if err := validateMaterial(material); err != nil {
		return gamecore.SetupArtifact{}, err
	}
	input := carddeck.ShuffleInput{
		HandID:         string(material.InstanceID),
		ServerSeed:     carddeck.Seed(material.ServerSeed),
		BeaconProvider: material.Beacon.Provider,
		BeaconRound:    material.Beacon.Round,
		BeaconDigest:   carddeck.BeaconDigest(material.Beacon.Digest),
	}
	for index, participant := range material.Participants {
		input.Contributions[index] = carddeck.ContributionDigest(participant.Contribution)
	}
	shuffle, err := carddeck.Shuffle(input)
	if err != nil {
		return gamecore.SetupArtifact{}, err
	}
	deal, err := carddeck.Deal(shuffle.Deck)
	if err != nil {
		return gamecore.SetupArtifact{}, err
	}
	setup := Setup{
		ShuffleSeed:   shuffle.Seed,
		Deck:          shuffle.Deck,
		DeckDigest:    shuffle.DeckDigest,
		Hands:         deal.Hands(),
		LandlordCards: deal.LandlordCards(),
		DealDigest:    deal.Digest(),
	}
	payload, err := encodeSetup(setup)
	if err != nil {
		return gamecore.SetupArtifact{}, err
	}
	return gamecore.NewSetupArtifact(descriptor, ArtifactVersion, payload)
}

func (m Module) VerifySetup(material gamecore.FairnessMaterial, artifact gamecore.SetupArtifact) error {
	if err := artifact.Validate(); err != nil {
		return err
	}
	if !artifact.Descriptor().Equal(descriptor) {
		return fmt.Errorf("%w: Doudizhu artifact descriptor mismatch", gamecore.ErrVerificationFailed)
	}
	if artifact.Version() != ArtifactVersion {
		return fmt.Errorf("%w: Doudizhu artifact version %q", gamecore.ErrUnsupportedVersion, artifact.Version())
	}
	if _, err := DecodeArtifact(artifact); err != nil {
		return err
	}
	expected, err := m.GenerateSetup(material)
	if err != nil {
		return err
	}
	actualDigest := artifact.Digest()
	expectedDigest := expected.Digest()
	if subtle.ConstantTimeCompare(actualDigest[:], expectedDigest[:]) != 1 || subtle.ConstantTimeCompare(artifact.Payload(), expected.Payload()) != 1 {
		return fmt.Errorf("%w: Doudizhu setup artifact mismatch", gamecore.ErrVerificationFailed)
	}
	return nil
}

func DecodeArtifact(artifact gamecore.SetupArtifact) (Setup, error) {
	if err := artifact.Validate(); err != nil {
		return Setup{}, err
	}
	if !artifact.Descriptor().Equal(descriptor) {
		return Setup{}, fmt.Errorf("%w: Doudizhu artifact descriptor mismatch", gamecore.ErrVerificationFailed)
	}
	if artifact.Version() != ArtifactVersion {
		return Setup{}, fmt.Errorf("%w: Doudizhu artifact version %q", gamecore.ErrUnsupportedVersion, artifact.Version())
	}
	return decodeSetup(artifact.Payload())
}

func validateMaterial(material gamecore.FairnessMaterial) error {
	if err := material.Validate(); err != nil {
		return err
	}
	if !material.Descriptor.Equal(descriptor) {
		return fmt.Errorf("%w: Doudizhu setup descriptor mismatch", gamecore.ErrInvalidArgument)
	}
	serverCommitment, err := carddeck.ComputeServerCommitment(string(material.InstanceID), carddeck.Seed(material.ServerSeed))
	if err != nil {
		return err
	}
	if serverCommitment != carddeck.Commitment(material.ServerCommitment) {
		return fmt.Errorf("%w: server commitment mismatch", gamecore.ErrVerificationFailed)
	}
	for _, participant := range material.Participants {
		commitment, err := carddeck.ComputeClientCommitment(string(material.InstanceID), participant.Position, carddeck.ContributionDigest(participant.Contribution))
		if err != nil {
			return err
		}
		if commitment != carddeck.Commitment(participant.Commitment) {
			return fmt.Errorf("%w: participant %d commitment mismatch", gamecore.ErrVerificationFailed, participant.Position)
		}
	}
	return nil
}

func encodeSetup(setup Setup) ([]byte, error) {
	if err := setup.Deck.Validate(); err != nil {
		return nil, err
	}
	deckDigest, err := setup.Deck.Digest()
	if err != nil {
		return nil, err
	}
	if deckDigest != setup.DeckDigest {
		return nil, fmt.Errorf("%w: deck digest mismatch", gamecore.ErrVerificationFailed)
	}
	deal, err := carddeck.Deal(setup.Deck)
	if err != nil {
		return nil, err
	}
	if deal.Hands() != setup.Hands || deal.LandlordCards() != setup.LandlordCards || deal.Digest() != setup.DealDigest {
		return nil, fmt.Errorf("%w: deal does not match deck", gamecore.ErrVerificationFailed)
	}
	var buffer bytes.Buffer
	writeString(&buffer, payloadMagic)
	writeString(&buffer, carddeck.CardVersion)
	writeString(&buffer, carddeck.DeckVersion)
	writeString(&buffer, carddeck.SeedVersion)
	writeString(&buffer, carddeck.RandomStreamVersion)
	writeString(&buffer, carddeck.ShuffleVersion)
	writeString(&buffer, carddeck.DealVersion)
	_, _ = buffer.Write(setup.ShuffleSeed[:])
	_, _ = buffer.Write(setup.DeckDigest[:])
	for _, card := range setup.Deck {
		_ = buffer.WriteByte(byte(card))
	}
	_, _ = buffer.Write(setup.DealDigest[:])
	for _, hand := range setup.Hands {
		for _, card := range hand {
			_ = buffer.WriteByte(byte(card))
		}
	}
	for _, card := range setup.LandlordCards {
		_ = buffer.WriteByte(byte(card))
	}
	return buffer.Bytes(), nil
}

func decodeSetup(payload []byte) (Setup, error) {
	reader := bytes.NewReader(payload)
	versions := []string{
		payloadMagic,
		carddeck.CardVersion,
		carddeck.DeckVersion,
		carddeck.SeedVersion,
		carddeck.RandomStreamVersion,
		carddeck.ShuffleVersion,
		carddeck.DealVersion,
	}
	for index, expected := range versions {
		actual, err := readString(reader)
		if err != nil {
			return Setup{}, err
		}
		if actual != expected {
			return Setup{}, fmt.Errorf("%w: payload version %d is %q, want %q", gamecore.ErrUnsupportedVersion, index, actual, expected)
		}
	}
	var setup Setup
	if _, err := reader.Read(setup.ShuffleSeed[:]); err != nil {
		return Setup{}, fmt.Errorf("%w: truncated shuffle seed", gamecore.ErrVerificationFailed)
	}
	if _, err := reader.Read(setup.DeckDigest[:]); err != nil {
		return Setup{}, fmt.Errorf("%w: truncated deck digest", gamecore.ErrVerificationFailed)
	}
	for index := range setup.Deck {
		value, err := reader.ReadByte()
		if err != nil {
			return Setup{}, fmt.Errorf("%w: truncated deck", gamecore.ErrVerificationFailed)
		}
		setup.Deck[index] = carddeck.Card(value)
	}
	if _, err := reader.Read(setup.DealDigest[:]); err != nil {
		return Setup{}, fmt.Errorf("%w: truncated deal digest", gamecore.ErrVerificationFailed)
	}
	for seat := range setup.Hands {
		for index := range setup.Hands[seat] {
			value, err := reader.ReadByte()
			if err != nil {
				return Setup{}, fmt.Errorf("%w: truncated hands", gamecore.ErrVerificationFailed)
			}
			setup.Hands[seat][index] = carddeck.Card(value)
		}
	}
	for index := range setup.LandlordCards {
		value, err := reader.ReadByte()
		if err != nil {
			return Setup{}, fmt.Errorf("%w: truncated landlord cards", gamecore.ErrVerificationFailed)
		}
		setup.LandlordCards[index] = carddeck.Card(value)
	}
	if reader.Len() != 0 {
		return Setup{}, fmt.Errorf("%w: trailing setup payload bytes", gamecore.ErrVerificationFailed)
	}
	if _, err := encodeSetup(setup); err != nil {
		return Setup{}, err
	}
	return setup, nil
}

func writeString(buffer *bytes.Buffer, value string) {
	var size [2]byte
	binary.BigEndian.PutUint16(size[:], uint16(len(value)))
	_, _ = buffer.Write(size[:])
	_, _ = buffer.WriteString(value)
}

func readString(reader *bytes.Reader) (string, error) {
	var size [2]byte
	if _, err := reader.Read(size[:]); err != nil {
		return "", fmt.Errorf("%w: truncated setup payload string size", gamecore.ErrVerificationFailed)
	}
	length := int(binary.BigEndian.Uint16(size[:]))
	if length == 0 || length > 128 || length > reader.Len() {
		return "", fmt.Errorf("%w: invalid setup payload string length %d", gamecore.ErrVerificationFailed, length)
	}
	value := make([]byte, length)
	if _, err := reader.Read(value); err != nil {
		return "", fmt.Errorf("%w: truncated setup payload string", gamecore.ErrVerificationFailed)
	}
	return string(value), nil
}
