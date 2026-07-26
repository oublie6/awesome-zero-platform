//go:build integration

package casbinmysql

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	mysql "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

func TestClusteredEnginesConvergeThroughNotification(t *testing.T) {
	db, redisClient := openClusterIntegrationDependencies(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	channel := fmt.Sprintf("integration:authz:%d", time.Now().UnixNano())
	engineA := newStartedClusterEngine(t, ctx, db, redisClient, "engine-a", channel, 250*time.Millisecond)
	engineB := newStartedClusterEngine(t, ctx, db, redisClient, "engine-b", channel, 250*time.Millisecond)
	defer engineA.Close()
	defer engineB.Close()

	userID := uuid.NewString()
	role := fmt.Sprintf("integration_role_%d", time.Now().UnixNano())
	resource := "/integration/cluster"
	action := "GET"
	defer cleanupClusterPolicy(engineA, userID, role)

	if changed, err := engineA.AddRoleForUser(ctx, userID, role); err != nil || !changed {
		t.Fatalf("AddRoleForUser() changed=%v error=%v", changed, err)
	}
	if changed, err := engineA.AddPermissionForRole(ctx, role, resource, action); err != nil || !changed {
		t.Fatalf("AddPermissionForRole() changed=%v error=%v", changed, err)
	}
	waitForAuthorization(t, engineB, userID, resource, action, true)

	if changed, err := engineA.DeletePermissionForRole(ctx, role, resource, action); err != nil || !changed {
		t.Fatalf("DeletePermissionForRole() changed=%v error=%v", changed, err)
	}
	waitForAuthorization(t, engineB, userID, resource, action, false)
}

func TestClusteredEngineRepairsMissedNotificationByVersionPolling(t *testing.T) {
	db, redisClient := openClusterIntegrationDependencies(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	engineA := newStartedClusterEngine(t, ctx, db, redisClient, "poll-a", "integration:authz:publisher", 50*time.Millisecond)
	// Deliberately subscribe to another channel so this instance can only learn
	// the change from authorization_policy_state polling.
	engineB := newStartedClusterEngine(t, ctx, db, redisClient, "poll-b", "integration:authz:missed", 50*time.Millisecond)
	defer engineA.Close()
	defer engineB.Close()

	userID := uuid.NewString()
	role := fmt.Sprintf("integration_poll_role_%d", time.Now().UnixNano())
	resource := "/integration/poll"
	action := "POST"
	defer cleanupClusterPolicy(engineA, userID, role)

	if _, err := engineA.AddRoleForUser(ctx, userID, role); err != nil {
		t.Fatalf("AddRoleForUser() error = %v", err)
	}
	if _, err := engineA.AddPermissionForRole(ctx, role, resource, action); err != nil {
		t.Fatalf("AddPermissionForRole() error = %v", err)
	}
	waitForAuthorization(t, engineB, userID, resource, action, true)

	infoA := engineA.EngineInfo(ctx)
	infoB := engineB.EngineInfo(ctx)
	if infoB.LocalPolicyVersion < infoA.LocalPolicyVersion {
		t.Fatalf("engine B version=%d, engine A version=%d", infoB.LocalPolicyVersion, infoA.LocalPolicyVersion)
	}
}

func TestClusteredPolicyWritesDoNotLoseIndependentUpdates(t *testing.T) {
	db, redisClient := openClusterIntegrationDependencies(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	channel := fmt.Sprintf("integration:authz:concurrent:%d", time.Now().UnixNano())
	engineA := newStartedClusterEngine(t, ctx, db, redisClient, "concurrent-a", channel, 50*time.Millisecond)
	engineB := newStartedClusterEngine(t, ctx, db, redisClient, "concurrent-b", channel, 50*time.Millisecond)
	defer engineA.Close()
	defer engineB.Close()

	roleA := fmt.Sprintf("integration_concurrent_a_%d", time.Now().UnixNano())
	roleB := fmt.Sprintf("integration_concurrent_b_%d", time.Now().UnixNano())
	defer cleanupClusterPolicy(engineA, "", roleA)
	defer cleanupClusterPolicy(engineA, "", roleB)

	var wg sync.WaitGroup
	errs := make(chan error, 2)
	wg.Add(2)
	go func() {
		defer wg.Done()
		_, err := engineA.AddPermissionForRole(ctx, roleA, "/integration/a", "GET")
		errs <- err
	}()
	go func() {
		defer wg.Done()
		_, err := engineB.AddPermissionForRole(ctx, roleB, "/integration/b", "GET")
		errs <- err
	}()
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent policy update error = %v", err)
		}
	}

	waitForRolePermission(t, engineA, roleA, "/integration/a", "GET")
	waitForRolePermission(t, engineA, roleB, "/integration/b", "GET")
	waitForRolePermission(t, engineB, roleA, "/integration/a", "GET")
	waitForRolePermission(t, engineB, roleB, "/integration/b", "GET")
}

func openClusterIntegrationDependencies(t *testing.T) (*sql.DB, *redis.Client) {
	t.Helper()
	if os.Getenv("APP_API_INTEGRATION") != "1" {
		t.Skip("set APP_API_INTEGRATION=1 to run integration tests")
	}

	mysqlAddr := envOrDefault("APP_MYSQL_ADDR", "127.0.0.1:3306")
	mysqlUser := envOrDefault("APP_MYSQL_USER", "app_local")
	mysqlPassword := envOrDefault("APP_MYSQL_PASSWORD", "local-dev-only-mysql-password")
	mysqlDatabase := envOrDefault("APP_MYSQL_DATABASE", "awesome_zero_platform")
	mysqlConfig := mysql.Config{
		User:                 mysqlUser,
		Passwd:               mysqlPassword,
		Net:                  "tcp",
		Addr:                 mysqlAddr,
		DBName:               mysqlDatabase,
		ParseTime:            true,
		Loc:                  time.UTC,
		AllowNativePasswords: true,
	}
	dsn := mysqlConfig.FormatDSN()
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := db.Ping(); err != nil {
		t.Fatalf("mysql ping error = %v", err)
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr:     envOrDefault("APP_REDIS_ADDR", "127.0.0.1:6379"),
		Username: os.Getenv("APP_REDIS_USERNAME"),
		Password: envOrDefault("APP_REDIS_PASSWORD", "local-dev-only-redis-password"),
	})
	t.Cleanup(func() { _ = redisClient.Close() })
	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		t.Fatalf("redis ping error = %v", err)
	}
	return db, redisClient
}

func newStartedClusterEngine(t *testing.T, ctx context.Context, db *sql.DB, redisClient *redis.Client, instanceID, channel string, poll time.Duration) *Engine {
	t.Helper()
	engine, err := NewClustered(db, redisClient, ClusterConfig{
		Enabled:        true,
		InstanceID:     instanceID,
		Channel:        channel,
		PollInterval:   poll,
		PublishTimeout: time.Second,
		ReloadTimeout:  2 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewClustered() error = %v", err)
	}
	if err := engine.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	return engine
}

func waitForAuthorization(t *testing.T, engine *Engine, userID, resource, action string, want bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		allowed, err := engine.Enforce(context.Background(), userID, resource, action)
		if err == nil && allowed == want {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	allowed, err := engine.Enforce(context.Background(), userID, resource, action)
	t.Fatalf("Enforce() allowed=%v error=%v, want %v", allowed, err, want)
}

func waitForRolePermission(t *testing.T, engine *Engine, role, resource, action string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		permissions, err := engine.PermissionsForRole(context.Background(), role)
		if err == nil {
			for _, permission := range permissions {
				if permission.Resource == resource && permission.Action == action {
					return
				}
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("permission %s %s not observed for role %s", resource, action, role)
}

func cleanupClusterPolicy(engine *Engine, userID, role string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if userID != "" {
		_, _ = engine.DeleteUser(ctx, userID)
	}
	if role != "" {
		_, _ = engine.DeleteRole(ctx, role)
	}
}

func envOrDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
