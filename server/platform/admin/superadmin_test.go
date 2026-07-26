package admin

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/oublie6/awesome-zero-platform/server/platform/authn"
	"github.com/oublie6/awesome-zero-platform/server/platform/authz"
	"github.com/oublie6/awesome-zero-platform/server/platform/identity"
)

func TestSetAccountEnabledProtectsLastActiveSuperAdministrator(t *testing.T) {
	identityManager := newFakeAdminIdentity(
		identity.Account{ID: "active-admin", Status: identity.StatusActive},
		identity.Account{ID: "disabled-admin", Status: identity.StatusDisabled},
	)
	authorization := newFakeAdminAuthorization()
	authorization.roles["active-admin"] = []string{SuperAdminRole}
	authorization.roles["disabled-admin"] = []string{SuperAdminRole}
	repository := &lockingAdminRepository{}
	service := newAdminTestService(t, identityManager, authorization, repository)

	_, err := service.SetAccountEnabled(context.Background(), "active-admin", false, Actor{})
	if !errors.Is(err, ErrProtectedRole) {
		t.Fatalf("SetAccountEnabled() error = %v, want ErrProtectedRole", err)
	}
	if got := identityManager.account("active-admin").Status; got != identity.StatusActive {
		t.Fatalf("active admin status = %q, want active", got)
	}

	identityManager.setStatus("disabled-admin", identity.StatusActive)
	account, err := service.SetAccountEnabled(context.Background(), "active-admin", false, Actor{})
	if err != nil {
		t.Fatalf("SetAccountEnabled() with second active admin error = %v", err)
	}
	if account.Status != identity.StatusDisabled {
		t.Fatalf("account status = %q, want disabled", account.Status)
	}
	if repository.lockCalls != 2 {
		t.Fatalf("super admin lock calls = %d, want 2", repository.lockCalls)
	}
}

func TestBootstrapIsSerializedAcrossReplicas(t *testing.T) {
	identityManager := newFakeAdminIdentity()
	authorization := newFakeAdminAuthorization()
	repository := &lockingAdminRepository{}
	service := newAdminTestService(t, identityManager, authorization, repository)

	start := make(chan struct{})
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			<-start
			_, err := service.Bootstrap(context.Background(), "bootstrap-token", BootstrapInput{
				Username:    fmt.Sprintf("admin-%d", index),
				DisplayName: "Administrator",
				Password:    "valid-password",
			}, Actor{})
			errs <- err
		}(i)
	}
	close(start)
	wg.Wait()
	close(errs)

	succeeded := 0
	completed := 0
	for err := range errs {
		switch {
		case err == nil:
			succeeded++
		case errors.Is(err, ErrBootstrapComplete):
			completed++
		default:
			t.Fatalf("Bootstrap() unexpected error = %v", err)
		}
	}
	if succeeded != 1 || completed != 1 {
		t.Fatalf("bootstrap results: succeeded=%d complete=%d", succeeded, completed)
	}
	users, err := authorization.UsersForRole(context.Background(), SuperAdminRole)
	if err != nil {
		t.Fatalf("UsersForRole() error = %v", err)
	}
	if len(users) != 1 {
		t.Fatalf("super admin users = %v, want one", users)
	}
}

func newAdminTestService(t *testing.T, identityManager *fakeAdminIdentity, authorization *fakeAdminAuthorization, repository *lockingAdminRepository) *Service {
	t.Helper()
	service, err := NewService(identityManager, authorization, fakeSessionAdmin{}, repository, Config{BootstrapToken: "bootstrap-token"})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	return service
}

type fakeAdminIdentity struct {
	mu       sync.Mutex
	accounts map[string]identity.Account
	nextID   int
}

func newFakeAdminIdentity(accounts ...identity.Account) *fakeAdminIdentity {
	result := &fakeAdminIdentity{accounts: make(map[string]identity.Account)}
	for _, account := range accounts {
		result.accounts[account.ID] = account
	}
	return result
}

func (f *fakeAdminIdentity) ListAccounts(context.Context, identity.AccountQuery) (identity.AccountPage, error) {
	return identity.AccountPage{}, nil
}
func (f *fakeAdminIdentity) CreateAccount(_ context.Context, input identity.CreateAccountInput) (identity.Account, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.nextID++
	status := input.Status
	if status == "" {
		status = identity.StatusActive
	}
	account := identity.Account{
		ID:          fmt.Sprintf("account-%d", f.nextID),
		Username:    input.Identity.Username,
		DisplayName: input.DisplayName,
		Status:      status,
	}
	f.accounts[account.ID] = account
	return account, nil
}
func (f *fakeAdminIdentity) GetAccountByID(_ context.Context, accountID string) (identity.Account, error) {
	return f.accountResult(accountID)
}
func (f *fakeAdminIdentity) GetAccountByIDFresh(_ context.Context, accountID string) (identity.Account, error) {
	return f.accountResult(accountID)
}
func (f *fakeAdminIdentity) UpdateProfile(context.Context, string, identity.UpdateProfileInput) (identity.Account, error) {
	return identity.Account{}, nil
}
func (f *fakeAdminIdentity) EnableAccount(_ context.Context, accountID string) (identity.Account, error) {
	f.setStatus(accountID, identity.StatusActive)
	return f.accountResult(accountID)
}
func (f *fakeAdminIdentity) DisableAccount(_ context.Context, accountID string) (identity.Account, error) {
	f.setStatus(accountID, identity.StatusDisabled)
	return f.accountResult(accountID)
}
func (f *fakeAdminIdentity) ResetPassword(context.Context, string, string) error { return nil }
func (f *fakeAdminIdentity) PasswordParams() identity.PasswordParams {
	return identity.PasswordParams{}
}
func (f *fakeAdminIdentity) account(accountID string) identity.Account {
	account, _ := f.accountResult(accountID)
	return account
}
func (f *fakeAdminIdentity) accountResult(accountID string) (identity.Account, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	account, ok := f.accounts[accountID]
	if !ok {
		return identity.Account{}, identity.ErrAccountNotFound
	}
	return account, nil
}
func (f *fakeAdminIdentity) setStatus(accountID string, status identity.AccountStatus) {
	f.mu.Lock()
	account := f.accounts[accountID]
	account.Status = status
	f.accounts[accountID] = account
	f.mu.Unlock()
}

type fakeAdminAuthorization struct {
	mu    sync.Mutex
	roles map[string][]string
}

func newFakeAdminAuthorization() *fakeAdminAuthorization {
	return &fakeAdminAuthorization{roles: make(map[string][]string)}
}
func (f *fakeAdminAuthorization) EngineInfo(context.Context) authz.EngineInfo {
	return authz.EngineInfo{ID: "fake"}
}
func (f *fakeAdminAuthorization) ModelText(context.Context) string { return "" }
func (f *fakeAdminAuthorization) ListRawRules(context.Context) ([]authz.RawRule, error) {
	return nil, nil
}
func (f *fakeAdminAuthorization) ValidateRawRules(context.Context, []authz.RawRule) error { return nil }
func (f *fakeAdminAuthorization) ReplaceRawRules(context.Context, []authz.RawRule) error  { return nil }
func (f *fakeAdminAuthorization) RolesForUser(_ context.Context, accountID string) ([]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.roles[accountID]...), nil
}
func (f *fakeAdminAuthorization) UsersForRole(_ context.Context, role string) ([]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	users := make([]string, 0)
	for accountID, roles := range f.roles {
		if contains(roles, role) {
			users = append(users, accountID)
		}
	}
	return users, nil
}
func (f *fakeAdminAuthorization) PermissionsForRole(context.Context, string) ([]authz.Permission, error) {
	return nil, nil
}
func (f *fakeAdminAuthorization) ReplacePermissionsForRole(context.Context, string, []authz.Permission) error {
	return nil
}
func (f *fakeAdminAuthorization) ReplaceRolesForUser(_ context.Context, accountID string, roles []string) error {
	f.mu.Lock()
	f.roles[accountID] = append([]string(nil), roles...)
	f.mu.Unlock()
	return nil
}
func (f *fakeAdminAuthorization) Explain(context.Context, string, string, string) (authz.Explanation, error) {
	return authz.Explanation{}, nil
}

type fakeSessionAdmin struct{}

func (fakeSessionAdmin) ListByAccount(context.Context, string) ([]authn.SessionView, error) {
	return nil, nil
}
func (fakeSessionAdmin) RevokeByAccount(context.Context, string) (int64, error) { return 0, nil }

type lockingAdminRepository struct {
	mu        sync.Mutex
	lockCalls int
}

func (r *lockingAdminRepository) WithSuperAdminLock(ctx context.Context, fn func(context.Context) error) error {
	r.mu.Lock()
	r.lockCalls++
	defer r.mu.Unlock()
	return fn(ctx)
}
func (*lockingAdminRepository) ListRoles(context.Context) ([]Role, error) { return nil, nil }
func (*lockingAdminRepository) GetRole(context.Context, string) (Role, error) {
	return Role{}, nil
}
func (*lockingAdminRepository) CreateRole(_ context.Context, role Role) (Role, error) {
	return role, nil
}
func (*lockingAdminRepository) UpdateRole(_ context.Context, role Role) (Role, error) {
	return role, nil
}
func (*lockingAdminRepository) DeleteRole(context.Context, string) error          { return nil }
func (*lockingAdminRepository) ListResources(context.Context) ([]Resource, error) { return nil, nil }
func (*lockingAdminRepository) GetResource(context.Context, string) (Resource, error) {
	return Resource{}, nil
}
func (*lockingAdminRepository) CreateResource(_ context.Context, resource Resource) (Resource, error) {
	return resource, nil
}
func (*lockingAdminRepository) UpdateResource(_ context.Context, resource Resource) (Resource, error) {
	return resource, nil
}
func (*lockingAdminRepository) DeleteResource(context.Context, string) error  { return nil }
func (*lockingAdminRepository) AppendAudit(context.Context, AuditEvent) error { return nil }
func (*lockingAdminRepository) ListAudit(context.Context, AuditQuery) (AuditPage, error) {
	return AuditPage{}, nil
}
