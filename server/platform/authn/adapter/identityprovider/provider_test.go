package identityprovider

import (
	"context"
	"errors"
	"testing"

	"github.com/oublie6/awesome-zero-platform/server/platform/authn"
	"github.com/oublie6/awesome-zero-platform/server/platform/identity"
)

func TestProviderAuthenticatesUsernameEmailAndPhone(t *testing.T) {
	account := identity.Account{
		ID:          "01984f63-ec7f-7a4a-b908-33e8ff14d465",
		Username:    "alice",
		Email:       "alice@example.com",
		Phone:       "+14155550123",
		DisplayName: "Alice",
		Status:      identity.StatusActive,
	}
	service := &fakeAccountService{account: account}
	provider := New(service)

	for _, identifier := range []string{"alice", "alice@example.com", "+14155550123"} {
		principal, err := provider.Authenticate(context.Background(), identifier, "correct-password")
		if err != nil {
			t.Fatalf("Authenticate(%q) error = %v", identifier, err)
		}
		if principal.AccountID != account.ID || principal.DisplayName != account.DisplayName {
			t.Fatalf("principal = %#v", principal)
		}
	}
	if service.usernameCalls != 1 || service.emailCalls != 1 || service.phoneCalls != 1 || service.verifyCalls != 3 {
		t.Fatalf("unexpected calls: %#v", service)
	}
}

func TestProviderSeparatesCredentialAndPersistenceFailures(t *testing.T) {
	credentialProvider := New(&fakeAccountService{findErr: identity.ErrAccountNotFound})
	if _, err := credentialProvider.Authenticate(context.Background(), "missing", "password"); !errors.Is(err, authn.ErrInvalidCredentials) {
		t.Fatalf("missing account error = %v, want ErrInvalidCredentials", err)
	}

	persistenceFailure := errors.New("mysql unavailable")
	failureProvider := New(&fakeAccountService{findErr: persistenceFailure})
	if _, err := failureProvider.Authenticate(context.Background(), "alice", "password"); err == nil || errors.Is(err, authn.ErrInvalidCredentials) {
		t.Fatalf("persistence error = %v, want non-credential failure", err)
	}
}

func TestProviderRejectsDisabledAccount(t *testing.T) {
	provider := New(&fakeAccountService{account: identity.Account{
		ID:     "01984f63-ec7f-7a4a-b908-33e8ff14d465",
		Status: identity.StatusDisabled,
	}})
	if _, err := provider.Authenticate(context.Background(), "alice", "password"); !errors.Is(err, authn.ErrInvalidCredentials) {
		t.Fatalf("Authenticate() error = %v, want ErrInvalidCredentials", err)
	}
	if _, err := provider.ResolveActive(context.Background(), "01984f63-ec7f-7a4a-b908-33e8ff14d465"); !errors.Is(err, authn.ErrAccountUnavailable) {
		t.Fatalf("ResolveActive() error = %v, want ErrAccountUnavailable", err)
	}
}

type fakeAccountService struct {
	account       identity.Account
	findErr       error
	verifyErr     error
	usernameCalls int
	emailCalls    int
	phoneCalls    int
	verifyCalls   int
}

func (f *fakeAccountService) FindAccountByUsername(context.Context, string) (identity.Account, error) {
	f.usernameCalls++
	return f.account, f.findErr
}

func (f *fakeAccountService) FindAccountByEmail(context.Context, string) (identity.Account, error) {
	f.emailCalls++
	return f.account, f.findErr
}

func (f *fakeAccountService) FindAccountByPhone(context.Context, string) (identity.Account, error) {
	f.phoneCalls++
	return f.account, f.findErr
}

func (f *fakeAccountService) GetAccountByID(context.Context, string) (identity.Account, error) {
	return f.account, f.findErr
}

func (f *fakeAccountService) VerifyPassword(context.Context, string, string) error {
	f.verifyCalls++
	return f.verifyErr
}
