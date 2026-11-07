package mysqlstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
)

func (s *Store) LoadHand(ctx context.Context, id domain.HandID) (domain.HandSnapshot, error) {
	if s == nil || s.database == nil || s.database.DB() == nil {
		return domain.HandSnapshot{}, fmt.Errorf("mysql doudizhu store is not configured")
	}
	var row struct {
		Snapshot []byte `db:"snapshot_json"`
	}
	if err := s.goZero.QueryRowCtx(ctx, &row, `
SELECT snapshot_json
FROM doudizhu_hands
WHERE hand_id = ?`, id); err != nil {
		return domain.HandSnapshot{}, translateNotFound(err)
	}
	var snapshot domain.HandSnapshot
	if err := json.Unmarshal(row.Snapshot, &snapshot); err != nil {
		return domain.HandSnapshot{}, fmt.Errorf("decode doudizhu hand snapshot: %w", err)
	}
	if _, err := domain.RestoreHand(snapshot); err != nil {
		return domain.HandSnapshot{}, fmt.Errorf("validate doudizhu hand snapshot: %w", err)
	}
	return snapshot, nil
}

func (s *Store) LoadRoom(ctx context.Context, id domain.RoomID) (domain.RoomSnapshot, error) {
	if s == nil || s.database == nil || s.database.DB() == nil {
		return domain.RoomSnapshot{}, fmt.Errorf("mysql doudizhu store is not configured")
	}
	var row struct {
		Snapshot []byte `db:"snapshot_json"`
	}
	if err := s.goZero.QueryRowCtx(ctx, &row, `
SELECT snapshot_json
FROM doudizhu_rooms
WHERE room_id = ?`, id); err != nil {
		return domain.RoomSnapshot{}, translateNotFound(err)
	}
	var snapshot domain.RoomSnapshot
	if err := json.Unmarshal(row.Snapshot, &snapshot); err != nil {
		return domain.RoomSnapshot{}, fmt.Errorf("decode doudizhu room snapshot: %w", err)
	}
	if snapshot.ID != id {
		return domain.RoomSnapshot{}, fmt.Errorf("doudizhu room identity mismatch")
	}
	if _, err := domain.RestoreRoom(snapshot); err != nil {
		return domain.RoomSnapshot{}, fmt.Errorf("validate doudizhu room snapshot: %w", err)
	}
	return snapshot, nil
}

func (s *Store) RoomClientSequence(ctx context.Context, roomID domain.RoomID, actor domain.AccountID) (uint64, error) {
	var row struct {
		Sequence uint64 `db:"last_sequence"`
	}
	err := s.goZero.QueryRowCtx(ctx, &row, `
SELECT last_sequence FROM doudizhu_client_sequences
WHERE aggregate_type = 'room' AND aggregate_id = ? AND actor_account_id = ?`, string(roomID), string(actor))
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("query room client sequence: %w", err)
	}
	return row.Sequence, nil
}

func (s *Store) HandClientSequence(ctx context.Context, handID domain.HandID, actor domain.AccountID) (uint64, error) {
	var row struct {
		Sequence uint64 `db:"last_sequence"`
	}
	err := s.goZero.QueryRowCtx(ctx, &row, `
SELECT last_sequence FROM doudizhu_client_sequences
WHERE aggregate_type = 'hand' AND aggregate_id = ? AND actor_account_id = ?`, string(handID), string(actor))
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("query hand client sequence: %w", err)
	}
	return row.Sequence, nil
}

// FindRecoverableRoomIDs uses authoritative room snapshots, never client
// device storage or guessed room IDs. JSON_CONTAINS matches an exact seat
// account identity (not wildcard LIKE/JSON_SEARCH). This uses the existing
// nonclosed-status index; cap results rather than silently truncating them.
// If the lobby grows substantially, migrate to a transactionally maintained
// indexed membership projection instead of weakening this authorization.
func (s *Store) FindRecoverableRoomIDs(ctx context.Context, actor domain.AccountID) ([]domain.RoomID, error) {
	if s == nil || s.database == nil || s.database.DB() == nil ||
		actor == "" || len(actor) > 128 {
		return nil, fmt.Errorf("invalid recoverable room query")
	}
	rows, err := s.database.DB().QueryContext(ctx, `
SELECT room_id FROM doudizhu_rooms
WHERE status IN ('HAND_ACTIVE', 'READY', 'WAITING_PLAYERS')
  AND JSON_CONTAINS(snapshot_json, JSON_OBJECT('AccountID', ?), '$.Seats') = 1
ORDER BY CASE status WHEN 'HAND_ACTIVE' THEN 0 WHEN 'READY' THEN 1 ELSE 2 END,
         updated_at DESC, room_id ASC
LIMIT 33`, string(actor))
	if err != nil {
		return nil, fmt.Errorf("find recoverable doudizhu rooms: %w", err)
	}
	defer rows.Close()
	ids := make([]domain.RoomID, 0, 8)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan recoverable room: %w", err)
		}
		if id == "" || len(id) > 128 {
			return nil, fmt.Errorf("invalid recoverable room identity")
		}
		ids = append(ids, domain.RoomID(id))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate recoverable rooms: %w", err)
	}
	if len(ids) > 32 {
		return nil, fmt.Errorf("too many recoverable rooms; narrow discovery")
	}
	return ids, nil
}
