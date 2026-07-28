package mysqlstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/application"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
)

type transaction struct{ tx *sql.Tx }

func (t *transaction) ClaimCommand(
	ctx context.Context,
	actor domain.AccountID,
	command application.Command,
	createdAt time.Time,
) (application.StoredCommandResult, bool, error) {
	commandJSON, err := json.Marshal(command)
	if err != nil {
		return application.StoredCommandResult{}, false, err
	}
	if _, err := t.tx.ExecContext(ctx, `
INSERT INTO doudizhu_command_results (
    actor_account_id, command_id, aggregate_type, aggregate_id, client_sequence,
    payload_digest, command_json, result_json, accepted, created_at, completed_at
) VALUES (?, ?, ?, ?, ?, ?, ?, NULL, NULL, ?, NULL)
ON DUPLICATE KEY UPDATE command_id = VALUES(command_id)`,
		actor, command.CommandID, command.AggregateType, command.AggregateID, command.ClientSeq,
		command.PayloadDigest[:], commandJSON, createdAt.UTC()); err != nil {
		return application.StoredCommandResult{}, false, err
	}

	var storedCommandJSON []byte
	var resultJSON sql.NullString
	var completedAt sql.NullTime
	if err := t.tx.QueryRowContext(ctx, `
SELECT command_json, result_json, completed_at
FROM doudizhu_command_results
WHERE actor_account_id = ? AND command_id = ?
FOR UPDATE`, actor, command.CommandID).Scan(&storedCommandJSON, &resultJSON, &completedAt); err != nil {
		return application.StoredCommandResult{}, false, err
	}
	var stored application.StoredCommandResult
	if err := json.Unmarshal(storedCommandJSON, &stored.Command); err != nil {
		return application.StoredCommandResult{}, false, fmt.Errorf("decode stored doudizhu command: %w", err)
	}
	if !resultJSON.Valid {
		if completedAt.Valid {
			return application.StoredCommandResult{}, false, fmt.Errorf("doudizhu command result is incomplete")
		}
		return stored, false, nil
	}
	if !completedAt.Valid {
		return application.StoredCommandResult{}, false, fmt.Errorf("doudizhu command completion time is missing")
	}
	if err := json.Unmarshal([]byte(resultJSON.String), &stored.Result); err != nil {
		return application.StoredCommandResult{}, false, fmt.Errorf("decode stored doudizhu result: %w", err)
	}
	return stored, true, nil
}

func (t *transaction) CompleteCommand(
	ctx context.Context,
	actor domain.AccountID,
	commandID string,
	result application.CommandResult,
	completedAt time.Time,
) error {
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return err
	}
	updated, err := t.tx.ExecContext(ctx, `
UPDATE doudizhu_command_results
SET result_json = ?, accepted = ?, completed_at = ?
WHERE actor_account_id = ? AND command_id = ? AND result_json IS NULL`,
		resultJSON, result.Accepted, completedAt.UTC(), actor, commandID)
	if err != nil {
		return err
	}
	rows, err := updated.RowsAffected()
	if err != nil {
		return err
	}
	if rows != 1 {
		return fmt.Errorf("complete doudizhu command: %w", application.ErrOptimisticConflict)
	}
	return nil
}

func (t *transaction) LockClientSequence(ctx context.Context, aggregateType domain.AggregateType, aggregateID string, actor domain.AccountID) (uint64, error) {
	if _, err := t.tx.ExecContext(ctx, `
INSERT INTO doudizhu_client_sequences (
    aggregate_type, aggregate_id, actor_account_id, last_sequence, updated_at
) VALUES (?, ?, ?, 0, UTC_TIMESTAMP(6))
ON DUPLICATE KEY UPDATE aggregate_id = VALUES(aggregate_id)`, aggregateType, aggregateID, actor); err != nil {
		return 0, err
	}
	var sequence uint64
	if err := t.tx.QueryRowContext(ctx, `
SELECT last_sequence
FROM doudizhu_client_sequences
WHERE aggregate_type = ? AND aggregate_id = ? AND actor_account_id = ?
FOR UPDATE`, aggregateType, aggregateID, actor).Scan(&sequence); err != nil {
		return 0, err
	}
	return sequence, nil
}

func (t *transaction) SaveClientSequence(ctx context.Context, aggregateType domain.AggregateType, aggregateID string, actor domain.AccountID, sequence uint64, updatedAt time.Time) error {
	result, err := t.tx.ExecContext(ctx, `
UPDATE doudizhu_client_sequences
SET last_sequence = ?, updated_at = ?
WHERE aggregate_type = ? AND aggregate_id = ? AND actor_account_id = ? AND last_sequence < ?`,
		sequence, updatedAt.UTC(), aggregateType, aggregateID, actor, sequence)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows != 1 {
		return application.ErrSequenceConflict
	}
	return nil
}

func (t *transaction) InsertRoom(ctx context.Context, snapshot domain.RoomSnapshot, createdAt time.Time) error {
	encoded, err := json.Marshal(snapshot)
	if err != nil {
		return err
	}
	_, err = t.tx.ExecContext(ctx, `
INSERT INTO doudizhu_rooms (
    room_id, owner_account_id, status, active_hand_id, aggregate_version,
    snapshot_json, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		snapshot.ID, snapshot.OwnerID, snapshot.Status, nullableString(string(snapshot.ActiveHandID)), snapshot.Version,
		encoded, createdAt.UTC(), createdAt.UTC())
	return translateInsertError(err)
}

func (t *transaction) LoadRoomForUpdate(ctx context.Context, id domain.RoomID) (domain.RoomSnapshot, error) {
	var encoded []byte
	if err := t.tx.QueryRowContext(ctx, `
SELECT snapshot_json
FROM doudizhu_rooms
WHERE room_id = ?
FOR UPDATE`, id).Scan(&encoded); err != nil {
		return domain.RoomSnapshot{}, translateNotFound(err)
	}
	var snapshot domain.RoomSnapshot
	if err := json.Unmarshal(encoded, &snapshot); err != nil {
		return domain.RoomSnapshot{}, fmt.Errorf("decode doudizhu room snapshot: %w", err)
	}
	return snapshot, nil
}

func (t *transaction) UpdateRoom(ctx context.Context, snapshot domain.RoomSnapshot, previousVersion uint64, updatedAt time.Time) error {
	encoded, err := json.Marshal(snapshot)
	if err != nil {
		return err
	}
	result, err := t.tx.ExecContext(ctx, `
UPDATE doudizhu_rooms
SET owner_account_id = ?, status = ?, active_hand_id = ?, aggregate_version = ?, snapshot_json = ?, updated_at = ?
WHERE room_id = ? AND aggregate_version = ?`,
		snapshot.OwnerID, snapshot.Status, nullableString(string(snapshot.ActiveHandID)), snapshot.Version, encoded, updatedAt.UTC(),
		snapshot.ID, previousVersion)
	return optimisticResult(result, err)
}

func (t *transaction) InsertHand(ctx context.Context, snapshot domain.HandSnapshot, createdAt time.Time) error {
	encoded, err := json.Marshal(snapshot)
	if err != nil {
		return err
	}
	_, err = t.tx.ExecContext(ctx, `
INSERT INTO doudizhu_hands (
    hand_id, room_id, phase, reveal_key_id, aggregate_version,
    snapshot_json, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		snapshot.ID, snapshot.RoomID, snapshot.Phase, snapshot.RevealKeyID, snapshot.Version,
		encoded, createdAt.UTC(), createdAt.UTC())
	return translateInsertError(err)
}

func (t *transaction) LoadHandForUpdate(ctx context.Context, id domain.HandID) (domain.HandSnapshot, error) {
	var encoded []byte
	if err := t.tx.QueryRowContext(ctx, `
SELECT snapshot_json
FROM doudizhu_hands
WHERE hand_id = ?
FOR UPDATE`, id).Scan(&encoded); err != nil {
		return domain.HandSnapshot{}, translateNotFound(err)
	}
	var snapshot domain.HandSnapshot
	if err := json.Unmarshal(encoded, &snapshot); err != nil {
		return domain.HandSnapshot{}, fmt.Errorf("decode doudizhu hand snapshot: %w", err)
	}
	return snapshot, nil
}

func (t *transaction) UpdateHand(ctx context.Context, snapshot domain.HandSnapshot, previousVersion uint64, updatedAt time.Time) error {
	encoded, err := json.Marshal(snapshot)
	if err != nil {
		return err
	}
	result, err := t.tx.ExecContext(ctx, `
UPDATE doudizhu_hands
SET phase = ?, reveal_key_id = ?, aggregate_version = ?, snapshot_json = ?, updated_at = ?
WHERE hand_id = ? AND aggregate_version = ?`,
		snapshot.Phase, snapshot.RevealKeyID, snapshot.Version, encoded, updatedAt.UTC(), snapshot.ID, previousVersion)
	return optimisticResult(result, err)
}

func (t *transaction) InsertContributionRecord(ctx context.Context, record application.ProtectedContributionRecord) error {
	_, err := t.tx.ExecContext(ctx, `
INSERT INTO doudizhu_contribution_records (
    record_id, hand_id, seat_number, actor_account_id, command_id,
    contribution_digest, protection_key_id, nonce, ciphertext, aad_digest, created_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		record.RecordID, record.HandID, record.Seat, record.ActorAccountID, record.CommandID,
		record.ContributionDigest[:], record.ProtectionKeyID, record.Nonce, record.Ciphertext, record.AADDigest[:], record.CreatedAt.UTC())
	return translateInsertError(err)
}

func (t *transaction) AppendOutbox(ctx context.Context, events []application.OutboxEvent) error {
	for _, event := range events {
		_, err := t.tx.ExecContext(ctx, `
INSERT INTO doudizhu_outbox_events (
    event_id, event_protocol, event_name, aggregate_type, aggregate_id,
    aggregate_version, occurred_at, causation_command_id, actor_account_id,
    payload_json, published_at, delivery_attempts
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NULL, 0)`,
			event.EventID, event.Protocol, event.Name, event.AggregateType, event.AggregateID,
			event.AggregateVersion, event.OccurredAt.UTC(), event.CausationCommandID, event.ActorAccountID,
			event.PayloadJSON)
		if err != nil {
			return translateInsertError(err)
		}
	}
	return nil
}

func optimisticResult(result sql.Result, err error) error {
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows != 1 {
		return application.ErrOptimisticConflict
	}
	return nil
}

func translateNotFound(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return application.ErrNotFound
	}
	return err
}

func translateInsertError(err error) error {
	if err == nil {
		return nil
	}
	// go-sql-driver/mysql formats duplicate-key failures as "Error 1062 ...".
	// Keeping the adapter independent of driver concrete types also makes SQL
	// tests possible with standard database/sql fakes.
	if strings.Contains(err.Error(), "Error 1062") || strings.Contains(strings.ToLower(err.Error()), "duplicate entry") {
		return application.ErrAlreadyExists
	}
	return err
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}
