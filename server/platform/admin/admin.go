package admin

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/oublie6/awesome-zero-platform/server/platform/authn"
	"github.com/oublie6/awesome-zero-platform/server/platform/authz"
	"github.com/oublie6/awesome-zero-platform/server/platform/identity"
)

const SuperAdminRole = "platform_super_admin"

var (
	ErrNotFound          = errors.New("admin resource not found")
	ErrConflict          = errors.New("admin resource conflict")
	ErrProtectedRole     = errors.New("protected role")
	ErrBootstrapDisabled = errors.New("admin bootstrap is disabled")
	ErrBootstrapComplete = errors.New("admin bootstrap is already complete")
)

type Config struct {
	BootstrapToken string
}

type Actor struct {
	AccountID string
	RequestID string
	ClientIP  string
	UserAgent string
}

type Role struct {
	Code        string    `json:"code"`
	DisplayName string    `json:"displayName"`
	Description string    `json:"description"`
	System      bool      `json:"system"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type Resource struct {
	Code        string    `json:"code"`
	DisplayName string    `json:"displayName"`
	Module      string    `json:"module"`
	Pattern     string    `json:"pattern"`
	Actions     []string  `json:"actions"`
	Description string    `json:"description"`
	System      bool      `json:"system"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type AuditEvent struct {
	ID           string         `json:"id"`
	ActorID      string         `json:"actorId,omitempty"`
	Action       string         `json:"action"`
	ResourceType string         `json:"resourceType"`
	ResourceID   string         `json:"resourceId,omitempty"`
	Outcome      string         `json:"outcome"`
	RequestID    string         `json:"requestId,omitempty"`
	ClientIP     string         `json:"clientIp,omitempty"`
	UserAgent    string         `json:"userAgent,omitempty"`
	Details      map[string]any `json:"details,omitempty"`
	CreatedAt    time.Time      `json:"createdAt"`
}

type AuditQuery struct {
	Search   string
	Action   string
	Outcome  string
	Page     int
	PageSize int
}

type AuditPage struct {
	Items    []AuditEvent `json:"items"`
	Total    int64        `json:"total"`
	Page     int          `json:"page"`
	PageSize int          `json:"pageSize"`
}

type BootstrapInput struct {
	Username    string
	DisplayName string
	Password    string
}

type IdentityManager interface {
	ListAccounts(context.Context, identity.AccountQuery) (identity.AccountPage, error)
	CreateAccount(context.Context, identity.CreateAccountInput) (identity.Account, error)
	GetAccountByID(context.Context, string) (identity.Account, error)
	UpdateProfile(context.Context, string, identity.UpdateProfileInput) (identity.Account, error)
	EnableAccount(context.Context, string) (identity.Account, error)
	DisableAccount(context.Context, string) (identity.Account, error)
	ResetPassword(context.Context, string, string) error
	PasswordParams() identity.PasswordParams
}

type Repository interface {
	ListRoles(context.Context) ([]Role, error)
	GetRole(context.Context, string) (Role, error)
	CreateRole(context.Context, Role) (Role, error)
	UpdateRole(context.Context, Role) (Role, error)
	DeleteRole(context.Context, string) error
	ListResources(context.Context) ([]Resource, error)
	GetResource(context.Context, string) (Resource, error)
	CreateResource(context.Context, Resource) (Resource, error)
	UpdateResource(context.Context, Resource) (Resource, error)
	DeleteResource(context.Context, string) error
	AppendAudit(context.Context, AuditEvent) error
	ListAudit(context.Context, AuditQuery) (AuditPage, error)
}

type superAdminLocker interface {
	WithSuperAdminLock(context.Context, func(context.Context) error) error
}

type freshIdentityReader interface {
	GetAccountByIDFresh(context.Context, string) (identity.Account, error)
}

type Service struct {
	identity      IdentityManager
	authorization authz.Administrator
	sessions      authn.SessionAdministrator
	repository    Repository
	bootstrapHash [32]byte
	bootstrapSet  bool
	now           func() time.Time
	newID         func() string
}

func NewService(identityManager IdentityManager, authorization authz.Administrator, sessions authn.SessionAdministrator, repository Repository, cfg Config) (*Service, error) {
	if identityManager == nil || authorization == nil || sessions == nil || repository == nil {
		return nil, fmt.Errorf("admin identity, authorization, sessions, and repository are required")
	}
	service := &Service{
		identity:      identityManager,
		authorization: authorization,
		sessions:      sessions,
		repository:    repository,
		now:           time.Now,
		newID:         func() string { return uuid.Must(uuid.NewV7()).String() },
	}
	if token := strings.TrimSpace(cfg.BootstrapToken); token != "" {
		service.bootstrapHash = sha256.Sum256([]byte(token))
		service.bootstrapSet = true
	}
	return service, nil
}

func (s *Service) BootstrapAvailable(ctx context.Context) (bool, error) {
	if !s.bootstrapSet {
		return false, nil
	}
	users, err := s.authorization.UsersForRole(ctx, SuperAdminRole)
	return len(users) == 0, err
}

func (s *Service) Bootstrap(ctx context.Context, token string, input BootstrapInput, actor Actor) (identity.Account, error) {
	if !s.bootstrapSet {
		return identity.Account{}, ErrBootstrapDisabled
	}
	candidate := sha256.Sum256([]byte(strings.TrimSpace(token)))
	if subtle.ConstantTimeCompare(candidate[:], s.bootstrapHash[:]) != 1 {
		return identity.Account{}, ErrBootstrapDisabled
	}

	var account identity.Account
	err := s.withSuperAdminLock(ctx, func(lockCtx context.Context) error {
		available, err := s.BootstrapAvailable(lockCtx)
		if err != nil {
			return err
		}
		if !available {
			return ErrBootstrapComplete
		}

		account, err = s.identity.CreateAccount(lockCtx, identity.CreateAccountInput{
			Identity:    identity.Identity{Username: input.Username},
			DisplayName: input.DisplayName,
			Status:      identity.StatusActive,
			Password:    input.Password,
		})
		if err != nil {
			return err
		}
		if err := s.authorization.ReplaceRolesForUser(lockCtx, account.ID, []string{SuperAdminRole}); err != nil {
			_, _ = s.identity.DisableAccount(lockCtx, account.ID)
			return fmt.Errorf("assign bootstrap role: %w", err)
		}
		return nil
	})
	if err != nil {
		return identity.Account{}, err
	}

	s.audit(ctx, actor, "admin.bootstrap", "account", account.ID, "success", nil)
	return account, nil
}

func (s *Service) ListAccounts(ctx context.Context, query identity.AccountQuery) (identity.AccountPage, error) {
	return s.identity.ListAccounts(ctx, query)
}

func (s *Service) GetAccount(ctx context.Context, accountID string) (identity.Account, error) {
	return s.identity.GetAccountByID(ctx, accountID)
}

func (s *Service) CreateAccount(ctx context.Context, input identity.CreateAccountInput, roles []string, actor Actor) (identity.Account, error) {
	account, err := s.identity.CreateAccount(ctx, input)
	if err != nil {
		return identity.Account{}, err
	}
	if len(roles) > 0 {
		if err := s.authorization.ReplaceRolesForUser(ctx, account.ID, uniqueStrings(roles)); err != nil {
			_, _ = s.identity.DisableAccount(ctx, account.ID)
			return identity.Account{}, err
		}
	}
	s.audit(ctx, actor, "account.create", "account", account.ID, "success", map[string]any{"roles": roles})
	return account, nil
}

func (s *Service) UpdateAccount(ctx context.Context, accountID string, input identity.UpdateProfileInput, actor Actor) (identity.Account, error) {
	account, err := s.identity.UpdateProfile(ctx, accountID, input)
	if err == nil {
		s.audit(ctx, actor, "account.update", "account", accountID, "success", nil)
	}
	return account, err
}

func (s *Service) SetAccountEnabled(ctx context.Context, accountID string, enabled bool, actor Actor) (identity.Account, error) {
	var (
		account identity.Account
		err     error
	)
	if enabled {
		account, err = s.identity.EnableAccount(ctx, accountID)
	} else {
		err = s.withSuperAdminLock(ctx, func(lockCtx context.Context) error {
			if err := s.ensureCanDisableSuperAdmin(lockCtx, accountID); err != nil {
				return err
			}
			account, err = s.identity.DisableAccount(lockCtx, accountID)
			if err != nil {
				return err
			}
			_, _ = s.sessions.RevokeByAccount(lockCtx, accountID)
			return nil
		})
	}
	if err == nil {
		s.audit(ctx, actor, "account.status", "account", accountID, "success", map[string]any{"enabled": enabled})
	}
	return account, err
}

func (s *Service) ResetPassword(ctx context.Context, accountID, password string, actor Actor) error {
	if err := s.identity.ResetPassword(ctx, accountID, password); err != nil {
		return err
	}
	revoked, err := s.sessions.RevokeByAccount(ctx, accountID)
	if err != nil {
		return err
	}
	s.audit(ctx, actor, "account.reset_password", "account", accountID, "success", map[string]any{"revokedSessions": revoked})
	return nil
}

func (s *Service) SessionsForAccount(ctx context.Context, accountID string) ([]authn.SessionView, error) {
	return s.sessions.ListByAccount(ctx, accountID)
}

func (s *Service) RevokeAccountSessions(ctx context.Context, accountID string, actor Actor) (int64, error) {
	count, err := s.sessions.RevokeByAccount(ctx, accountID)
	if err == nil {
		s.audit(ctx, actor, "session.revoke_all", "account", accountID, "success", map[string]any{"count": count})
	}
	return count, err
}

func (s *Service) ListRoles(ctx context.Context) ([]Role, error) { return s.repository.ListRoles(ctx) }
func (s *Service) GetRole(ctx context.Context, code string) (Role, error) {
	return s.repository.GetRole(ctx, code)
}
func (s *Service) CreateRole(ctx context.Context, role Role, actor Actor) (Role, error) {
	created, err := s.repository.CreateRole(ctx, role)
	if err == nil {
		s.audit(ctx, actor, "role.create", "role", created.Code, "success", nil)
	}
	return created, err
}
func (s *Service) UpdateRole(ctx context.Context, role Role, actor Actor) (Role, error) {
	updated, err := s.repository.UpdateRole(ctx, role)
	if err == nil {
		s.audit(ctx, actor, "role.update", "role", updated.Code, "success", nil)
	}
	return updated, err
}
func (s *Service) DeleteRole(ctx context.Context, code string, actor Actor) error {
	code = strings.TrimSpace(code)
	if code == SuperAdminRole {
		return ErrProtectedRole
	}
	if users, err := s.authorization.UsersForRole(ctx, code); err != nil {
		return err
	} else if len(users) > 0 {
		return fmt.Errorf("%w: role still has members", ErrConflict)
	}
	if err := s.repository.DeleteRole(ctx, code); err != nil {
		return err
	}
	if err := s.authorization.ReplacePermissionsForRole(ctx, code, nil); err != nil {
		return err
	}
	s.audit(ctx, actor, "role.delete", "role", code, "success", nil)
	return nil
}

func (s *Service) ListResources(ctx context.Context) ([]Resource, error) {
	return s.repository.ListResources(ctx)
}
func (s *Service) CreateResource(ctx context.Context, resource Resource, actor Actor) (Resource, error) {
	created, err := s.repository.CreateResource(ctx, resource)
	if err == nil {
		s.audit(ctx, actor, "resource.create", "authorization_resource", created.Code, "success", nil)
	}
	return created, err
}
func (s *Service) UpdateResource(ctx context.Context, resource Resource, actor Actor) (Resource, error) {
	updated, err := s.repository.UpdateResource(ctx, resource)
	if err == nil {
		s.audit(ctx, actor, "resource.update", "authorization_resource", updated.Code, "success", nil)
	}
	return updated, err
}
func (s *Service) DeleteResource(ctx context.Context, code string, actor Actor) error {
	resource, err := s.repository.GetResource(ctx, code)
	if err != nil {
		return err
	}
	if resource.System {
		return ErrProtectedRole
	}
	if err := s.repository.DeleteResource(ctx, code); err != nil {
		return err
	}
	s.audit(ctx, actor, "resource.delete", "authorization_resource", code, "success", nil)
	return nil
}

func (s *Service) RolesForAccount(ctx context.Context, accountID string) ([]string, error) {
	return s.authorization.RolesForUser(ctx, accountID)
}

func (s *Service) ReplaceAccountRoles(ctx context.Context, accountID string, roles []string, actor Actor) error {
	roles = uniqueStrings(roles)
	err := s.withSuperAdminLock(ctx, func(lockCtx context.Context) error {
		current, err := s.authorization.RolesForUser(lockCtx, accountID)
		if err != nil {
			return err
		}
		if contains(current, SuperAdminRole) && !contains(roles, SuperAdminRole) {
			active, err := s.activeSuperAdminCount(lockCtx)
			if err != nil {
				return err
			}
			account, err := s.getAccountFresh(lockCtx, accountID)
			if err != nil {
				return err
			}
			if account.Status == identity.StatusActive && active <= 1 {
				return ErrProtectedRole
			}
		}
		return s.authorization.ReplaceRolesForUser(lockCtx, accountID, roles)
	})
	if err != nil {
		return err
	}
	s.audit(ctx, actor, "account.roles.replace", "account", accountID, "success", map[string]any{"roles": roles})
	return nil
}

func (s *Service) PermissionsForRole(ctx context.Context, role string) ([]authz.Permission, error) {
	return s.authorization.PermissionsForRole(ctx, role)
}

func (s *Service) ReplaceRolePermissions(ctx context.Context, role string, permissions []authz.Permission, actor Actor) error {
	if role == SuperAdminRole && !containsPermission(permissions, "/*", ".*") {
		return ErrProtectedRole
	}
	if err := s.authorization.ReplacePermissionsForRole(ctx, role, permissions); err != nil {
		return err
	}
	s.audit(ctx, actor, "role.permissions.replace", "role", role, "success", map[string]any{"permissions": permissions})
	return nil
}

func (s *Service) EngineInfo(ctx context.Context) authz.EngineInfo {
	return s.authorization.EngineInfo(ctx)
}
func (s *Service) ModelText(ctx context.Context) string { return s.authorization.ModelText(ctx) }
func (s *Service) ListRawRules(ctx context.Context) ([]authz.RawRule, error) {
	return s.authorization.ListRawRules(ctx)
}
func (s *Service) ValidateRawRules(ctx context.Context, rules []authz.RawRule) error {
	if err := s.authorization.ValidateRawRules(ctx, rules); err != nil {
		return err
	}
	return protectRawRules(rules)
}
func (s *Service) ReplaceRawRules(ctx context.Context, rules []authz.RawRule, actor Actor) error {
	if err := s.ValidateRawRules(ctx, rules); err != nil {
		return err
	}
	if err := s.authorization.ReplaceRawRules(ctx, rules); err != nil {
		return err
	}
	s.audit(ctx, actor, "authorization.raw.replace", "authorization_engine", s.EngineInfo(ctx).ID, "success", map[string]any{"ruleCount": len(rules)})
	return nil
}
func (s *Service) Explain(ctx context.Context, subject, resource, action string) (authz.Explanation, error) {
	return s.authorization.Explain(ctx, subject, resource, action)
}

func (s *Service) ListAudit(ctx context.Context, query AuditQuery) (AuditPage, error) {
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PageSize <= 0 {
		query.PageSize = 20
	}
	if query.PageSize > 100 {
		query.PageSize = 100
	}
	return s.repository.ListAudit(ctx, query)
}

func (s *Service) PasswordParams() identity.PasswordParams { return s.identity.PasswordParams() }

func (s *Service) withSuperAdminLock(ctx context.Context, fn func(context.Context) error) error {
	if locker, ok := s.repository.(superAdminLocker); ok {
		return locker.WithSuperAdminLock(ctx, fn)
	}
	return fn(ctx)
}

func (s *Service) getAccountFresh(ctx context.Context, accountID string) (identity.Account, error) {
	if reader, ok := s.identity.(freshIdentityReader); ok {
		return reader.GetAccountByIDFresh(ctx, accountID)
	}
	return s.identity.GetAccountByID(ctx, accountID)
}

func (s *Service) activeSuperAdminCount(ctx context.Context) (int, error) {
	users, err := s.authorization.UsersForRole(ctx, SuperAdminRole)
	if err != nil {
		return 0, err
	}
	active := 0
	for _, accountID := range users {
		account, err := s.getAccountFresh(ctx, accountID)
		if err != nil {
			return 0, err
		}
		if account.Status == identity.StatusActive {
			active++
		}
	}
	return active, nil
}

func (s *Service) ensureCanDisableSuperAdmin(ctx context.Context, accountID string) error {
	roles, err := s.authorization.RolesForUser(ctx, accountID)
	if err != nil {
		return err
	}
	if !contains(roles, SuperAdminRole) {
		return nil
	}
	account, err := s.getAccountFresh(ctx, accountID)
	if err != nil {
		return err
	}
	if account.Status != identity.StatusActive {
		return nil
	}
	active, err := s.activeSuperAdminCount(ctx)
	if err != nil {
		return err
	}
	if active <= 1 {
		return ErrProtectedRole
	}
	return nil
}

func (s *Service) audit(ctx context.Context, actor Actor, action, resourceType, resourceID, outcome string, details map[string]any) {
	_ = s.repository.AppendAudit(ctx, AuditEvent{
		ID:           s.newID(),
		ActorID:      strings.TrimSpace(actor.AccountID),
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Outcome:      outcome,
		RequestID:    actor.RequestID,
		ClientIP:     actor.ClientIP,
		UserAgent:    actor.UserAgent,
		Details:      details,
		CreatedAt:    s.now().UTC(),
	})
}

func protectRawRules(rules []authz.RawRule) error {
	hasSuperPolicy := false
	hasSuperMember := false
	for _, rule := range rules {
		if rule.PType == "p" && len(rule.Values) >= 3 && rule.Values[0] == SuperAdminRole && rule.Values[1] == "/*" && rule.Values[2] == ".*" {
			hasSuperPolicy = true
		}
		if rule.PType == "g" && len(rule.Values) >= 2 && rule.Values[1] == SuperAdminRole {
			hasSuperMember = true
		}
	}
	if !hasSuperPolicy || !hasSuperMember {
		return fmt.Errorf("%w: raw policies must preserve the super administrator wildcard policy and at least one member", ErrProtectedRole)
	}
	return nil
}

func uniqueStrings(values []string) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func containsPermission(values []authz.Permission, resource, action string) bool {
	for _, value := range values {
		if value.Resource == resource && value.Action == action {
			return true
		}
	}
	return false
}
