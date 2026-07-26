package casbinmysql

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"github.com/casbin/casbin/v2/persist"
	"github.com/oublie6/awesome-zero-platform/server/platform/authz"
	"github.com/redis/go-redis/v9"
)

const modelText = `[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[role_definition]
g = _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub) && keyMatch2(r.obj, p.obj) && regexMatch(r.act, p.act)
`

type Engine struct {
	enforcer *casbin.SyncedEnforcer
	db       *sql.DB
	adapter  *mysqlAdapter

	adminMu sync.Mutex

	stateMu          sync.RWMutex
	loadedAt         time.Time
	localVersion     uint64
	databaseVersion  uint64
	lastSyncAt       time.Time
	lastSyncErr      error
	watcherConnected bool

	clusterEnabled bool
	cluster        ClusterConfig
	redis          *redis.Client
	lifecycleMu    sync.Mutex
	cancel         context.CancelFunc
	wg             sync.WaitGroup
}

func New(db *sql.DB) (*Engine, error) {
	if db == nil {
		return nil, fmt.Errorf("mysql database is required")
	}
	adapter := &mysqlAdapter{db: db}
	m, err := model.NewModelFromString(modelText)
	if err != nil {
		return nil, fmt.Errorf("create casbin model: %w", err)
	}
	enforcer, err := casbin.NewSyncedEnforcer(m, adapter)
	if err != nil {
		return nil, fmt.Errorf("create casbin enforcer: %w", err)
	}
	return &Engine{
		enforcer: enforcer,
		db:       db,
		adapter:  adapter,
		loadedAt: time.Now().UTC(),
	}, nil
}

func (e *Engine) Enforce(_ context.Context, subject, resource, action string) (bool, error) {
	return e.enforcer.Enforce(subject, resource, action)
}

func (e *Engine) AddRoleForUser(ctx context.Context, accountID, role string) (bool, error) {
	accountID = strings.TrimSpace(accountID)
	role = strings.TrimSpace(role)
	return e.mutateRules(ctx, func(rules []authz.RawRule) ([]authz.RawRule, error) {
		return addRule(rules, authz.RawRule{PType: "g", Values: []string{accountID, role}}), nil
	})
}

func (e *Engine) DeleteRoleForUser(ctx context.Context, accountID, role string) (bool, error) {
	accountID = strings.TrimSpace(accountID)
	role = strings.TrimSpace(role)
	return e.mutateRules(ctx, func(rules []authz.RawRule) ([]authz.RawRule, error) {
		return removeExactRule(rules, authz.RawRule{PType: "g", Values: []string{accountID, role}}), nil
	})
}

func (e *Engine) AddPermissionForRole(ctx context.Context, role, resource, action string) (bool, error) {
	role = strings.TrimSpace(role)
	resource = strings.TrimSpace(resource)
	action = strings.TrimSpace(action)
	return e.mutateRules(ctx, func(rules []authz.RawRule) ([]authz.RawRule, error) {
		return addRule(rules, authz.RawRule{PType: "p", Values: []string{role, resource, action}}), nil
	})
}

func (e *Engine) DeletePermissionForRole(ctx context.Context, role, resource, action string) (bool, error) {
	role = strings.TrimSpace(role)
	resource = strings.TrimSpace(resource)
	action = strings.TrimSpace(action)
	return e.mutateRules(ctx, func(rules []authz.RawRule) ([]authz.RawRule, error) {
		return removeExactRule(rules, authz.RawRule{PType: "p", Values: []string{role, resource, action}}), nil
	})
}

func (e *Engine) DeleteUser(ctx context.Context, accountID string) (bool, error) {
	accountID = strings.TrimSpace(accountID)
	return e.mutateRules(ctx, func(rules []authz.RawRule) ([]authz.RawRule, error) {
		filtered := rules[:0]
		for _, rule := range rules {
			values := trimmedValues(rule.Values)
			if len(values) > 0 && values[0] == accountID && (rule.PType == "p" || rule.PType == "g") {
				continue
			}
			filtered = append(filtered, rule)
		}
		return filtered, nil
	})
}

func (e *Engine) DeleteRole(ctx context.Context, role string) (bool, error) {
	role = strings.TrimSpace(role)
	return e.mutateRules(ctx, func(rules []authz.RawRule) ([]authz.RawRule, error) {
		filtered := rules[:0]
		for _, rule := range rules {
			values := trimmedValues(rule.Values)
			if rule.PType == "p" && len(values) > 0 && values[0] == role {
				continue
			}
			if rule.PType == "g" && len(values) > 1 && (values[0] == role || values[1] == role) {
				continue
			}
			filtered = append(filtered, rule)
		}
		return filtered, nil
	})
}

type mysqlAdapter struct {
	db *sql.DB
}

func (a *mysqlAdapter) LoadPolicy(m model.Model) error {
	rows, err := a.db.Query(`SELECT ptype, v0, v1, v2, v3, v4, v5
		FROM authorization_casbin_rules
		ORDER BY rule_id`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		values := make([]string, 6)
		var ptype string
		if err := rows.Scan(&ptype, &values[0], &values[1], &values[2], &values[3], &values[4], &values[5]); err != nil {
			return err
		}
		for len(values) > 0 && values[len(values)-1] == "" {
			values = values[:len(values)-1]
		}
		if err := persist.LoadPolicyLine(ptype+", "+strings.Join(values, ", "), m); err != nil {
			return fmt.Errorf("load casbin policy: %w", err)
		}
	}
	return rows.Err()
}

func (a *mysqlAdapter) SavePolicy(m model.Model) error {
	tx, err := a.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.Exec(`DELETE FROM authorization_casbin_rules`); err != nil {
		return err
	}
	for _, section := range []string{"p", "g"} {
		for ptype, assertion := range m[section] {
			for _, rule := range assertion.Policy {
				if err := insertRule(tx, ptype, rule); err != nil {
					return err
				}
			}
		}
	}
	return tx.Commit()
}

func (a *mysqlAdapter) AddPolicy(_ string, ptype string, rule []string) error {
	return insertRule(a.db, ptype, rule)
}

func (a *mysqlAdapter) RemovePolicy(_ string, ptype string, rule []string) error {
	values := padded(rule)
	_, err := a.db.Exec(`DELETE FROM authorization_casbin_rules
		WHERE ptype = ? AND v0 = ? AND v1 = ? AND v2 = ? AND v3 = ? AND v4 = ? AND v5 = ?`,
		ptype, values[0], values[1], values[2], values[3], values[4], values[5])
	return err
}

func (a *mysqlAdapter) RemoveFilteredPolicy(_ string, ptype string, fieldIndex int, fieldValues ...string) error {
	if fieldIndex < 0 || fieldIndex > 5 {
		return fmt.Errorf("casbin field index must be between 0 and 5")
	}
	query := `DELETE FROM authorization_casbin_rules WHERE ptype = ?`
	args := []any{ptype}
	for offset, value := range fieldValues {
		if value == "" {
			continue
		}
		column := fieldIndex + offset
		if column > 5 {
			return fmt.Errorf("casbin filtered policy exceeds v5")
		}
		query += fmt.Sprintf(" AND v%d = ?", column)
		args = append(args, value)
	}
	_, err := a.db.Exec(query, args...)
	return err
}

type execer interface {
	Exec(string, ...any) (sql.Result, error)
}

func insertRule(exec execer, ptype string, rule []string) error {
	values := padded(rule)
	ruleHash, err := fingerprint(ptype, values)
	if err != nil {
		return err
	}
	_, err = exec.Exec(`INSERT IGNORE INTO authorization_casbin_rules
		(rule_hash, ptype, v0, v1, v2, v3, v4, v5) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		ruleHash, ptype, values[0], values[1], values[2], values[3], values[4], values[5])
	return err
}

func fingerprint(ptype string, values [6]string) (string, error) {
	payload, err := json.Marshal([7]string{ptype, values[0], values[1], values[2], values[3], values[4], values[5]})
	if err != nil {
		return "", fmt.Errorf("encode casbin rule fingerprint: %w", err)
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

func padded(rule []string) [6]string {
	var values [6]string
	copy(values[:], rule)
	return values
}

func addRule(rules []authz.RawRule, candidate authz.RawRule) []authz.RawRule {
	candidate = normalizeRawRules([]authz.RawRule{candidate})[0]
	for _, rule := range normalizeRawRules(rules) {
		if rule.PType == candidate.PType && strings.Join(rule.Values, "\x00") == strings.Join(candidate.Values, "\x00") {
			return rules
		}
	}
	return append(rules, candidate)
}

func removeExactRule(rules []authz.RawRule, candidate authz.RawRule) []authz.RawRule {
	candidate = normalizeRawRules([]authz.RawRule{candidate})[0]
	filtered := rules[:0]
	for _, rule := range rules {
		normalized := normalizeRawRules([]authz.RawRule{rule})[0]
		if normalized.PType == candidate.PType && strings.Join(normalized.Values, "\x00") == strings.Join(candidate.Values, "\x00") {
			continue
		}
		filtered = append(filtered, rule)
	}
	return filtered
}

func (e *Engine) setLoadedAt(value time.Time) {
	e.stateMu.Lock()
	e.loadedAt = value.UTC()
	e.stateMu.Unlock()
}

func (e *Engine) setVersions(local, database uint64) {
	e.stateMu.Lock()
	e.localVersion = local
	e.databaseVersion = database
	e.stateMu.Unlock()
}

func (e *Engine) setDatabaseVersion(version uint64) {
	e.stateMu.Lock()
	e.databaseVersion = version
	e.stateMu.Unlock()
}

func (e *Engine) setSyncError(err error) {
	if err == nil {
		return
	}
	e.stateMu.Lock()
	e.lastSyncErr = err
	e.stateMu.Unlock()
}

func (e *Engine) setSyncSuccess(at time.Time) {
	e.stateMu.Lock()
	e.lastSyncErr = nil
	e.lastSyncAt = at.UTC()
	e.stateMu.Unlock()
}

func (e *Engine) setWatcherConnected(connected bool) {
	e.stateMu.Lock()
	e.watcherConnected = connected
	e.stateMu.Unlock()
}

func (e *Engine) syncSnapshot() (local, database uint64, syncErr error, lastSync time.Time, watcherConnected bool) {
	e.stateMu.RLock()
	defer e.stateMu.RUnlock()
	return e.localVersion, e.databaseVersion, e.lastSyncErr, e.lastSyncAt, e.watcherConnected
}

func (e *Engine) stateSnapshot() (loadedAt time.Time, local, database uint64, syncErr error, lastSync time.Time, watcherConnected bool) {
	e.stateMu.RLock()
	defer e.stateMu.RUnlock()
	return e.loadedAt, e.localVersion, e.databaseVersion, e.lastSyncErr, e.lastSyncAt, e.watcherConnected
}
