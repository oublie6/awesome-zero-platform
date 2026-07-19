//go:build integration

package bootstrap

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"
)

func TestIntegrationDependenciesAndHealth(t *testing.T) {
	if os.Getenv("APP_API_INTEGRATION") != "1" {
		t.Skip("set APP_API_INTEGRATION=1 to run integration tests")
	}

	port := reservePort(t)
	configPath := writeConfig(t, runtimeConfig(port, false, true))

	app, err := New(configPath)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer app.Stop()

	done := make(chan struct{})
	go func() {
		defer close(done)
		app.Start()
	}()

	baseURL := "http://127.0.0.1:" + strconv.Itoa(port)
	waitForHealthy(t, baseURL+"/health/live")

	live := doRequest(t, http.MethodGet, baseURL+"/health/live", "", nil)
	assertHealthStatus(t, live, http.StatusOK, "ok")

	ready := doRequest(t, http.MethodGet, baseURL+"/health/ready", "", nil)
	assertHealthStatus(t, ready, http.StatusOK, "ready")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var schemaVersion string
	if err := app.mysql.DB().QueryRowContext(ctx, "SELECT meta_value FROM foundation_schema_meta WHERE meta_key = ?", "schema_version").Scan(&schemaVersion); err != nil {
		t.Fatalf("query schema version: %v", err)
	}
	if schemaVersion != "0004" {
		t.Fatalf("schema version = %q, want 0004", schemaVersion)
	}

	var seedState string
	if err := app.mysql.DB().QueryRowContext(ctx, "SELECT meta_value FROM foundation_schema_meta WHERE meta_key = ?", "seed_state").Scan(&seedState); err != nil {
		t.Fatalf("query seed state: %v", err)
	}
	if seedState != "development" {
		t.Fatalf("seed state = %q, want development", seedState)
	}

	txKey := fmt.Sprintf("integration_tx_%d", time.Now().UnixNano())
	if err := app.mysql.WithinTransaction(ctx, func(txCtx context.Context, tx *sql.Tx) error {
		_, err := tx.ExecContext(txCtx, `
			INSERT INTO foundation_schema_meta (meta_key, meta_value)
			VALUES (?, ?)
			ON DUPLICATE KEY UPDATE
			    meta_value = VALUES(meta_value),
			    updated_at = CURRENT_TIMESTAMP(6)
		`, txKey, "ok")
		return err
	}); err != nil {
		t.Fatalf("WithinTransaction() error = %v", err)
	}
	defer func() {
		_, _ = app.mysql.DB().ExecContext(context.Background(), "DELETE FROM foundation_schema_meta WHERE meta_key = ?", txKey)
	}()

	var txValue string
	if err := app.mysql.DB().QueryRowContext(ctx, "SELECT meta_value FROM foundation_schema_meta WHERE meta_key = ?", txKey).Scan(&txValue); err != nil {
		t.Fatalf("query integration transaction row: %v", err)
	}
	if txValue != "ok" {
		t.Fatalf("transaction row value = %q, want ok", txValue)
	}

	redisKey := fmt.Sprintf("integration:redis:%d", time.Now().UnixNano())
	if err := app.redis.Client().Set(ctx, redisKey, "ok", time.Minute).Err(); err != nil {
		t.Fatalf("redis set: %v", err)
	}
	defer func() {
		_ = app.redis.Client().Del(context.Background(), redisKey).Err()
	}()

	redisValue, err := app.redis.Client().Get(ctx, redisKey).Result()
	if err != nil {
		t.Fatalf("redis get: %v", err)
	}
	if redisValue != "ok" {
		t.Fatalf("redis value = %q, want ok", redisValue)
	}

	if count, err := app.redis.Client().Exists(ctx, redisKey).Result(); err != nil {
		t.Fatalf("redis exists: %v", err)
	} else if count != 1 {
		t.Fatalf("redis exists count = %d, want 1", count)
	}

	select {
	case <-done:
		t.Fatal("server exited unexpectedly")
	default:
	}
}

func TestIntegrationReadinessPayloadStaysMinimal(t *testing.T) {
	if os.Getenv("APP_API_INTEGRATION") != "1" {
		t.Skip("set APP_API_INTEGRATION=1 to run integration tests")
	}

	port := reservePort(t)
	configPath := writeConfig(t, runtimeConfig(port, false, true))

	app, err := New(configPath)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer app.Stop()

	go app.Start()
	baseURL := "http://127.0.0.1:" + strconv.Itoa(port)
	waitForHealthy(t, baseURL+"/health/live")

	resp := doRequest(t, http.MethodGet, baseURL+"/health/ready", "", nil)
	if resp.status != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", resp.status, http.StatusOK, string(resp.body))
	}

	var payload map[string]string
	if err := json.Unmarshal(resp.body, &payload); err != nil {
		t.Fatalf("decode readiness payload: %v", err)
	}
	if len(payload) != 1 || payload["status"] != "ready" {
		t.Fatalf("unexpected readiness payload: %s", string(resp.body))
	}
}
