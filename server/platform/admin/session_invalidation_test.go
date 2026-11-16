package admin

import (
	"context"
	"errors"
	"testing"

	"github.com/oublie6/awesome-zero-platform/server/platform/authn"
	"github.com/oublie6/awesome-zero-platform/server/platform/identity"
)

type recorderSessionAdmin struct {
	calls int
	err   error
}

func (s *recorderSessionAdmin) ListByAccount(context.Context, string) ([]authn.SessionView, error) {
	return nil, nil
}
func (s *recorderSessionAdmin) RevokeByAccount(context.Context, string) (int64, error) {
	s.calls++
	return 0, s.err
}
func TestDisabledAccountCannotBeReenabledWhenRedisRevocationFails(t *testing.T) {
	id := newFakeAdminIdentity(identity.Account{ID: "affected-account", Status: identity.StatusDisabled})
	service := newAdminTestService(t, id, newFakeAdminAuthorization(), &lockingAdminRepository{})
	expected := errors.New("redis unavailable")
	tracker := &recorderSessionAdmin{err: expected}
	service.sessions = tracker
	_, err := service.SetAccountEnabled(context.Background(), "affected-account", true, Actor{})
	if !errors.Is(err, expected) {
		t.Fatalf("reenable error=%v", err)
	}
	if got := id.account("affected-account").Status; got != identity.StatusDisabled {
		t.Fatalf("reenabled with live old sessions: %v", got)
	}
	if tracker.calls != 1 {
		t.Fatalf("revocation attempted %d times", tracker.calls)
	}
}
func TestDisableReportsRevocationFailureWhileLeavingAccountDisabled(t *testing.T) {
	id := newFakeAdminIdentity(identity.Account{ID: "affected-account", Status: identity.StatusActive})
	service := newAdminTestService(t, id, newFakeAdminAuthorization(), &lockingAdminRepository{})
	expected := errors.New("redis unavailable")
	service.sessions = &recorderSessionAdmin{err: expected}
	_, err := service.SetAccountEnabled(context.Background(), "affected-account", false, Actor{})
	if !errors.Is(err, expected) {
		t.Fatalf("disabled falsely reported success: %v", err)
	}
	if got := id.account("affected-account").Status; got != identity.StatusDisabled {
		t.Fatalf("account not safely disabled: %v", got)
	}
}
func TestAdminPasswordResetRequiresPreAndPostRevocation(t *testing.T) {
	id := newFakeAdminIdentity(identity.Account{ID: "affected-account", Status: identity.StatusActive})
	service := newAdminTestService(t, id, newFakeAdminAuthorization(), &lockingAdminRepository{})
	tracker := &recorderSessionAdmin{}
	service.sessions = tracker
	if err := service.ResetPassword(context.Background(), "affected-account", "new-password-1234", Actor{}); err != nil {
		t.Fatal(err)
	}
	if tracker.calls != 2 {
		t.Fatalf("password reset revocations=%d, want 2", tracker.calls)
	}
}
