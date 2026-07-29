//go:build integration

package mysqlstore_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/application"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/livehand"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/randomizedsetup"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/infrastructure/mysqlstore"
	"github.com/oublie6/awesome-zero-platform/server/business/gamecore"
	"github.com/oublie6/awesome-zero-platform/server/business/gamecore/infrastructure/mysqlarchive"
)

func TestFinalEvidenceReadAuthorizationAndCorruptionWithRealMySQL(t *testing.T) {
	if os.Getenv("APP_API_INTEGRATION") != "1" {
		t.Skip("APP_API_INTEGRATION is not enabled")
	}
	db := openIntegrationDB(t)
	prefix := fmt.Sprintf("g29-evidence-%d", time.Now().UnixNano())
	handID := domain.HandID(prefix + "-hand")
	roomID := domain.RoomID(prefix + "-room")
	participant := domain.AccountID(prefix + "-seat-1")
	cleanupIntegrationRows(t, db, prefix)
	defer cleanupIntegrationRows(t, db, prefix)

	seats := [3]domain.HandSeat{
		{Seat: domain.SeatOne, AccountID: participant},
		{Seat: domain.SeatTwo, AccountID: domain.AccountID(prefix + "-seat-2")},
		{Seat: domain.SeatThree, AccountID: domain.AccountID(prefix + "-seat-3")},
	}
	var serverCommitment domain.ServerCommitment
	serverCommitment[0] = 1
	hand, _, err := domain.NewHand(
		handID,
		roomID,
		seats,
		serverCommitment,
		"reveal-key-1",
		domain.BeaconPlan{Provider: "integration-beacon", Round: "round-1"},
	)
	if err != nil {
		t.Fatal(err)
	}
	snapshot := hand.Snapshot()
	encodedSnapshot, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
INSERT INTO doudizhu_hands (
    hand_id, room_id, phase, reveal_key_id, reveal_public_key_sha256,
    aggregate_version, snapshot_json, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, UTC_TIMESTAMP(6), UTC_TIMESTAMP(6))`,
		snapshot.ID,
		snapshot.RoomID,
		snapshot.Phase,
		snapshot.RevealKeyID,
		snapshot.RevealPublicKeySHA256[:],
		snapshot.Version,
		encodedSnapshot,
	); err != nil {
		t.Fatal(err)
	}

	clock := &integrationClock{now: time.Now().UTC().Truncate(time.Microsecond)}
	archive, err := mysqlarchive.New(db, clock)
	if err != nil {
		t.Fatal(err)
	}
	payload := []byte(`{"v":"integration-final-record-v1"}`)
	record, err := gamecore.NewFinalRecord(
		gamecore.InstanceID(handID),
		randomizedsetup.Descriptor(),
		gamecore.FinalStatusAborted,
		2,
		payload,
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := archive.Archive(record); err != nil {
		t.Fatal(err)
	}

	store, err := mysqlstore.New(sqlDatabase{db})
	if err != nil {
		t.Fatal(err)
	}
	verifier := integrationFinalVerifier{expected: record}
	service, err := application.NewFinalEvidenceService(store, archive, verifier)
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.Get(context.Background(), participant, handID)
	if err != nil {
		t.Fatal(err)
	}
	if result.Version != application.FinalEvidenceResultV1 || result.HandID != handID ||
		result.Status != gamecore.FinalStatusAborted || result.FinalVersion != record.Version() ||
		!bytes.Equal(result.Payload, payload) || result.ArchivedAt.IsZero() ||
		result.Verification.HandID != string(handID) || result.Verification.RecordDigest != verifier.digest() {
		t.Fatalf("result=%#v", result)
	}

	if _, err := service.Get(context.Background(), domain.AccountID(prefix+"-outsider"), handID); !errors.Is(err, application.ErrFinalEvidenceForbidden) {
		t.Fatalf("outsider error=%v", err)
	}

	if _, err := db.Exec(`UPDATE game_final_records SET record_digest = ? WHERE instance_id = ?`, bytes.Repeat([]byte{0x7f}, 32), handID); err != nil {
		t.Fatal(err)
	}
	if _, _, err := archive.LoadFinalRecord(context.Background(), gamecore.InstanceID(handID)); !errors.Is(err, mysqlarchive.ErrArchiveCorrupt) {
		t.Fatalf("corrupt archive error=%v", err)
	}
}

type integrationFinalVerifier struct {
	expected gamecore.FinalRecord
}

func (v integrationFinalVerifier) Verify(record gamecore.FinalRecord) (livehand.FinalVerificationReport, error) {
	if record.Digest() != v.expected.Digest() || record.InstanceID() != v.expected.InstanceID() ||
		record.Status() != v.expected.Status() || record.Version() != v.expected.Version() ||
		!bytes.Equal(record.Payload(), v.expected.Payload()) {
		return livehand.FinalVerificationReport{}, fmt.Errorf("unexpected final record")
	}
	return livehand.FinalVerificationReport{
		Version:      livehand.FinalVerificationReportV1,
		HandID:       string(record.InstanceID()),
		Status:       record.Status(),
		FinalVersion: record.Version(),
		RecordDigest: v.digest(),
		Reason:       "integration",
	}, nil
}

func (v integrationFinalVerifier) digest() string {
	return fmt.Sprintf("%x", v.expected.Digest())
}
