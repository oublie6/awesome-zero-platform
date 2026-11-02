package mysqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/livehand"
	"github.com/oublie6/awesome-zero-platform/server/business/gamecore"
)

var ErrLivePruneNotEligible = errors.New("doudizhu live history cannot be pruned before final archive retention")

const MinimumLiveArchiveRetention = 30 * 24 * time.Hour

// PruneArchivedHand is an explicit bounded maintenance operation, never a
// background gameplay side effect. Full replay receipts are retained for at
// least thirty days after an independently verified completed final archive.
// Active, unarchived, aborted-in-flight or mismatched records fail closed.
//
// Call only during an authorized maintenance window. The caller must ensure
// external forensic/retention obligations allow deletion; final game evidence,
// durable room/fairness command results and the audit outbox are preserved.
// FinalArchiveReader is owned and implemented by gamecore. The Doudizhu
// persistence adapter is not permitted to query gamecore's database tables.
type FinalArchiveReader interface {
	LoadFinalRecord(context.Context, gamecore.InstanceID) (gamecore.FinalRecord, time.Time, error)
}

func (j *LiveJournal) PruneArchivedHand(ctx context.Context, id domain.HandID, olderThan time.Duration, archive FinalArchiveReader) error {
	if j == nil || j.db == nil || archive == nil || id == "" || len(id) > 128 ||
		olderThan < MinimumLiveArchiveRetention {
		return ErrLivePruneNotEligible
	}
	final, archived, err := archive.LoadFinalRecord(ctx, gamecore.InstanceID(id))
	if err != nil {
		return ErrLivePruneNotEligible
	}
	if final.InstanceID() != gamecore.InstanceID(id) ||
		final.Status() != gamecore.FinalStatusCompleted ||
		final.Version() < 2 ||
		archived.After(time.Now().UTC().Add(-olderThan)) {
		return ErrLivePruneNotEligible
	}
	if _, err := livehand.VerifyFinalRecord(final); err != nil {
		return ErrLiveReplayIntegrity
	}
	tx, err := j.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var committed uint64
	if err := tx.QueryRowContext(ctx, `SELECT committed_version FROM doudizhu_live_writers
WHERE hand_id=? FOR UPDATE`, id).Scan(&committed); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrLivePruneNotEligible
		}
		return err
	}
	var phase string
	if err := tx.QueryRowContext(ctx, `SELECT phase FROM doudizhu_hands
WHERE hand_id=? FOR UPDATE`, id).Scan(&phase); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrLivePruneNotEligible
		}
		return err
	}
	if phase != string(domain.HandCompleted) || committed != final.Version() {
		return ErrLivePruneNotEligible
	}
	var actionCount uint64
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM doudizhu_live_actions
WHERE hand_id=?`, id).Scan(&actionCount); err != nil {
		return err
	}
	if actionCount != committed-1 {
		return ErrLiveReplayIntegrity
	}
	// The final archive and room/fairness audit remain; remove only redundant
	// private recovery material and the complete per-hand transition log.
	for _, table := range []string{"doudizhu_live_checkpoints", "doudizhu_live_actions",
		"doudizhu_protected_server_seeds", "doudizhu_live_writers"} {
		if _, err := tx.ExecContext(ctx, "DELETE FROM "+table+" WHERE hand_id=?", id); err != nil {
			return fmt.Errorf("prune %s: %w", table, err)
		}
	}
	return tx.Commit()
}
