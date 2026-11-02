package runtime

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/livehand"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/infrastructure/mysqlstore"
	"github.com/oublie6/awesome-zero-platform/server/business/gamecore"
)

const (
	liveRecoveryCheckpointV1 = "doudizhu-live-recovery-checkpoint-v1"
	liveCheckpointInterval   = 32
	maxLiveCheckpointActions = 512
)

type checkpointReceipt struct {
	CommandID string                  `json:"commandId"`
	Actor     domain.AccountID        `json:"actor"`
	Command   gamecore.Command        `json:"command"`
	Outcome   gamecore.CommandOutcome `json:"outcome"`
}

type recoveryCheckpoint struct {
	Version        string              `json:"v"`
	HandID         domain.HandID       `json:"handId"`
	StateVersion   uint64              `json:"stateVersion"`
	PublicDigest   [32]byte            `json:"publicDigest"`
	PrivateDigests [3][32]byte         `json:"privateDigests"`
	Receipts       []checkpointReceipt `json:"receipts"`
}

// The checkpoint is a complete, bounded encrypted transition-and-receipt
// snapshot. Only the sealed blob reaches MySQL. The journal remains the
// authoritative source of committed versions and supplies an integrity check
// for each checkpoint record during recovery.
func (c *LiveHandCoordinator) checkpointAfterCommit(id domain.HandID, epoch uint64, outcome gamecore.CommandOutcome, next gamecore.LiveGame) {
	if c == nil || c.journal == nil || next == nil ||
		outcome.Terminal || outcome.Version <= 1 ||
		(outcome.Version-1)%liveCheckpointInterval != 0 {
		return
	}
	game, ok := next.(*livehand.Game)
	if !ok {
		return
	}
	c.mu.RLock()
	saved := c.receipts[id]
	byVersion := make([]checkpointReceipt, outcome.Version-1)
	for commandID, receipt := range saved {
		version := receipt.outcome.Version
		if version <= 1 || version > outcome.Version {
			continue
		}
		if byVersion[version-2].CommandID != "" {
			c.mu.RUnlock()
			return
		}
		byVersion[version-2] = checkpointReceipt{
			CommandID: commandID,
			Actor: receipt.actor, Command: receipt.command, Outcome: receipt.outcome,
		}
	}
	c.mu.RUnlock()
	for i, record := range byVersion {
		if record.CommandID == "" || record.Outcome.Version != uint64(i+2) {
			return // a concurrent accepted command has not published its receipt
		}
	}
	cp := recoveryCheckpoint{
		Version: liveRecoveryCheckpointV1, HandID: id,
		StateVersion: outcome.Version, Receipts: byVersion,
	}
	var err error
	cp.PublicDigest, cp.PrivateDigests, err = checkpointViewDigests(game)
	if err != nil {
		return
	}
	payload, err := json.Marshal(cp)
	if err != nil {
		return
	}
	// A checkpoint is auxiliary to an already committed command. Never block
	// live gameplay on disk I/O, and never allocate unbounded workers under
	// high room concurrency. If all slots are busy, the next checkpoint
	// interval can retry; full genesis replay remains the fallback.
	select {
	case c.checkpointSlots <- struct{}{}:
		go func() {
			defer func() { <-c.checkpointSlots }()
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			_ = c.journal.SaveCheckpoint(ctx, id, epoch, outcome.Version, payload)
		}()
	default:
	}
}

func checkpointViewDigests(game *livehand.Game) ([32]byte, [3][32]byte, error) {
	var pub [32]byte
	var priv [3][32]byte
	if game == nil {
		return pub, priv, gamecore.ErrVerificationFailed
	}
	view, err := game.View(gamecore.ViewRequest{PublicOnly: true})
	if err != nil {
		return pub, priv, err
	}
	pub = sha256.Sum256(view.Payload)
	for seat := uint8(1); seat <= 3; seat++ {
		secret, err := game.View(gamecore.ViewRequest{ViewerPosition: seat})
		if err != nil {
			return pub, priv, err
		}
		priv[seat-1] = sha256.Sum256(secret.Payload)
	}
	return pub, priv, nil
}

// restoreRecoveryCheckpoint checks the independent log identities, hashes
// and authenticated checkpoint content before exposing any live state.
// Returned receipts are private to recovery until the entire hand validates.
func restoreRecoveryCheckpoint(
	game *livehand.Game, checkpoint mysqlstore.LiveCheckpoint, committed []mysqlstore.LiveAction,
) (map[string]liveReceipt, error) {
	if checkpoint.Version < 2 || checkpoint.Version-1 > uint64(len(committed)) {
		return nil, gamecore.ErrVerificationFailed
	}
	if len(checkpoint.Payload) == 0 || len(checkpoint.Payload) > 512<<10 {
		return nil, gamecore.ErrVerificationFailed
	}
	var cp recoveryCheckpoint
	decoder := json.NewDecoder(bytes.NewReader(checkpoint.Payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&cp); err != nil {
		return nil, fmt.Errorf("%w: decode protected checkpoint: %v", gamecore.ErrVerificationFailed, err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return nil, gamecore.ErrVerificationFailed
	}
	if cp.Version != liveRecoveryCheckpointV1 ||
		cp.HandID != domain.HandID(game.InstanceID()) ||
		cp.StateVersion != checkpoint.Version ||
		len(cp.Receipts) != int(checkpoint.Version-1) ||
		len(cp.Receipts) > maxLiveCheckpointActions {
		return nil, gamecore.ErrVerificationFailed
	}
	receipts := make(map[string]liveReceipt, len(cp.Receipts))
	for index, record := range cp.Receipts {
		log := committed[index]
		if record.CommandID == "" || record.CommandID != log.CommandID ||
			record.Actor != log.Actor || record.Command.ActorPosition != log.ActorPosition ||
			record.Command.ExpectedVersion != log.ExpectedVersion ||
			!bytes.Equal(record.Command.Payload, log.Payload) ||
			record.Outcome.Version != log.Version {
			return nil, gamecore.ErrVerificationFailed
		}
		if _, duplicate := receipts[record.CommandID]; duplicate {
			return nil, gamecore.ErrVerificationFailed
		}
		replayed, err := game.Apply(record.Command)
		if err != nil || replayed.Version != log.Version {
			return nil, gamecore.ErrVerificationFailed
		}
		replayJSON, err := json.Marshal(replayed)
		if err != nil || sha256.Sum256(replayJSON) != log.ResultDigest {
			return nil, gamecore.ErrVerificationFailed
		}
		checkpointJSON, err := json.Marshal(record.Outcome)
		if err != nil || !bytes.Equal(checkpointJSON, replayJSON) {
			return nil, gamecore.ErrVerificationFailed
		}
		receipts[record.CommandID] = liveReceipt{
			actor: record.Actor, command: record.Command, outcome: cloneLiveOutcome(replayed),
		}
	}
	pub, priv, err := checkpointViewDigests(game)
	if err != nil || pub != cp.PublicDigest || priv != cp.PrivateDigests {
		return nil, gamecore.ErrVerificationFailed
	}
	return receipts, nil
}
