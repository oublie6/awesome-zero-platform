package mysqlstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/livehand"
	"github.com/oublie6/awesome-zero-platform/server/business/gamecore"
)

type TurnTimeouts struct{ Bidding, Playing time.Duration }

func (j *LiveJournal) EnableTurnDeadlines(ctx context.Context, limits TurnTimeouts) error {
	if j == nil || j.db == nil || limits.Bidding <= 0 || limits.Playing <= 0 ||
		limits.Bidding > 24*time.Hour || limits.Playing > 24*time.Hour {
		return errors.New("invalid turn deadline config")
	}
	// Fail closed if the additive migration has not actually been applied.
	var table string
	if err := j.db.QueryRowContext(ctx, `SELECT table_name FROM information_schema.tables
WHERE table_schema=DATABASE() AND table_name='doudizhu_turn_deadlines'`).Scan(&table); err != nil {
		return fmt.Errorf("turn deadline table unavailable: %w", err)
	}
	j.turnTimeouts = &limits
	return nil
}

type TurnDeadline struct {
	HandID   domain.HandID
	Version  uint64
	Phase    string
	Seat     uint8
	Deadline time.Time
}

func (j *LiveJournal) writeTurnDeadline(ctx context.Context, tx *sql.Tx, id domain.HandID,
	version uint64, phase string, seat uint8) error {
	if j.turnTimeouts == nil {
		return nil
	}
	timeout := j.turnTimeouts.Bidding
	if phase == livehand.PhasePlaying {
		timeout = j.turnTimeouts.Playing
	}
	if (phase != livehand.PhaseBidding && phase != livehand.PhasePlaying) || seat < 1 || seat > 3 || version == 0 {
		return errors.New("invalid committed turn deadline")
	}
	// DATETIME(3) is persisted in UTC, computed by MySQL inside the writer tx.
	_, err := tx.ExecContext(ctx, `INSERT INTO doudizhu_turn_deadlines(
hand_id,live_version,phase,seat,deadline_at,next_attempt_at,updated_at)
VALUES(?,?,?,?,DATE_ADD(UTC_TIMESTAMP(3),INTERVAL ? MICROSECOND),UTC_TIMESTAMP(3),UTC_TIMESTAMP(6))
ON DUPLICATE KEY UPDATE live_version=VALUES(live_version),phase=VALUES(phase),seat=VALUES(seat),
deadline_at=VALUES(deadline_at),next_attempt_at=UTC_TIMESTAMP(3),updated_at=UTC_TIMESTAMP(6)`,
		string(id), version, phase, seat, timeout.Microseconds())
	return err
}

// Keep a durable terminal repair cursor after the final live action commits.
// The separate domain lifecycle transaction may fail or lose its ACK.
func (j *LiveJournal) writeTerminalRepair(ctx context.Context, tx *sql.Tx,
	id domain.HandID, version uint64, phase string) error {
	result, err := tx.ExecContext(ctx, `UPDATE doudizhu_turn_deadlines SET
live_version=?,phase=?,deadline_at=UTC_TIMESTAMP(3),
next_attempt_at=UTC_TIMESTAMP(3),updated_at=UTC_TIMESTAMP(6)
WHERE hand_id=?`, version, phase, string(id))
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

func (j *LiveJournal) updateTurnDeadline(ctx context.Context, tx *sql.Tx, id domain.HandID, outcome gamecore.CommandOutcome) error {
	if outcome.Terminal {
		return j.writeTerminalRepair(ctx, tx, id, outcome.Version, livehand.PhaseCompleted)
	}
	var payload struct {
		Phase   string `json:"phase"`
		Bidding struct {
			CurrentBidder uint8 `json:"currentBidder"`
		} `json:"bidding"`
		Playing struct {
			CurrentSeat uint8 `json:"currentSeat"`
		} `json:"playing"`
		PlayingSeat uint8 `json:"playingSeat"`
	}
	if err := json.Unmarshal(outcome.Payload, &payload); err != nil {
		return err
	}
	switch payload.Phase {
	case livehand.PhaseBidding:
		return j.writeTurnDeadline(ctx, tx, id, outcome.Version, payload.Phase, payload.Bidding.CurrentBidder)
	case livehand.PhasePlaying:
		seat := payload.Playing.CurrentSeat
		if seat == 0 {
			seat = payload.PlayingSeat
		} // first landlord turn after final bid
		return j.writeTurnDeadline(ctx, tx, id, outcome.Version, payload.Phase, seat)
	case livehand.PhaseNoLandlord, livehand.PhaseCompleted:
		return j.writeTerminalRepair(ctx, tx, id, outcome.Version, payload.Phase)
	default:
		return fmt.Errorf("unexpected committed phase %q", payload.Phase)
	}
}
func (j *LiveJournal) GetTurnDeadline(ctx context.Context, id domain.HandID) (TurnDeadline, time.Time, error) {
	var result TurnDeadline
	var now time.Time
	if j == nil || j.turnTimeouts == nil {
		return result, now, errors.New("turn deadlines disabled")
	}
	var seat uint8
	err := j.db.QueryRowContext(ctx, `SELECT live_version,phase,seat,deadline_at,UTC_TIMESTAMP(3)
FROM doudizhu_turn_deadlines WHERE hand_id=?`, string(id)).
		Scan(&result.Version, &result.Phase, &seat, &result.Deadline, &now)
	if err != nil {
		return result, now, err
	}
	result.HandID = id
	result.Seat = seat
	result.Deadline = result.Deadline.UTC()
	return result, now.UTC(), nil
}
func (j *LiveJournal) PendingTurnDeliveries(ctx context.Context, limit int) ([]TurnDeadline, error) {
	if j == nil || j.turnTimeouts == nil {
		return nil, errors.New("turn deadlines disabled")
	}
	if limit < 1 || limit > 128 {
		return nil, errors.New("invalid pending delivery limit")
	}
	rows, err := j.db.QueryContext(ctx, `SELECT hand_id,live_version,phase,seat,deadline_at
FROM doudizhu_turn_deadlines WHERE next_attempt_at<=UTC_TIMESTAMP(3)
ORDER BY next_attempt_at,hand_id LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]TurnDeadline, 0, limit)
	for rows.Next() {
		var t TurnDeadline
		if err := rows.Scan(&t.HandID, &t.Version, &t.Phase, &t.Seat, &t.Deadline); err != nil {
			return nil, err
		}
		t.Deadline = t.Deadline.UTC()
		result = append(result, t)
	}
	return result, rows.Err()
}

// Delivery is advisory. Retry shortly after the deadline because
// go-queue/dq deletes jobs before callback and can drop callbacks after outage.
func (j *LiveJournal) MarkTurnDelivery(ctx context.Context, item TurnDeadline, succeeded bool) error {
	if j == nil || j.turnTimeouts == nil {
		return errors.New("turn deadlines disabled")
	}
	var query string
	if succeeded {
		query = `UPDATE doudizhu_turn_deadlines SET
next_attempt_at=GREATEST(DATE_ADD(UTC_TIMESTAMP(3),INTERVAL 2 SECOND),
    DATE_ADD(deadline_at,INTERVAL 2 SECOND))
WHERE hand_id=? AND live_version=? AND seat=? AND deadline_at=?`
	} else {
		query = `UPDATE doudizhu_turn_deadlines SET
next_attempt_at=DATE_ADD(UTC_TIMESTAMP(3),INTERVAL 2 SECOND)
WHERE hand_id=? AND live_version=? AND seat=? AND deadline_at=?`
	}
	_, err := j.db.ExecContext(ctx, query, string(item.HandID), item.Version, item.Seat, item.Deadline)
	return err
}

// Remove a repair cursor only after both the domain and a journal-verified
// final archive converge. The caller checks this before invoking the store.
// A cancelled/expired hand may still have a BIDDING or PLAYING cursor.
func (j *LiveJournal) ClearTerminalRepair(ctx context.Context, item TurnDeadline) error {
	if j == nil || j.turnTimeouts == nil || item.HandID == "" || item.Version == 0 ||
		item.Seat < 1 || item.Seat > 3 ||
		(item.Phase != livehand.PhaseNoLandlord && item.Phase != livehand.PhaseCompleted &&
			item.Phase != livehand.PhaseBidding && item.Phase != livehand.PhasePlaying) {
		return errors.New("invalid reconciled cursor")
	}
	_, err := j.db.ExecContext(ctx, `DELETE FROM doudizhu_turn_deadlines
WHERE hand_id=? AND live_version=? AND phase=? AND seat=? AND deadline_at=?`,
		string(item.HandID), item.Version, item.Phase, item.Seat, item.Deadline)
	return err
}

var ErrTurnExpired = errors.New("doudizhu turn deadline expired")

// requireCurrentTurn runs in the same MySQL tx after locking the writer row.
// Both human and system calls pass this gate. Only the internal zero-sequence
// timeout command may run after the deadline; all other late commands reject.
func (j *LiveJournal) requireCurrentTurn(ctx context.Context, tx *sql.Tx,
	id domain.HandID, commandID string, sequence, expectedVersion uint64, seat uint8) error {
	if j.turnTimeouts == nil {
		return nil
	}
	var version uint64
	var turnSeat uint8
	var deadline, now time.Time
	err := tx.QueryRowContext(ctx, `SELECT live_version,seat,deadline_at,UTC_TIMESTAMP(6)
FROM doudizhu_turn_deadlines WHERE hand_id=? FOR UPDATE`, string(id)).
		Scan(&version, &turnSeat, &deadline, &now)
	if err != nil {
		return fmt.Errorf("read committed turn guard: %w", err)
	}
	if version != expectedVersion || turnSeat != seat {
		return ErrLiveWatermarkConflict
	}
	automated := sequence == 0 && len(commandID) > 13 && commandID[:13] == "turn-timeout:"
	if automated {
		if now.Before(deadline) {
			return ErrTurnExpired
		} // early queue task cannot act
	} else if !now.Before(deadline) {
		return ErrTurnExpired // late user action cannot overtake an expired turn
	}
	return nil
}
