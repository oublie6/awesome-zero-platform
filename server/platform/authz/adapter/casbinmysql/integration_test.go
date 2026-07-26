//go:build integration

package casbinmysql

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/oublie6/awesome-zero-platform/server/foundation/database"
	"github.com/oublie6/awesome-zero-platform/server/platform/authz"
)

func TestCasbinMySQLIntegrationPersistence(t *testing.T) {
	if os.Getenv("APP_API_INTEGRATION") != "1" {
		t.Skip("set APP_API_INTEGRATION=1 to run integration tests")
	}

	handle, err := database.Open(context.Background(), database.Config{
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
	})
	if err != nil {
		t.Fatalf("database.Open() error = %v", err)
	}
	t.Cleanup(func() { _ = handle.Close() })

	cleanupPolicies(t, handle)
	t.Cleanup(func() { cleanupPolicies(t, handle) })

	engine, err := New(handle.DB())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	service, err := authz.NewService(engine, engine)
	if err != nil {
		t.Fatalf("authz.NewService() error = %v", err)
	}

	assigned, err := service.AssignRole(context.Background(), "account-1", "platform-admin")
	if err != nil || !assigned {
		t.Fatalf("AssignRole() = %v, %v", assigned, err)
	}
	granted, err := service.GrantPermission(context.Background(), "platform-admin", "/platform/:resource", "GET|POST")
	if err != nil || !granted {
		t.Fatalf("GrantPermission() = %v, %v", granted, err)
	}
	allowed, err := service.Enforce(context.Background(), "account-1", "/platform/users", "GET")
	if err != nil || !allowed {
		t.Fatalf("Enforce(allowed) = %v, %v", allowed, err)
	}
	denied, err := service.Enforce(context.Background(), "account-1", "/platform/users", "DELETE")
	if err != nil {
		t.Fatalf("Enforce(denied) error = %v", err)
	}
	if denied {
		t.Fatal("DELETE was unexpectedly allowed")
	}

	reloaded, err := New(handle.DB())
	if err != nil {
		t.Fatalf("New(reloaded) error = %v", err)
	}
	reloadedService, err := authz.NewService(reloaded, reloaded)
	if err != nil {
		t.Fatalf("authz.NewService(reloaded) error = %v", err)
	}
	allowed, err = reloadedService.Enforce(context.Background(), "account-1", "/platform/settings", "POST")
	if err != nil || !allowed {
		t.Fatalf("reloaded Enforce() = %v, %v", allowed, err)
	}

	if duplicate, err := reloadedService.AssignRole(context.Background(), "account-1", "platform-admin"); err != nil || duplicate {
		t.Fatalf("duplicate AssignRole() = %v, %v, want false, nil", duplicate, err)
	}
	var count int
	if err := handle.DB().QueryRow(`SELECT COUNT(*) FROM authorization_casbin_rules`).Scan(&count); err != nil {
		t.Fatalf("count policies: %v", err)
	}
	if count != 2 {
		t.Fatalf("policy count = %d, want 2", count)
	}

	revoked, err := reloadedService.RevokePermission(context.Background(), "platform-admin", "/platform/:resource", "GET|POST")
	if err != nil || !revoked {
		t.Fatalf("RevokePermission() = %v, %v", revoked, err)
	}
	allowed, err = reloadedService.Enforce(context.Background(), "account-1", "/platform/users", "GET")
	if err != nil {
		t.Fatalf("Enforce() after revoke error = %v", err)
	}
	if allowed {
		t.Fatal("permission remained allowed after revoke")
	}
}

func cleanupPolicies(t *testing.T, handle database.Handle) {
	t.Helper()
	if _, err := handle.DB().Exec(`DELETE FROM authorization_casbin_rules`); err != nil {
		t.Fatalf("cleanup authorization policies: %v", err)
	}
}
