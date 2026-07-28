package mysqlarchive

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/oublie6/awesome-zero-platform/server/business/gamecore"
)

var ErrArchiveConflict = errors.New("game final archive conflicts with an existing record")

type Clock interface {
	Now() time.Time
}

type Archive struct {
	db    *sql.DB
	clock Clock
}

func New(db *sql.DB, clock Clock) (*Archive, error) {
	if db == nil || clock == nil {
		return nil, fmt.Errorf("mysql game archive configuration is invalid")
	}
	return &Archive{db: db, clock: clock}, nil
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
	_, err := a.db.ExecContext(ctx, `
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
	var stored struct {
		gameID, ruleset, module, fairness, status string
		participant                           uint8
		version                               uint64
		payload, digest                       []byte
	}
	err = a.db.QueryRowContext(ctx, `
SELECT game_id, ruleset_version, module_version, fairness_suite_id,
       participant_count, final_status, final_version, payload, record_digest
FROM game_final_records
WHERE instance_id = ?`, string(record.InstanceID())).Scan(
		&stored.gameID, &stored.ruleset, &stored.module, &stored.fairness,
		&stored.participant, &stored.status, &stored.version, &stored.payload, &stored.digest,
	)
	if err != nil {
		return fmt.Errorf("read game final record: %w", err)
	}
	if stored.gameID != string(descriptor.GameID()) || stored.ruleset != string(descriptor.RulesetVersion()) ||
		stored.module != string(descriptor.ModuleVersion()) || stored.fairness != string(descriptor.FairnessSuiteID()) ||
		stored.participant != descriptor.ParticipantCount() || stored.status != string(record.Status()) ||
		stored.version != record.Version() || !bytes.Equal(stored.payload, payload) || !bytes.Equal(stored.digest, digest[:]) {
		return fmt.Errorf("%w: instance %s", ErrArchiveConflict, record.InstanceID())
	}
	return nil
}
