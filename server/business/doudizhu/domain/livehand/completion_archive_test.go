package livehand

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/carddeck"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/settlement"
	"github.com/oublie6/awesome-zero-platform/server/business/gamecore"
)

func TestWinningPlayArchivesCompletedHandAndRemovesLiveInstance(t *testing.T) {
	game := newDirectPlayingGame(t)
	game.transcript = validTranscriptForLiveTest(t, string(game.id))
	last := game.current[0][0]
	game.current[0] = []carddeck.Card{last}
	archive := &recordingFinalArchive{}
	directory, err := gamecore.NewLiveDirectory(archive)
	if err != nil {
		t.Fatal(err)
	}
	if err := directory.Add(game.Descriptor(), game); err != nil {
		t.Fatal(err)
	}

	outcome, err := directory.Apply(game.id, livePlayCommand(t, 1, 2, last))
	if err != nil {
		t.Fatal(err)
	}
	if !outcome.Terminal || outcome.Version != 3 || len(outcome.FinalPayload) == 0 || archive.calls != 1 || directory.Contains(game.id) {
		t.Fatalf("outcome=%#v archive=%d contains=%v", outcome, archive.calls, directory.Contains(game.id))
	}
	if archive.record.Status() != gamecore.FinalStatusCompleted || archive.record.Version() != 3 {
		t.Fatalf("record status=%s version=%d", archive.record.Status(), archive.record.Version())
	}
	var completed CompletedPayload
	if err := json.Unmarshal(archive.record.Payload(), &completed); err != nil {
		t.Fatal(err)
	}
	if completed.Version != CompletedPayloadV1 || completed.Status != string(gamecore.FinalStatusCompleted) || completed.StateVersion != 3 || completed.WinnerSeat != 1 || completed.LandlordSeat != 1 || completed.Playing.WinnerSeat != 1 || completed.Settlement.Version != settlement.RulesVersion || !completed.Settlement.Spring {
		t.Fatalf("completed=%#v", completed)
	}
	if len(completed.FinalHands[0]) != 0 || len(completed.FinalHands[1]) != 17 || len(completed.FinalHands[2]) != 17 || len(completed.LandlordCards) != 3 {
		t.Fatalf("final hands=%v landlord=%v", completed.FinalHands, completed.LandlordCards)
	}
	if len(completed.Transcript) == 0 || len(completed.SetupArtifact) == 0 || completed.TranscriptDigest == "" || completed.SetupDigest == "" {
		t.Fatalf("missing evidence: %#v", completed)
	}
}

func TestWinningPlayArchiveFailureKeepsSamePendingRecordWithoutReplay(t *testing.T) {
	game := newDirectPlayingGame(t)
	game.transcript = validTranscriptForLiveTest(t, string(game.id))
	last := game.current[0][0]
	game.current[0] = []carddeck.Card{last}
	archive := &failOnceFinalArchive{fail: true}
	directory, err := gamecore.NewLiveDirectory(archive)
	if err != nil {
		t.Fatal(err)
	}
	if err := directory.Add(game.Descriptor(), game); err != nil {
		t.Fatal(err)
	}

	outcome, err := directory.Apply(game.id, livePlayCommand(t, 1, 2, last))
	if err == nil || !outcome.Terminal || archive.calls != 1 || !directory.Contains(game.id) {
		t.Fatalf("outcome=%#v error=%v calls=%d contains=%v", outcome, err, archive.calls, directory.Contains(game.id))
	}
	pending, ok, err := directory.PendingFinalRecord(game.id)
	if err != nil || !ok {
		t.Fatalf("pending=%#v ok=%v error=%v", pending, ok, err)
	}
	firstDigest := pending.Digest()
	if game.version != 3 || len(game.current[0]) != 0 || game.play.Snapshot().Revision != 1 {
		t.Fatalf("version=%d cards=%d playing=%#v", game.version, len(game.current[0]), game.play.Snapshot())
	}
	if _, err := directory.Apply(game.id, livePlayCommand(t, 1, 2, last)); !errors.Is(err, gamecore.ErrFinalizationPending) {
		t.Fatalf("second apply error=%v", err)
	}
	if game.version != 3 || game.play.Snapshot().Revision != 1 {
		t.Fatalf("winning play replayed: version=%d playing=%#v", game.version, game.play.Snapshot())
	}

	archive.fail = false
	retried, err := directory.RetryArchive(game.id)
	if err != nil {
		t.Fatal(err)
	}
	if retried.Digest() != firstDigest || archive.calls != 2 || directory.Contains(game.id) {
		t.Fatalf("digest=%x/%x calls=%d contains=%v", retried.Digest(), firstDigest, archive.calls, directory.Contains(game.id))
	}
	if archive.records[0].Digest() != archive.records[1].Digest() {
		t.Fatalf("archive digest changed: %x != %x", archive.records[0].Digest(), archive.records[1].Digest())
	}
}

func TestNonWinningPlayDoesNotArchive(t *testing.T) {
	game := newDirectPlayingGame(t)
	archive := &recordingFinalArchive{}
	directory, err := gamecore.NewLiveDirectory(archive)
	if err != nil {
		t.Fatal(err)
	}
	if err := directory.Add(game.Descriptor(), game); err != nil {
		t.Fatal(err)
	}
	played := game.current[0][0]
	outcome, err := directory.Apply(game.id, livePlayCommand(t, 1, 2, played))
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Terminal || len(outcome.FinalPayload) != 0 || archive.calls != 0 || !directory.Contains(game.id) {
		t.Fatalf("outcome=%#v calls=%d contains=%v", outcome, archive.calls, directory.Contains(game.id))
	}
}

type failOnceFinalArchive struct {
	calls   int
	fail    bool
	records []gamecore.FinalRecord
}

func (a *failOnceFinalArchive) Archive(record gamecore.FinalRecord) error {
	a.calls++
	a.records = append(a.records, record)
	if a.fail {
		return errors.New("archive unavailable")
	}
	return nil
}
