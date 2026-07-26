package adminhttp

import (
	"errors"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/oublie6/awesome-zero-platform/server/apps/app-api/internal/securityhttp"
	"github.com/oublie6/awesome-zero-platform/server/apps/app-api/internal/svc"
	"github.com/oublie6/awesome-zero-platform/server/foundation/apperrors"
	platformresponse "github.com/oublie6/awesome-zero-platform/server/foundation/response"
	"github.com/oublie6/awesome-zero-platform/server/platform/admin"
	"github.com/oublie6/awesome-zero-platform/server/platform/authz"
	"github.com/oublie6/awesome-zero-platform/server/platform/identity"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/rest/httpx"
	"github.com/zeromicro/go-zero/rest/pathvar"
)

type bootstrapRequest struct {
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
	Password    string `json:"password"`
}

type createAccountRequest struct {
	Username    string   `json:"username"`
	Email       string   `json:"email"`
	Phone       string   `json:"phone"`
	DisplayName string   `json:"displayName"`
	Status      string   `json:"status"`
	Password    string   `json:"password"`
	Roles       []string `json:"roles"`
}

type updateAccountRequest struct {
	Username    *string `json:"username"`
	Email       *string `json:"email"`
	Phone       *string `json:"phone"`
	DisplayName *string `json:"displayName"`
}

type resetPasswordRequest struct {
	Password string `json:"password"`
}

type replaceRolesRequest struct {
	Roles []string `json:"roles"`
}

type roleRequest struct {
	Code        string `json:"code"`
	DisplayName string `json:"displayName"`
	Description string `json:"description"`
}

type resourceRequest struct {
	Code        string   `json:"code"`
	DisplayName string   `json:"displayName"`
	Module      string   `json:"module"`
	Pattern     string   `json:"pattern"`
	Actions     []string `json:"actions"`
	Description string   `json:"description"`
}

type replacePermissionsRequest struct {
	Permissions []authz.Permission `json:"permissions"`
}

type rawRulesRequest struct {
	Rules []authz.RawRule `json:"rules"`
}

type explainRequest struct {
	Subject  string `json:"subject"`
	Resource string `json:"resource"`
	Action   string `json:"action"`
}

type accountResponse struct {
	ID          string                 `json:"id"`
	Username    string                 `json:"username,omitempty"`
	Email       string                 `json:"email,omitempty"`
	Phone       string                 `json:"phone,omitempty"`
	DisplayName string                 `json:"displayName"`
	Status      identity.AccountStatus `json:"status"`
	CreatedAt   time.Time              `json:"createdAt"`
	UpdatedAt   time.Time              `json:"updatedAt"`
}

func Register(server *rest.Server, serverCtx *svc.ServiceContext) {
	if serverCtx == nil || serverCtx.Admin == nil {
		return
	}
	server.AddRoutes([]rest.Route{
		{Method: http.MethodGet, Path: "/admin/bootstrap/status", Handler: bootstrapStatusHandler(serverCtx)},
		{Method: http.MethodPost, Path: "/admin/bootstrap", Handler: bootstrapHandler(serverCtx)},
		{Method: http.MethodGet, Path: "/admin/me", Handler: protected(serverCtx, meHandler(serverCtx))},
		{Method: http.MethodGet, Path: "/admin/accounts", Handler: protected(serverCtx, listAccountsHandler(serverCtx))},
		{Method: http.MethodPost, Path: "/admin/accounts", Handler: protected(serverCtx, createAccountHandler(serverCtx))},
		{Method: http.MethodGet, Path: "/admin/accounts/:id", Handler: protected(serverCtx, getAccountHandler(serverCtx))},
		{Method: http.MethodPatch, Path: "/admin/accounts/:id", Handler: protected(serverCtx, updateAccountHandler(serverCtx))},
		{Method: http.MethodPost, Path: "/admin/accounts/:id/enable", Handler: protected(serverCtx, accountStatusHandler(serverCtx, true))},
		{Method: http.MethodPost, Path: "/admin/accounts/:id/disable", Handler: protected(serverCtx, accountStatusHandler(serverCtx, false))},
		{Method: http.MethodPost, Path: "/admin/accounts/:id/reset-password", Handler: protected(serverCtx, resetPasswordHandler(serverCtx))},
		{Method: http.MethodGet, Path: "/admin/accounts/:id/sessions", Handler: protected(serverCtx, accountSessionsHandler(serverCtx))},
		{Method: http.MethodPost, Path: "/admin/accounts/:id/revoke-sessions", Handler: protected(serverCtx, revokeSessionsHandler(serverCtx))},
		{Method: http.MethodGet, Path: "/admin/accounts/:id/roles", Handler: protected(serverCtx, accountRolesHandler(serverCtx))},
		{Method: http.MethodPut, Path: "/admin/accounts/:id/roles", Handler: protected(serverCtx, replaceAccountRolesHandler(serverCtx))},
		{Method: http.MethodGet, Path: "/admin/roles", Handler: protected(serverCtx, listRolesHandler(serverCtx))},
		{Method: http.MethodPost, Path: "/admin/roles", Handler: protected(serverCtx, createRoleHandler(serverCtx))},
		{Method: http.MethodGet, Path: "/admin/roles/:code", Handler: protected(serverCtx, getRoleHandler(serverCtx))},
		{Method: http.MethodPatch, Path: "/admin/roles/:code", Handler: protected(serverCtx, updateRoleHandler(serverCtx))},
		{Method: http.MethodDelete, Path: "/admin/roles/:code", Handler: protected(serverCtx, deleteRoleHandler(serverCtx))},
		{Method: http.MethodGet, Path: "/admin/roles/:code/permissions", Handler: protected(serverCtx, rolePermissionsHandler(serverCtx))},
		{Method: http.MethodPut, Path: "/admin/roles/:code/permissions", Handler: protected(serverCtx, replaceRolePermissionsHandler(serverCtx))},
		{Method: http.MethodGet, Path: "/admin/authorization/resources", Handler: protected(serverCtx, listResourcesHandler(serverCtx))},
		{Method: http.MethodPost, Path: "/admin/authorization/resources", Handler: protected(serverCtx, createResourceHandler(serverCtx))},
		{Method: http.MethodGet, Path: "/admin/authorization/resources/:code", Handler: protected(serverCtx, getResourceHandler(serverCtx))},
		{Method: http.MethodPatch, Path: "/admin/authorization/resources/:code", Handler: protected(serverCtx, updateResourceHandler(serverCtx))},
		{Method: http.MethodDelete, Path: "/admin/authorization/resources/:code", Handler: protected(serverCtx, deleteResourceHandler(serverCtx))},
		{Method: http.MethodGet, Path: "/admin/authorization/engine", Handler: protected(serverCtx, engineHandler(serverCtx))},
		{Method: http.MethodGet, Path: "/admin/authorization/engine/model", Handler: protected(serverCtx, modelHandler(serverCtx))},
		{Method: http.MethodGet, Path: "/admin/authorization/engine/policies", Handler: protected(serverCtx, policiesHandler(serverCtx))},
		{Method: http.MethodPost, Path: "/admin/authorization/engine/policies/validate", Handler: protected(serverCtx, validatePoliciesHandler(serverCtx))},
		{Method: http.MethodPut, Path: "/admin/authorization/engine/policies", Handler: protected(serverCtx, replacePoliciesHandler(serverCtx))},
		{Method: http.MethodPost, Path: "/admin/authorization/engine/explain", Handler: protected(serverCtx, explainHandler(serverCtx))},
		{Method: http.MethodGet, Path: "/admin/audit/events", Handler: protected(serverCtx, auditHandler(serverCtx))},
		{Method: http.MethodGet, Path: "/admin/system/overview", Handler: protected(serverCtx, systemOverviewHandler(serverCtx))},
		{Method: http.MethodGet, Path: "/admin/system/configuration", Handler: protected(serverCtx, systemConfigurationHandler(serverCtx))},
	})
}

func protected(serverCtx *svc.ServiceContext, next http.HandlerFunc) http.HandlerFunc {
	return securityhttp.RequireAuthentication(serverCtx.Authn, func(w http.ResponseWriter, r *http.Request) {
		authentication, ok := securityhttp.AuthenticationFromContext(r.Context())
		if !ok {
			writeError(r, w, apperrors.Unauthorized("authentication required"))
			return
		}
		allowed, err := serverCtx.Authorizer.Enforce(r.Context(), authentication.Principal.AccountID, r.URL.Path, r.Method)
		if err != nil {
			writeError(r, w, apperrors.Internal(err))
			return
		}
		if !allowed {
			writeError(r, w, apperrors.Forbidden("permission denied"))
			return
		}
		next(w, r)
	})
}

func bootstrapStatusHandler(serverCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		available, err := serverCtx.Admin.BootstrapAvailable(r.Context())
		if err != nil {
			writeError(r, w, mapError(err))
			return
		}
		write(r, w, http.StatusOK, map[string]any{"available": available})
	}
}

func bootstrapHandler(serverCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request bootstrapRequest
		if err := httpx.Parse(r, &request); err != nil {
			writeError(r, w, apperrors.InvalidParameter("invalid request body").WithCause(err))
			return
		}
		account, err := serverCtx.Admin.Bootstrap(r.Context(), r.Header.Get("X-Admin-Bootstrap-Token"), admin.BootstrapInput{
			Username: request.Username, DisplayName: request.DisplayName, Password: request.Password,
		}, actorFromRequest(r))
		if err != nil {
			writeError(r, w, mapError(err))
			return
		}
		write(r, w, http.StatusCreated, accountDTO(account))
	}
}

func meHandler(serverCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authentication, _ := securityhttp.AuthenticationFromContext(r.Context())
		account, err := serverCtx.Admin.GetAccount(r.Context(), authentication.Principal.AccountID)
		if err != nil {
			writeError(r, w, mapError(err))
			return
		}
		roles, err := serverCtx.Admin.RolesForAccount(r.Context(), account.ID)
		if err != nil {
			writeError(r, w, mapError(err))
			return
		}
		permissions := make([]authz.Permission, 0)
		for _, role := range roles {
			rows, err := serverCtx.Admin.PermissionsForRole(r.Context(), role)
			if err != nil {
				writeError(r, w, mapError(err))
				return
			}
			permissions = append(permissions, rows...)
		}
		write(r, w, http.StatusOK, map[string]any{"account": accountDTO(account), "roles": roles, "permissions": permissions})
	}
}

func listAccountsHandler(serverCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		page, _ := strconv.Atoi(query.Get("page"))
		pageSize, _ := strconv.Atoi(query.Get("pageSize"))
		result, err := serverCtx.Admin.ListAccounts(r.Context(), identity.AccountQuery{
			Search: query.Get("search"), Status: identity.AccountStatus(query.Get("status")), Page: page, PageSize: pageSize,
		})
		if err != nil {
			writeError(r, w, mapError(err))
			return
		}
		items := make([]accountResponse, 0, len(result.Items))
		for _, account := range result.Items {
			items = append(items, accountDTO(account))
		}
		write(r, w, http.StatusOK, map[string]any{"items": items, "total": result.Total, "page": result.Page, "pageSize": result.PageSize})
	}
}

func createAccountHandler(serverCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request createAccountRequest
		if err := httpx.Parse(r, &request); err != nil {
			writeError(r, w, apperrors.InvalidParameter("invalid request body").WithCause(err))
			return
		}
		account, err := serverCtx.Admin.CreateAccount(r.Context(), identity.CreateAccountInput{
			Identity:    identity.Identity{Username: request.Username, Email: request.Email, Phone: request.Phone},
			DisplayName: request.DisplayName, Status: identity.AccountStatus(request.Status), Password: request.Password,
		}, request.Roles, actorFromRequest(r))
		if err != nil {
			writeError(r, w, mapError(err))
			return
		}
		write(r, w, http.StatusCreated, accountDTO(account))
	}
}

func getAccountHandler(serverCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		account, err := serverCtx.Admin.GetAccount(r.Context(), routeParam(r, "id"))
		if err != nil {
			writeError(r, w, mapError(err))
			return
		}
		roles, _ := serverCtx.Admin.RolesForAccount(r.Context(), account.ID)
		write(r, w, http.StatusOK, map[string]any{"account": accountDTO(account), "roles": roles})
	}
}

func updateAccountHandler(serverCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request updateAccountRequest
		if err := httpx.Parse(r, &request); err != nil {
			writeError(r, w, apperrors.InvalidParameter("invalid request body").WithCause(err))
			return
		}
		account, err := serverCtx.Admin.UpdateAccount(r.Context(), routeParam(r, "id"), identity.UpdateProfileInput{
			Username: request.Username, Email: request.Email, Phone: request.Phone, DisplayName: request.DisplayName,
		}, actorFromRequest(r))
		if err != nil {
			writeError(r, w, mapError(err))
			return
		}
		write(r, w, http.StatusOK, accountDTO(account))
	}
}

func accountStatusHandler(serverCtx *svc.ServiceContext, enabled bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		account, err := serverCtx.Admin.SetAccountEnabled(r.Context(), routeParam(r, "id"), enabled, actorFromRequest(r))
		if err != nil {
			writeError(r, w, mapError(err))
			return
		}
		write(r, w, http.StatusOK, accountDTO(account))
	}
}

func resetPasswordHandler(serverCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request resetPasswordRequest
		if err := httpx.Parse(r, &request); err != nil {
			writeError(r, w, apperrors.InvalidParameter("invalid request body").WithCause(err))
			return
		}
		if err := serverCtx.Admin.ResetPassword(r.Context(), routeParam(r, "id"), request.Password, actorFromRequest(r)); err != nil {
			writeError(r, w, mapError(err))
			return
		}
		write(r, w, http.StatusOK, map[string]any{"reset": true})
	}
}

func accountSessionsHandler(serverCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessions, err := serverCtx.Admin.SessionsForAccount(r.Context(), routeParam(r, "id"))
		if err != nil {
			writeError(r, w, mapError(err))
			return
		}
		write(r, w, http.StatusOK, sessions)
	}
}

func revokeSessionsHandler(serverCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		count, err := serverCtx.Admin.RevokeAccountSessions(r.Context(), routeParam(r, "id"), actorFromRequest(r))
		if err != nil {
			writeError(r, w, mapError(err))
			return
		}
		write(r, w, http.StatusOK, map[string]any{"revoked": count})
	}
}

func accountRolesHandler(serverCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roles, err := serverCtx.Admin.RolesForAccount(r.Context(), routeParam(r, "id"))
		if err != nil {
			writeError(r, w, mapError(err))
			return
		}
		write(r, w, http.StatusOK, roles)
	}
}

func replaceAccountRolesHandler(serverCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request replaceRolesRequest
		if err := httpx.Parse(r, &request); err != nil {
			writeError(r, w, apperrors.InvalidParameter("invalid request body").WithCause(err))
			return
		}
		if err := serverCtx.Admin.ReplaceAccountRoles(r.Context(), routeParam(r, "id"), request.Roles, actorFromRequest(r)); err != nil {
			writeError(r, w, mapError(err))
			return
		}
		write(r, w, http.StatusOK, map[string]any{"roles": request.Roles})
	}
}

func listRolesHandler(serverCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roles, err := serverCtx.Admin.ListRoles(r.Context())
		if err != nil {
			writeError(r, w, mapError(err))
			return
		}
		write(r, w, http.StatusOK, roles)
	}
}

func createRoleHandler(serverCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request roleRequest
		if err := httpx.Parse(r, &request); err != nil {
			writeError(r, w, apperrors.InvalidParameter("invalid request body").WithCause(err))
			return
		}
		role, err := serverCtx.Admin.CreateRole(r.Context(), admin.Role{Code: request.Code, DisplayName: request.DisplayName, Description: request.Description}, actorFromRequest(r))
		if err != nil {
			writeError(r, w, mapError(err))
			return
		}
		write(r, w, http.StatusCreated, role)
	}
}

func getRoleHandler(serverCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		role, err := serverCtx.Admin.GetRole(r.Context(), routeParam(r, "code"))
		if err != nil {
			writeError(r, w, mapError(err))
			return
		}
		write(r, w, http.StatusOK, role)
	}
}

func updateRoleHandler(serverCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request roleRequest
		if err := httpx.Parse(r, &request); err != nil {
			writeError(r, w, apperrors.InvalidParameter("invalid request body").WithCause(err))
			return
		}
		role, err := serverCtx.Admin.UpdateRole(r.Context(), admin.Role{Code: routeParam(r, "code"), DisplayName: request.DisplayName, Description: request.Description}, actorFromRequest(r))
		if err != nil {
			writeError(r, w, mapError(err))
			return
		}
		write(r, w, http.StatusOK, role)
	}
}

func deleteRoleHandler(serverCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := serverCtx.Admin.DeleteRole(r.Context(), routeParam(r, "code"), actorFromRequest(r)); err != nil {
			writeError(r, w, mapError(err))
			return
		}
		write(r, w, http.StatusOK, map[string]any{"deleted": true})
	}
}

func rolePermissionsHandler(serverCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		permissions, err := serverCtx.Admin.PermissionsForRole(r.Context(), routeParam(r, "code"))
		if err != nil {
			writeError(r, w, mapError(err))
			return
		}
		write(r, w, http.StatusOK, permissions)
	}
}

func replaceRolePermissionsHandler(serverCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request replacePermissionsRequest
		if err := httpx.Parse(r, &request); err != nil {
			writeError(r, w, apperrors.InvalidParameter("invalid request body").WithCause(err))
			return
		}
		if err := serverCtx.Admin.ReplaceRolePermissions(r.Context(), routeParam(r, "code"), request.Permissions, actorFromRequest(r)); err != nil {
			writeError(r, w, mapError(err))
			return
		}
		write(r, w, http.StatusOK, request.Permissions)
	}
}

func listResourcesHandler(serverCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resources, err := serverCtx.Admin.ListResources(r.Context())
		if err != nil {
			writeError(r, w, mapError(err))
			return
		}
		write(r, w, http.StatusOK, resources)
	}
}

func createResourceHandler(serverCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request resourceRequest
		if err := httpx.Parse(r, &request); err != nil {
			writeError(r, w, apperrors.InvalidParameter("invalid request body").WithCause(err))
			return
		}
		resource, err := serverCtx.Admin.CreateResource(r.Context(), admin.Resource{
			Code: request.Code, DisplayName: request.DisplayName, Module: request.Module, Pattern: request.Pattern, Actions: request.Actions, Description: request.Description,
		}, actorFromRequest(r))
		if err != nil {
			writeError(r, w, mapError(err))
			return
		}
		write(r, w, http.StatusCreated, resource)
	}
}

func getResourceHandler(serverCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resources, err := serverCtx.Admin.ListResources(r.Context())
		if err != nil {
			writeError(r, w, mapError(err))
			return
		}
		code := routeParam(r, "code")
		for _, resource := range resources {
			if resource.Code == code {
				write(r, w, http.StatusOK, resource)
				return
			}
		}
		writeError(r, w, apperrors.NotFound("resource not found"))
	}
}

func updateResourceHandler(serverCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request resourceRequest
		if err := httpx.Parse(r, &request); err != nil {
			writeError(r, w, apperrors.InvalidParameter("invalid request body").WithCause(err))
			return
		}
		resource, err := serverCtx.Admin.UpdateResource(r.Context(), admin.Resource{
			Code: routeParam(r, "code"), DisplayName: request.DisplayName, Module: request.Module, Pattern: request.Pattern, Actions: request.Actions, Description: request.Description,
		}, actorFromRequest(r))
		if err != nil {
			writeError(r, w, mapError(err))
			return
		}
		write(r, w, http.StatusOK, resource)
	}
}

func deleteResourceHandler(serverCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := serverCtx.Admin.DeleteResource(r.Context(), routeParam(r, "code"), actorFromRequest(r)); err != nil {
			writeError(r, w, mapError(err))
			return
		}
		write(r, w, http.StatusOK, map[string]any{"deleted": true})
	}
}

func engineHandler(serverCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		write(r, w, http.StatusOK, serverCtx.Admin.EngineInfo(r.Context()))
	}
}

func modelHandler(serverCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		write(r, w, http.StatusOK, map[string]any{"model": serverCtx.Admin.ModelText(r.Context())})
	}
}

func policiesHandler(serverCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rules, err := serverCtx.Admin.ListRawRules(r.Context())
		if err != nil {
			writeError(r, w, mapError(err))
			return
		}
		write(r, w, http.StatusOK, rules)
	}
}

func validatePoliciesHandler(serverCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request rawRulesRequest
		if err := httpx.Parse(r, &request); err != nil {
			writeError(r, w, apperrors.InvalidParameter("invalid request body").WithCause(err))
			return
		}
		if err := serverCtx.Admin.ValidateRawRules(r.Context(), request.Rules); err != nil {
			writeError(r, w, mapError(err))
			return
		}
		write(r, w, http.StatusOK, map[string]any{"valid": true})
	}
}

func replacePoliciesHandler(serverCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request rawRulesRequest
		if err := httpx.Parse(r, &request); err != nil {
			writeError(r, w, apperrors.InvalidParameter("invalid request body").WithCause(err))
			return
		}
		if err := serverCtx.Admin.ReplaceRawRules(r.Context(), request.Rules, actorFromRequest(r)); err != nil {
			writeError(r, w, mapError(err))
			return
		}
		write(r, w, http.StatusOK, map[string]any{"saved": true, "ruleCount": len(request.Rules)})
	}
}

func explainHandler(serverCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request explainRequest
		if err := httpx.Parse(r, &request); err != nil {
			writeError(r, w, apperrors.InvalidParameter("invalid request body").WithCause(err))
			return
		}
		explanation, err := serverCtx.Admin.Explain(r.Context(), request.Subject, request.Resource, request.Action)
		if err != nil {
			writeError(r, w, mapError(err))
			return
		}
		write(r, w, http.StatusOK, explanation)
	}
}

func auditHandler(serverCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		page, _ := strconv.Atoi(query.Get("page"))
		pageSize, _ := strconv.Atoi(query.Get("pageSize"))
		result, err := serverCtx.Admin.ListAudit(r.Context(), admin.AuditQuery{
			Search: query.Get("search"), Action: query.Get("action"), Outcome: query.Get("outcome"), Page: page, PageSize: pageSize,
		})
		if err != nil {
			writeError(r, w, mapError(err))
			return
		}
		write(r, w, http.StatusOK, result)
	}
}

func systemOverviewHandler(serverCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		mysqlStatus := "ready"
		if err := serverCtx.MySQL.Ping(r.Context()); err != nil {
			mysqlStatus = "unavailable"
		}
		redisStatus := "ready"
		if err := serverCtx.Redis.Ping(r.Context()); err != nil {
			redisStatus = "unavailable"
		}
		bootstrapAvailable, _ := serverCtx.Admin.BootstrapAvailable(r.Context())
		write(r, w, http.StatusOK, map[string]any{
			"service":            serverCtx.Config.Name,
			"mysql":              mysqlStatus,
			"redis":              redisStatus,
			"authentication":     serverCtx.Authn != nil,
			"authorization":      serverCtx.Authorizer != nil,
			"admin":              serverCtx.Admin != nil,
			"metrics":            serverCtx.Config.Observability.Metrics.Enabled,
			"bootstrapAvailable": bootstrapAvailable,
			"engine":             serverCtx.Admin.EngineInfo(r.Context()),
		})
	}
}

func systemConfigurationHandler(serverCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		params := serverCtx.Admin.PasswordParams()
		write(r, w, http.StatusOK, map[string]any{
			"http": map[string]any{"host": serverCtx.Config.Host, "port": serverCtx.Config.Port, "maxBodyBytes": serverCtx.Config.HTTP.MaxBodyBytes},
			"authentication": map[string]any{
				"issuer":           serverCtx.Config.Authentication.Issuer,
				"accessTTL":        serverCtx.Config.Authentication.AccessTTL.String(),
				"refreshTTL":       serverCtx.Config.Authentication.RefreshTTL.String(),
				"sessionKeyPrefix": serverCtx.Config.Authentication.SessionKeyPrefix,
				"secretConfigured": strings.TrimSpace(serverCtx.Config.Authentication.AccessTokenSecret) != "",
			},
			"admin":         map[string]any{"enabled": serverCtx.Config.Admin.Enabled, "bootstrapTokenConfigured": strings.TrimSpace(serverCtx.Config.Admin.BootstrapToken) != ""},
			"observability": map[string]any{"metricsEnabled": serverCtx.Config.Observability.Metrics.Enabled, "metricsPath": serverCtx.Config.Observability.Metrics.Path},
			"password":      map[string]any{"memoryKiB": params.MemoryKiB, "iterations": params.Iterations, "parallelism": params.Parallelism, "saltLength": params.SaltLength, "keyLength": params.KeyLength},
		})
	}
}

func accountDTO(account identity.Account) accountResponse {
	return accountResponse{
		ID: account.ID, Username: account.Username, Email: account.Email, Phone: account.Phone,
		DisplayName: account.DisplayName, Status: account.Status, CreatedAt: account.CreatedAt, UpdatedAt: account.UpdatedAt,
	}
}

func routeParam(r *http.Request, name string) string { return strings.TrimSpace(pathvar.Vars(r)[name]) }

func actorFromRequest(r *http.Request) admin.Actor {
	authentication, _ := securityhttp.AuthenticationFromContext(r.Context())
	clientIP := r.RemoteAddr
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		clientIP = host
	}
	return admin.Actor{
		AccountID: authentication.Principal.AccountID,
		RequestID: r.Header.Get("X-Request-Id"),
		ClientIP:  clientIP,
		UserAgent: r.UserAgent(),
	}
}

func mapError(err error) error {
	switch {
	case errors.Is(err, admin.ErrNotFound), errors.Is(err, identity.ErrAccountNotFound):
		return apperrors.NotFound("resource not found")
	case errors.Is(err, admin.ErrConflict), errors.Is(err, identity.ErrIdentityConflict), errors.Is(err, admin.ErrBootstrapComplete):
		return apperrors.Conflict("resource conflict")
	case errors.Is(err, admin.ErrProtectedRole):
		return apperrors.Conflict("protected administrator state cannot be removed")
	case errors.Is(err, admin.ErrBootstrapDisabled):
		return apperrors.NotFound("admin bootstrap is unavailable")
	case errors.Is(err, identity.ErrInvalidAccountState), errors.Is(err, identity.ErrInvalidCredentials):
		return apperrors.Conflict("account state does not allow this operation")
	default:
		if strings.Contains(err.Error(), "must") || strings.Contains(err.Error(), "required") || strings.Contains(err.Error(), "invalid") || strings.Contains(err.Error(), "duplicate") {
			return apperrors.InvalidParameter(err.Error())
		}
		return apperrors.Internal(err)
	}
}

func write(r *http.Request, w http.ResponseWriter, status int, data any) {
	platformresponse.WriteJSON(r.Context(), w, status, platformresponse.Success(r.Context(), data))
}

func writeError(r *http.Request, w http.ResponseWriter, err error) {
	platformresponse.WriteError(r.Context(), w, err)
}
