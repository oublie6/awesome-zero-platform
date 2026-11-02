package mysqlstore

import (
	"context"
	"encoding/json"
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
