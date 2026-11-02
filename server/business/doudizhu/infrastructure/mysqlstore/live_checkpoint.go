package mysqlstore

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
)

const (
	liveCheckpointDomain  = "doudizhu/live-checkpoint/aes256gcm/v1:"
	maxLiveCheckpointSize = 512 << 10
)

type LiveCheckpoint struct {
	Version uint64
	Payload []byte
}

// ConfigureCheckpoints derives an independent AES-GCM key from the existing
// production protection root. No plaintext private hand state is stored in
// MySQL or emitted into telemetry.
func (j *LiveJournal) ConfigureCheckpoints(rootKey []byte) error {
	if j == nil || j.db == nil || len(rootKey) != 32 {
		return fmt.Errorf("durable checkpoint requires mysql and 256-bit key")
	}
	h := sha256.New()
	h.Write([]byte(liveCheckpointDomain))
	h.Write(rootKey)
	key := h.Sum(nil)
	defer clear(key)
	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}
	j.checkpointAEAD = aead
	j.checkpointReaders = nil
	return nil
}

func checkpointAAD(id domain.HandID, version uint64) []byte {
	// Include both stable hand identity and committed version in AEAD
	// associated data; older valid checkpoints cannot be relabelled.
	result := make([]byte, 0, len(liveCheckpointDomain)+128+8)
	result = append(result, liveCheckpointDomain...)
	result = append(result, id...)
	var v [8]byte
	binary.BigEndian.PutUint64(v[:], version)
	result = append(result, v[:]...)
	return result
}

// SaveCheckpoint persists a complete replayable, authenticated snapshot at a
// committed version. It cannot move a checkpoint backwards or let a stale
// owner publish one after a fenced takeover. It never changes the journal's
// authoritative committed watermark.
func (j *LiveJournal) SaveCheckpoint(ctx context.Context, id domain.HandID, epoch, version uint64, payload []byte) error {
	if j == nil || j.db == nil || j.checkpointAEAD == nil || id == "" || len(id) > 128 ||
		epoch == 0 || version <= 1 || version > maxLiveActionsPerHand+1 ||
		len(payload) == 0 || len(payload) > maxLiveCheckpointSize {
		return fmt.Errorf("invalid protected live checkpoint")
	}
	nonce := make([]byte, j.checkpointAEAD.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return fmt.Errorf("generate checkpoint nonce: %w", err)
	}
	ciphertext := j.checkpointAEAD.Seal(nil, nonce, payload, checkpointAAD(id, version))
	digest := sha256.Sum256(payload)
	tx, err := j.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var currentEpoch, committed uint64
	if err := tx.QueryRowContext(ctx, `SELECT owner_epoch,committed_version FROM doudizhu_live_writers
WHERE hand_id=? FOR UPDATE`, id).Scan(&currentEpoch, &committed); err != nil {
		return err
	}
	if epoch != currentEpoch {
		return ErrLiveWriterFenced
	}
	if version > committed {
		return ErrLiveWatermarkConflict
	}
	var checkpointVersion uint64
	err = tx.QueryRowContext(ctx, `SELECT checkpoint_version FROM doudizhu_live_checkpoints
WHERE hand_id=? FOR UPDATE`, id).Scan(&checkpointVersion)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if err == nil && checkpointVersion >= version {
		return tx.Commit()
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO doudizhu_live_checkpoints
(hand_id,checkpoint_version,nonce,ciphertext,state_digest,created_at)
VALUES(?,?,?,?,?,UTC_TIMESTAMP(6))
ON DUPLICATE KEY UPDATE checkpoint_version=VALUES(checkpoint_version),nonce=VALUES(nonce),
ciphertext=VALUES(ciphertext),state_digest=VALUES(state_digest),created_at=UTC_TIMESTAMP(6)`,
		id, version, nonce, ciphertext, digest[:])
	if err != nil {
		return err
	}
	return tx.Commit()
}

// LoadCheckpoint authenticates and bounds the snapshot before decoding any
// application state. Missing checkpoints are allowed: genesis+journal replay
// remains the authoritative recovery fallback. Malformed records are fatal.
func (j *LiveJournal) LoadCheckpoint(ctx context.Context, id domain.HandID) (LiveCheckpoint, error) {
	if j == nil || j.db == nil || j.checkpointAEAD == nil || id == "" || len(id) > 128 {
		return LiveCheckpoint{}, fmt.Errorf("protected live checkpoint unavailable")
	}
	var version uint64
	var nonce, ciphertext, digest []byte
	err := j.db.QueryRowContext(ctx, `SELECT checkpoint_version,nonce,ciphertext,state_digest
FROM doudizhu_live_checkpoints WHERE hand_id=?`, id).Scan(&version, &nonce, &ciphertext, &digest)
	if errors.Is(err, sql.ErrNoRows) {
		return LiveCheckpoint{}, nil
	}
	if err != nil {
		return LiveCheckpoint{}, err
	}
	if version <= 1 || version > maxLiveActionsPerHand+1 ||
		len(nonce) != j.checkpointAEAD.NonceSize() ||
		len(ciphertext) <= j.checkpointAEAD.Overhead() ||
		len(ciphertext) > maxLiveCheckpointSize+j.checkpointAEAD.Overhead() ||
		len(digest) != sha256.Size {
		return LiveCheckpoint{}, ErrLiveReplayIntegrity
	}
	plaintext, err := j.openCheckpoint(nonce, ciphertext, checkpointAAD(id, version))
	if err != nil {
		return LiveCheckpoint{}, ErrLiveReplayIntegrity
	}
	checksum := sha256.Sum256(plaintext)
	if !equalDigest(checksum[:], digest) || len(plaintext) == 0 || len(plaintext) > maxLiveCheckpointSize {
		clear(plaintext)
		return LiveCheckpoint{}, ErrLiveReplayIntegrity
	}
	return LiveCheckpoint{Version: version, Payload: plaintext}, nil
}

func equalDigest(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	var diff byte
	for i := range a {
		diff |= a[i] ^ b[i]
	}
	return diff == 0
}
