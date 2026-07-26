package casbinmysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/oublie6/awesome-zero-platform/server/platform/authz"
	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"
)

const globalPolicyKey = "global"

type ClusterConfig struct {
	Enabled        bool
	InstanceID     string
	Channel        string
	PollInterval   time.Duration
	PublishTimeout time.Duration
	ReloadTimeout  time.Duration
}

type policyChangeMessage struct {
	Version   uint64    `json:"version"`
	Source    string    `json:"source"`
	ChangedAt time.Time `json:"changedAt"`
}

type policyQueryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func NewClustered(db *sql.DB, redisClient *redis.Client, cfg ClusterConfig) (*Engine, error) {
	engine, err := New(db)
	if err != nil {
		return nil, err
	}
	if !cfg.Enabled {
		return engine, nil
	}
	if redisClient == nil {
		return nil, fmt.Errorf("clustered authorization redis client is required")
	}
	cfg.prepare()
	engine.clusterEnabled = true
	engine.cluster = cfg
	engine.redis = redisClient
	return engine, nil
}

func (c *ClusterConfig) prepare() {
	c.InstanceID = strings.TrimSpace(c.InstanceID)
	if c.InstanceID == "" {
		c.InstanceID = strings.TrimSpace(os.Getenv("HOSTNAME"))
	}
	if c.InstanceID == "" {
		c.InstanceID = "app-api"
	}
	if strings.TrimSpace(c.Channel) == "" {
		c.Channel = "awesome-zero-platform:authz:policy-changed"
	}
	if c.PollInterval <= 0 {
		c.PollInterval = 20 * time.Second
	}
	if c.PublishTimeout <= 0 {
		c.PublishTimeout = 2 * time.Second
	}
	if c.ReloadTimeout <= 0 {
		c.ReloadTimeout = 5 * time.Second
	}
}

// Start loads the durable version and starts Pub/Sub plus database-version
// reconciliation. Normal Enforce calls remain in-memory operations.
func (e *Engine) Start(ctx context.Context) error {
	if e == nil || !e.clusterEnabled {
		return nil
	}
	version, err := e.currentPolicyVersion(ctx)
	if err != nil {
		return fmt.Errorf("read initial authorization policy version: %w", err)
	}
	e.setVersions(version, version)
	e.setSyncSuccess(time.Now().UTC())

	runCtx, cancel := context.WithCancel(ctx)
	e.lifecycleMu.Lock()
	if e.cancel != nil {
		e.lifecycleMu.Unlock()
		cancel()
		return fmt.Errorf("authorization policy synchronizer already started")
	}
	e.cancel = cancel
	e.lifecycleMu.Unlock()

	e.wg.Add(2)
	go e.watchPolicyChanges(runCtx)
	go e.pollPolicyVersion(runCtx)
	return nil
}

func (e *Engine) Close() error {
	if e == nil {
		return nil
	}
	e.lifecycleMu.Lock()
	cancel := e.cancel
	e.cancel = nil
	e.lifecycleMu.Unlock()
	if cancel != nil {
		cancel()
	}
	e.wg.Wait()
	return nil
}

func (e *Engine) Name() string { return "authorization-policy" }

// Ping makes an out-of-date or failed policy synchronizer fail readiness while
// preserving the prior in-memory Enforcer until a reload succeeds.
func (e *Engine) Ping(ctx context.Context) error {
	if e == nil || !e.clusterEnabled {
		return nil
	}
	version, err := e.currentPolicyVersion(ctx)
	if err != nil {
		return err
	}
	e.setDatabaseVersion(version)
	local, _, syncErr, _, _ := e.syncSnapshot()
	if version > local {
		return fmt.Errorf("authorization policy is stale: local=%d database=%d", local, version)
	}
	if syncErr != nil {
		// A successful authoritative version check proves that any previous
		// transient subscription error did not leave this Enforcer stale.
		e.setSyncSuccess(time.Now().UTC())
	}
	return nil
}

func (e *Engine) mutateRules(ctx context.Context, transform func([]authz.RawRule) ([]authz.RawRule, error)) (bool, error) {
	if transform == nil {
		return false, fmt.Errorf("authorization policy transform is required")
	}
	e.adminMu.Lock()
	defer e.adminMu.Unlock()

	if !e.clusterEnabled {
		current, err := e.snapshotRules()
		if err != nil {
			return false, err
		}
		next, err := transform(copyRules(current))
		if err != nil {
			return false, err
		}
		next = normalizeRawRules(next)
		if err := e.ValidateRawRules(ctx, next); err != nil {
			return false, err
		}
		if err := e.validatePolicySafety(current, next); err != nil {
			return false, err
		}
		if err := validateActiveSuperAdminSafety(ctx, e.db, next); err != nil {
			return false, err
		}
		if reflect.DeepEqual(normalizeRawRules(current), next) {
			return false, nil
		}
		if err := e.applyRawRules(next); err != nil {
			return false, err
		}
		e.setLoadedAt(time.Now().UTC())
		return true, nil
	}

	tx, err := e.db.BeginTx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("begin authorization policy transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	version, err := lockPolicyVersion(ctx, tx)
	if err != nil {
		return false, err
	}
	current, err := loadRawRules(ctx, tx)
	if err != nil {
		return false, err
	}
	next, err := transform(copyRules(current))
	if err != nil {
		return false, err
	}
	next = normalizeRawRules(next)
	if err := e.ValidateRawRules(ctx, next); err != nil {
		return false, err
	}
	if err := e.validatePolicySafety(current, next); err != nil {
		return false, err
	}
	if err := validateActiveSuperAdminSafety(ctx, tx, next); err != nil {
		return false, err
	}
	if reflect.DeepEqual(normalizeRawRules(current), next) {
		return false, nil
	}
	if err := replaceRawRulesTx(ctx, tx, next); err != nil {
		return false, err
	}
	nextVersion := version + 1
	if _, err := tx.ExecContext(ctx, `UPDATE authorization_policy_state
		SET version = ?, updated_at = CURRENT_TIMESTAMP(6)
		WHERE policy_key = ?`, nextVersion, globalPolicyKey); err != nil {
		return false, fmt.Errorf("advance authorization policy version: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("commit authorization policy transaction: %w", err)
	}

	if err := e.reloadCommittedVersion(ctx, nextVersion); err != nil {
		return true, err
	}
	e.publishPolicyChange(nextVersion)
	return true, nil
}

func lockPolicyVersion(ctx context.Context, tx *sql.Tx) (uint64, error) {
	var version uint64
	if err := tx.QueryRowContext(ctx, `SELECT version FROM authorization_policy_state
		WHERE policy_key = ? FOR UPDATE`, globalPolicyKey).Scan(&version); err != nil {
		return 0, fmt.Errorf("lock authorization policy version: %w", err)
	}
	return version, nil
}

func (e *Engine) currentPolicyVersion(ctx context.Context) (uint64, error) {
	var version uint64
	if err := e.db.QueryRowContext(ctx, `SELECT version FROM authorization_policy_state
		WHERE policy_key = ?`, globalPolicyKey).Scan(&version); err != nil {
		return 0, fmt.Errorf("read authorization policy version: %w", err)
	}
	return version, nil
}

func loadRawRules(ctx context.Context, queryer policyQueryer) ([]authz.RawRule, error) {
	rows, err := queryer.QueryContext(ctx, `SELECT ptype, v0, v1, v2, v3, v4, v5
		FROM authorization_casbin_rules ORDER BY rule_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	rules := make([]authz.RawRule, 0)
	for rows.Next() {
		values := make([]string, 6)
		var ptype string
		if err := rows.Scan(&ptype, &values[0], &values[1], &values[2], &values[3], &values[4], &values[5]); err != nil {
			return nil, err
		}
		for len(values) > 0 && values[len(values)-1] == "" {
			values = values[:len(values)-1]
		}
		rules = append(rules, authz.RawRule{PType: ptype, Values: values})
	}
	return rules, rows.Err()
}

func replaceRawRulesTx(ctx context.Context, tx *sql.Tx, rules []authz.RawRule) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM authorization_casbin_rules`); err != nil {
		return fmt.Errorf("clear authorization policies: %w", err)
	}
	for _, rule := range rules {
		values := padded(trimmedValues(rule.Values))
		ruleHash, err := fingerprint(strings.TrimSpace(rule.PType), values)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO authorization_casbin_rules
			(rule_hash, ptype, v0, v1, v2, v3, v4, v5)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			ruleHash, strings.TrimSpace(rule.PType), values[0], values[1], values[2], values[3], values[4], values[5]); err != nil {
			return fmt.Errorf("insert authorization policy: %w", err)
		}
	}
	return nil
}

func (e *Engine) reloadCommittedVersion(parent context.Context, version uint64) error {
	ctx, cancel := context.WithTimeout(parent, e.cluster.ReloadTimeout)
	defer cancel()
	if err := e.enforcer.LoadPolicy(); err != nil {
		err = fmt.Errorf("reload committed authorization policy version %d: %w", version, err)
		e.setSyncError(err)
		return err
	}
	now := time.Now().UTC()
	e.setVersions(version, version)
	e.setLoadedAt(now)
	e.setSyncSuccess(now)
	return nil
}

func (e *Engine) reconcilePolicyVersion(parent context.Context, hintedVersion uint64) {
	ctx, cancel := context.WithTimeout(parent, e.cluster.ReloadTimeout)
	defer cancel()

	version, err := e.currentPolicyVersion(ctx)
	if err != nil {
		e.setSyncError(err)
		return
	}
	// The notification is only a wake-up hint. MySQL is the source of truth;
	// never advance local state to a version observed only in Pub/Sub payloads.
	if hintedVersion > version {
		logx.WithContext(ctx).Infof("authorization policy notification is ahead of database: hinted=%d database=%d", hintedVersion, version)
	}
	e.setDatabaseVersion(version)
	local, _, _, _, _ := e.syncSnapshot()
	if version <= local {
		e.setSyncSuccess(time.Now().UTC())
		return
	}

	e.adminMu.Lock()
	defer e.adminMu.Unlock()
	local, _, _, _, _ = e.syncSnapshot()
	if version <= local {
		e.setSyncSuccess(time.Now().UTC())
		return
	}
	if err := e.enforcer.LoadPolicy(); err != nil {
		e.setSyncError(fmt.Errorf("reload authorization policy version %d: %w", version, err))
		return
	}
	now := time.Now().UTC()
	e.setVersions(version, version)
	e.setLoadedAt(now)
	e.setSyncSuccess(now)
}

func (e *Engine) watchPolicyChanges(ctx context.Context) {
	defer e.wg.Done()
	for ctx.Err() == nil {
		pubsub := e.redis.Subscribe(ctx, e.cluster.Channel)
		if _, err := pubsub.Receive(ctx); err != nil {
			_ = pubsub.Close()
			e.setWatcherConnected(false)
			if !errors.Is(err, context.Canceled) {
				logx.WithContext(ctx).Errorf("subscribe authorization policy changes: %v", err)
			}
			if !sleepContext(ctx, time.Second) {
				return
			}
			continue
		}
		e.setWatcherConnected(true)
		e.reconcilePolicyVersion(ctx, 0)

		for ctx.Err() == nil {
			message, err := pubsub.ReceiveMessage(ctx)
			if err != nil {
				_ = pubsub.Close()
				e.setWatcherConnected(false)
				if !errors.Is(err, context.Canceled) {
					logx.WithContext(ctx).Errorf("receive authorization policy change: %v", err)
				}
				break
			}
			var change policyChangeMessage
			if err := json.Unmarshal([]byte(message.Payload), &change); err != nil {
				logx.WithContext(ctx).Errorf("ignore invalid authorization policy notification: %v", err)
				continue
			}
			local, _, _, _, _ := e.syncSnapshot()
			if change.Version <= local || change.Source == e.cluster.InstanceID {
				continue
			}
			e.reconcilePolicyVersion(ctx, change.Version)
		}
	}
}

func (e *Engine) pollPolicyVersion(ctx context.Context) {
	defer e.wg.Done()
	ticker := time.NewTicker(e.cluster.PollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			e.reconcilePolicyVersion(ctx, 0)
		}
	}
}

func (e *Engine) publishPolicyChange(version uint64) {
	if !e.clusterEnabled || e.redis == nil {
		return
	}
	payload, err := json.Marshal(policyChangeMessage{
		Version:   version,
		Source:    e.cluster.InstanceID,
		ChangedAt: time.Now().UTC(),
	})
	if err != nil {
		logx.Errorf("encode authorization policy notification: %v", err)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), e.cluster.PublishTimeout)
	defer cancel()
	if err := e.redis.Publish(ctx, e.cluster.Channel, payload).Err(); err != nil {
		logx.Errorf("publish authorization policy notification: %v", err)
	}
}

func normalizeRawRules(rules []authz.RawRule) []authz.RawRule {
	result := copyRules(rules)
	for index := range result {
		result[index].PType = strings.TrimSpace(result[index].PType)
		result[index].Values = trimmedValues(result[index].Values)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].PType == result[j].PType {
			return strings.Join(result[i].Values, "\x00") < strings.Join(result[j].Values, "\x00")
		}
		return result[i].PType < result[j].PType
	})
	return result
}

func copyRules(rules []authz.RawRule) []authz.RawRule {
	result := make([]authz.RawRule, len(rules))
	for index, rule := range rules {
		result[index] = authz.RawRule{PType: rule.PType, Values: append([]string(nil), rule.Values...)}
	}
	return result
}

func sleepContext(ctx context.Context, duration time.Duration) bool {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
