package mysqlstore

import (
 "context"
 "database/sql"
 "encoding/json"
 "errors"
 "fmt"
 "github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
)

func (j *LiveJournal) HasLiveWriter(ctx context.Context, id domain.HandID) (bool, error) {
 var marker int
 err := j.db.QueryRowContext(ctx, "SELECT 1 FROM doudizhu_live_writers WHERE hand_id=?", id).Scan(&marker)
 if errors.Is(err, sql.ErrNoRows) { return false, nil }
 return err == nil, err
}

func (j *LiveJournal) HasLiveCommand(ctx context.Context, id domain.HandID, commandID string) (bool, error) {
 if id == "" || commandID == "" || len(commandID) > 128 { return false, nil }
 var marker int
 err := j.db.QueryRowContext(ctx, "SELECT 1 FROM doudizhu_live_actions WHERE hand_id=? AND command_id=? LIMIT 1", id, commandID).Scan(&marker)
 if errors.Is(err, sql.ErrNoRows) { return false, nil }
 return err == nil, err
}

func (j *LiveJournal) ReadTerminalHistory(ctx context.Context, id domain.HandID) (domain.HandSnapshot, domain.HandSnapshot, []LiveAction, error) {
 var snapshot, genesis domain.HandSnapshot
 var raw []byte
 if err := j.db.QueryRowContext(ctx, "SELECT snapshot_json FROM doudizhu_hands WHERE hand_id=?", id).Scan(&raw); err != nil {
  return snapshot, genesis, nil, err
 }
 if err := json.Unmarshal(raw, &snapshot); err != nil { return snapshot, genesis, nil, fmt.Errorf("%w: terminal hand JSON: %v", ErrLiveReplayIntegrity, err) }
 if snapshot.ID != id || !snapshot.Phase.Terminal() { return snapshot, genesis, nil, ErrLiveReplayIntegrity }
 if _, err := domain.RestoreHand(snapshot); err != nil { return snapshot, genesis, nil, fmt.Errorf("%w: terminal hand: %v", ErrLiveReplayIntegrity, err) }
 var watermark uint64
 if err := j.db.QueryRowContext(ctx, "SELECT committed_version,genesis_snapshot FROM doudizhu_live_writers WHERE hand_id=?", id).Scan(&watermark, &raw); err != nil {
  return snapshot, genesis, nil, err
 }
 if err := json.Unmarshal(raw, &genesis); err != nil { return snapshot, genesis, nil, fmt.Errorf("%w: genesis JSON: %v", ErrLiveReplayIntegrity, err) }
 if genesis.ID != id || genesis.Phase != domain.HandDealing { return snapshot, genesis, nil, ErrLiveReplayIntegrity }
 if _, err := domain.RestoreHand(genesis); err != nil { return snapshot, genesis, nil, fmt.Errorf("%w: genesis: %v", ErrLiveReplayIntegrity, err) }
 actions, err := j.Read(ctx, id)
 if err != nil { return snapshot, genesis, nil, err }
 if watermark != uint64(len(actions))+1 { return snapshot, genesis, nil, ErrLiveReplayIntegrity }
 return snapshot, genesis, actions, nil
}
