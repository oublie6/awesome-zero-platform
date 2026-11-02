package mysqlstore

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"sort"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
)

const maxProtectionRoots = 8

func checkedRootIDs(active string, roots map[string][]byte) ([]string, error) {
	if active == "" || len(active) > 128 || len(roots) == 0 || len(roots) > maxProtectionRoots {
		return nil, fmt.Errorf("invalid protection keyring")
	}
	ids := make([]string, 0, len(roots))
	for id, key := range roots {
		if id == "" || len(id) > 128 || len(key) != 32 {
			return nil, fmt.Errorf("invalid protection key")
		}
		ids = append(ids, id)
	}
	if _, ok := roots[active]; !ok {
		return nil, fmt.Errorf("missing active protection key")
	}
	sort.Strings(ids)
	return ids, nil
}

func deriveCheckpointAEAD(root []byte) (cipher.AEAD, error) {
	if len(root) != 32 {
		return nil, fmt.Errorf("invalid checkpoint key length")
	}
	h := sha256.New()
	_, _ = h.Write([]byte(liveCheckpointDomain))
	_, _ = h.Write(root)
	key := h.Sum(nil)
	defer clear(key)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

// ConfigureCheckpointKeyring must be called during initialization only.
// Active keys write; up to seven retained historical keys only decrypt.
func (j *LiveJournal) ConfigureCheckpointKeyring(active string, roots map[string][]byte) error {
	ids, err := checkedRootIDs(active, roots)
	if err != nil {
		return err
	}
	if err := j.ConfigureCheckpoints(roots[active]); err != nil {
		return err
	}
	readers := make([]cipher.AEAD, 0, len(ids)-1)
	for _, id := range ids {
		if id == active {
			continue
		}
		aead, err := deriveCheckpointAEAD(roots[id])
		if err != nil {
			return err
		}
		readers = append(readers, aead)
	}
	j.checkpointReaders = readers
	return nil
}

func (j *LiveJournal) openCheckpoint(nonce, ciphertext, aad []byte) ([]byte, error) {
	if j == nil || j.checkpointAEAD == nil {
		return nil, ErrLiveReplayIntegrity
	}
	if plain, err := j.checkpointAEAD.Open(nil, nonce, ciphertext, aad); err == nil {
		return plain, nil
	}
	for _, reader := range j.checkpointReaders {
		if len(nonce) == reader.NonceSize() {
			if plain, err := reader.Open(nil, nonce, ciphertext, aad); err == nil {
				return plain, nil
			}
		}
	}
	return nil, ErrLiveReplayIntegrity
}

// ReencryptCheckpoint rewrites authenticated checkpoint ciphertext under the
// active key. A hand's owner epoch and committed watermark are locked/fenced.
// It does not change the journal, accepted requests or game state.
func (j *LiveJournal) ReencryptCheckpoint(ctx context.Context, id domain.HandID, epoch uint64) error {
	if j == nil || j.db == nil || j.checkpointAEAD == nil || id == "" || len(id) > 128 || epoch == 0 {
		return fmt.Errorf("invalid checkpoint rotation request")
	}
	tx, err := j.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var owner, committed uint64
	if err := tx.QueryRowContext(ctx, `SELECT owner_epoch,committed_version FROM doudizhu_live_writers
WHERE hand_id=? FOR UPDATE`, id).Scan(&owner, &committed); err != nil {
		return err
	}
	if epoch != owner {
		return ErrLiveWriterFenced
	}
	var version uint64
	var nonce, ciphertext, digest []byte
	err = tx.QueryRowContext(ctx, `SELECT checkpoint_version,nonce,ciphertext,state_digest
FROM doudizhu_live_checkpoints WHERE hand_id=? FOR UPDATE`, id).Scan(&version, &nonce, &ciphertext, &digest)
	if errors.Is(err, sql.ErrNoRows) {
		return tx.Commit()
	}
	if err != nil {
		return err
	}
	if version < 2 || version > committed || version > maxLiveActionsPerHand+1 ||
		len(nonce) != j.checkpointAEAD.NonceSize() ||
		len(ciphertext) <= j.checkpointAEAD.Overhead() ||
		len(ciphertext) > maxLiveCheckpointSize+j.checkpointAEAD.Overhead() ||
		len(digest) != sha256.Size {
		return ErrLiveReplayIntegrity
	}
	plain, err := j.openCheckpoint(nonce, ciphertext, checkpointAAD(id, version))
	if err != nil {
		return err
	}
	defer clear(plain)
	calculated := sha256.Sum256(plain)
	if len(plain) == 0 || len(plain) > maxLiveCheckpointSize || !bytes.Equal(calculated[:], digest) {
		return ErrLiveReplayIntegrity
	}
	fresh := make([]byte, j.checkpointAEAD.NonceSize())
	if _, err := io.ReadFull(rand.Reader, fresh); err != nil {
		return err
	}
	sealed := j.checkpointAEAD.Seal(nil, fresh, plain, checkpointAAD(id, version))
	if _, err := tx.ExecContext(ctx, `UPDATE doudizhu_live_checkpoints
SET nonce=?,ciphertext=?,created_at=UTC_TIMESTAMP(6) WHERE hand_id=? AND checkpoint_version=?`,
		fresh, sealed, id, version); err != nil {
		return err
	}
	return tx.Commit()
}

// NewProtectedSeedStoreKeyring keeps old keys read-only until each affected
// committed seed has been authenticated and re-encrypted under the active key.
func NewProtectedSeedStoreKeyring(db *sql.DB, active string, roots map[string][]byte) (*ProtectedSeedStore, error) {
	ids, err := checkedRootIDs(active, roots)
	if err != nil {
		return nil, err
	}
	store, err := NewProtectedSeedStore(db, roots[active])
	if err != nil {
		return nil, err
	}
	for _, id := range ids {
		if id == active {
			continue
		}
		old, err := NewProtectedSeedStore(db, roots[id])
		if err != nil {
			return nil, err
		}
		store.readers = append(store.readers, old.aead)
	}
	return store, nil
}

func (s *ProtectedSeedStore) openSeed(nonce, ciphertext, aad []byte) ([]byte, error) {
	if s == nil || s.aead == nil {
		return nil, fmt.Errorf("seed key unavailable")
	}
	if plain, err := s.aead.Open(nil, nonce, ciphertext, aad); err == nil {
		return plain, nil
	}
	for _, reader := range s.readers {
		if len(nonce) == reader.NonceSize() {
			if plain, err := reader.Open(nil, nonce, ciphertext, aad); err == nil {
				return plain, nil
			}
		}
	}
	return nil, fmt.Errorf("protected seed authentication failed")
}

// ReencryptSeed is an explicit, row-locked maintenance operation. It never
// replaces the player's previously committed fairness seed with new entropy.
func (s *ProtectedSeedStore) ReencryptSeed(ctx context.Context, id domain.HandID, db *sql.DB) error {
	if s == nil || s.aead == nil || db == nil || id == "" || len(id) > 128 {
		return fmt.Errorf("invalid seed rotation request")
	}
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var nonce, ciphertext []byte
	if err := tx.QueryRowContext(ctx, `SELECT nonce,ciphertext FROM doudizhu_protected_server_seeds
WHERE hand_id=? FOR UPDATE`, id).Scan(&nonce, &ciphertext); err != nil {
		return err
	}
	if len(nonce) != s.aead.NonceSize() || len(ciphertext) != 32+s.aead.Overhead() {
		return fmt.Errorf("invalid protected seed encoding")
	}
	plain, err := s.openSeed(nonce, ciphertext, []byte(serverSeedDomain+string(id)))
	if err != nil {
		return err
	}
	defer clear(plain)
	if len(plain) != 32 {
		return fmt.Errorf("invalid protected seed plaintext")
	}
	fresh := make([]byte, s.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, fresh); err != nil {
		return err
	}
	sealed := s.aead.Seal(nil, fresh, plain, []byte(serverSeedDomain+string(id)))
	if _, err := tx.ExecContext(ctx, `UPDATE doudizhu_protected_server_seeds SET nonce=?,ciphertext=?
WHERE hand_id=?`, fresh, sealed, id); err != nil {
		return err
	}
	return tx.Commit()
}
