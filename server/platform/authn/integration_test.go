//go:build integration

package authn_test

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/oublie6/awesome-zero-platform/server/foundation/cache"
	"github.com/oublie6/awesome-zero-platform/server/foundation/database"
	"github.com/oublie6/awesome-zero-platform/server/platform/authn"
	"github.com/oublie6/awesome-zero-platform/server/platform/authn/adapter/identityprovider"
	"github.com/oublie6/awesome-zero-platform/server/platform/authn/adapter/jwthmac"
	"github.com/oublie6/awesome-zero-platform/server/platform/authn/adapter/redissession"
	"github.com/oublie6/awesome-zero-platform/server/platform/identity"
)

func TestAuthenticationIntegrationLifecycle(t *testing.T) {
	if os.Getenv("APP_API_INTEGRATION") != "1" {
		t.Skip("set APP_API_INTEGRATION=1 to run integration tests")
	}

	ctx := context.Background()
	mysql, err := database.Open(ctx, integrationDatabaseConfig())
	if err != nil {
		t.Fatalf("database.Open() error = %v", err)
	}
	t.Cleanup(func() { _ = mysql.Close() })
	redisHandle, err := cache.Open(ctx, integrationRedisConfig())
	if err != nil {
		t.Fatalf("cache.Open() error = %v", err)
	}
	t.Cleanup(func() { _ = redisHandle.Close() })

	cleanupIdentity(t, mysql)
	t.Cleanup(func() { cleanupIdentity(t, mysql) })

	identityService := identity.NewService(mysql)
	account, err := identityService.CreateAccount(ctx, identity.CreateAccountInput{
		Identity: identity.Identity{
			Username: "auth.integration",
			Email:    "auth.integration@example.com",
			Phone:    "+14155550888",
		},
		DisplayName: "Authentication Integration",
		Password:    "integration-password-123",
	})
	if err != nil {
		t.Fatalf("CreateAccount() error = %v", err)
	}

	prefix := "authn:integration:" + strings.ReplaceAll(account.ID, "-", "") + ":"
	t.Cleanup(func() { cleanupRedisPrefix(t, redisHandle, prefix) })
	store, err := redissession.New(redisHandle.Client(), prefix)
	if err != nil {
		t.Fatalf("redissession.New() error = %v", err)
	}
	codec, err := jwthmac.New("integration-access-token-secret-0123456789abcdef", "authn-integration")
	if err != nil {
		t.Fatalf("jwthmac.New() error = %v", err)
	}
	service, err := authn.NewService(
		identityprovider.New(identityService),
		codec,
		store,
		authn.Config{AccessTTL: 5 * time.Minute, RefreshTTL: time.Hour},
	)
	if err != nil {
		t.Fatalf("authn.NewService() error = %v", err)
	}

	authentication, tokens, err := service.Login(ctx, "AUTH.INTEGRATION", "integration-password-123")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if authentication.Principal.AccountID != account.ID {
		t.Fatalf("login account id = %q, want %q", authentication.Principal.AccountID, account.ID)
	}
	if _, err := service.AuthenticateAccess(ctx, tokens.AccessToken); err != nil {
		t.Fatalf("AuthenticateAccess() error = %v", err)
	}

	_, rotated, err := service.Refresh(ctx, tokens.RefreshToken)
	if err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}
	if rotated.RefreshToken == tokens.RefreshToken {
		t.Fatal("refresh token was not rotated")
	}
	if _, _, err := service.Refresh(ctx, tokens.RefreshToken); !errors.Is(err, authn.ErrInvalidRefresh) {
		t.Fatalf("reused refresh token error = %v, want ErrInvalidRefresh", err)
	}

	if _, err := identityService.DisableAccount(ctx, account.ID); err != nil {
		t.Fatalf("DisableAccount() error = %v", err)
	}
	if _, err := service.AuthenticateAccess(ctx, rotated.AccessToken); !errors.Is(err, authn.ErrAccountUnavailable) {
		t.Fatalf("disabled account access error = %v, want ErrAccountUnavailable", err)
	}
	if _, err := service.AuthenticateAccess(ctx, rotated.AccessToken); !errors.Is(err, authn.ErrInvalidToken) {
		t.Fatalf("revoked disabled session error = %v, want ErrInvalidToken", err)
	}

	if _, err := identityService.EnableAccount(ctx, account.ID); err != nil {
		t.Fatalf("EnableAccount() error = %v", err)
	}
	_, nextTokens, err := service.Login(ctx, "auth.integration@example.com", "integration-password-123")
	if err != nil {
		t.Fatalf("second Login() error = %v", err)
	}
	if err := service.Logout(ctx, nextTokens.AccessToken); err != nil {
		t.Fatalf("Logout() error = %v", err)
	}
	if _, err := service.AuthenticateAccess(ctx, nextTokens.AccessToken); !errors.Is(err, authn.ErrInvalidToken) {
		t.Fatalf("logged-out access error = %v, want ErrInvalidToken", err)
	}
}

func integrationDatabaseConfig() database.Config {
	return database.Config{
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
}

func integrationRedisConfig() cache.Config {
	return cache.Config{
		Addr:             "127.0.0.1:6379",
		Password:         "local-dev-only-redis-password",
		DB:               0,
		PoolSize:         4,
		DialTimeout:      3 * time.Second,
		ReadTimeout:      3 * time.Second,
		WriteTimeout:     3 * time.Second,
		StartupTimeout:   3 * time.Second,
		ReadinessTimeout: 2 * time.Second,
	}
}

func cleanupIdentity(t *testing.T, mysql database.Handle) {
	t.Helper()
	if _, err := mysql.DB().Exec(`DELETE FROM identity_password_credentials`); err != nil {
		t.Fatalf("cleanup identity credentials: %v", err)
	}
	if _, err := mysql.DB().Exec(`DELETE FROM identity_accounts`); err != nil {
		t.Fatalf("cleanup identity accounts: %v", err)
	}
}

func cleanupRedisPrefix(t *testing.T, handle cache.Handle, prefix string) {
	t.Helper()
	ctx := context.Background()
	var cursor uint64
	for {
		keys, next, err := handle.Client().Scan(ctx, cursor, prefix+"*", 100).Result()
		if err != nil {
			t.Fatalf("scan redis sessions: %v", err)
		}
		if len(keys) > 0 {
			if err := handle.Client().Del(ctx, keys...).Err(); err != nil {
				t.Fatalf("delete redis sessions: %v", err)
			}
		}
		cursor = next
		if cursor == 0 {
			return
		}
	}
}
