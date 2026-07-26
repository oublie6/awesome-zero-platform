package authz

import (
	"context"
	"testing"
)

func TestServiceDelegatesThroughContracts(t *testing.T) {
	engine := &fakeEngine{allowed: true}
	service, err := NewService(engine, engine)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	assigned, err := service.AssignRole(context.Background(), "account-1", "admin")
	if err != nil || !assigned {
		t.Fatalf("AssignRole() = %v, %v", assigned, err)
	}
	granted, err := service.GrantPermission(context.Background(), "admin", "/platform/*", "GET|POST")
	if err != nil || !granted {
		t.Fatalf("GrantPermission() = %v, %v", granted, err)
	}
	allowed, err := service.Enforce(context.Background(), "account-1", "/platform/users", "GET")
	if err != nil || !allowed {
		t.Fatalf("Enforce() = %v, %v", allowed, err)
	}

	if engine.subject != "account-1" || engine.role != "admin" || engine.resource != "/platform/users" || engine.action != "GET" {
		t.Fatalf("unexpected delegated values: %#v", engine)
	}
}

func TestServiceRejectsEmptyPolicyValues(t *testing.T) {
	engine := &fakeEngine{}
	service, err := NewService(engine, engine)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	if _, err := service.AssignRole(context.Background(), "", "admin"); err == nil {
		t.Fatal("AssignRole() expected validation error")
	}
	if _, err := service.Enforce(context.Background(), "account-1", "", "GET"); err == nil {
		t.Fatal("Enforce() expected validation error")
	}
}

type fakeEngine struct {
	allowed  bool
	subject  string
	role     string
	resource string
	action   string
}

func (f *fakeEngine) Enforce(_ context.Context, subject, resource, action string) (bool, error) {
	f.subject, f.resource, f.action = subject, resource, action
	return f.allowed, nil
}
func (f *fakeEngine) AddRoleForUser(_ context.Context, subject, role string) (bool, error) {
	f.subject, f.role = subject, role
	return true, nil
}
func (f *fakeEngine) DeleteRoleForUser(context.Context, string, string) (bool, error) {
	return true, nil
}
func (f *fakeEngine) AddPermissionForRole(_ context.Context, role, resource, action string) (bool, error) {
	f.role, f.resource, f.action = role, resource, action
	return true, nil
}
func (f *fakeEngine) DeletePermissionForRole(context.Context, string, string, string) (bool, error) {
	return true, nil
}
func (f *fakeEngine) DeleteUser(context.Context, string) (bool, error) { return true, nil }
func (f *fakeEngine) DeleteRole(context.Context, string) (bool, error) { return true, nil }
