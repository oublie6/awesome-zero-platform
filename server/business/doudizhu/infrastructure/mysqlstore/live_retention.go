package mysqlstore

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
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
func (j *LiveJournal) PruneArchivedHand(ctx context.Context, id domain.HandID, olderThan time.Duration) error {
	if j == nil || j.db == nil || id == "" || len(id) > 128 || olderThan < MinimumLiveArchiveRetention {
		return ErrLivePruneNotEligible
	}
	tx, err := j.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil { return err }
	defer tx.Rollback()
	var committed uint64
	if err := tx.QueryRowContext(ctx, `SELECT committed_version FROM doudizhu_live_writers
WHERE hand_id=? FOR UPDATE`, id).Scan(&committed); err != nil { return err }
	var phase string
	if err := tx.QueryRowContext(ctx, `SELECT phase FROM doudizhu_hands
WHERE hand_id=? FOR UPDATE`, id).Scan(&phase); err != nil { return err }
	if phase != string(domain.HandCompleted) { return ErrLivePruneNotEligible }
	var gameID,ruleset,module,fairness,status string
	var participants uint8
	var finalVersion uint64
	var payload,digest []byte
	var archived time.Time
	if err := tx.QueryRowContext(ctx, `SELECT game_id,ruleset_version,module_version,
fairness_suite_id,participant_count,final_status,final_version,payload,record_digest,archived_at
FROM game_final_records WHERE instance_id=? FOR UPDATE`,id).Scan(&gameID,&ruleset,&module,
		&fairness,&participants,&status,&finalVersion,&payload,&digest,&archived); err != nil { return err }
	if status != string(gamecore.FinalStatusCompleted) || committed < 2 || finalVersion != committed ||
		archived.After(time.Now().UTC().Add(-olderThan)) { return ErrLivePruneNotEligible }
	desc,err:=gamecore.NewDescriptor(gamecore.GameID(gameID),gamecore.RulesetVersion(ruleset),
		gamecore.ModuleVersion(module),gamecore.FairnessSuiteID(fairness),participants)
	if err != nil { return ErrLiveReplayIntegrity }
	record,err:=gamecore.NewFinalRecord(gamecore.InstanceID(id),desc,
		gamecore.FinalStatus(status),finalVersion,payload)
	if err != nil { return ErrLiveReplayIntegrity }
	actual:=record.Digest()
	if !bytes.Equal(actual[:],digest) { return ErrLiveReplayIntegrity }
	var actionCount uint64
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM doudizhu_live_actions
WHERE hand_id=?`, id).Scan(&actionCount); err != nil { return err }
	if actionCount != committed-1 { return ErrLiveReplayIntegrity }
	// The final archive and room/fairness audit remain; remove only redundant
	// private recovery material and the complete per-hand transition log.
	for _,table:=range []string{"doudizhu_live_checkpoints","doudizhu_live_actions",
		"doudizhu_protected_server_seeds","doudizhu_live_writers"} {
		if _,err:=tx.ExecContext(ctx,"DELETE FROM "+table+" WHERE hand_id=?",id);err!=nil{return fmt.Errorf("prune %s: %w",table,err)}
	}
	return tx.Commit()
}
