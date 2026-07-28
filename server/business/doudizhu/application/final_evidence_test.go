package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/livehand"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/randomizedsetup"
	"github.com/oublie6/awesome-zero-platform/server/business/gamecore"
)

func TestFinalEvidenceServiceAllowsOnlySeatedParticipants(t *testing.T) {
	record := finalEvidenceTestRecord(t)
	archivedAt := time.Date(2026, 7, 29, 2, 3, 4, 0, time.FixedZone("test", 8*60*60))
	hands := &finalEvidenceHandReader{snapshot: finalEvidenceHandSnapshot()}
	records := &finalEvidenceRecordLoader{record: record, archivedAt: archivedAt}
	verifier := &finalEvidenceVerifier{report: livehand.FinalVerificationReport{
		Version: livehand.FinalVerificationReportV1, HandID: "hand-evidence", Status: gamecore.FinalStatusCompleted, Completed: true,
	}}
	service, err := NewFinalEvidenceService(hands, records, verifier)
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.Get(context.Background(), "player-2", "hand-evidence")
	if err != nil {
		t.Fatal(err)
	}
	if result.Version != FinalEvidenceResultV1 || result.HandID != "hand-evidence" || result.Status != gamecore.FinalStatusCompleted ||
		result.FinalVersion != record.Version() || !result.ArchivedAt.Equal(archivedAt.UTC()) || string(result.Payload) != `{"evidence":true}` ||
		result.Verification.Version != livehand.FinalVerificationReportV1 || records.calls != 1 || verifier.calls != 1 {
		t.Fatalf("result=%#v records=%d verifier=%d", result, records.calls, verifier.calls)
	}
	result.Payload[0] = '['
	if string(record.Payload()) != `{"evidence":true}` {
		t.Fatal("returned payload mutated immutable final record")
	}
}

func TestFinalEvidenceServiceRejectsOutsiderBeforeArchiveLookup(t *testing.T) {
	hands := &finalEvidenceHandReader{snapshot: finalEvidenceHandSnapshot()}
	records := &finalEvidenceRecordLoader{}
	verifier := &finalEvidenceVerifier{}
	service, err := NewFinalEvidenceService(hands, records, verifier)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Get(context.Background(), "outsider", "hand-evidence"); !errors.Is(err, ErrFinalEvidenceForbidden) {
		t.Fatalf("error=%v", err)
	}
	if records.calls != 0 || verifier.calls != 0 {
		t.Fatalf("archive or verifier called for outsider: records=%d verifier=%d", records.calls, verifier.calls)
	}
}

func TestFinalEvidenceServicePropagatesReadAndVerificationFailures(t *testing.T) {
	readFailure := errors.New("archive unavailable")
	service, err := NewFinalEvidenceService(
		&finalEvidenceHandReader{snapshot: finalEvidenceHandSnapshot()},
		&finalEvidenceRecordLoader{err: readFailure},
		&finalEvidenceVerifier{},
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Get(context.Background(), "player-1", "hand-evidence"); !errors.Is(err, readFailure) {
		t.Fatalf("error=%v", err)
	}

	verifyFailure := livehand.ErrInvalidFinalEvidence
	service, err = NewFinalEvidenceService(
		&finalEvidenceHandReader{snapshot: finalEvidenceHandSnapshot()},
		&finalEvidenceRecordLoader{record: finalEvidenceTestRecord(t), archivedAt: time.Now()},
		&finalEvidenceVerifier{err: verifyFailure},
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Get(context.Background(), "player-1", "hand-evidence"); !errors.Is(err, verifyFailure) {
		t.Fatalf("error=%v", err)
	}
}

func TestFinalEvidenceServiceRejectsInvalidQueriesAndDependencies(t *testing.T) {
	if _, err := NewFinalEvidenceService(nil, &finalEvidenceRecordLoader{}, &finalEvidenceVerifier{}); !errors.Is(err, ErrFinalEvidenceInvalid) {
		t.Fatalf("constructor error=%v", err)
	}
	service, err := NewFinalEvidenceService(&finalEvidenceHandReader{snapshot: finalEvidenceHandSnapshot()}, &finalEvidenceRecordLoader{}, &finalEvidenceVerifier{})
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		actor domain.AccountID
		hand  domain.HandID
	}{
		{actor: "", hand: "hand-evidence"},
		{actor: " player-1", hand: "hand-evidence"},
		{actor: "player-1", hand: ""},
		{actor: "player-1", hand: "hand-evidence "},
	} {
		if _, err := service.Get(context.Background(), test.actor, test.hand); !errors.Is(err, ErrFinalEvidenceInvalid) {
			t.Fatalf("query=%#v error=%v", test, err)
		}
	}
}

func finalEvidenceHandSnapshot() domain.HandSnapshot {
	return domain.HandSnapshot{
		ID: "hand-evidence",
		Seats: [3]domain.HandSeat{
			{Seat: 1, AccountID: "player-1"},
			{Seat: 2, AccountID: "player-2"},
			{Seat: 3, AccountID: "player-3"},
		},
	}
}

func finalEvidenceTestRecord(t *testing.T) gamecore.FinalRecord {
	t.Helper()
	record, err := gamecore.NewFinalRecord("hand-evidence", randomizedsetup.Descriptor(), gamecore.FinalStatusCompleted, 7, []byte(`{"evidence":true}`))
	if err != nil {
		t.Fatal(err)
	}
	return record
}

type finalEvidenceHandReader struct {
	snapshot domain.HandSnapshot
	err      error
	calls    int
}

func (reader *finalEvidenceHandReader) LoadHand(context.Context, domain.HandID) (domain.HandSnapshot, error) {
	reader.calls++
	return reader.snapshot, reader.err
}

type finalEvidenceRecordLoader struct {
	record     gamecore.FinalRecord
	archivedAt time.Time
	err        error
	calls      int
}

func (loader *finalEvidenceRecordLoader) LoadFinalRecord(context.Context, gamecore.InstanceID) (gamecore.FinalRecord, time.Time, error) {
	loader.calls++
	return loader.record, loader.archivedAt, loader.err
}

type finalEvidenceVerifier struct {
	report livehand.FinalVerificationReport
	err    error
	calls  int
}

func (verifier *finalEvidenceVerifier) Verify(gamecore.FinalRecord) (livehand.FinalVerificationReport, error) {
	verifier.calls++
	return verifier.report, verifier.err
}
