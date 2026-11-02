package mysqlstore

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"io"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/carddeck"
	foundationdb "github.com/oublie6/awesome-zero-platform/server/foundation/database"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

const serverSeedDomain = "doudizhu/server-seed/aes256gcm/v1:"

type ProtectedSeedStore struct {
	sql     sqlx.SqlConn
	aead    cipher.AEAD
	readers []cipher.AEAD
}

func NewProtectedSeedStore(db *sql.DB, rootKey []byte) (*ProtectedSeedStore, error) {
	if db == nil || len(rootKey) != 32 {
		return nil, fmt.Errorf("protected seed store requires database and 256-bit root key")
	}
	// Domain separation: the server-seed encryption key never equals the
	// configured contribution-encryption root key.
	derivation := append([]byte(serverSeedDomain), rootKey...)
	key := sha256.Sum256(derivation)
	clear(derivation)
	block, err := aes.NewCipher(key[:])
	clear(key[:])
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &ProtectedSeedStore{sql: foundationdb.WrapSqlConn(db), aead: aead}, nil
}
func (s *ProtectedSeedStore) Save(ctx context.Context, id domain.HandID, seed carddeck.Seed) error {
	if s == nil || s.aead == nil || id == "" || len(id) > 128 {
		return fmt.Errorf("invalid seed store or hand identity")
	}
	nonce := make([]byte, s.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return fmt.Errorf("seed nonce entropy: %w", err)
	}
	ciphertext := s.aead.Seal(nil, nonce, seed[:], []byte(serverSeedDomain+string(id)))
	_, err := s.sql.ExecCtx(ctx, `INSERT INTO doudizhu_protected_server_seeds(hand_id,nonce,ciphertext,created_at)
VALUES(?,?,?,UTC_TIMESTAMP(6))`, string(id), nonce, ciphertext)
	if err != nil {
		return fmt.Errorf("persist encrypted seed: %w", err)
	}
	return nil
}
func (s *ProtectedSeedStore) Load(ctx context.Context, id domain.HandID) (carddeck.Seed, error) {
	var seed carddeck.Seed
	if s == nil || s.aead == nil || id == "" || len(id) > 128 {
		return seed, fmt.Errorf("invalid seed query")
	}
	var row struct {
		Nonce      []byte `db:"nonce"`
		Ciphertext []byte `db:"ciphertext"`
	}
	if err := s.sql.QueryRowCtx(ctx, &row, `SELECT nonce,ciphertext FROM doudizhu_protected_server_seeds WHERE hand_id = ?`, string(id)); err != nil {
		return seed, fmt.Errorf("load encrypted server seed: %w", err)
	}
	if len(row.Nonce) != s.aead.NonceSize() {
		return seed, fmt.Errorf("invalid protected seed nonce")
	}
	plaintext, err := s.openSeed(row.Nonce, row.Ciphertext, []byte(serverSeedDomain+string(id)))
	if err != nil {
		return seed, fmt.Errorf("protected server seed authentication failed")
	}
	defer clear(plaintext)
	if len(plaintext) != len(seed) {
		return seed, fmt.Errorf("invalid protected server seed length")
	}
	copy(seed[:], plaintext)
	return seed, nil
}
func (s *ProtectedSeedStore) Delete(ctx context.Context, id domain.HandID) error {
	if s == nil || s.sql == nil || id == "" || len(id) > 128 {
		return fmt.Errorf("invalid protected seed delete")
	}
	_, err := s.sql.ExecCtx(ctx, `DELETE FROM doudizhu_protected_server_seeds WHERE hand_id = ?`, string(id))
	if err != nil {
		return fmt.Errorf("delete protected server seed: %w", err)
	}
	return nil
}
