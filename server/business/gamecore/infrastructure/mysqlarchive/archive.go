package mysqlarchive

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/oublie6/awesome-zero-platform/server/business/gamecore"
	foundationdb "github.com/oublie6/awesome-zero-platform/server/foundation/database"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var (
	ErrArchiveConflict = errors.New("game final archive conflicts with an existing record")
	ErrArchiveNotFound = errors.New("game final archive record not found")
	ErrArchiveCorrupt  = errors.New("game final archive record is corrupt")
)

type Clock interface {
	Now() time.Time
}

type Archive struct {
	db    *sql.DB
	conn  sqlx.SqlConn
	clock Clock
}

type StoredRecord struct {
	Record     gamecore.FinalRecord
	ArchivedAt time.Time
}

type storedRow struct {
	instanceID, gameID, ruleset, module, fairness, status string
	participant                                           uint8
	version                                               uint64
	payload, digest                                       []byte
	archivedAt                                            time.Time
}

func New(db *sql.DB, clock Clock) (*Archive, error) {
	if db == nil || clock == nil {
		return nil, fmt.Errorf("mysql game archive configuration is invalid")
	}
	return &Archive{db: db, conn: foundationdb.WrapSqlConn(db), clock: clock}, nil
}

func (a *Archive) Archive(record gamecore.FinalRecord) error {
	if a == nil || a.db == nil || a.clock == nil {
		return fmt.Errorf("mysql game archive is not configured")
	}
	if err := record.Validate(); err != nil {
		return err
	}
	descriptor := record.Descriptor()
	payload := record.Payload()
	digest := record.Digest()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := a.conn.ExecCtx(ctx, `
INSERT INTO game_final_records (
    instance_id, game_id, ruleset_version, module_version, fairness_suite_id,
    participant_count, final_status, final_version, payload, record_digest, archived_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE instance_id = VALUES(instance_id)`,
		string(record.InstanceID()), string(descriptor.GameID()), string(descriptor.RulesetVersion()),
		string(descriptor.ModuleVersion()), string(descriptor.FairnessSuiteID()), descriptor.ParticipantCount(),
		string(record.Status()), record.Version(), payload, digest[:], a.clock.Now().UTC(),
	)
	if err != nil {
		return fmt.Errorf("insert game final record: %w", err)
	}
	stored, err := a.queryRow(ctx, record.InstanceID())
	if err != nil {
		return err
	}
	if stored.gameID != string(descriptor.GameID()) || stored.ruleset != string(descriptor.RulesetVersion()) ||
		stored.module != string(descriptor.ModuleVersion()) || stored.fairness != string(descriptor.FairnessSuiteID()) ||
		stored.participant != descriptor.ParticipantCount() || stored.status != string(record.Status()) ||
		stored.version != record.Version() || !bytes.Equal(stored.payload, payload) || !bytes.Equal(stored.digest, digest[:]) {
		return fmt.Errorf("%w: instance %s", ErrArchiveConflict, record.InstanceID())
	}
	return nil
}

func (a *Archive) Load(ctx context.Context, id gamecore.InstanceID) (StoredRecord, error) {
	if a == nil || a.db == nil || a.clock == nil {
		return StoredRecord{}, fmt.Errorf("mysql game archive is not configured")
	}
	if ctx == nil {
		return StoredRecord{}, fmt.Errorf("load game final record: nil context")
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	stored, err := a.queryRow(ctx, id)
	if err != nil {
		return StoredRecord{}, err
	}
	record, err := restoreRecord(stored)
	if err != nil {
		return StoredRecord{}, err
	}
	return StoredRecord{Record: record, ArchivedAt: stored.archivedAt.UTC()}, nil
}

func (a *Archive) LoadFinalRecord(ctx context.Context, id gamecore.InstanceID) (gamecore.FinalRecord, time.Time, error) {
	stored, err := a.Load(ctx, id)
	if err != nil {
		return gamecore.FinalRecord{}, time.Time{}, err
	}
	return stored.Record, stored.ArchivedAt, nil
}

// archiveQueryRow maps database columns through go-zero's sqlx scanner.
type archiveQueryRow struct {
    InstanceID string `db:"instance_id"`
    GameID string `db:"game_id"`
    Ruleset string `db:"ruleset_version"`
    Module string `db:"module_version"`
    Fairness string `db:"fairness_suite_id"`
    Participant uint8 `db:"participant_count"`
    Status string `db:"final_status"`
    Version uint64 `db:"final_version"`
    Payload []byte `db:"payload"`
    Digest []byte `db:"record_digest"`
    ArchivedAt time.Time `db:"archived_at"`
}

func (a *Archive) queryRow(ctx context.Context, id gamecore.InstanceID) (storedRow, error) {
    var row archiveQueryRow
    err := a.conn.QueryRowCtx(ctx, &row, `
SELECT instance_id, game_id, ruleset_version, module_version, fairness_suite_id,
       participant_count, final_status, final_version, payload, record_digest, archived_at
FROM game_final_records
WHERE instance_id = ?`, string(id))
	if errors.Is(err, sql.ErrNoRows) {
		return storedRow{}, fmt.Errorf("%w: instance %s", ErrArchiveNotFound, id)
	}
	if err != nil {
		return storedRow{}, fmt.Errorf("read game final record: %w", err)
	}
	return storedRow{
        instanceID: row.InstanceID, gameID: row.GameID, ruleset: row.Ruleset,
        module: row.Module, fairness: row.Fairness, participant: row.Participant,
        status: row.Status, version: row.Version, payload: row.Payload,
        digest: row.Digest, archivedAt: row.ArchivedAt,
    }, nil
}

func restoreRecord(stored storedRow) (gamecore.FinalRecord, error) {
	descriptor, err := gamecore.NewDescriptor(
		gamecore.GameID(stored.gameID),
		gamecore.RulesetVersion(stored.ruleset),
		gamecore.ModuleVersion(stored.module),
		gamecore.FairnessSuiteID(stored.fairness),
		stored.participant,
	)
	if err != nil {
		return gamecore.FinalRecord{}, fmt.Errorf("%w: descriptor: %v", ErrArchiveCorrupt, err)
	}
	record, err := gamecore.NewFinalRecord(
		gamecore.InstanceID(stored.instanceID),
		descriptor,
		gamecore.FinalStatus(stored.status),
		stored.version,
		stored.payload,
	)
	if err != nil {
		return gamecore.FinalRecord{}, fmt.Errorf("%w: final record: %v", ErrArchiveCorrupt, err)
	}
	digest := record.Digest()
	if len(stored.digest) != len(digest) || !bytes.Equal(stored.digest, digest[:]) {
		return gamecore.FinalRecord{}, fmt.Errorf("%w: digest mismatch for instance %s", ErrArchiveCorrupt, stored.instanceID)
	}
	return record, nil
}
