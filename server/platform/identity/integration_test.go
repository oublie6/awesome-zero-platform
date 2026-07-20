//go:build integration

package identity

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/oublie6/awesome-zero-platform/server/foundation/database"
)

func TestIdentityIntegrationLifecycle(t *testing.T) {
	if os.Getenv("APP_API_INTEGRATION") != "1" {
		t.Skip("set APP_API_INTEGRATION=1 to run integration tests")
	}

	service, db := newIntegrationService(t)
	cleanupIdentityTables(t, db)

	account, err := service.CreateAccount(context.Background(), CreateAccountInput{
		Identity: Identity{
			Username: "Alice.Admin",
			Email:    "Alice.Admin@example.com",
			Phone:    "+14155550123",
		},
		DisplayName: "Alice Admin",
		Password:    "initial-password-123",
	})
	if err != nil {
		t.Fatalf("CreateAccount() error = %v", err)
	}

	var credentialHash string
	if err := db.QueryRowContext(context.Background(), `
		SELECT password_hash
		FROM identity_password_credentials
		WHERE account_id = ?
	`, account.ID).Scan(&credentialHash); err != nil {
		t.Fatalf("query credential hash: %v", err)
	}
	if strings.Contains(credentialHash, "initial-password-123") {
		t.Fatal("password hash unexpectedly contains plaintext password")
	}

	if _, err := service.GetAccountByID(context.Background(), account.ID); err != nil {
		t.Fatalf("GetAccountByID() error = %v", err)
	}
	if _, err := service.FindAccountByUsername(context.Background(), "alice.admin"); err != nil {
		t.Fatalf("FindAccountByUsername() error = %v", err)
	}
	if _, err := service.FindAccountByEmail(context.Background(), "alice.admin@example.com"); err != nil {
		t.Fatalf("FindAccountByEmail() error = %v", err)
	}
	if _, err := service.FindAccountByPhone(context.Background(), "+14155550123"); err != nil {
		t.Fatalf("FindAccountByPhone() error = %v", err)
	}

	updated, err := service.UpdateProfile(context.Background(), account.ID, UpdateProfileInput{
		Username:    stringPtr("alice.ops"),
		Email:       stringPtr("alice.ops@example.com"),
		Phone:       stringPtr("+14155550124"),
		DisplayName: stringPtr("Alice Operations"),
	})
	if err != nil {
		t.Fatalf("UpdateProfile() error = %v", err)
	}
	if updated.Username != "alice.ops" || updated.Email != "alice.ops@example.com" || updated.Phone != "+14155550124" {
		t.Fatalf("updated account = %#v", updated)
	}

	disabled, err := service.DisableAccount(context.Background(), account.ID)
	if err != nil {
		t.Fatalf("DisableAccount() error = %v", err)
	}
	if disabled.Status != StatusDisabled {
		t.Fatalf("disabled status = %q, want %q", disabled.Status, StatusDisabled)
	}
	if _, err := service.DisableAccount(context.Background(), account.ID); !errors.Is(err, ErrInvalidAccountState) {
		t.Fatalf("second DisableAccount() error = %v, want invalid account state", err)
	}
	if err := service.VerifyPassword(context.Background(), account.ID, "initial-password-123"); !errors.Is(err, ErrInvalidAccountState) {
		t.Fatalf("VerifyPassword() on disabled account error = %v, want invalid account state", err)
	}

	enabled, err := service.EnableAccount(context.Background(), account.ID)
	if err != nil {
		t.Fatalf("EnableAccount() error = %v", err)
	}
	if enabled.Status != StatusActive {
		t.Fatalf("enabled status = %q, want %q", enabled.Status, StatusActive)
	}

	if err := service.VerifyPassword(context.Background(), account.ID, "initial-password-123"); err != nil {
		t.Fatalf("VerifyPassword() error = %v", err)
	}
	if err := service.VerifyPassword(context.Background(), account.ID, "wrong-password-123"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("VerifyPassword() wrong password error = %v, want invalid credentials", err)
	}

	if err := service.ChangePassword(context.Background(), account.ID, "initial-password-123", "new-password-456"); err != nil {
		t.Fatalf("ChangePassword() error = %v", err)
	}
	if err := service.VerifyPassword(context.Background(), account.ID, "initial-password-123"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("VerifyPassword() old password error = %v, want invalid credentials", err)
	}
	if err := service.VerifyPassword(context.Background(), account.ID, "new-password-456"); err != nil {
		t.Fatalf("VerifyPassword() new password error = %v", err)
	}
}

func TestIdentityIntegrationRejectsInvalidAndDuplicateValues(t *testing.T) {
	if os.Getenv("APP_API_INTEGRATION") != "1" {
		t.Skip("set APP_API_INTEGRATION=1 to run integration tests")
	}

	service, db := newIntegrationService(t)
	cleanupIdentityTables(t, db)

	if _, err := service.CreateAccount(context.Background(), CreateAccountInput{
		Identity:    Identity{},
		DisplayName: "No Identity",
		Password:    "valid-password-123",
	}); err == nil {
		t.Fatal("expected missing identity error, got nil")
	}

	if _, err := service.CreateAccount(context.Background(), CreateAccountInput{
		Identity: Identity{
			Email: "not-an-email",
		},
		DisplayName: "Bad Email",
		Password:    "valid-password-123",
	}); err == nil {
		t.Fatal("expected invalid email error, got nil")
	}

	base := CreateAccountInput{
		Identity: Identity{
			Username: "duplicate.user",
			Email:    "duplicate.user@example.com",
			Phone:    "+14155550125",
		},
		DisplayName: "Duplicate User",
		Password:    "valid-password-123",
	}
	account, err := service.CreateAccount(context.Background(), base)
	if err != nil {
		t.Fatalf("CreateAccount() error = %v", err)
	}

	if _, err := service.GetAccountByID(context.Background(), "00000000-0000-0000-0000-000000000000"); !errors.Is(err, ErrAccountNotFound) {
		t.Fatalf("GetAccountByID() not found error = %v, want account not found", err)
	}
	if _, err := service.FindAccountByUsername(context.Background(), "missing.user"); !errors.Is(err, ErrAccountNotFound) {
		t.Fatalf("FindAccountByUsername() not found error = %v, want account not found", err)
	}
	if _, err := service.FindAccountByEmail(context.Background(), "missing.user@example.com"); !errors.Is(err, ErrAccountNotFound) {
		t.Fatalf("FindAccountByEmail() not found error = %v, want account not found", err)
	}
	if _, err := service.FindAccountByPhone(context.Background(), "+14155550999"); !errors.Is(err, ErrAccountNotFound) {
		t.Fatalf("FindAccountByPhone() not found error = %v, want account not found", err)
	}

	conflictCases := []struct {
		name  string
		input CreateAccountInput
	}{
		{
			name: "duplicate username",
			input: CreateAccountInput{
				Identity: Identity{
					Username: "duplicate.user",
					Email:    "other1@example.com",
				},
				DisplayName: "Other One",
				Password:    "valid-password-123",
			},
		},
		{
			name: "duplicate email",
			input: CreateAccountInput{
				Identity: Identity{
					Username: "other-two",
					Email:    "duplicate.user@example.com",
				},
				DisplayName: "Other Two",
				Password:    "valid-password-123",
			},
		},
		{
			name: "duplicate phone",
			input: CreateAccountInput{
				Identity: Identity{
					Username: "other-three",
					Phone:    "+14155550125",
				},
				DisplayName: "Other Three",
				Password:    "valid-password-123",
			},
		},
	}

	for _, tt := range conflictCases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.CreateAccount(context.Background(), tt.input)
			if !errors.Is(err, ErrIdentityConflict) {
				t.Fatalf("CreateAccount() conflict error = %v, want identity conflict", err)
			}
			if err == nil {
				t.Fatal("expected conflict error, got nil")
			}
			if strings.Contains(err.Error(), "Duplicate entry") || strings.Contains(err.Error(), "identity_accounts") ||
				strings.Contains(err.Error(), "127.0.0.1") || strings.Contains(err.Error(), "duplicate.user") {
				t.Fatalf("conflict error leaked sensitive details: %v", err)
			}
		})
	}

	if err := service.VerifyPassword(context.Background(), account.ID, ""); err == nil {
		t.Fatal("expected empty password validation error, got nil")
	}
	if err := service.VerifyPassword(context.Background(), account.ID, strings.Repeat("x", PasswordMaxBytes+1)); err == nil {
		t.Fatal("expected oversized password validation error, got nil")
	}
}

func TestIdentityIntegrationTransactionRollbackAndForeignKey(t *testing.T) {
	if os.Getenv("APP_API_INTEGRATION") != "1" {
		t.Skip("set APP_API_INTEGRATION=1 to run integration tests")
	}

	baseService, db := newIntegrationService(t)
	cleanupIdentityTables(t, db)

	failing := newService(serviceDependencies{
		transactions: baseService.transactions,
		accounts:     baseService.accounts,
		credentials:  failingCredentialStore{},
		hasher:       NewTestArgon2idHasher(),
		generateID: func() (string, error) {
			return "019824ef-b2c5-7cc2-8dc5-5f43a9f53e4b", nil
		},
		now: time.Now,
	})

	_, err := failing.CreateAccount(context.Background(), CreateAccountInput{
		Identity: Identity{
			Username: "rollback.user",
			Email:    "rollback.user@example.com",
		},
		DisplayName: "Rollback User",
		Password:    "valid-password-123",
	})
	if !errors.Is(err, ErrPersistence) {
		t.Fatalf("CreateAccount() rollback error = %v, want persistence failure", err)
	}

	var accountCount int
	if err := db.QueryRowContext(context.Background(), `
		SELECT COUNT(*)
		FROM identity_accounts
		WHERE username_key = ?
	`, "rollback.user").Scan(&accountCount); err != nil {
		t.Fatalf("count rolled back account: %v", err)
	}
	if accountCount != 0 {
		t.Fatalf("rolled back account count = %d, want 0", accountCount)
	}

	_, err = db.ExecContext(context.Background(), `
		INSERT INTO identity_password_credentials (account_id, password_hash, password_changed_at, created_at, updated_at)
		VALUES (?, ?, UTC_TIMESTAMP(6), UTC_TIMESTAMP(6), UTC_TIMESTAMP(6))
	`, "019824ef-b2c5-7cc2-8dc5-5f43a9f53e4c", "$argon2id$v=19$m=8192,t=1,p=1$ZmFrZXNhbHQ$ZmFrZWhhc2g")
	if err == nil {
		t.Fatal("expected foreign-key error, got nil")
	}
}

func TestIdentityIntegrationConcurrentCreateSameUsername(t *testing.T) {
	if os.Getenv("APP_API_INTEGRATION") != "1" {
		t.Skip("set APP_API_INTEGRATION=1 to run integration tests")
	}

	service, db := newIntegrationService(t)
	cleanupIdentityTables(t, db)

	username := fmt.Sprintf("concurrent.%d", time.Now().UnixNano())

	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		success  int
		conflict int
	)

	for range 5 {
		wg.Add(1)
		go func() {
			defer wg.Done()

			_, err := service.CreateAccount(context.Background(), CreateAccountInput{
				Identity: Identity{
					Username: username,
					Email:    fmt.Sprintf("%s-%d@example.com", username, time.Now().UnixNano()),
				},
				DisplayName: "Concurrent User",
				Password:    "valid-password-123",
			})

			mu.Lock()
			defer mu.Unlock()
			switch {
			case err == nil:
				success++
			case errors.Is(err, ErrIdentityConflict):
				conflict++
			default:
				t.Errorf("unexpected concurrent CreateAccount() error: %v", err)
			}
		}()
	}

	wg.Wait()

	if success != 1 {
		t.Fatalf("success count = %d, want 1", success)
	}
	if conflict != 4 {
		t.Fatalf("conflict count = %d, want 4", conflict)
	}
}

type failingCredentialStore struct{}

func (failingCredentialStore) InsertPasswordCredentialTx(context.Context, *sql.Tx, storedCredential) error {
	return ErrPersistence
}

func (failingCredentialStore) GetPasswordCredentialByAccountID(context.Context, string) (storedCredential, error) {
	return storedCredential{}, ErrPersistence
}

func (failingCredentialStore) GetPasswordCredentialByAccountIDTx(context.Context, *sql.Tx, string) (storedCredential, error) {
	return storedCredential{}, ErrPersistence
}

func (failingCredentialStore) UpdatePasswordCredentialTx(context.Context, *sql.Tx, storedCredential) error {
	return ErrPersistence
}

func newIntegrationService(t *testing.T) (*Service, *sql.DB) {
	t.Helper()

	cfg := database.Config{
		Addr:             "127.0.0.1:3306",
		Database:         "awesome_zero_platform",
		User:             "app_local",
		Password:         "local-dev-only-mysql-password",
		Charset:          "utf8mb4",
		ParseTime:        true,
		Location:         "UTC",
		TimeZone:         "+00:00",
		Timeout:          3 * time.Second,
		MaxOpenConns:     4,
		MaxIdleConns:     2,
		ConnMaxLifetime:  5 * time.Minute,
		StartupTimeout:   3 * time.Second,
		ReadinessTimeout: 2 * time.Second,
	}

	handle, err := database.Open(context.Background(), cfg)
	if err != nil {
		t.Fatalf("database.Open() error = %v", err)
	}
	t.Cleanup(func() {
		_ = handle.Close()
	})

	return newService(serviceDependencies{
		transactions: handle,
		accounts:     NewMySQLStore(handle.DB()),
		credentials:  NewMySQLStore(handle.DB()),
		hasher:       NewTestArgon2idHasher(),
		now:          time.Now,
	}), handle.DB()
}

func cleanupIdentityTables(t *testing.T, db *sql.DB) {
	t.Helper()

	if _, err := db.Exec(`
		DELETE FROM identity_password_credentials
	`); err != nil {
		t.Fatalf("cleanup credentials: %v", err)
	}
	if _, err := db.Exec(`
		DELETE FROM identity_accounts
	`); err != nil {
		t.Fatalf("cleanup accounts: %v", err)
	}
}

func stringPtr(input string) *string {
	return &input
}
