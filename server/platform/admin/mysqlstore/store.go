package mysqlstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	mysql "github.com/go-sql-driver/mysql"
	foundationcache "github.com/oublie6/awesome-zero-platform/server/foundation/cache"
	foundationdb "github.com/oublie6/awesome-zero-platform/server/foundation/database"
	"github.com/oublie6/awesome-zero-platform/server/platform/admin"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type Store struct {
	db    *sql.DB
	conn  sqlx.SqlConn
	cache *foundationcache.ModelCache
}

func New(db *sql.DB) (*Store, error) {
	return NewCached(db, nil)
}

func NewCached(db *sql.DB, modelCache *foundationcache.ModelCache) (*Store, error) {
	if db == nil {
		return nil, fmt.Errorf("mysql database is required")
	}
	return &Store{db: db, conn: foundationdb.WrapSqlConn(db), cache: modelCache}, nil
}

type roleRow struct {
	Code        string    `db:"role_code"`
	DisplayName string    `db:"display_name"`
	Description string    `db:"description"`
	System      bool      `db:"is_system"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
}

func (row roleRow) role() admin.Role {
	return admin.Role{
		Code: row.Code, DisplayName: row.DisplayName, Description: row.Description,
		System: row.System, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}

type resourceRow struct {
	Code        string    `db:"resource_code"`
	DisplayName string    `db:"display_name"`
	Module      string    `db:"module_name"`
	Pattern     string    `db:"resource_pattern"`
	Actions     []byte    `db:"actions_json"`
	Description string    `db:"description"`
	System      bool      `db:"is_system"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
}

func (row resourceRow) resource() (admin.Resource, error) {
	var actions []string
	if err := json.Unmarshal(row.Actions, &actions); err != nil {
		return admin.Resource{}, err
	}
	return admin.Resource{
		Code: row.Code, DisplayName: row.DisplayName, Module: row.Module,
		Pattern: row.Pattern, Actions: actions, Description: row.Description,
		System: row.System, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}, nil
}

type auditRow struct {
	ID           string    `db:"event_id"`
	ActorID      string    `db:"actor_account_id"`
	Action       string    `db:"action_name"`
	ResourceType string    `db:"resource_type"`
	ResourceID   string    `db:"resource_id"`
	Outcome      string    `db:"outcome"`
	RequestID    string    `db:"request_id"`
	ClientIP     string    `db:"client_ip"`
	UserAgent    string    `db:"user_agent"`
	Details      []byte    `db:"details_json"`
	CreatedAt    time.Time `db:"created_at"`
}

func (row auditRow) event() admin.AuditEvent {
	event := admin.AuditEvent{
		ID: row.ID, ActorID: row.ActorID, Action: row.Action,
		ResourceType: row.ResourceType, ResourceID: row.ResourceID,
		Outcome: row.Outcome, RequestID: row.RequestID,
		ClientIP: row.ClientIP, UserAgent: row.UserAgent, CreatedAt: row.CreatedAt,
	}
	if len(row.Details) > 0 {
		_ = json.Unmarshal(row.Details, &event.Details)
	}
	return event
}

func (s *Store) ListRoles(ctx context.Context) ([]admin.Role, error) {
	var records []roleRow
	if err := s.conn.QueryRowsCtx(ctx, &records, `SELECT role_code, display_name, description, is_system, created_at, updated_at
        FROM authorization_roles ORDER BY is_system DESC, role_code`); err != nil {
		return nil, err
	}
	roles := make([]admin.Role, 0, len(records))
	for _, record := range records {
		roles = append(roles, record.role())
	}
	return roles, nil
}

func (s *Store) GetRole(ctx context.Context, code string) (admin.Role, error) {
	code = strings.TrimSpace(code)
	if s.cache == nil {
		return s.GetRoleFresh(ctx, code)
	}
	var role admin.Role
	err := s.cache.TakeCtx(ctx, &role, s.roleKey(code), func(value any) error {
		loaded, err := s.GetRoleFresh(ctx, code)
		if err != nil {
			return err
		}
		*value.(*admin.Role) = loaded
		return nil
	})
	return role, err
}

func (s *Store) GetRoleFresh(ctx context.Context, code string) (admin.Role, error) {
	var record roleRow
	err := s.conn.QueryRowCtx(ctx, &record, `SELECT role_code, display_name, description, is_system, created_at, updated_at
        FROM authorization_roles WHERE role_code = ?`, strings.TrimSpace(code))
	if errors.Is(err, sql.ErrNoRows) {
		return admin.Role{}, admin.ErrNotFound
	}
	if err != nil {
		return admin.Role{}, err
	}
	return record.role(), nil
}

func (s *Store) CreateRole(ctx context.Context, role admin.Role) (admin.Role, error) {
	role.Code = strings.TrimSpace(role.Code)
	role.DisplayName = strings.TrimSpace(role.DisplayName)
	role.Description = strings.TrimSpace(role.Description)
	if role.Code == "" || role.DisplayName == "" {
		return admin.Role{}, fmt.Errorf("role code and display name are required")
	}
	_, err := s.conn.ExecCtx(ctx, `INSERT INTO authorization_roles
		(role_code, display_name, description, is_system) VALUES (?, ?, ?, ?)`,
		role.Code, role.DisplayName, role.Description, role.System,
	)
	if err != nil {
		return admin.Role{}, translate(err)
	}
	if err := s.invalidate(ctx, s.roleKey(role.Code)); err != nil {
		return admin.Role{}, err
	}
	return s.GetRoleFresh(ctx, role.Code)
}

func (s *Store) UpdateRole(ctx context.Context, role admin.Role) (admin.Role, error) {
	role.Code = strings.TrimSpace(role.Code)
	role.DisplayName = strings.TrimSpace(role.DisplayName)
	role.Description = strings.TrimSpace(role.Description)
	if role.Code == "" || role.DisplayName == "" {
		return admin.Role{}, fmt.Errorf("role code and display name are required")
	}
	result, err := s.conn.ExecCtx(ctx, `UPDATE authorization_roles
		SET display_name = ?, description = ?, updated_at = CURRENT_TIMESTAMP(6)
		WHERE role_code = ?`, role.DisplayName, role.Description, role.Code)
	if err != nil {
		return admin.Role{}, translate(err)
	}
	if err := requireAffected(result); err != nil {
		return admin.Role{}, err
	}
	if err := s.invalidate(ctx, s.roleKey(role.Code)); err != nil {
		return admin.Role{}, err
	}
	return s.GetRoleFresh(ctx, role.Code)
}

func (s *Store) DeleteRole(ctx context.Context, code string) error {
	code = strings.TrimSpace(code)
	result, err := s.conn.ExecCtx(ctx, `DELETE FROM authorization_roles WHERE role_code = ? AND is_system = 0`, code)
	if err != nil {
		return err
	}
	if err := requireAffected(result); err != nil {
		return err
	}
	return s.invalidate(ctx, s.roleKey(code))
}

func (s *Store) ListResources(ctx context.Context) ([]admin.Resource, error) {
	var records []resourceRow
	if err := s.conn.QueryRowsCtx(ctx, &records, `SELECT resource_code, display_name, module_name, resource_pattern,
        actions_json, description, is_system, created_at, updated_at
        FROM authorization_resources ORDER BY module_name, resource_code`); err != nil {
		return nil, err
	}
	resources := make([]admin.Resource, 0, len(records))
	for _, record := range records {
		resource, err := record.resource()
		if err != nil {
			return nil, err
		}
		resources = append(resources, resource)
	}
	return resources, nil
}

func (s *Store) GetResource(ctx context.Context, code string) (admin.Resource, error) {
	code = strings.TrimSpace(code)
	if s.cache == nil {
		return s.GetResourceFresh(ctx, code)
	}
	var resource admin.Resource
	err := s.cache.TakeCtx(ctx, &resource, s.resourceKey(code), func(value any) error {
		loaded, err := s.GetResourceFresh(ctx, code)
		if err != nil {
			return err
		}
		*value.(*admin.Resource) = loaded
		return nil
	})
	return resource, err
}

func (s *Store) GetResourceFresh(ctx context.Context, code string) (admin.Resource, error) {
	var record resourceRow
	err := s.conn.QueryRowCtx(ctx, &record, `SELECT resource_code, display_name, module_name, resource_pattern,
        actions_json, description, is_system, created_at, updated_at
        FROM authorization_resources WHERE resource_code = ?`, strings.TrimSpace(code))
	if errors.Is(err, sql.ErrNoRows) {
		return admin.Resource{}, admin.ErrNotFound
	}
	if err != nil {
		return admin.Resource{}, err
	}
	return record.resource()
}

func (s *Store) CreateResource(ctx context.Context, resource admin.Resource) (admin.Resource, error) {
	if err := validateResource(&resource); err != nil {
		return admin.Resource{}, err
	}
	actions, _ := json.Marshal(resource.Actions)
	_, err := s.conn.ExecCtx(ctx, `INSERT INTO authorization_resources
		(resource_code, display_name, module_name, resource_pattern, actions_json, description, is_system)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		resource.Code, resource.DisplayName, resource.Module, resource.Pattern, actions, resource.Description, resource.System,
	)
	if err != nil {
		return admin.Resource{}, translate(err)
	}
	if err := s.invalidate(ctx, s.resourceKey(resource.Code)); err != nil {
		return admin.Resource{}, err
	}
	return s.GetResourceFresh(ctx, resource.Code)
}

func (s *Store) UpdateResource(ctx context.Context, resource admin.Resource) (admin.Resource, error) {
	if err := validateResource(&resource); err != nil {
		return admin.Resource{}, err
	}
	actions, _ := json.Marshal(resource.Actions)
	result, err := s.conn.ExecCtx(ctx, `UPDATE authorization_resources
		SET display_name = ?, module_name = ?, resource_pattern = ?, actions_json = ?, description = ?, updated_at = CURRENT_TIMESTAMP(6)
		WHERE resource_code = ?`,
		resource.DisplayName, resource.Module, resource.Pattern, actions, resource.Description, resource.Code,
	)
	if err != nil {
		return admin.Resource{}, translate(err)
	}
	if err := requireAffected(result); err != nil {
		return admin.Resource{}, err
	}
	if err := s.invalidate(ctx, s.resourceKey(resource.Code)); err != nil {
		return admin.Resource{}, err
	}
	return s.GetResourceFresh(ctx, resource.Code)
}

func (s *Store) DeleteResource(ctx context.Context, code string) error {
	code = strings.TrimSpace(code)
	result, err := s.conn.ExecCtx(ctx, `DELETE FROM authorization_resources WHERE resource_code = ? AND is_system = 0`, code)
	if err != nil {
		return err
	}
	if err := requireAffected(result); err != nil {
		return err
	}
	return s.invalidate(ctx, s.resourceKey(code))
}

func (s *Store) AppendAudit(ctx context.Context, event admin.AuditEvent) error {
	details, err := json.Marshal(event.Details)
	if err != nil {
		return err
	}
	_, err = s.conn.ExecCtx(ctx, `INSERT INTO platform_audit_events
		(event_id, actor_account_id, action_name, resource_type, resource_id, outcome,
		 request_id, client_ip, user_agent, details_json, created_at)
		VALUES (?, NULLIF(?, ''), ?, ?, NULLIF(?, ''), ?, NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), ?, ?)`,
		event.ID, event.ActorID, event.Action, event.ResourceType, event.ResourceID, event.Outcome,
		event.RequestID, event.ClientIP, event.UserAgent, details, event.CreatedAt.UTC(),
	)
	return err
}

func (s *Store) ListAudit(ctx context.Context, query admin.AuditQuery) (admin.AuditPage, error) {
	where := make([]string, 0, 3)
	args := make([]any, 0, 8)
	if search := strings.TrimSpace(query.Search); search != "" {
		pattern := "%" + search + "%"
		where = append(where, `(actor_account_id LIKE ? OR resource_id LIKE ? OR request_id LIKE ?)`)
		args = append(args, pattern, pattern, pattern)
	}
	if action := strings.TrimSpace(query.Action); action != "" {
		where = append(where, `action_name = ?`)
		args = append(args, action)
	}
	if outcome := strings.TrimSpace(query.Outcome); outcome != "" {
		where = append(where, `outcome = ?`)
		args = append(args, outcome)
	}
	whereSQL := ""
	if len(where) > 0 {
		whereSQL = " WHERE " + strings.Join(where, " AND ")
	}
	var total int64
	if err := s.conn.QueryRowCtx(ctx, &total, `SELECT COUNT(*) FROM platform_audit_events`+whereSQL, args...); err != nil {
		return admin.AuditPage{}, err
	}
	listArgs := append([]any(nil), args...)
	listArgs = append(listArgs, query.PageSize, (query.Page-1)*query.PageSize)
	var records []auditRow
	if err := s.conn.QueryRowsCtx(ctx, &records, `SELECT event_id, COALESCE(actor_account_id, ''), action_name,
        resource_type, COALESCE(resource_id, ''), outcome, COALESCE(request_id, ''),
        COALESCE(client_ip, ''), COALESCE(user_agent, ''), details_json, created_at
        FROM platform_audit_events`+whereSQL+`
        ORDER BY created_at DESC, event_id DESC LIMIT ? OFFSET ?`, listArgs...); err != nil {
		return admin.AuditPage{}, err
	}
	items := make([]admin.AuditEvent, 0, len(records))
	for _, record := range records {
		items = append(items, record.event())
	}
	return admin.AuditPage{Items: items, Total: total, Page: query.Page, PageSize: query.PageSize}, nil
}

type scanner interface {
	Scan(...any) error
}

func scanRole(row scanner) (admin.Role, error) {
	var role admin.Role
	err := row.Scan(&role.Code, &role.DisplayName, &role.Description, &role.System, &role.CreatedAt, &role.UpdatedAt)
	return role, err
}

func scanResource(row scanner) (admin.Resource, error) {
	var resource admin.Resource
	var actions []byte
	if err := row.Scan(&resource.Code, &resource.DisplayName, &resource.Module, &resource.Pattern,
		&actions, &resource.Description, &resource.System, &resource.CreatedAt, &resource.UpdatedAt); err != nil {
		return admin.Resource{}, err
	}
	if err := json.Unmarshal(actions, &resource.Actions); err != nil {
		return admin.Resource{}, err
	}
	return resource, nil
}

func validateResource(resource *admin.Resource) error {
	resource.Code = strings.TrimSpace(resource.Code)
	resource.DisplayName = strings.TrimSpace(resource.DisplayName)
	resource.Module = strings.TrimSpace(resource.Module)
	resource.Pattern = strings.TrimSpace(resource.Pattern)
	resource.Description = strings.TrimSpace(resource.Description)
	if resource.Code == "" || resource.DisplayName == "" || resource.Module == "" || resource.Pattern == "" {
		return fmt.Errorf("resource code, display name, module, and pattern are required")
	}
	if len(resource.Actions) == 0 {
		return fmt.Errorf("resource actions are required")
	}
	seen := map[string]struct{}{}
	actions := resource.Actions[:0]
	for _, action := range resource.Actions {
		action = strings.TrimSpace(action)
		if action == "" {
			return fmt.Errorf("resource action must not be empty")
		}
		if _, exists := seen[action]; exists {
			continue
		}
		seen[action] = struct{}{}
		actions = append(actions, action)
	}
	resource.Actions = actions
	return nil
}

func (s *Store) roleKey(code string) string {
	if s.cache == nil {
		return ""
	}
	return s.cache.Key("role", code)
}

func (s *Store) resourceKey(code string) string {
	if s.cache == nil {
		return ""
	}
	return s.cache.Key("resource", code)
}

func (s *Store) invalidate(ctx context.Context, keys ...string) error {
	if s.cache == nil {
		return nil
	}
	return s.cache.DelCtx(ctx, keys...)
}

func requireAffected(result sql.Result) error {
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count != 1 {
		return admin.ErrNotFound
	}
	return nil
}

func translate(err error) error {
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
		return fmt.Errorf("%w: duplicate value", admin.ErrConflict)
	}
	return err
}

var _ admin.Repository = (*Store)(nil)
