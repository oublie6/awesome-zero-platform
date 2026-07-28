package livehand

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"testing"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/bidding"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/carddeck"
	"github.com/oublie6/awesome-zero-platform/server/business/gamecore"
)

func TestNoLandlordHandCanAbortAndArchiveExactlyOnce(t *testing.T) {
	game := newDirectBiddingGame(t, 1)
	game.transcript = validTranscriptForLiveTest(t, string(game.id))
	archive := &recordingFinalArchive{}
	directory, err := gamecore.NewLiveDirectory(archive)
	if err != nil {
		t.Fatal(err)
	}
	if err := directory.Add(game.Descriptor(), game); err != nil {
		t.Fatal(err)
	}
	for index, position := range []uint8{1, 2, 3} {
		if _, err := directory.Apply(game.id, bidCommand(t, position, uint64(index+1), bidding.ScorePass)); err != nil {
			t.Fatal(err)
		}
	}
	if game.phase != PhaseNoLandlord || game.version != 4 {
		t.Fatalf("phase=%s version=%d", game.phase, game.version)
	}

	record, err := directory.Abort(game.id, "NO_LANDLORD")
	if err != nil {
		t.Fatal(err)
	}
	if record.Status() != gamecore.FinalStatusAborted || record.Version() != 5 || archive.calls != 1 || directory.Contains(game.id) {
		t.Fatalf("record=%#v archive calls=%d contains=%v", record, archive.calls, directory.Contains(game.id))
	}
	var terminal TerminalPayload
	if err := json.Unmarshal(record.Payload(), &terminal); err != nil {
		t.Fatal(err)
	}
	if terminal.Reason != "NO_LANDLORD" || terminal.StateVersion != 5 || len(terminal.LandlordCards) != 3 {
		t.Fatalf("terminal=%#v", terminal)
	}
	for index, hand := range terminal.CurrentHands {
		if len(hand) != 17 {
			t.Fatalf("seat %d terminal cards=%d want=17", index+1, len(hand))
		}
	}
	if terminal.TranscriptDigest != hex.EncodeToString(game.transcript.TranscriptDigest[:]) {
		t.Fatalf("transcript digest=%s want=%x", terminal.TranscriptDigest, game.transcript.TranscriptDigest)
	}
	if _, err := directory.Abort(game.id, "NO_LANDLORD"); !errors.Is(err, gamecore.ErrInstanceNotFound) {
		t.Fatalf("second abort error=%v", err)
	}
	if archive.calls != 1 {
		t.Fatalf("archive calls=%d want=1", archive.calls)
	}
}

type recordingFinalArchive struct {
	calls  int
	record gamecore.FinalRecord
}

func (a *recordingFinalArchive) Archive(record gamecore.FinalRecord) error {
	a.calls++
	a.record = record
	return nil
}

func validTranscriptForLiveTest(t *testing.T, handID string) carddeck.Transcript {
	t.Helper()
	var seed carddeck.Seed
	for index := range seed {
		seed[index] = byte(index + 1)
	}
	serverCommitment, err := carddeck.ComputeServerCommitment(handID, seed)
	if err != nil {
		t.Fatal(err)
	}
	input := carddeck.TranscriptInput{
		HandID:           handID,
		ServerSeed:       seed,
		ServerCommitment: serverCommitment,
		Beacon: carddeck.BeaconEvidence{
			Provider: "test-beacon",
			Round:    "round-1",
			ProofRef: "proof-1",
		},
		RevealKey: carddeck.RevealKeyAudit{KeyID: "reveal-key-1"},
	}
	input.Beacon.Digest[0] = 1
	input.RevealKey.PublicKeySHA256[0] = 2
	for index := range input.Contributions {
		seat := uint8(index + 1)
		var digest carddeck.ContributionDigest
		digest[0] = byte(index + 3)
		commitment, err := carddeck.ComputeClientCommitment(handID, seat, digest)
		if err != nil {
			t.Fatal(err)
		}
		input.Contributions[index] = carddeck.ContributionEvidence{Seat: seat, Digest: digest, Commitment: commitment}
	}
	transcript, err := carddeck.BuildTranscript(input)
	if err != nil {
		t.Fatal(err)
	}
	if err := carddeck.VerifyTranscript(transcript); err != nil {
		t.Fatal(err)
	}
	return transcript
}
