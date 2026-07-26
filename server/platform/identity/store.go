package identity

import (
	"context"
	"database/sql"
	"errors"
	"time"

	mysql "github.com/go-sql-driver/mysql"
	foundationcache "github.com/oublie6/awesome-zero-platform/server/foundation/cache"
)

type accountStore interface {
	InsertAccountTx(context.Context, *sql.Tx, accountRecord) error
	GetAccountByID(context.Context, string) (Account, error)
	GetAccountByIDTx(context.Context, *sql.Tx, string) (Account, error)
	GetAccountByUsername(context.Context, string) (Account, error)
	GetAccountByEmail(context.Context, string) (Account, error)
	GetAccountByPhone(context.Context, string) (Account, error)
	UpdateAccountProfile(context.Context, accountRecord) error
	UpdateAccountStatus(context.Context, string, AccountStatus, time.Time) error
}

type credentialStore interface {
	InsertPasswordCredentialTx(context.Context, *sql.Tx, storedCredential) error
	GetPasswordCredentialByAccountID(context.Context, string) (storedCredential, error)
	GetPasswordCredentialByAccountIDTx(context.Context, *sql.Tx, string) (storedCredential, error)
	UpdatePasswordCredentialTx(context.Context, *sql.Tx, storedCredential) error
}

type accountFreshReader interface {
	GetAccountByIDFresh(context.Context, string) (Account, error)
	GetAccountByUsernameFresh(context.Context, string) (Account, error)
	GetAccountByEmailFresh(context.Context, string) (Account, error)
	GetAccountByPhoneFresh(context.Context, string) (Account, error)
}

type accountCacheInvalidator interface {
	InvalidateAccountCache(context.Context, accountRecord) error
}

type queryRower interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type execer interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

type accountRecord struct {
	ID          string
	Username    string
	UsernameKey string
	Email       string
	EmailKey    string
	Phone       string
	PhoneKey    string
	DisplayName string
	Status      AccountStatus
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type storedCredential struct {
	AccountID         string
	Hash              string
	PasswordChangedAt time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type MySQLStore struct {
	db    *sql.DB
	cache *foundationcache.ModelCache
}

func NewMySQLStore(db *sql.DB) *MySQLStore {
	return &MySQLStore{db: db}
}

func NewCachedMySQLStore(db *sql.DB, modelCache *foundationcache.ModelCache) *MySQLStore {
	return &MySQLStore{db: db, cache: modelCache}
}

func (s *MySQLStore) InsertAccountTx(ctx context.Context, tx *sql.Tx, record accountRecord) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO identity_accounts (
			account_id,
			username,
			username_key,
			email,
			email_key,
			phone,
			phone_key,
			display_name,
			status,
			created_at,
			updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		record.ID,
		nullString(record.Username),
		nullString(record.UsernameKey),
		nullString(record.Email),
		nullString(record.EmailKey),
		nullString(record.Phone),
		nullString(record.PhoneKey),
		record.DisplayName,
		record.Status,
		record.CreatedAt.UTC(),
		record.UpdatedAt.UTC(),
	)
	if err != nil {
		return translateMySQLError(err)
	}

	return nil
}

func (s *MySQLStore) GetAccountByID(ctx context.Context, accountID string) (Account, error) {
	var account Account
	err := s.takeAccount(ctx, &account, s.accountIDKey(accountID), func() (Account, error) {
		return s.GetAccountByIDFresh(ctx, accountID)
	})
	return account, err
}

func (s *MySQLStore) GetAccountByIDFresh(ctx context.Context, accountID string) (Account, error) {
	return s.getAccount(ctx, s.db, `
		SELECT account_id, username, email, phone, display_name, status, created_at, updated_at
		FROM identity_accounts
		WHERE account_id = ?
	`, accountID)
}

func (s *MySQLStore) GetAccountByIDTx(ctx context.Context, tx *sql.Tx, accountID string) (Account, error) {
	return s.getAccount(ctx, tx, `
		SELECT account_id, username, email, phone, display_name, status, created_at, updated_at
		FROM identity_accounts
		WHERE account_id = ?
		FOR UPDATE
	`, accountID)
}

func (s *MySQLStore) GetAccountByUsername(ctx context.Context, usernameKey string) (Account, error) {
	var account Account
	err := s.takeAccount(ctx, &account, s.usernameKey(usernameKey), func() (Account, error) {
		return s.GetAccountByUsernameFresh(ctx, usernameKey)
	})
	return account, err
}

func (s *MySQLStore) GetAccountByUsernameFresh(ctx context.Context, usernameKey string) (Account, error) {
	return s.getAccount(ctx, s.db, `
		SELECT account_id, username, email, phone, display_name, status, created_at, updated_at
		FROM identity_accounts
		WHERE username_key = ?
	`, usernameKey)
}

func (s *MySQLStore) GetAccountByEmail(ctx context.Context, emailKey string) (Account, error) {
	var account Account
	err := s.takeAccount(ctx, &account, s.emailKey(emailKey), func() (Account, error) {
		return s.GetAccountByEmailFresh(ctx, emailKey)
	})
	return account, err
}

func (s *MySQLStore) GetAccountByEmailFresh(ctx context.Context, emailKey string) (Account, error) {
	return s.getAccount(ctx, s.db, `
		SELECT account_id, username, email, phone, display_name, status, created_at, updated_at
		FROM identity_accounts
		WHERE email_key = ?
	`, emailKey)
}

func (s *MySQLStore) GetAccountByPhone(ctx context.Context, phoneKey string) (Account, error) {
	var account Account
	err := s.takeAccount(ctx, &account, s.phoneKey(phoneKey), func() (Account, error) {
		return s.GetAccountByPhoneFresh(ctx, phoneKey)
	})
	return account, err
}

func (s *MySQLStore) GetAccountByPhoneFresh(ctx context.Context, phoneKey string) (Account, error) {
	return s.getAccount(ctx, s.db, `
		SELECT account_id, username, email, phone, display_name, status, created_at, updated_at
		FROM identity_accounts
		WHERE phone_key = ?
	`, phoneKey)
}

func (s *MySQLStore) UpdateAccountProfile(ctx context.Context, record accountRecord) error {
	previous, err := s.GetAccountByIDFresh(ctx, record.ID)
	if err != nil {
		return err
	}
	result, err := s.db.ExecContext(ctx, `
		UPDATE identity_accounts
		SET username = ?,
		    username_key = ?,
		    email = ?,
		    email_key = ?,
		    phone = ?,
		    phone_key = ?,
		    display_name = ?,
		    updated_at = ?
		WHERE account_id = ?
	`,
		nullString(record.Username),
		nullString(record.UsernameKey),
		nullString(record.Email),
		nullString(record.EmailKey),
		nullString(record.Phone),
		nullString(record.PhoneKey),
		record.DisplayName,
		record.UpdatedAt.UTC(),
		record.ID,
	)
	if err != nil {
		return translateMySQLError(err)
	}
	if err := requireRowsAffected(result); err != nil {
		return err
	}

	return s.invalidateAccounts(ctx, recordFromAccount(previous), record)
}

func (s *MySQLStore) UpdateAccountStatus(ctx context.Context, accountID string, status AccountStatus, updatedAt time.Time) error {
	previous, err := s.GetAccountByIDFresh(ctx, accountID)
	if err != nil {
		return err
	}
	result, err := s.db.ExecContext(ctx, `
		UPDATE identity_accounts
		SET status = ?, updated_at = ?
		WHERE account_id = ?
	`, status, updatedAt.UTC(), accountID)
	if err != nil {
		return translateMySQLError(err)
	}
	if err := requireRowsAffected(result); err != nil {
		return err
	}

	next := recordFromAccount(previous)
	next.Status = status
	next.UpdatedAt = updatedAt.UTC()
	return s.invalidateAccounts(ctx, recordFromAccount(previous), next)
}

func (s *MySQLStore) InvalidateAccountCache(ctx context.Context, record accountRecord) error {
	return s.invalidateAccounts(ctx, record)
}

func (s *MySQLStore) InsertPasswordCredentialTx(ctx context.Context, tx *sql.Tx, credential storedCredential) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO identity_password_credentials (
			account_id,
			password_hash,
			password_changed_at,
			created_at,
			updated_at
		) VALUES (?, ?, ?, ?, ?)
	`, credential.AccountID, credential.Hash, credential.PasswordChangedAt.UTC(), credential.CreatedAt.UTC(), credential.UpdatedAt.UTC())
	if err != nil {
		return translateMySQLError(err)
	}

	return nil
}

func (s *MySQLStore) GetPasswordCredentialByAccountID(ctx context.Context, accountID string) (storedCredential, error) {
	return s.getPasswordCredential(ctx, s.db, `
		SELECT account_id, password_hash, password_changed_at, created_at, updated_at
		FROM identity_password_credentials
		WHERE account_id = ?
	`, accountID)
}

func (s *MySQLStore) GetPasswordCredentialByAccountIDTx(ctx context.Context, tx *sql.Tx, accountID string) (storedCredential, error) {
	return s.getPasswordCredential(ctx, tx, `
		SELECT account_id, password_hash, password_changed_at, created_at, updated_at
		FROM identity_password_credentials
		WHERE account_id = ?
		FOR UPDATE
	`, accountID)
}

func (s *MySQLStore) UpdatePasswordCredentialTx(ctx context.Context, tx *sql.Tx, credential storedCredential) error {
	result, err := tx.ExecContext(ctx, `
		UPDATE identity_password_credentials
		SET password_hash = ?,
		    password_changed_at = ?,
		    updated_at = ?
		WHERE account_id = ?
	`, credential.Hash, credential.PasswordChangedAt.UTC(), credential.UpdatedAt.UTC(), credential.AccountID)
	if err != nil {
		return translateMySQLError(err)
	}
	if err := requireRowsAffected(result); err != nil {
		return wrapIdentityError(ErrPersistence, err)
	}

	return nil
}

func (s *MySQLStore) takeAccount(ctx context.Context, target *Account, key string, loader func() (Account, error)) error {
	if s.cache == nil {
		account, err := loader()
		if err == nil {
			*target = account
		}
		return err
	}
	return s.cache.TakeCtx(ctx, target, key, func(value any) error {
		account, err := loader()
		if err != nil {
			return err
		}
		*value.(*Account) = account
		return nil
	})
}

func (s *MySQLStore) invalidateAccounts(ctx context.Context, records ...accountRecord) error {
	if s.cache == nil {
		return nil
	}
	keys := make([]string, 0, len(records)*4)
	for _, record := range records {
		keys = append(keys,
			s.accountIDKey(record.ID),
			s.usernameKey(record.UsernameKey),
			s.emailKey(record.EmailKey),
			s.phoneKey(record.PhoneKey),
		)
	}
	return s.cache.DelCtx(ctx, keys...)
}

func (s *MySQLStore) accountIDKey(value string) string {
	if s.cache == nil {
		return ""
	}
	return s.cache.Key("id", value)
}

func (s *MySQLStore) usernameKey(value string) string {
	if s.cache == nil || value == "" {
		return ""
	}
	return s.cache.Key("username", value)
}

func (s *MySQLStore) emailKey(value string) string {
	if s.cache == nil || value == "" {
		return ""
	}
	return s.cache.Key("email", value)
}

func (s *MySQLStore) phoneKey(value string) string {
	if s.cache == nil || value == "" {
		return ""
	}
	return s.cache.Key("phone", value)
}

func (s *MySQLStore) getAccount(ctx context.Context, rower queryRower, query string, args ...any) (Account, error) {
	var (
		account                Account
		username, email, phone sql.NullString
	)

	err := rower.QueryRowContext(ctx, query, args...).Scan(
		&account.ID,
		&username,
		&email,
		&phone,
		&account.DisplayName,
		&account.Status,
		&account.CreatedAt,
		&account.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Account{}, ErrAccountNotFound
		}
		return Account{}, wrapIdentityError(ErrPersistence, err)
	}

	account.Username = nullStringValue(username)
	account.Email = nullStringValue(email)
	account.Phone = nullStringValue(phone)

	return account, nil
}

func (s *MySQLStore) getPasswordCredential(ctx context.Context, rower queryRower, query string, args ...any) (storedCredential, error) {
	var credential storedCredential

	err := rower.QueryRowContext(ctx, query, args...).Scan(
		&credential.AccountID,
		&credential.Hash,
		&credential.PasswordChangedAt,
		&credential.CreatedAt,
		&credential.UpdatedAt,
	)
	if err != nil {
		return storedCredential{}, wrapIdentityError(ErrPersistence, err)
	}

	return credential, nil
}

func requireRowsAffected(result sql.Result) error {
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return wrapIdentityError(ErrPersistence, err)
	}
	if rowsAffected != 1 {
		return ErrAccountNotFound
	}

	return nil
}

func translateMySQLError(err error) error {
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) {
		if mysqlErr.Number == 1062 {
			return wrapIdentityError(ErrIdentityConflict, err)
		}
	}

	return wrapIdentityError(ErrPersistence, err)
}

func recordFromAccount(account Account) accountRecord {
	record := accountRecord{
		ID:          account.ID,
		Username:    account.Username,
		Email:       account.Email,
		Phone:       account.Phone,
		DisplayName: account.DisplayName,
		Status:      account.Status,
		CreatedAt:   account.CreatedAt,
		UpdatedAt:   account.UpdatedAt,
	}
	if account.Username != "" {
		_, record.UsernameKey, _ = normalizeUsername(account.Username)
	}
	if account.Email != "" {
		_, record.EmailKey, _ = normalizeEmail(account.Email)
	}
	if account.Phone != "" {
		_, record.PhoneKey, _ = normalizePhone(account.Phone)
	}
	return record
}

func nullString(input string) any {
	if input == "" {
		return nil
	}

	return input
}

func nullStringValue(input sql.NullString) string {
	if !input.Valid {
		return ""
	}

	return input.String
}
