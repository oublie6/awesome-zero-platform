package carddeck

import (
	"bytes"
	"testing"
)

func TestParseTranscriptRoundTripsCanonicalEvidence(t *testing.T) {
	original := mustTranscriptForDecode(t)
	canonical, err := original.CanonicalBytes()
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := ParseTranscript(canonical)
	if err != nil {
		t.Fatal(err)
	}
	if parsed != original {
		t.Fatalf("parsed transcript differs\nparsed=%#v\noriginal=%#v", parsed, original)
	}
	roundTrip, err := parsed.CanonicalBytes()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(roundTrip, canonical) {
		t.Fatal("canonical transcript did not round-trip")
	}
}

func TestParseTranscriptRejectsTamperingAndNonCanonicalBytes(t *testing.T) {
	transcript := mustTranscriptForDecode(t)
	canonical, err := transcript.CanonicalBytes()
	if err != nil {
		t.Fatal(err)
	}
	contributionSeatOffset, deckCountOffset, firstHandSeatOffset := transcriptOffsets(transcript)

	tests := []struct {
		name   string
		mutate func([]byte) []byte
	}{
		{name: "empty", mutate: func([]byte) []byte { return nil }},
		{name: "truncated", mutate: func(value []byte) []byte { return value[:len(value)-1] }},
		{name: "trailing", mutate: func(value []byte) []byte { return append(value, 0) }},
		{name: "wrong domain", mutate: func(value []byte) []byte { value[0] ^= 1; return value }},
		{name: "zero version length", mutate: func(value []byte) []byte {
			offset := len(transcriptDomain) + 1
			for index := 0; index < 4; index++ {
				value[offset+index] = 0
			}
			return value
		}},
		{name: "wrong contribution seat", mutate: func(value []byte) []byte {
			value[contributionSeatOffset] = 2
			return value
		}},
		{name: "wrong deck count", mutate: func(value []byte) []byte {
			value[deckCountOffset] = 0
			value[deckCountOffset+1] = DeckSize - 1
			return value
		}},
		{name: "wrong hand seat", mutate: func(value []byte) []byte {
			value[firstHandSeatOffset] = 3
			return value
		}},
		{name: "tampered deck", mutate: func(value []byte) []byte {
			firstDeckCard := deckCountOffset + 2
			value[firstDeckCard] = value[firstDeckCard+1]
			return value
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			value := append([]byte(nil), canonical...)
			value = test.mutate(value)
			if _, err := ParseTranscript(value); err == nil {
				t.Fatal("expected parse failure")
			}
		})
	}
}

func mustTranscriptForDecode(t *testing.T) Transcript {
	t.Helper()
	var serverSeed Seed
	for index := range serverSeed {
		serverSeed[index] = byte(index + 1)
	}
	serverCommitment, err := ComputeServerCommitment("decode-hand-1", serverSeed)
	if err != nil {
		t.Fatal(err)
	}
	var contributions [3]ContributionEvidence
	for index := range contributions {
		var digest ContributionDigest
		for position := range digest {
			digest[position] = byte((index+1)*17 + position)
		}
		seat := uint8(index + 1)
		commitment, err := ComputeClientCommitment("decode-hand-1", seat, digest)
		if err != nil {
			t.Fatal(err)
		}
		contributions[index] = ContributionEvidence{Seat: seat, Digest: digest, Commitment: commitment}
	}
	var beaconDigest BeaconDigest
	var publicKeyDigest Digest
	for index := range beaconDigest {
		beaconDigest[index] = byte(200 - index)
		publicKeyDigest[index] = byte(100 + index)
	}
	transcript, err := BuildTranscript(TranscriptInput{
		HandID:           "decode-hand-1",
		ServerSeed:       serverSeed,
		ServerCommitment: serverCommitment,
		Contributions:    contributions,
		Beacon: BeaconEvidence{
			Provider: "test-beacon", Round: "2026-07-29T00:00:00Z",
			Digest: beaconDigest, ProofRef: "test-proof-ref",
		},
		RevealKey: RevealKeyAudit{KeyID: "reveal-key-1", PublicKeySHA256: publicKeyDigest},
	})
	if err != nil {
		t.Fatal(err)
	}
	return transcript
}

func transcriptOffsets(transcript Transcript) (contributionSeat, deckCount, firstHandSeat int) {
	offset := len(transcriptDomain) + 1
	versions := []string{
		transcript.Versions.Card, transcript.Versions.Deck, transcript.Versions.Seed,
		transcript.Versions.Random, transcript.Versions.Shuffle, transcript.Versions.Deal,
		transcript.Versions.Transcript, transcript.HandID,
	}
	for _, value := range versions {
		offset += 4 + len(value)
	}
	offset += 32 + 32
	contributionSeat = offset
	offset += 3 * (1 + 32 + 32)
	offset += 4 + len(transcript.Beacon.Provider)
	offset += 4 + len(transcript.Beacon.Round)
	offset += 32
	offset += 4 + len(transcript.Beacon.ProofRef)
	offset += 4 + len(transcript.RevealKey.KeyID)
	offset += 32
	offset += 32
	deckCount = offset
	firstHandSeat = deckCount + 2 + DeckSize
	return contributionSeat, deckCount, firstHandSeat
}
