package livehand

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/bidding"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/carddeck"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/playing"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/randomizedsetup"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/settlement"
	"github.com/oublie6/awesome-zero-platform/server/business/gamecore"
)

const FinalVerificationReportV1 = "doudizhu-final-verification-report-v1"

var ErrInvalidFinalEvidence = errors.New("doudizhu live hand: invalid final evidence")

type FinalVerificationReport struct {
	Version          string               `json:"v"`
	HandID           string               `json:"handId"`
	Status           gamecore.FinalStatus `json:"status"`
	FinalVersion     uint64               `json:"finalVersion"`
	RecordDigest     string               `json:"recordDigest"`
	SetupDigest      string               `json:"setupDigest"`
	TranscriptDigest string               `json:"transcriptDigest"`
	Completed        bool                 `json:"completed"`
	Reason           string               `json:"reason,omitempty"`
	LandlordSeat     uint8                `json:"landlordSeat,omitempty"`
	WinnerSeat       uint8                `json:"winnerSeat,omitempty"`
	SeatPoints       [3]int64             `json:"seatPoints,omitempty"`
	RemainingCards   [3]int               `json:"remainingCards"`
}

func VerifyFinalRecord(record gamecore.FinalRecord) (FinalVerificationReport, error) {
	if err := record.Validate(); err != nil {
		return FinalVerificationReport{}, invalidEvidence("final record", err)
	}
	if !record.Descriptor().Equal(randomizedsetup.Descriptor()) {
		return FinalVerificationReport{}, invalidEvidence("descriptor", gamecore.ErrVerificationFailed)
	}
	switch record.Status() {
	case gamecore.FinalStatusCompleted:
		return verifyCompletedRecord(record)
	case gamecore.FinalStatusAborted:
		return verifyAbortedRecord(record)
	default:
		return FinalVerificationReport{}, invalidEvidence("status", gamecore.ErrUnsupportedVersion)
	}
}

func verifyCompletedRecord(record gamecore.FinalRecord) (FinalVerificationReport, error) {
	var payload CompletedPayload
	if err := decodeStrictJSON(record.Payload(), &payload); err != nil {
		return FinalVerificationReport{}, invalidEvidence("completed payload", err)
	}
	if payload.Version != CompletedPayloadV1 || payload.Status != string(gamecore.FinalStatusCompleted) ||
		payload.HandID != string(record.InstanceID()) || payload.StateVersion != record.Version() {
		return FinalVerificationReport{}, invalidEvidence("completed identity", gamecore.ErrVerificationFailed)
	}
	setup, transcript, setupDigest, transcriptDigest, err := verifyCommonEvidence(payload.HandID, payload.SetupArtifact, payload.SetupDigest, payload.Transcript, payload.TranscriptDigest)
	if err != nil {
		return FinalVerificationReport{}, err
	}

	auction, err := replayBidding(setup.DealDigest, payload.Bidding)
	if err != nil {
		return FinalVerificationReport{}, invalidEvidence("bidding", err)
	}
	if auction.NoLandlord || auction.Landlord == 0 || auction.Landlord != payload.LandlordSeat ||
		auction.HighestScore != payload.WinningScore {
		return FinalVerificationReport{}, invalidEvidence("landlord result", gamecore.ErrVerificationFailed)
	}

	playingSnapshot, hands, err := replayPlaying(setup, auction.Landlord, payload.Playing)
	if err != nil {
		return FinalVerificationReport{}, invalidEvidence("playing", err)
	}
	if !playingSnapshot.Complete || playingSnapshot.WinnerSeat != payload.WinnerSeat {
		return FinalVerificationReport{}, invalidEvidence("winner", gamecore.ErrVerificationFailed)
	}
	if err := compareHandCodes(payload.FinalHands, hands); err != nil {
		return FinalVerificationReport{}, invalidEvidence("final hands", err)
	}
	if err := compareCodes(payload.LandlordCards, setup.LandlordCards[:]); err != nil {
		return FinalVerificationReport{}, invalidEvidence("landlord cards", err)
	}

	calculated, err := settlement.Calculate(settlement.Input{
		LandlordSeat: auction.Landlord,
		WinningScore: auction.HighestScore,
		Playing:      playingSnapshot,
	})
	if err != nil {
		return FinalVerificationReport{}, invalidEvidence("settlement", err)
	}
	if !reflect.DeepEqual(calculated, payload.Settlement) || payload.Settlement.LandlordSeat != payload.LandlordSeat ||
		payload.Settlement.WinnerSeat != payload.WinnerSeat {
		return FinalVerificationReport{}, invalidEvidence("settlement mismatch", gamecore.ErrVerificationFailed)
	}

	return FinalVerificationReport{
		Version:          FinalVerificationReportV1,
		HandID:           payload.HandID,
		Status:           record.Status(),
		FinalVersion:     record.Version(),
		RecordDigest:     encodeDigest(record.Digest()),
		SetupDigest:      setupDigest,
		TranscriptDigest: transcriptDigest,
		Completed:        true,
		LandlordSeat:     payload.LandlordSeat,
		WinnerSeat:       payload.WinnerSeat,
		SeatPoints:       calculated.SeatPoints,
		RemainingCards:   handCounts(hands),
	}, nil
}

func verifyAbortedRecord(record gamecore.FinalRecord) (FinalVerificationReport, error) {
	var payload TerminalPayload
	if err := decodeStrictJSON(record.Payload(), &payload); err != nil {
		return FinalVerificationReport{}, invalidEvidence("aborted payload", err)
	}
	if payload.Version != TerminalPayloadV1 || payload.Status != string(gamecore.FinalStatusAborted) ||
		payload.HandID != string(record.InstanceID()) || payload.StateVersion != record.Version() ||
		strings.TrimSpace(payload.Reason) == "" || payload.Reason != strings.TrimSpace(payload.Reason) || len(payload.Reason) > 128 {
		return FinalVerificationReport{}, invalidEvidence("aborted identity", gamecore.ErrVerificationFailed)
	}
	setup, _, setupDigest, transcriptDigest, err := verifyCommonEvidence(payload.HandID, payload.SetupArtifact, payload.SetupDigest, payload.Transcript, payload.TranscriptDigest)
	if err != nil {
		return FinalVerificationReport{}, err
	}
	if err := compareCodes(payload.LandlordCards, setup.LandlordCards[:]); err != nil {
		return FinalVerificationReport{}, invalidEvidence("landlord cards", err)
	}
	hands, err := verifyAbortedHands(payload.CurrentHands, setup)
	if err != nil {
		return FinalVerificationReport{}, invalidEvidence("current hands", err)
	}
	return FinalVerificationReport{
		Version:          FinalVerificationReportV1,
		HandID:           payload.HandID,
		Status:           record.Status(),
		FinalVersion:     record.Version(),
		RecordDigest:     encodeDigest(record.Digest()),
		SetupDigest:      setupDigest,
		TranscriptDigest: transcriptDigest,
		Reason:           payload.Reason,
		RemainingCards:   handCounts(hands),
	}, nil
}

func verifyCommonEvidence(handID, artifactText, artifactDigestText, transcriptText, transcriptDigestText string) (randomizedsetup.Setup, carddeck.Transcript, string, string, error) {
	artifactPayload, err := base64.RawURLEncoding.DecodeString(artifactText)
	if err != nil || len(artifactPayload) == 0 {
		return randomizedsetup.Setup{}, carddeck.Transcript{}, "", "", invalidEvidence("setup artifact encoding", err)
	}
	artifactDigest, err := decodeDigest(artifactDigestText)
	if err != nil {
		return randomizedsetup.Setup{}, carddeck.Transcript{}, "", "", invalidEvidence("setup digest", err)
	}
	artifact, err := gamecore.RestoreSetupArtifact(randomizedsetup.Descriptor(), randomizedsetup.ArtifactVersion, artifactPayload, artifactDigest)
	if err != nil {
		return randomizedsetup.Setup{}, carddeck.Transcript{}, "", "", invalidEvidence("setup artifact", err)
	}
	setup, err := randomizedsetup.DecodeArtifact(artifact)
	if err != nil {
		return randomizedsetup.Setup{}, carddeck.Transcript{}, "", "", invalidEvidence("setup payload", err)
	}

	transcriptBytes, err := base64.RawURLEncoding.DecodeString(transcriptText)
	if err != nil || len(transcriptBytes) == 0 {
		return randomizedsetup.Setup{}, carddeck.Transcript{}, "", "", invalidEvidence("transcript encoding", err)
	}
	transcript, err := carddeck.ParseTranscript(transcriptBytes)
	if err != nil {
		return randomizedsetup.Setup{}, carddeck.Transcript{}, "", "", invalidEvidence("transcript", err)
	}
	transcriptDigest, err := decodeDigest(transcriptDigestText)
	if err != nil || transcriptDigest != gamecore.Digest(transcript.TranscriptDigest) {
		return randomizedsetup.Setup{}, carddeck.Transcript{}, "", "", invalidEvidence("transcript digest", gamecore.ErrVerificationFailed)
	}
	if transcript.HandID != handID || transcript.Deck != setup.Deck || transcript.DeckDigest != setup.DeckDigest ||
		transcript.Deal.Hands() != setup.Hands || transcript.Deal.LandlordCards() != setup.LandlordCards || transcript.DealDigest != setup.DealDigest {
		return randomizedsetup.Setup{}, carddeck.Transcript{}, "", "", invalidEvidence("setup transcript identity", gamecore.ErrVerificationFailed)
	}
	return setup, transcript, artifactDigestText, transcriptDigestText, nil
}

func replayBidding(dealDigest carddeck.DealDigest, expected bidding.Snapshot) (bidding.Snapshot, error) {
	if expected.Version != bidding.RulesVersion || len(expected.Actions) < 1 || len(expected.Actions) > 3 {
		return bidding.Snapshot{}, gamecore.ErrVerificationFailed
	}
	state, err := bidding.New(dealDigest)
	if err != nil {
		return bidding.Snapshot{}, err
	}
	var actual bidding.Snapshot
	for _, action := range expected.Actions {
		actual, err = state.Submit(action.Position, action.Score)
		if err != nil {
			return bidding.Snapshot{}, err
		}
	}
	if !reflect.DeepEqual(actual, expected) || !actual.Complete {
		return bidding.Snapshot{}, gamecore.ErrVerificationFailed
	}
	return actual, nil
}

func replayPlaying(setup randomizedsetup.Setup, landlord uint8, expected playing.Snapshot) (playing.Snapshot, [3][]carddeck.Card, error) {
	if landlord < 1 || landlord > 3 || expected.Version != playing.StateVersion || len(expected.History) == 0 {
		return playing.Snapshot{}, [3][]carddeck.Card{}, gamecore.ErrVerificationFailed
	}
	var hands [3][]carddeck.Card
	for index := range setup.Hands {
		hands[index] = append([]carddeck.Card(nil), setup.Hands[index][:]...)
	}
	hands[landlord-1] = append(hands[landlord-1], setup.LandlordCards[:]...)
	state, err := playing.NewState(landlord)
	if err != nil {
		return playing.Snapshot{}, [3][]carddeck.Card{}, err
	}
	var actual playing.Snapshot
	for _, action := range expected.History {
		switch action.Type {
		case playing.ActionPlay:
			remaining, removeErr := removeCardsForVerification(hands[action.Seat-1], action.Cards)
			if removeErr != nil {
				return playing.Snapshot{}, [3][]carddeck.Card{}, removeErr
			}
			actual, err = state.Play(action.Seat, action.Cards, len(remaining) == 0)
			if err == nil {
				hands[action.Seat-1] = remaining
			}
		case playing.ActionPass:
			actual, err = state.Pass(action.Seat)
		default:
			return playing.Snapshot{}, [3][]carddeck.Card{}, gamecore.ErrVerificationFailed
		}
		if err != nil {
			return playing.Snapshot{}, [3][]carddeck.Card{}, err
		}
	}
	if !reflect.DeepEqual(actual, expected) {
		return playing.Snapshot{}, [3][]carddeck.Card{}, gamecore.ErrVerificationFailed
	}
	return actual, hands, nil
}

func verifyAbortedHands(encoded [3][]string, setup randomizedsetup.Setup) ([3][]carddeck.Card, error) {
	var result [3][]carddeck.Card
	var seen [carddeck.DeckSize]bool
	landlordOwner := -1
	for seat := range encoded {
		if len(encoded[seat]) > carddeck.CardsPerSeat+carddeck.LandlordCardCount {
			return result, gamecore.ErrVerificationFailed
		}
		allowed := make(map[carddeck.Card]bool, carddeck.CardsPerSeat)
		for _, card := range setup.Hands[seat] {
			allowed[card] = true
		}
		for _, code := range encoded[seat] {
			card, err := carddeck.ParseCard(code)
			if err != nil || seen[card] {
				return result, gamecore.ErrVerificationFailed
			}
			if !allowed[card] {
				if !containsPhysicalCard(setup.LandlordCards[:], card) {
					return result, gamecore.ErrVerificationFailed
				}
				if landlordOwner >= 0 && landlordOwner != seat {
					return result, gamecore.ErrVerificationFailed
				}
				landlordOwner = seat
			}
			seen[card] = true
			result[seat] = append(result[seat], card)
		}
	}
	return result, nil
}

func compareHandCodes(encoded [3][]string, expected [3][]carddeck.Card) error {
	for index := range encoded {
		if err := compareCodes(encoded[index], expected[index]); err != nil {
			return fmt.Errorf("seat %d: %w", index+1, err)
		}
	}
	return nil
}

func compareCodes(encoded []string, expected []carddeck.Card) error {
	if len(encoded) != len(expected) {
		return gamecore.ErrVerificationFailed
	}
	for index, code := range encoded {
		card, err := carddeck.ParseCard(code)
		if err != nil || card != expected[index] {
			return gamecore.ErrVerificationFailed
		}
	}
	return nil
}

func removeCardsForVerification(hand, cards []carddeck.Card) ([]carddeck.Card, error) {
	remaining := append([]carddeck.Card(nil), hand...)
	for _, wanted := range cards {
		found := -1
		for index, held := range remaining {
			if held == wanted {
				found = index
				break
			}
		}
		if found < 0 {
			return nil, gamecore.ErrVerificationFailed
		}
		remaining = append(remaining[:found], remaining[found+1:]...)
	}
	return remaining, nil
}

func containsPhysicalCard(cards []carddeck.Card, wanted carddeck.Card) bool {
	for _, card := range cards {
		if card == wanted {
			return true
		}
	}
	return false
}

func decodeStrictJSON(payload []byte, destination any) error {
	if len(payload) == 0 {
		return gamecore.ErrVerificationFailed
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return gamecore.ErrVerificationFailed
		}
		return err
	}
	return nil
}

func decodeDigest(value string) (gamecore.Digest, error) {
	var digest gamecore.Digest
	if len(value) != hex.EncodedLen(len(digest)) || value != strings.ToLower(value) {
		return digest, gamecore.ErrVerificationFailed
	}
	decoded, err := hex.DecodeString(value)
	if err != nil || len(decoded) != len(digest) {
		return digest, gamecore.ErrVerificationFailed
	}
	copy(digest[:], decoded)
	return digest, nil
}

func encodeDigest(value gamecore.Digest) string { return hex.EncodeToString(value[:]) }

func handCounts(hands [3][]carddeck.Card) [3]int {
	return [3]int{len(hands[0]), len(hands[1]), len(hands[2])}
}

func invalidEvidence(part string, cause error) error {
	if cause == nil {
		cause = gamecore.ErrVerificationFailed
	}
	return fmt.Errorf("%w: %s: %v", ErrInvalidFinalEvidence, part, cause)
}
