package identity

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/oublie6/awesome-zero-platform/server/foundation/database"
)

type txRunner interface {
	WithinTransaction(context.Context, func(context.Context, *sql.Tx) error) error
}

type serviceDependencies struct {
	transactions txRunner
	accounts     accountStore
	credentials  credentialStore
	hasher       PasswordHasher
	generateID   func() (string, error)
	now          func() time.Time
}

type Service struct {
	transactions txRunner
	accounts     accountStore
	credentials  credentialStore
	hasher       PasswordHasher
	generateID   func() (string, error)
	now          func() time.Time
}

func NewService(mysql database.Handle) *Service {
	store := NewMySQLStore(mysql.DB())

	return newService(serviceDependencies{
		transactions: mysql,
		accounts:     store,
		credentials:  store,
		hasher:       NewArgon2idHasher(),
		generateID: func() (string, error) {
			id, err := uuid.NewV7()
			if err != nil {
				return "", err
			}

			return id.String(), nil
		},
		now: time.Now,
	})
}

func newService(deps serviceDependencies) *Service {
	if deps.hasher == nil {
		deps.hasher = NewArgon2idHasher()
	}
	if deps.generateID == nil {
		deps.generateID = func() (string, error) {
			id, err := uuid.NewV7()
			if err != nil {
				return "", err
			}

			return id.String(), nil
		}
	}
	if deps.now == nil {
		deps.now = time.Now
	}

	return &Service{
		transactions: deps.transactions,
		accounts:     deps.accounts,
		credentials:  deps.credentials,
		hasher:       deps.hasher,
		generateID:   deps.generateID,
		now:          deps.now,
	}
}

func (s *Service) CreateAccount(ctx context.Context, input CreateAccountInput) (Account, error) {
	normalized, displayName, status, err := s.validateCreateAccountInput(input)
	if err != nil {
		return Account{}, err
	}

	accountID, err := s.generateID()
	if err != nil {
		return Account{}, wrapIdentityError(ErrPersistence, err)
	}
	if err := validateAccountID(accountID); err != nil {
		return Account{}, wrapIdentityError(ErrPersistence, err)
	}

	passwordHash, err := s.hasher.Hash(input.Password)
	if err != nil {
		return Account{}, wrapIdentityError(ErrPersistence, err)
	}

	now := s.now().UTC()
	record := accountRecord{
		ID:          accountID,
		Username:    normalized.Username,
		UsernameKey: normalized.UsernameKey,
		Email:       normalized.Email,
		EmailKey:    normalized.EmailKey,
		Phone:       normalized.Phone,
		PhoneKey:    normalized.PhoneKey,
		DisplayName: displayName,
		Status:      status,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	credential := storedCredential{
		AccountID:         accountID,
		Hash:              passwordHash,
		PasswordChangedAt: now,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	if err := s.transactions.WithinTransaction(ctx, func(txCtx context.Context, tx *sql.Tx) error {
		if err := s.accounts.InsertAccountTx(txCtx, tx, record); err != nil {
			return err
		}
		if err := s.credentials.InsertPasswordCredentialTx(txCtx, tx, credential); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return Account{}, err
	}

	return accountFromRecord(record), nil
}

func (s *Service) GetAccountByID(ctx context.Context, accountID string) (Account, error) {
	normalizedID, err := normalizeAccountID(accountID)
	if err != nil {
		return Account{}, err
	}

	return s.accounts.GetAccountByID(ctx, normalizedID)
}

func (s *Service) FindAccountByUsername(ctx context.Context, username string) (Account, error) {
	_, usernameKey, err := normalizeUsername(username)
	if err != nil {
		return Account{}, err
	}

	return s.accounts.GetAccountByUsername(ctx, usernameKey)
}

func (s *Service) FindAccountByEmail(ctx context.Context, email string) (Account, error) {
	_, emailKey, err := normalizeEmail(email)
	if err != nil {
		return Account{}, err
	}

	return s.accounts.GetAccountByEmail(ctx, emailKey)
}

func (s *Service) FindAccountByPhone(ctx context.Context, phone string) (Account, error) {
	_, phoneKey, err := normalizePhone(phone)
	if err != nil {
		return Account{}, err
	}

	return s.accounts.GetAccountByPhone(ctx, phoneKey)
}

func (s *Service) UpdateProfile(ctx context.Context, accountID string, input UpdateProfileInput) (Account, error) {
	normalizedID, err := normalizeAccountID(accountID)
	if err != nil {
		return Account{}, err
	}

	account, err := s.GetAccountByID(ctx, normalizedID)
	if err != nil {
		return Account{}, err
	}

	normalized, displayName, err := input.apply(account)
	if err != nil {
		return Account{}, err
	}

	account.Username = normalized.Username
	account.Email = normalized.Email
	account.Phone = normalized.Phone
	account.DisplayName = displayName
	account.UpdatedAt = s.now().UTC()

	if err := s.accounts.UpdateAccountProfile(ctx, accountRecord{
		ID:          account.ID,
		Username:    normalized.Username,
		UsernameKey: normalized.UsernameKey,
		Email:       normalized.Email,
		EmailKey:    normalized.EmailKey,
		Phone:       normalized.Phone,
		PhoneKey:    normalized.PhoneKey,
		DisplayName: displayName,
		UpdatedAt:   account.UpdatedAt,
	}); err != nil {
		return Account{}, err
	}

	return s.accounts.GetAccountByID(ctx, normalizedID)
}

func (s *Service) EnableAccount(ctx context.Context, accountID string) (Account, error) {
	return s.updateAccountStatus(ctx, accountID, StatusActive)
}

func (s *Service) DisableAccount(ctx context.Context, accountID string) (Account, error) {
	return s.updateAccountStatus(ctx, accountID, StatusDisabled)
}

func (s *Service) VerifyPassword(ctx context.Context, accountID string, password string) error {
	normalizedID, err := normalizeAccountID(accountID)
	if err != nil {
		return err
	}
	if err := validatePasswordInput(password); err != nil {
		return err
	}

	account, err := s.accounts.GetAccountByID(ctx, normalizedID)
	if err != nil {
		return err
	}
	if account.Status != StatusActive {
		return ErrInvalidAccountState
	}

	credential, err := s.credentials.GetPasswordCredentialByAccountID(ctx, normalizedID)
	if err != nil {
		return err
	}

	ok, err := s.hasher.Verify(credential.Hash, password)
	if err != nil {
		return wrapIdentityError(ErrPersistence, err)
	}
	if !ok {
		return ErrInvalidCredentials
	}

	return nil
}

func (s *Service) ChangePassword(ctx context.Context, accountID, currentPassword, newPassword string) error {
	normalizedID, err := normalizeAccountID(accountID)
	if err != nil {
		return err
	}
	if err := validatePasswordInput(currentPassword); err != nil {
		return err
	}
	if err := validatePasswordInput(newPassword); err != nil {
		return err
	}

	now := s.now().UTC()

	return s.transactions.WithinTransaction(ctx, func(txCtx context.Context, tx *sql.Tx) error {
		account, err := s.accounts.GetAccountByIDTx(txCtx, tx, normalizedID)
		if err != nil {
			return err
		}
		if account.Status != StatusActive {
			return ErrInvalidAccountState
		}

		credential, err := s.credentials.GetPasswordCredentialByAccountIDTx(txCtx, tx, normalizedID)
		if err != nil {
			return err
		}

		ok, err := s.hasher.Verify(credential.Hash, currentPassword)
		if err != nil {
			return wrapIdentityError(ErrPersistence, err)
		}
		if !ok {
			return ErrInvalidCredentials
		}

		passwordHash, err := s.hasher.Hash(newPassword)
		if err != nil {
			return wrapIdentityError(ErrPersistence, err)
		}

		credential.Hash = passwordHash
		credential.PasswordChangedAt = now
		credential.UpdatedAt = now

		return s.credentials.UpdatePasswordCredentialTx(txCtx, tx, credential)
	})
}

func (s *Service) PasswordParams() PasswordParams {
	return s.hasher.Params()
}

func (s *Service) validateCreateAccountInput(input CreateAccountInput) (NormalizedIdentity, string, AccountStatus, error) {
	normalized, err := input.Identity.Normalize()
	if err != nil {
		return NormalizedIdentity{}, "", "", err
	}
	if err := validatePasswordInput(input.Password); err != nil {
		return NormalizedIdentity{}, "", "", err
	}
	if err := validateDisplayName(input.DisplayName); err != nil {
		return NormalizedIdentity{}, "", "", err
	}

	status := input.Status
	if status == "" {
		status = StatusActive
	}
	if err := status.Validate(); err != nil {
		return NormalizedIdentity{}, "", "", err
	}

	return normalized, strings.TrimSpace(input.DisplayName), status, nil
}

func (s *Service) updateAccountStatus(ctx context.Context, accountID string, nextStatus AccountStatus) (Account, error) {
	normalizedID, err := normalizeAccountID(accountID)
	if err != nil {
		return Account{}, err
	}

	account, err := s.GetAccountByID(ctx, normalizedID)
	if err != nil {
		return Account{}, err
	}
	if account.Status == nextStatus {
		return Account{}, ErrInvalidAccountState
	}

	if err := s.accounts.UpdateAccountStatus(ctx, normalizedID, nextStatus, s.now().UTC()); err != nil {
		return Account{}, err
	}

	return s.accounts.GetAccountByID(ctx, normalizedID)
}

func accountFromRecord(record accountRecord) Account {
	return Account{
		ID:          record.ID,
		Username:    record.Username,
		Email:       record.Email,
		Phone:       record.Phone,
		DisplayName: record.DisplayName,
		Status:      record.Status,
		CreatedAt:   record.CreatedAt,
		UpdatedAt:   record.UpdatedAt,
	}
}
