package livehand

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"testing"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/bidding"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/carddeck"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/playing"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/randomizedsetup"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/settlement"
	"github.com/oublie6/awesome-zero-platform/server/business/gamecore"
)

func TestVerifyCompletedFinalRecordReplaysWholeHand(t *testing.T) {
	record, expected := validCompletedRecord(t)
	report, err := VerifyFinalRecord(record)
	if err != nil {
		t.Fatal(err)
	}
	if report.Version != FinalVerificationReportV1 || !report.Completed || report.Status != gamecore.FinalStatusCompleted ||
		report.HandID != string(record.InstanceID()) || report.FinalVersion != record.Version() ||
		report.LandlordSeat != expected.LandlordSeat || report.WinnerSeat != expected.WinnerSeat ||
		report.SeatPoints != expected.Settlement.SeatPoints || report.RemainingCards[expected.WinnerSeat-1] != 0 {
		t.Fatalf("report=%#v expected=%#v", report, expected)
	}
}

func TestVerifyCompletedFinalRecordRejectsSemanticTampering(t *testing.T) {
	record, payload := validCompletedRecord(t)
	tests := []struct {
		name   string
		mutate func(*CompletedPayload)
	}{
		{name: "settlement", mutate: func(value *CompletedPayload) { value.Settlement.SeatPoints[0]++ }},
		{name: "winner", mutate: func(value *CompletedPayload) { value.WinnerSeat = value.WinnerSeat%3 + 1 }},
		{name: "final hand", mutate: func(value *CompletedPayload) { value.FinalHands[1] = value.FinalHands[1][1:] }},
		{name: "bidding", mutate: func(value *CompletedPayload) { value.Bidding.HighestScore = bidding.ScoreOne }},
		{name: "invalid action seat", mutate: func(value *CompletedPayload) { value.Playing.History[0].Seat = 0 }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			changed := payload
			changed.Bidding.Actions = append([]bidding.Action(nil), payload.Bidding.Actions...)
			changed.Playing.History = clonePlayingHistory(payload.Playing.History)
			for index := range payload.FinalHands {
				changed.FinalHands[index] = append([]string(nil), payload.FinalHands[index]...)
			}
			changed.LandlordCards = append([]string(nil), payload.LandlordCards...)
			test.mutate(&changed)
			encoded, err := json.Marshal(changed)
			if err != nil {
				t.Fatal(err)
			}
			tampered, err := gamecore.NewFinalRecord(record.InstanceID(), record.Descriptor(), record.Status(), record.Version(), encoded)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := VerifyFinalRecord(tampered); !errors.Is(err, ErrInvalidFinalEvidence) {
				t.Fatalf("error=%v", err)
			}
		})
	}
}

func TestVerifyAbortedFinalRecordChecksRemainingCardOwnership(t *testing.T) {
	record, payload := validAbortedRecord(t)
	report, err := VerifyFinalRecord(record)
	if err != nil {
		t.Fatal(err)
	}
	if report.Completed || report.Status != gamecore.FinalStatusAborted || report.Reason != payload.Reason ||
		report.RemainingCards != [3]int{carddeck.CardsPerSeat, carddeck.CardsPerSeat, carddeck.CardsPerSeat} ||
		report.SeatPoints != [3]int64{} {
		t.Fatalf("report=%#v", report)
	}

	payload.CurrentHands[1] = append(payload.CurrentHands[1], payload.CurrentHands[0][0])
	payload.CurrentHands[0] = payload.CurrentHands[0][1:]
	encoded, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	tampered, err := gamecore.NewFinalRecord(record.InstanceID(), record.Descriptor(), record.Status(), record.Version(), encoded)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyFinalRecord(tampered); !errors.Is(err, ErrInvalidFinalEvidence) {
		t.Fatalf("error=%v", err)
	}
}

func TestVerifyFinalRecordRejectsUnknownPayloadFields(t *testing.T) {
	record, _ := validAbortedRecord(t)
	var payload map[string]any
	if err := json.Unmarshal(record.Payload(), &payload); err != nil {
		t.Fatal(err)
	}
	payload["settlement"] = map[string]any{"forged": true}
	encoded, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	tampered, err := gamecore.NewFinalRecord(record.InstanceID(), record.Descriptor(), record.Status(), record.Version(), encoded)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyFinalRecord(tampered); !errors.Is(err, ErrInvalidFinalEvidence) {
		t.Fatalf("error=%v", err)
	}
}

func validCompletedRecord(t *testing.T) (gamecore.FinalRecord, CompletedPayload) {
	t.Helper()
	material, artifact, setup, transcript := finalEvidenceFixture(t, "completed-evidence-hand")
	_ = material

	auction, err := bidding.New(setup.DealDigest)
	if err != nil {
		t.Fatal(err)
	}
	first := auction.Snapshot().CurrentBidder
	biddingSnapshot, err := auction.Submit(first, bidding.ScoreThree)
	if err != nil {
		t.Fatal(err)
	}
	landlord := biddingSnapshot.Landlord

	var hands [3][]carddeck.Card
	for index := range setup.Hands {
		hands[index] = append([]carddeck.Card(nil), setup.Hands[index][:]...)
	}
	hands[landlord-1] = append(hands[landlord-1], setup.LandlordCards[:]...)
	turns, err := playing.NewState(landlord)
	if err != nil {
		t.Fatal(err)
	}
	var playingSnapshot playing.Snapshot
	for len(hands[landlord-1]) > 0 {
		card := hands[landlord-1][0]
		hands[landlord-1] = hands[landlord-1][1:]
		playingSnapshot, err = turns.Play(landlord, []carddeck.Card{card}, len(hands[landlord-1]) == 0)
		if err != nil {
			t.Fatal(err)
		}
		if playingSnapshot.Complete {
			break
		}
		firstFarmer := landlord%3 + 1
		if _, err := turns.Pass(firstFarmer); err != nil {
			t.Fatal(err)
		}
		secondFarmer := firstFarmer%3 + 1
		playingSnapshot, err = turns.Pass(secondFarmer)
		if err != nil {
			t.Fatal(err)
		}
	}
	result, err := settlement.Calculate(settlement.Input{LandlordSeat: landlord, WinningScore: biddingSnapshot.HighestScore, Playing: playingSnapshot})
	if err != nil {
		t.Fatal(err)
	}
	transcriptBytes, err := transcript.CanonicalBytes()
	if err != nil {
		t.Fatal(err)
	}
	var finalHands [3][]string
	for index := range hands {
		finalHands[index], err = cardCodes(hands[index])
		if err != nil {
			t.Fatal(err)
		}
	}
	landlordCards, err := cardCodes(setup.LandlordCards[:])
	if err != nil {
		t.Fatal(err)
	}
	payload := CompletedPayload{
		Version:          CompletedPayloadV1,
		HandID:           string(material.InstanceID),
		Status:           string(gamecore.FinalStatusCompleted),
		StateVersion:     uint64(len(playingSnapshot.History)) + 2,
		SetupArtifact:    base64.RawURLEncoding.EncodeToString(artifact.Payload()),
		SetupDigest:      encodeDigest(artifact.Digest()),
		Transcript:       base64.RawURLEncoding.EncodeToString(transcriptBytes),
		TranscriptDigest: hex.EncodeToString(transcript.TranscriptDigest[:]),
		Bidding:          biddingSnapshot,
		Playing:          playingSnapshot,
		Settlement:       result,
		FinalHands:       finalHands,
		LandlordCards:    landlordCards,
		LandlordSeat:     landlord,
		WinningScore:     biddingSnapshot.HighestScore,
		WinnerSeat:       playingSnapshot.WinnerSeat,
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	record, err := gamecore.NewFinalRecord(material.InstanceID, randomizedsetup.Descriptor(), gamecore.FinalStatusCompleted, payload.StateVersion, encoded)
	if err != nil {
		t.Fatal(err)
	}
	return record, payload
}

func validAbortedRecord(t *testing.T) (gamecore.FinalRecord, TerminalPayload) {
	t.Helper()
	material, artifact, setup, transcript := finalEvidenceFixture(t, "aborted-evidence-hand")
	transcriptBytes, err := transcript.CanonicalBytes()
	if err != nil {
		t.Fatal(err)
	}
	var hands [3][]string
	for index := range setup.Hands {
		hands[index], err = cardCodes(setup.Hands[index][:])
		if err != nil {
			t.Fatal(err)
		}
	}
	landlordCards, err := cardCodes(setup.LandlordCards[:])
	if err != nil {
		t.Fatal(err)
	}
	payload := TerminalPayload{
		Version:          TerminalPayloadV1,
		HandID:           string(material.InstanceID),
		Status:           string(gamecore.FinalStatusAborted),
		Reason:           "player_cancelled",
		StateVersion:     2,
		SetupArtifact:    base64.RawURLEncoding.EncodeToString(artifact.Payload()),
		SetupDigest:      encodeDigest(artifact.Digest()),
		Transcript:       base64.RawURLEncoding.EncodeToString(transcriptBytes),
		TranscriptDigest: hex.EncodeToString(transcript.TranscriptDigest[:]),
		CurrentHands:     hands,
		LandlordCards:    landlordCards,
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	record, err := gamecore.NewFinalRecord(material.InstanceID, randomizedsetup.Descriptor(), gamecore.FinalStatusAborted, payload.StateVersion, encoded)
	if err != nil {
		t.Fatal(err)
	}
	return record, payload
}

func finalEvidenceFixture(t *testing.T, handID string) (gamecore.FairnessMaterial, gamecore.SetupArtifact, randomizedsetup.Setup, carddeck.Transcript) {
	t.Helper()
	serverSeed := digestFromText("server-seed:" + handID)
	serverCommitment, err := carddeck.ComputeServerCommitment(handID, carddeck.Seed(serverSeed))
	if err != nil {
		t.Fatal(err)
	}
	material := gamecore.FairnessMaterial{
		Descriptor:       randomizedsetup.Descriptor(),
		InstanceID:       gamecore.InstanceID(handID),
		ServerSeed:       gamecore.Seed(serverSeed),
		ServerCommitment: gamecore.Digest(serverCommitment),
		Beacon: gamecore.BeaconEvidence{
			Provider: "test-beacon",
			Round:    "42",
			Digest:   digestFromText("beacon:" + handID),
			ProofRef: "proof-42",
		},
		RevealKey: gamecore.RevealKeyAudit{KeyID: "test-key", PublicKeySHA256: digestFromText("public-key")},
	}
	for position := uint8(1); position <= 3; position++ {
		contribution := digestFromText(string(rune('0'+position)) + ":" + handID)
		commitment, err := carddeck.ComputeClientCommitment(handID, position, carddeck.ContributionDigest(contribution))
		if err != nil {
			t.Fatal(err)
		}
		material.Participants = append(material.Participants, gamecore.ParticipantFairness{
			Position: position, Contribution: contribution, Commitment: gamecore.Digest(commitment),
		})
	}
	artifact, err := randomizedsetup.NewModule().GenerateSetup(material)
	if err != nil {
		t.Fatal(err)
	}
	setup, err := randomizedsetup.DecodeArtifact(artifact)
	if err != nil {
		t.Fatal(err)
	}
	transcript, err := buildTranscript(material)
	if err != nil {
		t.Fatal(err)
	}
	return material, artifact, setup, transcript
}

func digestFromText(value string) gamecore.Digest { return gamecore.Digest(sha256.Sum256([]byte(value))) }

func clonePlayingHistory(source []playing.Action) []playing.Action {
	result := make([]playing.Action, len(source))
	for index, action := range source {
		result[index] = action
		result[index].Cards = append([]carddeck.Card(nil), action.Cards...)
		if action.Pattern != nil {
			pattern := *action.Pattern
			result[index].Pattern = &pattern
		}
	}
	return result
}
