package mysqlstore

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
	"github.com/oublie6/awesome-zero-platform/server/business/gamecore"
)

var (
	ErrLiveWriterFenced      = errors.New("doudizhu live writer epoch fenced")
	ErrLiveWatermarkConflict = errors.New("doudizhu live committed version mismatch")
	ErrLiveReplayIntegrity   = errors.New("doudizhu live action log integrity failure")
)

// The log contains only publicly applied bid/play/pass payloads and a result
// hash. It does not store unrevealed cards, the server seed or final private
// hands. Those must be reconstructed from the separate protected seed and
// authoritative genesis snapshot.
type LiveAction struct {
	HandID          domain.HandID
	Version         uint64
	Epoch           uint64
	CommandID       string
	Actor           domain.AccountID
	ActorPosition   uint8
	ExpectedVersion uint64
	Payload         []byte
	ResultDigest    [32]byte
}

// One Doudizhu hand has at most 54 played cards and bounded bidding turns.
// Leave generous room for protocol evolution while preventing a corrupt or
// malicious journal from exhausting memory during startup replay.
const (
	maxLiveActionsPerHand     = 512
	maxLiveActionPayloadBytes = 16 << 10
)

type LiveJournal struct{ db *sql.DB }

func NewLiveJournal(db *sql.DB) (*LiveJournal, error) {
	if db == nil {
		return nil, fmt.Errorf("live journal requires mysql")
	}
	return &LiveJournal{db: db}, nil
}
func (j *LiveJournal) Create(ctx context.Context, genesis domain.HandSnapshot) error {
	if j == nil || j.db == nil || genesis.ID == "" || genesis.Phase != domain.HandDealing {
		return fmt.Errorf("invalid live genesis")
	}
	if _, err := domain.RestoreHand(genesis); err != nil {
		return err
	}
	b, err := json.Marshal(genesis)
	if err != nil {
		return err
	}
	_, err = j.db.ExecContext(ctx, `INSERT INTO doudizhu_live_writers(hand_id,owner_epoch,committed_version,genesis_snapshot,updated_at)
VALUES(?,1,1,?,UTC_TIMESTAMP(6))`, string(genesis.ID), b)
	if err != nil {
		return fmt.Errorf("create live journal: %w", err)
	}
	return nil
}

// DeleteUnused is only for compensating a setup whose enclosing durable
// MarkHandDealt command has not committed. It refuses to delete any action.
func (j *LiveJournal) DeleteUnused(ctx context.Context, id domain.HandID) error {
	result, err := j.db.ExecContext(ctx, `DELETE FROM doudizhu_live_writers WHERE hand_id=? AND committed_version=1
AND NOT EXISTS (SELECT 1 FROM doudizhu_live_actions WHERE hand_id=?)`, id, id)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected != 1 {
		return ErrLiveWatermarkConflict
	}
	return nil
}

// Acquire is a MySQL-fenced owner takeover. Every append checks the returned
// epoch under the same row lock as its new committed watermark.
func (j *LiveJournal) Acquire(ctx context.Context, id domain.HandID) (uint64, uint64, domain.HandSnapshot, error) {
	var genesis domain.HandSnapshot
	tx, err := j.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return 0, 0, genesis, err
	}
	defer tx.Rollback()
	var epoch, version uint64
	var raw []byte
	if err = tx.QueryRowContext(ctx, `SELECT owner_epoch,committed_version,genesis_snapshot FROM doudizhu_live_writers
WHERE hand_id=? FOR UPDATE`, id).Scan(&epoch, &version, &raw); err != nil {
		return 0, 0, genesis, err
	}
	if epoch == ^uint64(0) {
		return 0, 0, genesis, ErrLiveWriterFenced
	}
	if err = json.Unmarshal(raw, &genesis); err != nil {
		return 0, 0, genesis, fmt.Errorf("%w: genesis JSON: %v", ErrLiveReplayIntegrity, err)
	}
	if genesis.ID != id || genesis.Phase != domain.HandDealing {
		return 0, 0, genesis, ErrLiveReplayIntegrity
	}
	if _, err = domain.RestoreHand(genesis); err != nil {
		return 0, 0, genesis, fmt.Errorf("%w: %v", ErrLiveReplayIntegrity, err)
	}
	epoch++
	if _, err = tx.ExecContext(ctx, `UPDATE doudizhu_live_writers SET owner_epoch=?,updated_at=UTC_TIMESTAMP(6) WHERE hand_id=?`, epoch, id); err != nil {
		return 0, 0, genesis, err
	}
	if err = tx.Commit(); err != nil {
		return 0, 0, genesis, fmt.Errorf("ambiguous owner takeover: %w", err)
	}
	return epoch, version, genesis, nil
}
func (j *LiveJournal) Append(ctx context.Context, id domain.HandID, epoch uint64, commandID string, actor domain.AccountID, command gamecore.Command, outcome gamecore.CommandOutcome) error {
	if j == nil || j.db == nil || id == "" || epoch == 0 || commandID == "" || len(commandID) > 128 ||
		actor == "" || len(actor) > 128 || command.ActorPosition < 1 || command.ActorPosition > 3 ||
		outcome.Version == 0 || outcome.Version != command.ExpectedVersion+1 ||
		len(command.Payload) == 0 || len(command.Payload) > maxLiveActionPayloadBytes ||
		outcome.Version > maxLiveActionsPerHand+1 || !json.Valid(command.Payload) {
		return fmt.Errorf("invalid durable live action")
	}
	if outcome.Terminal && len(outcome.FinalPayload) == 0 {
		return fmt.Errorf("invalid final live action")
	}
	encodedResult, err := json.Marshal(outcome)
	if err != nil {
		return err
	}
	digest := sha256.Sum256(encodedResult)
	tx, err := j.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var foundEpoch, committed uint64
	if err = tx.QueryRowContext(ctx, `SELECT owner_epoch,committed_version FROM doudizhu_live_writers
WHERE hand_id=? FOR UPDATE`, id).Scan(&foundEpoch, &committed); err != nil {
		return err
	}
	if foundEpoch != epoch {
		return ErrLiveWriterFenced
	}
	if committed != command.ExpectedVersion {
		return ErrLiveWatermarkConflict
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO doudizhu_live_actions(
hand_id,state_version,owner_epoch,command_id,actor_account_id,actor_position,expected_version,
command_payload,result_digest,committed_at) VALUES(?,?,?,?,?,?,?,?,?,UTC_TIMESTAMP(6))`,
		id, outcome.Version, epoch, commandID, actor, command.ActorPosition, command.ExpectedVersion, command.Payload, digest[:]); err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `UPDATE doudizhu_live_writers
SET committed_version=?,updated_at=UTC_TIMESTAMP(6)
WHERE hand_id=? AND owner_epoch=? AND committed_version=?`, outcome.Version, id, epoch, committed)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows != 1 {
		return ErrLiveWatermarkConflict
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("ambiguous live action commit: %w", err)
	}
	return nil
}
func (j *LiveJournal) Read(ctx context.Context, id domain.HandID) ([]LiveAction, error) {
	rows, err := j.db.QueryContext(ctx, `SELECT state_version,owner_epoch,command_id,actor_account_id,
actor_position,expected_version,command_payload,result_digest
FROM doudizhu_live_actions WHERE hand_id=? ORDER BY state_version ASC LIMIT ?`, id, maxLiveActionsPerHand+1)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]LiveAction, 0, 64)
	expected := uint64(2)
	var previousEpoch uint64
	for rows.Next() {
		if len(out) >= maxLiveActionsPerHand {
			return nil, ErrLiveReplayIntegrity
		}
		var action LiveAction
		var digest []byte
		var pos uint64
		action.HandID = id
		if err := rows.Scan(&action.Version, &action.Epoch, &action.CommandID, &action.Actor, &pos,
			&action.ExpectedVersion, &action.Payload, &digest); err != nil {
			return nil, err
		}
		if pos < 1 || pos > 3 {
			return nil, ErrLiveReplayIntegrity
		}
		action.ActorPosition = uint8(pos)
		if err := validateReplayAction(action, expected, previousEpoch, digest); err != nil {
			return nil, err
		}
		previousEpoch = action.Epoch
		copy(action.ResultDigest[:], digest)
		out = append(out, action)
		expected++
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// Fail closed before building the in-memory receipt index. A malformed action
// must not be replayed as a different actor, command identity or writer epoch.
func validateReplayAction(action LiveAction, expectedVersion, previousEpoch uint64, digest []byte) error {
	if action.Version != expectedVersion || action.ExpectedVersion != expectedVersion-1 ||
		action.Epoch == 0 || action.Epoch < previousEpoch ||
		action.CommandID == "" || len(action.CommandID) > 128 ||
		action.Actor == "" || len(action.Actor) > 128 ||
		action.ActorPosition < 1 || action.ActorPosition > 3 ||
		len(action.Payload) == 0 || len(action.Payload) > maxLiveActionPayloadBytes ||
		!json.Valid(action.Payload) || len(digest) != sha256.Size {
		return ErrLiveReplayIntegrity
	}
	return nil
}

func (j *LiveJournal) List(ctx context.Context) ([]domain.HandID, error) {
	rows, err := j.db.QueryContext(ctx, `SELECT hand_id FROM doudizhu_live_writers ORDER BY hand_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.HandID, 0, 64)
	for rows.Next() {
		var id domain.HandID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}
