package authz

import (
	"context"
	"time"
)

// RawRule is the plugin-neutral transport representation of a native policy row.
// Casbin currently uses p and g rules, but callers should inspect EngineInfo before
// assuming a specific policy type or field count.
type RawRule struct {
	PType  string   `json:"ptype"`
	Values []string `json:"values"`
}

type Permission struct {
	Resource string `json:"resource"`
	Action   string `json:"action"`
}

type EngineInfo struct {
	ID                    string    `json:"id"`
	Name                  string    `json:"name"`
	Version               string    `json:"version"`
	ModelType             string    `json:"modelType"`
	PolicyTypes           []string  `json:"policyTypes"`
	SupportsRawPolicy     bool      `json:"supportsRawPolicy"`
	SupportsModelInspect  bool      `json:"supportsModelInspection"`
	SupportsModelEdit     bool      `json:"supportsModelEditing"`
	SupportsExplain       bool      `json:"supportsPolicyExplanation"`
	SupportsBatchImport   bool      `json:"supportsBatchImport"`
	SupportsRoleHierarchy bool      `json:"supportsRoleHierarchy"`
	LoadedAt              time.Time `json:"loadedAt"`
}

type Explanation struct {
	Subject        string    `json:"subject"`
	Resource       string    `json:"resource"`
	Action         string    `json:"action"`
	Allowed        bool      `json:"allowed"`
	Roles          []string  `json:"roles"`
	MatchedRules   []RawRule `json:"matchedRules"`
	CandidateRules []RawRule `json:"candidateRules"`
}

// Administrator exposes read and controlled mutation capabilities used by the
// platform Admin application. It is intentionally separate from Authorizer and
// PolicyManager so normal request paths never depend on expert tooling.
type Administrator interface {
	EngineInfo(context.Context) EngineInfo
	ModelText(context.Context) string
	ListRawRules(context.Context) ([]RawRule, error)
	ValidateRawRules(context.Context, []RawRule) error
	ReplaceRawRules(context.Context, []RawRule) error
	RolesForUser(context.Context, string) ([]string, error)
	UsersForRole(context.Context, string) ([]string, error)
	PermissionsForRole(context.Context, string) ([]Permission, error)
	ReplacePermissionsForRole(context.Context, string, []Permission) error
	ReplaceRolesForUser(context.Context, string, []string) error
	Explain(context.Context, string, string, string) (Explanation, error)
}
