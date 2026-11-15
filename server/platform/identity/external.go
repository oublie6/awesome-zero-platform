package identity

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrExternalUnbound  = errors.New("external identity is not linked")
	ErrExternalConflict = errors.New("external identity binding conflict")
	ErrPendingInvalid   = errors.New("external identity proof invalid or expired")
)

// ExternalBinding owns the verified third-party subject mapping. A provider
// subject never authorizes an AccountID supplied by an unauthenticated client.
type ExternalBinding struct {
	db       *sql.DB
	identity *Service
	key      []byte
}

func NewExternalBinding(db *sql.DB, service *Service, subjectHMACKey []byte) (*ExternalBinding, error) {
	if db == nil || service == nil || len(subjectHMACKey) < 32 {
		return nil, fmt.Errorf("verified binding needs db, identity and 256-bit key")
	}
	return &ExternalBinding{db: db, identity: service, key: append([]byte(nil), subjectHMACKey...)}, nil
}
func (b *ExternalBinding) digest(provider, subject string) ([]byte, error) {
	if b == nil || provider != "wechat" || len(subject) < 4 || len(subject) > 256 || strings.TrimSpace(subject) != subject {
		return nil, ErrPendingInvalid
	}
	mac := hmac.New(sha256.New, b.key)
	mac.Write([]byte(provider))
	mac.Write([]byte{0})
	mac.Write([]byte(subject))
	return mac.Sum(nil), nil
}
func (b *ExternalBinding) ResolveVerified(ctx context.Context, provider, subject string) (Account, error) {
	digest, err := b.digest(provider, subject)
	if err != nil {
		return Account{}, err
	}
	var id string
	err = b.db.QueryRowContext(ctx, "SELECT account_id FROM identity_external_subjects WHERE provider=? AND subject_hmac=?", provider, digest).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return Account{}, ErrExternalUnbound
	}
	if err != nil {
		return Account{}, err
	}
	account, err := b.identity.GetAccountByIDFresh(ctx, id)
	if err != nil || account.Status != StatusActive {
		return Account{}, ErrInvalidAccountState
	}
	return account, nil
}

// BeginVerified creates a short-lived one-purpose proof only after the trusted
// provider adapter has verified the code. No auto-created identity or session.
func (b *ExternalBinding) BeginVerified(ctx context.Context, provider, subject string) (string, error) {
	digest, err := b.digest(provider, subject)
	if err != nil {
		return "", err
	}
	token := make([]byte, 32)
	if _, err = rand.Read(token); err != nil {
		return "", err
	}
	sum := sha256.Sum256(token)
	now := time.Now().UTC()
	_, err = b.db.ExecContext(ctx, "INSERT INTO identity_external_pending(token_digest,provider,subject_hmac,expires_at,created_at) VALUES(?,?,?,?,?)",
		sum[:], provider, digest, now.Add(5*time.Minute), now)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(token), nil
}
func pendingDigest(token string) ([32]byte, error) {
	decoded, err := base64.RawURLEncoding.Strict().DecodeString(token)
	if err != nil || len(decoded) != 32 || base64.RawURLEncoding.EncodeToString(decoded) != token {
		return [32]byte{}, ErrPendingInvalid
	}
	return sha256.Sum256(decoded), nil
}
func (b *ExternalBinding) pendingTx(ctx context.Context, tx *sql.Tx, token string) (string, []byte, error) {
	hash, err := pendingDigest(token)
	if err != nil {
		return "", nil, err
	}
	var provider string
	var subject []byte
	var expiry time.Time
	err = tx.QueryRowContext(ctx, "SELECT provider,subject_hmac,expires_at FROM identity_external_pending WHERE token_digest=? FOR UPDATE", hash[:]).Scan(&provider, &subject, &expiry)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil, ErrPendingInvalid
	}
	if err != nil {
		return "", nil, err
	}
	if !time.Now().UTC().Before(expiry) {
		return "", nil, ErrPendingInvalid
	}
	return provider, subject, nil
}
func (b *ExternalBinding) claimTx(ctx context.Context, tx *sql.Tx, provider string, subject []byte, accountID string) error {
	result, err := tx.ExecContext(ctx, "INSERT INTO identity_external_subjects(provider,subject_hmac,account_id,created_at) VALUES(?,?,?,UTC_TIMESTAMP(6))",
		provider, subject, accountID)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrExternalConflict, err)
	}
	if rows, _ := result.RowsAffected(); rows != 1 {
		return ErrExternalConflict
	}
	return nil
}
func (b *ExternalBinding) discardTx(ctx context.Context, tx *sql.Tx, token string) error {
	hash, _ := pendingDigest(token)
	_, err := tx.ExecContext(ctx, "DELETE FROM identity_external_pending WHERE token_digest=?", hash[:])
	return err
}

// BindPending requires that HTTP already authenticated the target AccountID
// with its current password (or a recently reauthenticated session).
func (b *ExternalBinding) BindPending(ctx context.Context, token, accountID string) (err error) {
	account, err := b.identity.GetAccountByIDFresh(ctx, accountID)
	if err != nil || account.Status != StatusActive {
		return ErrInvalidAccountState
	}
	tx, err := b.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	provider, subject, err := b.pendingTx(ctx, tx, token)
	if err != nil {
		return err
	}
	if err = b.claimTx(ctx, tx, provider, subject, accountID); err != nil {
		return err
	}
	if err = b.discardTx(ctx, tx, token); err != nil {
		return err
	}
	return tx.Commit()
}

// CreateFromPending creates the real AccountID, password credential and
// provider mapping atomically, so a race cannot leave orphan accounts.
func (b *ExternalBinding) CreateFromPending(ctx context.Context, token string, input CreateAccountInput) (account Account, err error) {
	normalized, display, status, err := b.identity.validateCreateAccountInput(input)
	if err != nil {
		return Account{}, err
	}
	if normalized.UsernameKey == "" || normalized.EmailKey == "" {
		return Account{}, ErrInvalidCredentials
	}
	id, err := b.identity.generateID()
	if err != nil {
		return Account{}, err
	}
	if err = validateAccountID(id); err != nil {
		return Account{}, err
	}
	hash, err := b.identity.hasher.Hash(input.Password)
	if err != nil {
		return Account{}, err
	}
	now := b.identity.now().UTC()
	record := accountRecord{ID: id, Username: normalized.Username, UsernameKey: normalized.UsernameKey,
		Email: normalized.Email, EmailKey: normalized.EmailKey, Phone: normalized.Phone, PhoneKey: normalized.PhoneKey,
		DisplayName: display, Status: status, CreatedAt: now, UpdatedAt: now}
	credential := storedCredential{AccountID: id, Hash: hash, PasswordChangedAt: now, CreatedAt: now, UpdatedAt: now}
	tx, err := b.db.BeginTx(ctx, nil)
	if err != nil {
		return Account{}, err
	}
	defer tx.Rollback()
	provider, subject, err := b.pendingTx(ctx, tx, token)
	if err != nil {
		return Account{}, err
	}
	if err = b.identity.accounts.InsertAccountTx(ctx, tx, record); err != nil {
		return Account{}, err
	}
	if err = b.identity.credentials.InsertPasswordCredentialTx(ctx, tx, credential); err != nil {
		return Account{}, err
	}
	if err = b.claimTx(ctx, tx, provider, subject, id); err != nil {
		return Account{}, err
	}
	if err = b.discardTx(ctx, tx, token); err != nil {
		return Account{}, err
	}
	if err = tx.Commit(); err != nil {
		return Account{}, err
	}
	return accountFromRecord(record), nil
}

// Unlink affects only external mapping. Every account has an existing password
// credential; the HTTP boundary must require fresh password verification.
func (b *ExternalBinding) Unlink(ctx context.Context, provider, accountID string) error {
	if provider != "wechat" {
		return ErrPendingInvalid
	}
	account, err := b.identity.GetAccountByIDFresh(ctx, accountID)
	if err != nil || account.Status != StatusActive {
		return ErrInvalidAccountState
	}
	result, err := b.db.ExecContext(ctx, "DELETE FROM identity_external_subjects WHERE provider=? AND account_id=?", provider, accountID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows != 1 {
		return ErrExternalUnbound
	}
	return nil
}

// Relink verifies the provider proof and requires both source and target
// platform credentials at the HTTP layer. The target must be free of this provider.
func (b *ExternalBinding) Relink(ctx context.Context, token, sourceID, targetID string) error {
	if sourceID == targetID {
		return ErrExternalConflict
	}
	source, err := b.identity.GetAccountByIDFresh(ctx, sourceID)
	if err != nil || source.Status != StatusActive {
		return ErrInvalidAccountState
	}
	target, err := b.identity.GetAccountByIDFresh(ctx, targetID)
	if err != nil || target.Status != StatusActive {
		return ErrInvalidAccountState
	}
	tx, err := b.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	provider, subject, err := b.pendingTx(ctx, tx, token)
	if err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, "UPDATE identity_external_subjects SET account_id=? WHERE provider=? AND subject_hmac=? AND account_id=?", targetID, provider, subject, sourceID)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrExternalConflict, err)
	}
	rows, err := result.RowsAffected()
	if err != nil || rows != 1 {
		return ErrExternalConflict
	}
	if err = b.discardTx(ctx, tx, token); err != nil {
		return err
	}
	return tx.Commit()
}
