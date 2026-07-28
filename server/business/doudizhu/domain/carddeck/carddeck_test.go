package carddeck

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"testing"
)

func TestCardRoundTripAndCanonicalDeck(t *testing.T) {
	deck := CanonicalDeck()
	if err := deck.Validate(); err != nil {
		t.Fatal(err)
	}
	for id, card := range deck {
		if int(card) != id {
			t.Fatalf("deck[%d]=%d", id, card)
		}
		code, err := card.Code()
		if err != nil {
			t.Fatal(err)
		}
		parsed, err := ParseCard(code)
		if err != nil || parsed != card {
			t.Fatalf("card=%d code=%q parsed=%d err=%v", card, code, parsed, err)
		}
	}
	for _, code := range []string{"", "xj", " YJ", "X3", "C1", "C03", "S11"} {
		if _, err := ParseCard(code); err == nil {
			t.Fatalf("expected error for %q", code)
		}
	}
	invalid := deck
	invalid[1] = invalid[0]
	if err := invalid.Validate(); err == nil {
		t.Fatal("expected duplicate-card error")
	}
}

func TestCopiesDoNotShareSlices(t *testing.T) {
	deck := CanonicalDeck()
	cards := deck.Cards()
	cards[0] = BigJoker
	if reflect.DeepEqual(cards, deck.Cards()) {
		t.Fatal("cards accessor shared mutable slice")
	}
}

func TestServerCommitmentBindsHandAndSeed(t *testing.T) {
	var seed Seed
	seed[0] = 1
	first, err := ComputeServerCommitment("hand-a", seed)
	if err != nil {
		t.Fatal(err)
	}
	second, _ := ComputeServerCommitment("hand-b", seed)
	seed[1] = 1
	third, _ := ComputeServerCommitment("hand-a", seed)
	if first == second || first == third {
		t.Fatal("server commitment did not bind all inputs")
	}
}

func TestShuffleInputBindsEveryField(t *testing.T) {
	base := goldenTranscriptInput(t)
	input := shuffleInputFromTranscript(base)
	encoded, err := input.CanonicalBytes()
	if err != nil {
		t.Fatal(err)
	}
	mutations := map[string]func(*ShuffleInput){
		"hand":        func(value *ShuffleInput) { value.HandID += "x" },
		"server-seed": func(value *ShuffleInput) { value.ServerSeed[0] ^= 1 },
		"seat-1":      func(value *ShuffleInput) { value.Contributions[0][0] ^= 1 },
		"seat-2":      func(value *ShuffleInput) { value.Contributions[1][0] ^= 1 },
		"seat-3":      func(value *ShuffleInput) { value.Contributions[2][0] ^= 1 },
		"provider":    func(value *ShuffleInput) { value.BeaconProvider += "x" },
		"round":       func(value *ShuffleInput) { value.BeaconRound += "x" },
		"beacon":      func(value *ShuffleInput) { value.BeaconDigest[0] ^= 1 },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			changed := input
			mutate(&changed)
			other, err := changed.CanonicalBytes()
			if err != nil {
				t.Fatal(err)
			}
			if reflect.DeepEqual(encoded, other) {
				t.Fatal("canonical encoding did not change")
			}
		})
	}
}

func TestUniformRejectsScriptedLowSample(t *testing.T) {
	source := &scriptedSource{values: []uint64{0, 5}}
	value, err := Uniform(source, 3)
	if err != nil {
		t.Fatal(err)
	}
	if value != 2 || source.calls != 2 {
		t.Fatalf("value=%d calls=%d", value, source.calls)
	}
	if _, err := Uniform(source, 0); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("err=%v", err)
	}
}

type scriptedSource struct {
	values []uint64
	calls  int
}

func (source *scriptedSource) Uint64() uint64 {
	value := source.values[source.calls]
	source.calls++
	return value
}

func TestDealRoundRobinAndOrderDigests(t *testing.T) {
	deck := CanonicalDeck()
	deal, err := Deal(deck)
	if err != nil {
		t.Fatal(err)
	}
	for seat := uint8(1); seat <= 3; seat++ {
		hand, _ := deal.Hand(seat)
		for round, card := range hand {
			expected := Card(round*3 + int(seat-1))
			if card != expected {
				t.Fatalf("seat=%d round=%d card=%d expected=%d", seat, round, card, expected)
			}
		}
	}
	if deal.LandlordCards() != [3]Card{51, 52, 53} {
		t.Fatalf("landlord=%v", deal.LandlordCards())
	}
	if deal.ReconstructDeck() != deck {
		t.Fatal("deal did not reconstruct deck")
	}
	swapped := deck
	swapped[0], swapped[1] = swapped[1], swapped[0]
	firstDigest, _ := deck.Digest()
	secondDigest, _ := swapped.Digest()
	if firstDigest == secondDigest {
		t.Fatal("deck digest did not bind order")
	}
	firstDeal, _ := Deal(deck)
	secondDeal, _ := Deal(swapped)
	if firstDeal.Digest() == secondDeal.Digest() {
		t.Fatal("deal digest did not bind order")
	}
}

func TestGoldenVectorFile(t *testing.T) {
	input := goldenTranscriptInput(t)
	transcript, err := BuildTranscript(input)
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifyTranscript(transcript); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile("testdata/golden-v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var vector goldenVector
	if err := json.Unmarshal(data, &vector); err != nil {
		t.Fatal(err)
	}
	assertHex(t, "server commitment", input.ServerCommitment[:], vector.ServerCommitment)
	assertHex(t, "shuffle seed", transcript.ShuffleSeedDigest[:], vector.ShuffleSeedDigest)
	assertHex(t, "deck digest", transcript.DeckDigest[:], vector.DeckDigest)
	assertHex(t, "deal digest", transcript.DealDigest[:], vector.DealDigest)
	assertHex(t, "transcript digest", transcript.TranscriptDigest[:], vector.TranscriptDigest)
	stream, _ := NewStream(transcript.ShuffleSeedDigest)
	first64 := make([]byte, 64)
	_, _ = stream.Read(first64)
	assertHex(t, "random stream", first64, vector.RandomStreamFirst64)
	deckCodes, _ := transcript.Deck.Codes()
	if !reflect.DeepEqual(deckCodes, vector.Deck) {
		t.Fatalf("deck codes mismatch\n got=%v\nwant=%v", deckCodes, vector.Deck)
	}
	for seat := uint8(1); seat <= 3; seat++ {
		hand, _ := transcript.Deal.Hand(seat)
		if !reflect.DeepEqual(cardCodes(hand[:]), vector.Hands[seat-1]) {
			t.Fatalf("seat %d hand mismatch", seat)
		}
	}
	landlord := transcript.Deal.LandlordCards()
	if !reflect.DeepEqual(cardCodes(landlord[:]), vector.LandlordCards) {
		t.Fatal("landlord cards mismatch")
	}
	repeat, err := BuildTranscript(input)
	if err != nil || repeat != transcript {
		t.Fatalf("repeat err=%v equal=%v", err, repeat == transcript)
	}
}

func TestTranscriptTamperingFails(t *testing.T) {
	original, err := BuildTranscript(goldenTranscriptInput(t))
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string]func(*Transcript){
		"version":           func(value *Transcript) { value.Versions.Shuffle = "other" },
		"hand":              func(value *Transcript) { value.HandID += "x" },
		"server-seed":       func(value *Transcript) { value.ServerSeed[0] ^= 1 },
		"server-commit":     func(value *Transcript) { value.ServerCommitment[0] ^= 1 },
		"contribution":      func(value *Transcript) { value.Contributions[1].Digest[0] ^= 1 },
		"client-commit":     func(value *Transcript) { value.Contributions[1].Commitment[0] ^= 1 },
		"beacon-provider":   func(value *Transcript) { value.Beacon.Provider += "x" },
		"beacon-round":      func(value *Transcript) { value.Beacon.Round += "x" },
		"beacon-digest":     func(value *Transcript) { value.Beacon.Digest[0] ^= 1 },
		"beacon-proof":      func(value *Transcript) { value.Beacon.ProofRef += "x" },
		"reveal-key-id":     func(value *Transcript) { value.RevealKey.KeyID += "x" },
		"reveal-key-hash":   func(value *Transcript) { value.RevealKey.PublicKeySHA256[0] ^= 1 },
		"seed-digest":       func(value *Transcript) { value.ShuffleSeedDigest[0] ^= 1 },
		"deck":              func(value *Transcript) { value.Deck[0], value.Deck[1] = value.Deck[1], value.Deck[0] },
		"deck-digest":       func(value *Transcript) { value.DeckDigest[0] ^= 1 },
		"deal":              func(value *Transcript) { value.Deal.hands[0][0], value.Deal.hands[1][0] = value.Deal.hands[1][0], value.Deal.hands[0][0] },
		"deal-digest":       func(value *Transcript) { value.DealDigest[0] ^= 1 },
		"transcript-digest": func(value *Transcript) { value.TranscriptDigest[0] ^= 1 },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			value := original
			mutate(&value)
			if err := VerifyTranscript(value); err == nil {
				t.Fatal("expected verification failure")
			}
		})
	}
}

func TestBuildTranscriptRejectsInconsistentCommitment(t *testing.T) {
	input := goldenTranscriptInput(t)
	input.Contributions[0].Commitment[0] ^= 1
	if _, err := BuildTranscript(input); !errors.Is(err, ErrVerificationFailed) {
		t.Fatalf("err=%v", err)
	}
}

func TestShuffleProducesPermutations(t *testing.T) {
	for iteration := 1; iteration <= 128; iteration++ {
		var seed Seed
		var beacon BeaconDigest
		var contributions [3]ContributionDigest
		for index := range seed {
			seed[index] = byte(iteration + index)
			beacon[index] = byte(iteration*3 + index + 1)
			for seat := range contributions {
				contributions[seat][index] = byte(iteration + seat + index + 7)
			}
		}
		result, err := Shuffle(ShuffleInput{
			HandID: "hand-property", ServerSeed: seed, Contributions: contributions,
			BeaconProvider: "provider", BeaconRound: "round", BeaconDigest: beacon,
		})
		if err != nil {
			t.Fatal(err)
		}
		if err := result.Deck.Validate(); err != nil {
			t.Fatalf("iteration=%d: %v", iteration, err)
		}
	}
}

func goldenTranscriptInput(t *testing.T) TranscriptInput {
	t.Helper()
	var serverSeed Seed
	var beacon BeaconDigest
	var revealHash Digest
	var contributions [3]ContributionEvidence
	for index := 0; index < 32; index++ {
		serverSeed[index] = byte(index + 1)
		beacon[index] = byte(0x80 + index)
		revealHash[index] = byte(0x40 + index)
		for seat := range contributions {
			contributions[seat].Seat = uint8(seat + 1)
			contributions[seat].Digest[index] = byte((seat+1)*17 + index)
		}
	}
	handID := "hand-golden-0023"
	serverCommitment, err := ComputeServerCommitment(handID, serverSeed)
	if err != nil {
		t.Fatal(err)
	}
	for index := range contributions {
		commitment, err := ComputeClientCommitment(handID, uint8(index+1), contributions[index].Digest)
		if err != nil {
			t.Fatal(err)
		}
		contributions[index].Commitment = commitment
	}
	return TranscriptInput{
		HandID: handID, ServerSeed: serverSeed, ServerCommitment: serverCommitment,
		Contributions: contributions,
		Beacon: BeaconEvidence{Provider: "test-beacon", Round: "round-2026-07-28", Digest: beacon, ProofRef: "proof:test:0023"},
		RevealKey: RevealKeyAudit{KeyID: "reveal-key-golden", PublicKeySHA256: revealHash},
	}
}

func shuffleInputFromTranscript(input TranscriptInput) ShuffleInput {
	result := ShuffleInput{
		HandID: input.HandID, ServerSeed: input.ServerSeed,
		BeaconProvider: input.Beacon.Provider, BeaconRound: input.Beacon.Round,
		BeaconDigest: input.Beacon.Digest,
	}
	for index := range input.Contributions {
		result.Contributions[index] = input.Contributions[index].Digest
	}
	return result
}

func cardCodes(cards []Card) []string {
	result := make([]string, len(cards))
	for index, card := range cards {
		result[index], _ = card.Code()
	}
	return result
}

func assertHex(t *testing.T, name string, actual []byte, expected string) {
	t.Helper()
	if hex.EncodeToString(actual) != expected {
		t.Fatalf("%s=%s want=%s", name, hex.EncodeToString(actual), expected)
	}
}

type goldenVector struct {
	ServerCommitment    string      `json:"serverCommitment"`
	ShuffleSeedDigest   string      `json:"shuffleSeedDigest"`
	RandomStreamFirst64 string      `json:"randomStreamFirst64"`
	Deck                []string    `json:"deck"`
	DeckDigest          string      `json:"deckDigest"`
	Hands               [3][]string `json:"hands"`
	LandlordCards       []string    `json:"landlordCards"`
	DealDigest          string      `json:"dealDigest"`
	TranscriptDigest    string      `json:"transcriptDigest"`
}
