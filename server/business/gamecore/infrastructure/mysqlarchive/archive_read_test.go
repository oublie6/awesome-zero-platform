package mysqlarchive

import (
	"errors"
	"testing"
	"time"

	"github.com/oublie6/awesome-zero-platform/server/business/gamecore"
)

func TestRestoreRecordReconstructsExactFinalRecord(t *testing.T) {
	original := mustFinalRecord(t)
	stored := rowFromRecord(original)
	stored.archivedAt = time.Date(2026, 7, 29, 2, 3, 4, 5000, time.FixedZone("test", 8*60*60))

	restored, err := restoreRecord(stored)
	if err != nil {
		t.Fatal(err)
	}
	if err := restored.Validate(); err != nil {
		t.Fatal(err)
	}
	if restored.InstanceID() != original.InstanceID() || !restored.Descriptor().Equal(original.Descriptor()) || restored.Status() != original.Status() || restored.Version() != original.Version() || restored.Digest() != original.Digest() || string(restored.Payload()) != string(original.Payload()) {
		t.Fatalf("restored=%#v original=%#v", restored, original)
	}

	payload := restored.Payload()
	payload[0] ^= 0xff
	if string(restored.Payload()) != string(original.Payload()) {
		t.Fatal("restored payload was not copy-isolated")
	}
}

func TestRestoreRecordRejectsCorruptRows(t *testing.T) {
	original := mustFinalRecord(t)
	tests := []struct {
		name   string
		mutate func(*storedRow)
	}{
		{name: "invalid instance", mutate: func(row *storedRow) { row.instanceID = " bad " }},
		{name: "invalid descriptor", mutate: func(row *storedRow) { row.ruleset = "" }},
		{name: "invalid participant count", mutate: func(row *storedRow) { row.participant = 0 }},
		{name: "invalid status", mutate: func(row *storedRow) { row.status = "settled" }},
		{name: "zero version", mutate: func(row *storedRow) { row.version = 0 }},
		{name: "empty payload", mutate: func(row *storedRow) { row.payload = nil }},
		{name: "tampered payload", mutate: func(row *storedRow) { row.payload[0] ^= 0xff }},
		{name: "short digest", mutate: func(row *storedRow) { row.digest = row.digest[:31] }},
		{name: "tampered digest", mutate: func(row *storedRow) { row.digest[0] ^= 0xff }},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			row := rowFromRecord(original)
			test.mutate(&row)
			if _, err := restoreRecord(row); !errors.Is(err, ErrArchiveCorrupt) {
				t.Fatalf("error=%v", err)
			}
		})
	}
}

func rowFromRecord(record gamecore.FinalRecord) storedRow {
	descriptor := record.Descriptor()
	digest := record.Digest()
	return storedRow{
		instanceID:  string(record.InstanceID()),
		gameID:      string(descriptor.GameID()),
		ruleset:     string(descriptor.RulesetVersion()),
		module:      string(descriptor.ModuleVersion()),
		fairness:    string(descriptor.FairnessSuiteID()),
		status:      string(record.Status()),
		participant: descriptor.ParticipantCount(),
		version:     record.Version(),
		payload:     record.Payload(),
		digest:      append([]byte(nil), digest[:]...),
	}
}

func mustFinalRecord(t *testing.T) gamecore.FinalRecord {
	t.Helper()
	descriptor, err := gamecore.NewDescriptor(
		"fair-doudizhu",
		"doudizhu-rules-v1",
		"doudizhu-module-v1",
		"commit-reveal-hmac-sha256-v1",
		3,
	)
	if err != nil {
		t.Fatal(err)
	}
	record, err := gamecore.NewFinalRecord(
		"hand-archive-read-1",
		descriptor,
		gamecore.FinalStatusCompleted,
		7,
		[]byte(`{"v":"doudizhu-live-completed-v1","winnerSeat":1}`),
	)
	if err != nil {
		t.Fatal(err)
	}
	return record
}
