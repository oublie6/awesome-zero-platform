//go:build integration

package redissession

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/oublie6/awesome-zero-platform/server/foundation/cache"
	"github.com/oublie6/awesome-zero-platform/server/platform/authn"
)

// TestGoZeroClientSessionRoundTrip exercises go-zero Redis commands and the
// native WATCH transaction against the same Redis DB and key namespace.
func TestGoZeroClientSessionRoundTrip(t *testing.T) {
	if os.Getenv("APP_API_INTEGRATION") != "1" {
		t.Skip("set APP_API_INTEGRATION=1 to run integration tests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	cfg := cache.Config{
		Addr:     os.Getenv("APP_REDIS_ADDR"),
		Username: os.Getenv("APP_REDIS_USERNAME"),
		Password: os.Getenv("APP_REDIS_PASSWORD"),
		DB:       0, PoolSize: 5,
		DialTimeout:      2 * time.Second,
		ReadTimeout:      2 * time.Second,
		WriteTimeout:     2 * time.Second,
		StartupTimeout:   3 * time.Second,
		ReadinessTimeout: 2 * time.Second,
	}
	if cfg.Addr == "" {
		cfg.Addr = "127.0.0.1:6379"
	}
	if cfg.Password == "" {
		cfg.Password = "local-dev-only-redis-password"
	}
	resource, err := cache.Open(ctx, cfg)
	if err != nil {
		t.Fatalf("open Redis clients: %v", err)
	}
	defer resource.Close()
	if resource.GoZero() == nil {
		t.Fatal("expected go-zero Redis client on DB 0")
	}

	prefix := fmt.Sprintf("integration:gozero:session:%d:", time.Now().UnixNano())
	store, err := New(resource.Client(), prefix, resource.GoZero())
	if err != nil {
		t.Fatalf("construct session store: %v", err)
	}
	now := time.Now().UTC()
	session := authn.Session{
		ID: "session-a", AccountID: "user-a", RefreshDigest: "digest-before",
		CreatedAt: now, ExpiresAt: now.Add(35*time.Second + 750*time.Millisecond),
	}
	// Run cleanup before the deferred resource.Close; otherwise the Redis
	// client is already closed when the test tries to remove its session.
	defer func() {
		_ = resource.Client().Del(context.Background(), prefix+session.ID).Err()
	}()
	if err := store.Create(ctx, session); err != nil {
		t.Fatalf("go-zero create session: %v", err)
	}
	if err := store.Create(ctx, session); !errors.Is(err, authn.ErrSessionConflict) {
		t.Fatalf("duplicate session error=%v, want ErrSessionConflict", err)
	}
	remaining, err := resource.Client().PTTL(ctx, prefix+session.ID).Result()
	if err != nil || remaining <= 0 || remaining > 36*time.Second {
		t.Fatalf("go-zero session PTTL=%v error=%v", remaining, err)
	}
	got, err := store.Get(ctx, session.ID)
	if err != nil || got.RefreshDigest != session.RefreshDigest {
		t.Fatalf("go-zero session get=%+v error=%v", got, err)
	}
    // Optimized realtime path reads the session and epoch in a single
    // pipeline and must enforce the exact trusted AccountID.
    optimized,err:=store.GetForAccount(ctx,session.ID,session.AccountID)
    if err!=nil || optimized.ID!=session.ID || optimized.RevocationEpoch!=0{
        t.Fatalf("optimized realtime get=%+v error=%v",optimized,err)
    }
    if _,err:=store.GetForAccount(ctx,session.ID,"other-account");!errors.Is(err,authn.ErrSessionNotFound){
        t.Fatalf("another account used this session: %v",err)
    }

	views, err := store.ListByAccount(ctx, session.AccountID)
	if err != nil || len(views) != 1 || views[0].ID != session.ID {
		t.Fatalf("go-zero session list=%+v error=%v", views, err)
	}
	newExpiry := time.Now().UTC().Add(2 * time.Minute)
	rotated, err := store.Rotate(ctx, session.ID, session.RefreshDigest, "digest-after", newExpiry)
	if err != nil || rotated.RefreshDigest != "digest-after" {
		t.Fatalf("native WATCH rotate=%+v error=%v", rotated, err)
	}
	if _, err := store.Rotate(ctx, session.ID, session.RefreshDigest, "stale", newExpiry); !errors.Is(err, authn.ErrInvalidRefresh) {
		t.Fatalf("stale refresh error=%v, want ErrInvalidRefresh", err)
	}
	got, err = store.Get(ctx, session.ID)
	if err != nil || got.RefreshDigest != "digest-after" {
		t.Fatalf("go-zero get after native WATCH=%+v error=%v", got, err)
	}
	staleBytes, err := resource.Client().Get(ctx, prefix+session.ID).Bytes()
	if err != nil {
		t.Fatal(err)
	}
	revoked, err := store.RevokeByAccount(ctx, session.AccountID)
	if err != nil || revoked != 1 {
		t.Fatalf("go-zero revoke by account=%d error=%v", revoked, err)
	}
	if _, err := store.Get(ctx, session.ID); !errors.Is(err, authn.ErrSessionNotFound) {
		t.Fatalf("get revoked session error=%v, want ErrSessionNotFound", err)
	}
	// Simulate the Redis SCAN missing a stale session key.
	if err := resource.Client().Set(ctx, prefix+session.ID, staleBytes, time.Minute).Err(); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(ctx, session.ID); !errors.Is(err, authn.ErrSessionNotFound) {
		t.Fatalf("stale session revived after account-wide revoke: %v", err)
	}
    if _,err:=store.GetForAccount(ctx,session.ID,session.AccountID);!errors.Is(err,authn.ErrSessionNotFound){
        t.Fatalf("pipelined realtime read revived stale epoch: %v",err)
    }

	if _, err := store.Rotate(ctx, session.ID, "digest-after", "digest-raced", time.Now().Add(time.Minute)); !errors.Is(err, authn.ErrInvalidRefresh) {
		t.Fatalf("refresh revived after account-wide revoke: %v", err)
	}
	_ = resource.Client().Del(ctx, prefix+session.ID).Err()
	if err := store.Revoke(ctx, session.ID); !errors.Is(err, authn.ErrSessionNotFound) {
		t.Fatalf("revoke missing session error=%v, want ErrSessionNotFound", err)
	}
}
