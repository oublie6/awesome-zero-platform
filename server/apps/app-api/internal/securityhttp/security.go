package securityhttp

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/oublie6/awesome-zero-platform/server/apps/app-api/internal/svc"
	"github.com/oublie6/awesome-zero-platform/server/foundation/apperrors"
	platformresponse "github.com/oublie6/awesome-zero-platform/server/foundation/response"
	"github.com/oublie6/awesome-zero-platform/server/platform/authn"
	"github.com/oublie6/awesome-zero-platform/server/platform/authz"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/rest/httpx"
)

type contextKey struct{}

type loginRequest struct {
	Identifier string `json:"identifier"`
	Password   string `json:"password"`
}

type refreshRequest struct {
	RefreshToken string `json:"refreshToken"`
}

type tokenResponse struct {
	AccessToken      string          `json:"accessToken"`
	RefreshToken     string          `json:"refreshToken"`
	TokenType        string          `json:"tokenType"`
	AccessExpiresAt  time.Time       `json:"accessExpiresAt"`
	RefreshExpiresAt time.Time       `json:"refreshExpiresAt"`
	Account          accountResponse `json:"account"`
}

type accountResponse struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
}

type sessionResponse struct {
	SessionID string          `json:"sessionId"`
	ExpiresAt time.Time       `json:"expiresAt"`
	Account   accountResponse `json:"account"`
}

func Register(server *rest.Server, serverCtx *svc.ServiceContext) {
	server.AddRoutes([]rest.Route{
		{Method: http.MethodPost, Path: "/auth/login", Handler: loginHandler(serverCtx)},
		{Method: http.MethodPost, Path: "/auth/refresh", Handler: refreshHandler(serverCtx)},
		{Method: http.MethodPost, Path: "/auth/logout", Handler: RequireAuthentication(serverCtx.Authn, logoutHandler(serverCtx))},
		{Method: http.MethodGet, Path: "/auth/session", Handler: RequireAuthentication(serverCtx.Authn, sessionHandler())},
	})
}

func loginHandler(serverCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if serverCtx.Authn == nil {
			platformresponse.WriteError(r.Context(), w, apperrors.Internal(errors.New("authentication service is unavailable")))
			return
		}
		var request loginRequest
		if err := httpx.Parse(r, &request); err != nil {
			platformresponse.WriteError(r.Context(), w, apperrors.InvalidParameter("invalid request body").WithCause(err))
			return
		}
		if strings.TrimSpace(request.Identifier) == "" || request.Password == "" {
			platformresponse.WriteError(r.Context(), w, apperrors.InvalidParameter("identifier and password are required"))
			return
		}

		authentication, tokens, err := serverCtx.Authn.Login(r.Context(), request.Identifier, request.Password)
		if err != nil {
			platformresponse.WriteError(r.Context(), w, mapAuthenticationError(err, true))
			return
		}
		writeTokens(r.Context(), w, authentication, tokens)
	}
}

func refreshHandler(serverCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if serverCtx.Authn == nil {
			platformresponse.WriteError(r.Context(), w, apperrors.Internal(errors.New("authentication service is unavailable")))
			return
		}
		var request refreshRequest
		if err := httpx.Parse(r, &request); err != nil {
			platformresponse.WriteError(r.Context(), w, apperrors.InvalidParameter("invalid request body").WithCause(err))
			return
		}
		if strings.TrimSpace(request.RefreshToken) == "" {
			platformresponse.WriteError(r.Context(), w, apperrors.InvalidParameter("refresh token is required"))
			return
		}

		authentication, tokens, err := serverCtx.Authn.Refresh(r.Context(), request.RefreshToken)
		if err != nil {
			platformresponse.WriteError(r.Context(), w, mapAuthenticationError(err, false))
			return
		}
		writeTokens(r.Context(), w, authentication, tokens)
	}
}

func logoutHandler(serverCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := serverCtx.Authn.Logout(r.Context(), bearerToken(r)); err != nil {
			platformresponse.WriteError(r.Context(), w, mapAuthenticationError(err, false))
			return
		}
		platformresponse.WriteJSON(r.Context(), w, http.StatusOK, platformresponse.Success(r.Context(), map[string]any{"loggedOut": true}))
	}
}

func sessionHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authentication, ok := AuthenticationFromContext(r.Context())
		if !ok {
			platformresponse.WriteError(r.Context(), w, apperrors.Unauthorized("authentication required"))
			return
		}
		platformresponse.WriteJSON(r.Context(), w, http.StatusOK, platformresponse.Success(r.Context(), sessionResponse{
			SessionID: authentication.SessionID,
			ExpiresAt: authentication.ExpiresAt,
			Account:   accountResponse{ID: authentication.Principal.AccountID, DisplayName: authentication.Principal.DisplayName},
		}))
	}
}

func RequireAuthentication(service *authn.Service, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if service == nil {
			platformresponse.WriteError(r.Context(), w, apperrors.Internal(errors.New("authentication service is unavailable")))
			return
		}
		token := bearerToken(r)
		if token == "" {
			platformresponse.WriteError(r.Context(), w, apperrors.Unauthorized("authentication required"))
			return
		}
		authentication, err := service.AuthenticateAccess(r.Context(), token)
		if err != nil {
			platformresponse.WriteError(r.Context(), w, mapAuthenticationError(err, false))
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), contextKey{}, authentication)))
	}
}

func RequireAuthorization(authorizer authz.Authorizer, resource, action string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authentication, ok := AuthenticationFromContext(r.Context())
		if !ok {
			platformresponse.WriteError(r.Context(), w, apperrors.Unauthorized("authentication required"))
			return
		}
		if authorizer == nil {
			platformresponse.WriteError(r.Context(), w, apperrors.Internal(errors.New("authorization service is unavailable")))
			return
		}
		allowed, err := authorizer.Enforce(r.Context(), authentication.Principal.AccountID, resource, action)
		if err != nil {
			platformresponse.WriteError(r.Context(), w, apperrors.Internal(err))
			return
		}
		if !allowed {
			platformresponse.WriteError(r.Context(), w, apperrors.Forbidden("permission denied"))
			return
		}
		next(w, r)
	}
}

func AuthenticationFromContext(ctx context.Context) (authn.Authentication, bool) {
	authentication, ok := ctx.Value(contextKey{}).(authn.Authentication)
	return authentication, ok
}

func bearerToken(r *http.Request) string {
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	scheme, value, ok := strings.Cut(header, " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") {
		return ""
	}
	return strings.TrimSpace(value)
}

func writeTokens(ctx context.Context, w http.ResponseWriter, authentication authn.Authentication, tokens authn.TokenPair) {
	platformresponse.WriteJSON(ctx, w, http.StatusOK, platformresponse.Success(ctx, tokenResponse{
		AccessToken:      tokens.AccessToken,
		RefreshToken:     tokens.RefreshToken,
		TokenType:        "Bearer",
		AccessExpiresAt:  tokens.AccessExpiresAt,
		RefreshExpiresAt: tokens.RefreshExpiresAt,
		Account:          accountResponse{ID: authentication.Principal.AccountID, DisplayName: authentication.Principal.DisplayName},
	}))
}

func mapAuthenticationError(err error, login bool) error {
	switch {
	case errors.Is(err, authn.ErrInvalidCredentials):
		return apperrors.Unauthorized("invalid credentials")
	case errors.Is(err, authn.ErrInvalidToken), errors.Is(err, authn.ErrInvalidRefresh), errors.Is(err, authn.ErrSessionNotFound), errors.Is(err, authn.ErrAccountUnavailable):
		if login {
			return apperrors.Unauthorized("invalid credentials")
		}
		return apperrors.Unauthorized("authentication required")
	default:
		return apperrors.Internal(err)
	}
}
