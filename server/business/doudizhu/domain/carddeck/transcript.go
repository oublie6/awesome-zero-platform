package carddeck

import (
	"bytes"
	"crypto/sha256"
	"crypto/subtle"
	"fmt"
)

const TranscriptVersion = "fair-doudizhu-fairness-transcript-v1"

type TranscriptDigest Digest

type AlgorithmVersions struct {
	Card       string
	Deck       string
	Seed       string
	Random     string
	Shuffle    string
	Deal       string
	Transcript string
}

func CurrentAlgorithmVersions() AlgorithmVersions {
	return AlgorithmVersions{
		Card: CardVersion, Deck: DeckVersion, Seed: SeedVersion,
		Random: RandomStreamVersion, Shuffle: ShuffleVersion,
		Deal: DealVersion, Transcript: TranscriptVersion,
	}
}

type ContributionEvidence struct {
	Seat       uint8
	Digest     ContributionDigest
	Commitment Commitment
}

type BeaconEvidence struct {
	Provider string
	Round    string
	Digest   BeaconDigest
	ProofRef string
}

type RevealKeyAudit struct {
	KeyID           string
	PublicKeySHA256 Digest
}

type TranscriptInput struct {
	HandID           string
	ServerSeed       Seed
	ServerCommitment Commitment
	Contributions    [3]ContributionEvidence
	Beacon           BeaconEvidence
	RevealKey        RevealKeyAudit
}

type Transcript struct {
	Versions          AlgorithmVersions
	HandID            string
	ServerSeed        Seed
	ServerCommitment  Commitment
	Contributions     [3]ContributionEvidence
	Beacon            BeaconEvidence
	RevealKey         RevealKeyAudit
	ShuffleSeedDigest Seed
	Deck              Deck
	DeckDigest        DeckDigest
	Deal              DealResult
	DealDigest        DealDigest
	TranscriptDigest  TranscriptDigest
}

func BuildTranscript(input TranscriptInput) (Transcript, error) {
	if err := validateTranscriptInput(input); err != nil {
		return Transcript{}, err
	}
	shuffleInput := ShuffleInput{
		HandID: input.HandID, ServerSeed: input.ServerSeed,
		BeaconProvider: input.Beacon.Provider, BeaconRound: input.Beacon.Round,
		BeaconDigest: input.Beacon.Digest,
	}
	for index, contribution := range input.Contributions {
		shuffleInput.Contributions[index] = contribution.Digest
	}
	shuffle, err := Shuffle(shuffleInput)
	if err != nil {
		return Transcript{}, err
	}
	deal, err := Deal(shuffle.Deck)
	if err != nil {
		return Transcript{}, err
	}
	transcript := Transcript{
		Versions: CurrentAlgorithmVersions(), HandID: input.HandID,
		ServerSeed: input.ServerSeed, ServerCommitment: input.ServerCommitment,
		Contributions: input.Contributions, Beacon: input.Beacon, RevealKey: input.RevealKey,
		ShuffleSeedDigest: shuffle.Seed, Deck: shuffle.Deck, DeckDigest: shuffle.DeckDigest,
		Deal: deal, DealDigest: deal.Digest(),
	}
	canonical, err := transcript.CanonicalBytes()
	if err != nil {
		return Transcript{}, err
	}
	transcript.TranscriptDigest = digestTranscript(canonical)
	return transcript, nil
}

func VerifyTranscript(transcript Transcript) error {
	if transcript.Versions != CurrentAlgorithmVersions() {
		return fmt.Errorf("%w: algorithm versions", ErrUnsupportedVersion)
	}
	input := TranscriptInput{
		HandID: transcript.HandID, ServerSeed: transcript.ServerSeed,
		ServerCommitment: transcript.ServerCommitment, Contributions: transcript.Contributions,
		Beacon: transcript.Beacon, RevealKey: transcript.RevealKey,
	}
	expected, err := BuildTranscript(input)
	if err != nil {
		return err
	}
	if !equal32(expected.ShuffleSeedDigest[:], transcript.ShuffleSeedDigest[:]) {
		return fmt.Errorf("%w: shuffle seed digest", ErrVerificationFailed)
	}
	if expected.Deck != transcript.Deck {
		return fmt.Errorf("%w: deck", ErrVerificationFailed)
	}
	if !equal32(expected.DeckDigest[:], transcript.DeckDigest[:]) {
		return fmt.Errorf("%w: deck digest", ErrVerificationFailed)
	}
	if expected.Deal.hands != transcript.Deal.hands || expected.Deal.landlordCards != transcript.Deal.landlordCards {
		return fmt.Errorf("%w: deal", ErrVerificationFailed)
	}
	if !equal32(expected.DealDigest[:], transcript.DealDigest[:]) {
		return fmt.Errorf("%w: deal digest", ErrVerificationFailed)
	}
	if !equal32(expected.TranscriptDigest[:], transcript.TranscriptDigest[:]) {
		return fmt.Errorf("%w: transcript digest", ErrVerificationFailed)
	}
	return nil
}

func (transcript Transcript) CanonicalBytes() ([]byte, error) {
	if transcript.Versions != CurrentAlgorithmVersions() {
		return nil, fmt.Errorf("%w: algorithm versions", ErrUnsupportedVersion)
	}
	if err := validateTranscriptInput(TranscriptInput{
		HandID: transcript.HandID, ServerSeed: transcript.ServerSeed,
		ServerCommitment: transcript.ServerCommitment, Contributions: transcript.Contributions,
		Beacon: transcript.Beacon, RevealKey: transcript.RevealKey,
	}); err != nil {
		return nil, err
	}
	if err := transcript.Deck.Validate(); err != nil {
		return nil, err
	}
	if err := transcript.Deal.Validate(); err != nil {
		return nil, err
	}
	var buffer bytes.Buffer
	writeDomainToBuffer(&buffer, transcriptDomain)
	for _, version := range []string{
		transcript.Versions.Card, transcript.Versions.Deck, transcript.Versions.Seed,
		transcript.Versions.Random, transcript.Versions.Shuffle, transcript.Versions.Deal,
		transcript.Versions.Transcript,
	} {
		writeString(&buffer, version)
	}
	writeString(&buffer, transcript.HandID)
	_, _ = buffer.Write(transcript.ServerSeed[:])
	_, _ = buffer.Write(transcript.ServerCommitment[:])
	for _, contribution := range transcript.Contributions {
		_ = buffer.WriteByte(contribution.Seat)
		_, _ = buffer.Write(contribution.Digest[:])
		_, _ = buffer.Write(contribution.Commitment[:])
	}
	writeString(&buffer, transcript.Beacon.Provider)
	writeString(&buffer, transcript.Beacon.Round)
	_, _ = buffer.Write(transcript.Beacon.Digest[:])
	writeString(&buffer, transcript.Beacon.ProofRef)
	writeString(&buffer, transcript.RevealKey.KeyID)
	_, _ = buffer.Write(transcript.RevealKey.PublicKeySHA256[:])
	_, _ = buffer.Write(transcript.ShuffleSeedDigest[:])
	writeU16(&buffer, len(transcript.Deck))
	for _, card := range transcript.Deck {
		_ = buffer.WriteByte(byte(card))
	}
	for seat, hand := range transcript.Deal.hands {
		_ = buffer.WriteByte(byte(seat + 1))
		writeU16(&buffer, len(hand))
		for _, card := range hand {
			_ = buffer.WriteByte(byte(card))
		}
	}
	writeU16(&buffer, len(transcript.Deal.landlordCards))
	for _, card := range transcript.Deal.landlordCards {
		_ = buffer.WriteByte(byte(card))
	}
	_, _ = buffer.Write(transcript.DeckDigest[:])
	_, _ = buffer.Write(transcript.DealDigest[:])
	return buffer.Bytes(), nil
}

func validateTranscriptInput(input TranscriptInput) error {
	if err := validateText("handId", input.HandID, false); err != nil {
		return err
	}
	serverCommitment, err := ComputeServerCommitment(input.HandID, input.ServerSeed)
	if err != nil {
		return err
	}
	if !equal32(serverCommitment[:], input.ServerCommitment[:]) {
		return fmt.Errorf("%w: server commitment", ErrVerificationFailed)
	}
	for index, contribution := range input.Contributions {
		expectedSeat := uint8(index + 1)
		if contribution.Seat != expectedSeat {
			return fmt.Errorf("%w: contribution seat %d", ErrInvalidArgument, contribution.Seat)
		}
		commitment, err := ComputeClientCommitment(input.HandID, expectedSeat, contribution.Digest)
		if err != nil {
			return err
		}
		if !equal32(commitment[:], contribution.Commitment[:]) {
			return fmt.Errorf("%w: client commitment for seat %d", ErrVerificationFailed, expectedSeat)
		}
	}
	if err := validateText("beaconProvider", input.Beacon.Provider, false); err != nil {
		return err
	}
	if err := validateText("beaconRound", input.Beacon.Round, false); err != nil {
		return err
	}
	if allZero([32]byte(input.Beacon.Digest)) {
		return fmt.Errorf("%w: beacon digest is zero", ErrInvalidArgument)
	}
	if err := validateText("beaconProofRef", input.Beacon.ProofRef, false); err != nil {
		return err
	}
	if err := validateText("revealKeyId", input.RevealKey.KeyID, false); err != nil {
		return err
	}
	if allZero([32]byte(input.RevealKey.PublicKeySHA256)) {
		return fmt.Errorf("%w: reveal public key hash is zero", ErrInvalidArgument)
	}
	return nil
}

func digestTranscript(canonical []byte) TranscriptDigest {
	h := sha256.New()
	writeDomain(h, transcriptDigestDomain)
	writeBytes(h, canonical)
	var digest TranscriptDigest
	copy(digest[:], h.Sum(nil))
	return digest
}

func equal32(left, right []byte) bool {
	return len(left) == 32 && len(right) == 32 && subtle.ConstantTimeCompare(left, right) == 1
}
