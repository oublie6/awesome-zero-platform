package carddeck

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

func ParseTranscript(canonical []byte) (Transcript, error) {
	if len(canonical) == 0 {
		return Transcript{}, fmt.Errorf("%w: empty transcript", ErrInvalidArgument)
	}
	reader := bytes.NewReader(canonical)
	domain, err := readCanonicalDomain(reader)
	if err != nil {
		return Transcript{}, err
	}
	if domain != transcriptDomain {
		return Transcript{}, fmt.Errorf("%w: transcript domain %q", ErrUnsupportedVersion, domain)
	}

	versions := make([]string, 7)
	for index := range versions {
		versions[index], err = readCanonicalString(reader)
		if err != nil {
			return Transcript{}, fmt.Errorf("transcript version %d: %w", index, err)
		}
	}
	handID, err := readCanonicalString(reader)
	if err != nil {
		return Transcript{}, fmt.Errorf("transcript hand id: %w", err)
	}

	transcript := Transcript{
		Versions: AlgorithmVersions{
			Card: versions[0], Deck: versions[1], Seed: versions[2], Random: versions[3],
			Shuffle: versions[4], Deal: versions[5], Transcript: versions[6],
		},
		HandID: handID,
	}
	if err := readFixed(reader, transcript.ServerSeed[:], "server seed"); err != nil {
		return Transcript{}, err
	}
	if err := readFixed(reader, transcript.ServerCommitment[:], "server commitment"); err != nil {
		return Transcript{}, err
	}
	for index := range transcript.Contributions {
		seat, err := reader.ReadByte()
		if err != nil {
			return Transcript{}, fmt.Errorf("%w: truncated contribution seat", ErrVerificationFailed)
		}
		transcript.Contributions[index].Seat = seat
		if err := readFixed(reader, transcript.Contributions[index].Digest[:], "contribution digest"); err != nil {
			return Transcript{}, err
		}
		if err := readFixed(reader, transcript.Contributions[index].Commitment[:], "contribution commitment"); err != nil {
			return Transcript{}, err
		}
	}
	if transcript.Beacon.Provider, err = readCanonicalString(reader); err != nil {
		return Transcript{}, fmt.Errorf("beacon provider: %w", err)
	}
	if transcript.Beacon.Round, err = readCanonicalString(reader); err != nil {
		return Transcript{}, fmt.Errorf("beacon round: %w", err)
	}
	if err := readFixed(reader, transcript.Beacon.Digest[:], "beacon digest"); err != nil {
		return Transcript{}, err
	}
	if transcript.Beacon.ProofRef, err = readCanonicalString(reader); err != nil {
		return Transcript{}, fmt.Errorf("beacon proof reference: %w", err)
	}
	if transcript.RevealKey.KeyID, err = readCanonicalString(reader); err != nil {
		return Transcript{}, fmt.Errorf("reveal key id: %w", err)
	}
	if err := readFixed(reader, transcript.RevealKey.PublicKeySHA256[:], "reveal public key digest"); err != nil {
		return Transcript{}, err
	}
	if err := readFixed(reader, transcript.ShuffleSeedDigest[:], "shuffle seed digest"); err != nil {
		return Transcript{}, err
	}

	deckCount, err := readCanonicalU16(reader, "deck count")
	if err != nil {
		return Transcript{}, err
	}
	if deckCount != DeckSize {
		return Transcript{}, fmt.Errorf("%w: deck count %d", ErrVerificationFailed, deckCount)
	}
	for index := range transcript.Deck {
		value, err := reader.ReadByte()
		if err != nil {
			return Transcript{}, fmt.Errorf("%w: truncated deck", ErrVerificationFailed)
		}
		transcript.Deck[index] = Card(value)
	}

	var deal DealResult
	for seatIndex := range deal.hands {
		seat, err := reader.ReadByte()
		if err != nil {
			return Transcript{}, fmt.Errorf("%w: truncated deal seat", ErrVerificationFailed)
		}
		if seat != uint8(seatIndex+1) {
			return Transcript{}, fmt.Errorf("%w: deal seat %d at index %d", ErrVerificationFailed, seat, seatIndex)
		}
		count, err := readCanonicalU16(reader, "hand count")
		if err != nil {
			return Transcript{}, err
		}
		if count != CardsPerSeat {
			return Transcript{}, fmt.Errorf("%w: hand count %d for seat %d", ErrVerificationFailed, count, seat)
		}
		for cardIndex := range deal.hands[seatIndex] {
			value, err := reader.ReadByte()
			if err != nil {
				return Transcript{}, fmt.Errorf("%w: truncated hand for seat %d", ErrVerificationFailed, seat)
			}
			deal.hands[seatIndex][cardIndex] = Card(value)
		}
	}
	landlordCount, err := readCanonicalU16(reader, "landlord card count")
	if err != nil {
		return Transcript{}, err
	}
	if landlordCount != LandlordCardCount {
		return Transcript{}, fmt.Errorf("%w: landlord card count %d", ErrVerificationFailed, landlordCount)
	}
	for index := range deal.landlordCards {
		value, err := reader.ReadByte()
		if err != nil {
			return Transcript{}, fmt.Errorf("%w: truncated landlord cards", ErrVerificationFailed)
		}
		deal.landlordCards[index] = Card(value)
	}
	if err := readFixed(reader, transcript.DeckDigest[:], "deck digest"); err != nil {
		return Transcript{}, err
	}
	if err := readFixed(reader, transcript.DealDigest[:], "deal digest"); err != nil {
		return Transcript{}, err
	}
	if reader.Len() != 0 {
		return Transcript{}, fmt.Errorf("%w: trailing transcript bytes", ErrVerificationFailed)
	}
	deal.digest = transcript.DealDigest
	transcript.Deal = deal
	transcript.TranscriptDigest = digestTranscript(canonical)

	if err := VerifyTranscript(transcript); err != nil {
		return Transcript{}, err
	}
	roundTrip, err := transcript.CanonicalBytes()
	if err != nil {
		return Transcript{}, err
	}
	if !bytes.Equal(roundTrip, canonical) {
		return Transcript{}, fmt.Errorf("%w: non-canonical transcript encoding", ErrVerificationFailed)
	}
	return transcript, nil
}

func readCanonicalDomain(reader *bytes.Reader) (string, error) {
	var domain []byte
	for len(domain) <= maxTextBytes {
		value, err := reader.ReadByte()
		if err != nil {
			return "", fmt.Errorf("%w: truncated transcript domain", ErrVerificationFailed)
		}
		if value == 0 {
			if len(domain) == 0 {
				return "", fmt.Errorf("%w: empty transcript domain", ErrVerificationFailed)
			}
			return string(domain), nil
		}
		domain = append(domain, value)
	}
	return "", fmt.Errorf("%w: transcript domain exceeds %d bytes", ErrVerificationFailed, maxTextBytes)
}

func readCanonicalString(reader *bytes.Reader) (string, error) {
	var size [4]byte
	if _, err := io.ReadFull(reader, size[:]); err != nil {
		return "", fmt.Errorf("%w: truncated string size", ErrVerificationFailed)
	}
	length := int(binary.BigEndian.Uint32(size[:]))
	if length < 1 || length > maxTextBytes || length > reader.Len() {
		return "", fmt.Errorf("%w: invalid string length %d", ErrVerificationFailed, length)
	}
	value := make([]byte, length)
	if _, err := io.ReadFull(reader, value); err != nil {
		return "", fmt.Errorf("%w: truncated string", ErrVerificationFailed)
	}
	return string(value), nil
}

func readCanonicalU16(reader *bytes.Reader, name string) (int, error) {
	var encoded [2]byte
	if _, err := io.ReadFull(reader, encoded[:]); err != nil {
		return 0, fmt.Errorf("%w: truncated %s", ErrVerificationFailed, name)
	}
	return int(binary.BigEndian.Uint16(encoded[:])), nil
}

func readFixed(reader *bytes.Reader, destination []byte, name string) error {
	if _, err := io.ReadFull(reader, destination); err != nil {
		return fmt.Errorf("%w: truncated %s", ErrVerificationFailed, name)
	}
	return nil
}
