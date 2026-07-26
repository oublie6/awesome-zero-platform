package casbinmysql

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/casbin/casbin/v2/util"
	"github.com/google/uuid"
	"github.com/oublie6/awesome-zero-platform/server/platform/authz"
)

const (
	protectedSuperAdminRole = "platform_super_admin"
	protectedWildcardObject = "/*"
	protectedWildcardAction = ".*"
)

func (e *Engine) EngineInfo(context.Context) authz.EngineInfo {
	loadedAt, localVersion, databaseVersion, syncErr, lastSync, watcherConnected := e.stateSnapshot()
	rules, _ := e.snapshotRules()
	info := authz.EngineInfo{
		ID:                    "casbin",
		Name:                  "Casbin",
		Version:               "v2",
		ModelType:             "RBAC with keyMatch2 resources and regex actions",
		PolicyTypes:           []string{"p", "g"},
		SupportsRawPolicy:     true,
		SupportsModelInspect:  true,
		SupportsModelEdit:     false,
		SupportsExplain:       true,
		SupportsBatchImport:   true,
		SupportsRoleHierarchy: true,
		LoadedAt:              loadedAt,
		LocalPolicyVersion:    localVersion,
		DatabasePolicyVersion: databaseVersion,
		PolicyRuleCount:       len(rules),
		SyncHealthy:           syncErr == nil && (!e.clusterEnabled || localVersion >= databaseVersion),
		WatcherConnected:      watcherConnected,
		LastSyncAt:            lastSync,
	}
	if e.clusterEnabled {
		info.InstanceID = e.cluster.InstanceID
	}
	if syncErr != nil {
		info.LastSyncError = syncErr.Error()
	}
	return info
}

func (e *Engine) ModelText(context.Context) string { return modelText }

func (e *Engine) ListRawRules(context.Context) ([]authz.RawRule, error) {
	return e.snapshotRules()
}

func (e *Engine) ValidateRawRules(_ context.Context, rules []authz.RawRule) error {
	seen := make(map[string]struct{}, len(rules))
	for index, rule := range rules {
		ptype := strings.TrimSpace(rule.PType)
		values := trimmedValues(rule.Values)
		switch ptype {
		case "p":
			if len(values) != 3 {
				return fmt.Errorf("rule %d: p requires subject, resource, and action", index+1)
			}
			if _, err := regexp.Compile(values[2]); err != nil {
				return fmt.Errorf("rule %d: invalid action regular expression: %w", index+1, err)
			}
		case "g":
			if len(values) != 2 {
				return fmt.Errorf("rule %d: g requires subject and role", index+1)
			}
		default:
			return fmt.Errorf("rule %d: unsupported policy type %q", index+1, ptype)
		}
		for _, value := range values {
			if value == "" {
				return fmt.Errorf("rule %d: values must not be empty", index+1)
			}
		}
		key := ptype + "\x00" + strings.Join(values, "\x00")
		if _, exists := seen[key]; exists {
			return fmt.Errorf("rule %d: duplicate policy", index+1)
		}
		seen[key] = struct{}{}
	}
	return nil
}

func (e *Engine) ReplaceRawRules(ctx context.Context, rules []authz.RawRule) error {
	_, err := e.mutateRules(ctx, func([]authz.RawRule) ([]authz.RawRule, error) {
		return copyRules(rules), nil
	})
	return err
}

func (e *Engine) RolesForUser(_ context.Context, accountID string) ([]string, error) {
	roles, err := e.enforcer.GetImplicitRolesForUser(strings.TrimSpace(accountID))
	if err != nil {
		return nil, err
	}
	sort.Strings(roles)
	return roles, nil
}

func (e *Engine) UsersForRole(_ context.Context, role string) ([]string, error) {
	users, err := e.enforcer.GetUsersForRole(strings.TrimSpace(role))
	if err != nil {
		return nil, err
	}
	sort.Strings(users)
	return users, nil
}

func (e *Engine) PermissionsForRole(_ context.Context, role string) ([]authz.Permission, error) {
	rows, err := e.enforcer.GetPermissionsForUser(strings.TrimSpace(role))
	if err != nil {
		return nil, err
	}
	permissions := make([]authz.Permission, 0, len(rows))
	for _, row := range rows {
		if len(row) < 3 {
			continue
		}
		permissions = append(permissions, authz.Permission{Resource: row[1], Action: row[2]})
	}
	sort.Slice(permissions, func(i, j int) bool {
		if permissions[i].Resource == permissions[j].Resource {
			return permissions[i].Action < permissions[j].Action
		}
		return permissions[i].Resource < permissions[j].Resource
	})
	return permissions, nil
}

func (e *Engine) ReplacePermissionsForRole(ctx context.Context, role string, permissions []authz.Permission) error {
	role = strings.TrimSpace(role)
	if role == "" {
		return fmt.Errorf("role must not be empty")
	}
	_, err := e.mutateRules(ctx, func(rules []authz.RawRule) ([]authz.RawRule, error) {
		filtered := rules[:0]
		for _, rule := range rules {
			if rule.PType == "p" && len(rule.Values) >= 1 && strings.TrimSpace(rule.Values[0]) == role {
				continue
			}
			filtered = append(filtered, rule)
		}
		for _, permission := range permissions {
			filtered = append(filtered, authz.RawRule{PType: "p", Values: []string{role, permission.Resource, permission.Action}})
		}
		return filtered, nil
	})
	return err
}

func (e *Engine) ReplaceRolesForUser(ctx context.Context, accountID string, roles []string) error {
	accountID = strings.TrimSpace(accountID)
	if accountID == "" {
		return fmt.Errorf("account id must not be empty")
	}
	_, err := e.mutateRules(ctx, func(rules []authz.RawRule) ([]authz.RawRule, error) {
		filtered := rules[:0]
		for _, rule := range rules {
			if rule.PType == "g" && len(rule.Values) >= 1 && strings.TrimSpace(rule.Values[0]) == accountID {
				continue
			}
			filtered = append(filtered, rule)
		}
		seen := map[string]struct{}{}
		for _, role := range roles {
			role = strings.TrimSpace(role)
			if role == "" {
				return nil, fmt.Errorf("role must not be empty")
			}
			if _, exists := seen[role]; exists {
				continue
			}
			seen[role] = struct{}{}
			filtered = append(filtered, authz.RawRule{PType: "g", Values: []string{accountID, role}})
		}
		return filtered, nil
	})
	return err
}

func (e *Engine) Explain(ctx context.Context, subject, resource, action string) (authz.Explanation, error) {
	subject = strings.TrimSpace(subject)
	resource = strings.TrimSpace(resource)
	action = strings.TrimSpace(action)
	if subject == "" || resource == "" || action == "" {
		return authz.Explanation{}, fmt.Errorf("subject, resource, and action are required")
	}
	allowed, err := e.Enforce(ctx, subject, resource, action)
	if err != nil {
		return authz.Explanation{}, err
	}
	roles, err := e.RolesForUser(ctx, subject)
	if err != nil {
		return authz.Explanation{}, err
	}
	principals := map[string]struct{}{subject: {}}
	for _, role := range roles {
		principals[role] = struct{}{}
	}

	explanation := authz.Explanation{
		Subject:  subject,
		Resource: resource,
		Action:   action,
		Allowed:  allowed,
		Roles:    roles,
	}
	policies, err := e.enforcer.GetPolicy()
	if err != nil {
		return authz.Explanation{}, err
	}
	for _, row := range policies {
		if len(row) < 3 {
			continue
		}
		if _, relevant := principals[row[0]]; !relevant {
			continue
		}
		rule := authz.RawRule{PType: "p", Values: append([]string(nil), row...)}
		explanation.CandidateRules = append(explanation.CandidateRules, rule)
		actionMatched, regexErr := regexp.MatchString(row[2], action)
		if regexErr == nil && util.KeyMatch2(resource, row[1]) && actionMatched {
			explanation.MatchedRules = append(explanation.MatchedRules, rule)
		}
	}
	return explanation, nil
}

func (e *Engine) snapshotRules() ([]authz.RawRule, error) {
	policies, err := e.enforcer.GetPolicy()
	if err != nil {
		return nil, err
	}
	groups, err := e.enforcer.GetGroupingPolicy()
	if err != nil {
		return nil, err
	}
	rules := make([]authz.RawRule, 0, len(policies)+len(groups))
	for _, policy := range policies {
		rules = append(rules, authz.RawRule{PType: "p", Values: append([]string(nil), policy...)})
	}
	for _, group := range groups {
		rules = append(rules, authz.RawRule{PType: "g", Values: append([]string(nil), group...)})
	}
	sort.Slice(rules, func(i, j int) bool {
		if rules[i].PType == rules[j].PType {
			return strings.Join(rules[i].Values, "\x00") < strings.Join(rules[j].Values, "\x00")
		}
		return rules[i].PType < rules[j].PType
	})
	return rules, nil
}

func (e *Engine) applyRawRules(rules []authz.RawRule) error {
	previous, err := e.snapshotRules()
	if err != nil {
		return err
	}
	e.enforcer.EnableAutoSave(false)
	defer e.enforcer.EnableAutoSave(true)
	e.enforcer.ClearPolicy()
	for _, rule := range rules {
		values := trimmedValues(rule.Values)
		var err error
		switch strings.TrimSpace(rule.PType) {
		case "p":
			_, err = e.enforcer.AddNamedPolicy("p", stringSliceToAny(values)...)
		case "g":
			_, err = e.enforcer.AddNamedGroupingPolicy("g", stringSliceToAny(values)...)
		default:
			err = fmt.Errorf("unsupported policy type %q", rule.PType)
		}
		if err != nil {
			_ = e.restoreRawRules(previous)
			return err
		}
	}
	if err := e.enforcer.SavePolicy(); err != nil {
		_ = e.restoreRawRules(previous)
		return fmt.Errorf("save casbin policy: %w", err)
	}
	return nil
}

func (e *Engine) restoreRawRules(rules []authz.RawRule) error {
	e.enforcer.ClearPolicy()
	for _, rule := range rules {
		values := trimmedValues(rule.Values)
		switch strings.TrimSpace(rule.PType) {
		case "p":
			if _, err := e.enforcer.AddNamedPolicy("p", stringSliceToAny(values)...); err != nil {
				return err
			}
		case "g":
			if _, err := e.enforcer.AddNamedGroupingPolicy("g", stringSliceToAny(values)...); err != nil {
				return err
			}
		}
	}
	return nil
}

func (e *Engine) validatePolicySafety(current, next []authz.RawRule) error {
	currentWildcard := hasProtectedWildcard(current)
	nextWildcard := hasProtectedWildcard(next)
	currentAdmins := directSuperAdminCount(current)
	nextAdmins := directSuperAdminCount(next)

	if currentWildcard && !nextWildcard {
		return fmt.Errorf("protected platform super administrator wildcard permission cannot be removed")
	}
	if nextAdmins > 0 && !nextWildcard {
		return fmt.Errorf("platform super administrator membership requires the protected wildcard permission")
	}
	if currentAdmins > 0 && nextAdmins == 0 {
		return fmt.Errorf("the last direct platform super administrator cannot be removed")
	}
	return nil
}

func hasProtectedWildcard(rules []authz.RawRule) bool {
	for _, rule := range rules {
		values := trimmedValues(rule.Values)
		if strings.TrimSpace(rule.PType) == "p" && len(values) == 3 &&
			values[0] == protectedSuperAdminRole && values[1] == protectedWildcardObject && values[2] == protectedWildcardAction {
			return true
		}
	}
	return false
}

func directSuperAdminCount(rules []authz.RawRule) int {
	seen := map[string]struct{}{}
	for _, rule := range rules {
		values := trimmedValues(rule.Values)
		if strings.TrimSpace(rule.PType) != "g" || len(values) != 2 || values[1] != protectedSuperAdminRole {
			continue
		}
		if _, err := uuid.Parse(values[0]); err != nil {
			continue
		}
		seen[values[0]] = struct{}{}
	}
	return len(seen)
}

func trimmedValues(values []string) []string {
	result := make([]string, len(values))
	for i, value := range values {
		result[i] = strings.TrimSpace(value)
	}
	return result
}

func stringSliceToAny(values []string) []interface{} {
	result := make([]interface{}, len(values))
	for i, value := range values {
		result[i] = value
	}
	return result
}
