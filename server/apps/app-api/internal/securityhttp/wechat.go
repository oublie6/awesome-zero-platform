package securityhttp

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/oublie6/awesome-zero-platform/server/apps/app-api/internal/svc"
	"github.com/oublie6/awesome-zero-platform/server/foundation/apperrors"
	platformresponse "github.com/oublie6/awesome-zero-platform/server/foundation/response"
	"github.com/oublie6/awesome-zero-platform/server/platform/authn"
	"github.com/oublie6/awesome-zero-platform/server/platform/authn/adapter/wechatcode"
	"github.com/oublie6/awesome-zero-platform/server/platform/identity"
	"github.com/zeromicro/go-zero/rest"
)

type wechatIdentityVerifier interface {
	Exchange(context.Context, string) (string, error)
}
type wechatAccountBinder interface {
	ResolveVerified(context.Context, string, string) (identity.Account, error)
	BeginVerified(context.Context, string, string) (string, error)
	BindPending(context.Context, string, string) error
	CreateFromPending(context.Context, string, identity.CreateAccountInput) (identity.Account, error)
	Unlink(context.Context, string, string) error
	Relink(context.Context, string, string, string) error
}
type wechatSessionIssuer interface {
	IssueVerifiedAccount(context.Context, string) (authn.Authentication, authn.TokenPair, error)
}

type wechatHandlers struct {
	verifier     wechatIdentityVerifier
	binder       wechatAccountBinder
	issuer       wechatSessionIssuer
	identities   *identity.Service
	auth         *authn.Service
	sessionAdmin authn.SessionAdministrator
	quota        *registrationQuota
}

func registerWeChatCodeLogin(server *rest.Server, ctx *svc.ServiceContext) {
	h := &wechatHandlers{quota: &registrationQuota{}}
	if ctx != nil && ctx.Identity != nil && ctx.MySQL != nil && ctx.MySQL.DB() != nil && ctx.Authn != nil {
		key, err := hex.DecodeString(os.Getenv("APP_WECHAT_SUBJECT_HMAC_KEY_HEX"))
		if err == nil && len(key) >= 32 {
			h.verifier, err = wechatcode.New(os.Getenv("APP_WECHAT_APP_ID"), os.Getenv("APP_WECHAT_APP_SECRET"))
			if err == nil {
				h.binder, err = identity.NewExternalBinding(ctx.MySQL.DB(), ctx.Identity, key)
				if err == nil {
					h.issuer = ctx.Authn
					h.identities = ctx.Identity
					h.auth = ctx.Authn
					h.sessionAdmin = ctx.SessionAdmin
				}
			}
		}
		clear(key)
	}
	server.AddRoutes([]rest.Route{
		{Method: http.MethodPost, Path: "/auth/wechat/code", Handler: h.code},
		{Method: http.MethodPost, Path: "/auth/wechat/prepare", Handler: h.prepare},
		{Method: http.MethodPost, Path: "/auth/wechat/bind", Handler: h.bind},
		{Method: http.MethodPost, Path: "/auth/wechat/register", Handler: h.register},
	})
	server.AddRoutes(rest.WithMiddleware(RequireAuthenticationMiddleware(h.auth),
		rest.Route{Method: http.MethodPost, Path: "/auth/wechat/unlink", Handler: h.unlink},
		rest.Route{Method: http.MethodPost, Path: "/auth/wechat/relink", Handler: h.relink},
	))
}
func decodeWeChat(w http.ResponseWriter, r *http.Request, target any) error {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return errors.New("unexpected trailing body")
	}
	return nil
}
func (h *wechatHandlers) allowUnauthenticated(r *http.Request) bool {
	if h == nil || h.quota == nil {
		return false
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil || net.ParseIP(host) == nil {
		host = "unknown"
	}
	// These limits intentionally do not trust attacker-controlled X-Forwarded-For.
	return h.quota.allow(host)
}
func (h *wechatHandlers) configured() bool {
	return h != nil && h.verifier != nil && h.binder != nil && h.issuer != nil && h.identities != nil
}
func (h *wechatHandlers) code(w http.ResponseWriter, r *http.Request) {
	if !h.configured() {
		platformresponse.WriteError(r.Context(), w, apperrors.Internal(errors.New("WeChat provider unavailable")))
		return
	}
	var req struct {
		Code string `json:"code"`
	}
	if decodeWeChat(w, r, &req) != nil || len(req.Code) < 4 || len(req.Code) > 128 || strings.TrimSpace(req.Code) != req.Code || strings.ContainsAny(req.Code, "\r\n") {
		platformresponse.WriteError(r.Context(), w, apperrors.InvalidParameter("invalid WeChat code"))
		return
	}
	subject, err := h.verifier.Exchange(r.Context(), req.Code)
	if err != nil {
		platformresponse.WriteError(r.Context(), w, apperrors.Unauthorized("WeChat identity verification failed"))
		return
	}
	account, err := h.binder.ResolveVerified(r.Context(), "wechat", subject)
	if err == nil {
		authentication, tokens, err := h.issuer.IssueVerifiedAccount(r.Context(), account.ID)
		if err != nil {
			platformresponse.WriteError(r.Context(), w, mapAuthenticationError(err, false))
			return
		}
		writeTokens(r.Context(), w, authentication, tokens)
		return
	}
	if !errors.Is(err, identity.ErrExternalUnbound) {
		platformresponse.WriteError(r.Context(), w, apperrors.Unauthorized("verified WeChat identity unavailable"))
		return
	}
	proof, err := h.binder.BeginVerified(r.Context(), "wechat", subject)
	if err != nil {
		platformresponse.WriteError(r.Context(), w, apperrors.Internal(err))
		return
	}
	platformresponse.WriteJSON(r.Context(), w, http.StatusOK, platformresponse.Success(r.Context(), map[string]any{
		"status": "binding_required", "pendingToken": proof, "expiresInSeconds": 300,
	}))
}

// A verified code can also produce a one-purpose proof for an already
// linked identity, required for re-link to another authenticated AccountID.
func (h *wechatHandlers) prepare(w http.ResponseWriter, r *http.Request) {
	if !h.configured() {
		platformresponse.WriteError(r.Context(), w, apperrors.Internal(errors.New("WeChat provider unavailable")))
		return
	}
	var req struct {
		Code string `json:"code"`
	}
	if decodeWeChat(w, r, &req) != nil || len(req.Code) < 4 || len(req.Code) > 128 || strings.TrimSpace(req.Code) != req.Code || strings.ContainsAny(req.Code, "\r\n") {
		platformresponse.WriteError(r.Context(), w, apperrors.InvalidParameter("invalid WeChat code"))
		return
	}
	subject, err := h.verifier.Exchange(r.Context(), req.Code)
	if err != nil {
		platformresponse.WriteError(r.Context(), w, apperrors.Unauthorized("WeChat identity verification failed"))
		return
	}
	proof, err := h.binder.BeginVerified(r.Context(), "wechat", subject)
	if err != nil {
		platformresponse.WriteError(r.Context(), w, apperrors.Internal(err))
		return
	}
	platformresponse.WriteJSON(r.Context(), w, http.StatusOK, platformresponse.Success(r.Context(), map[string]any{"pendingToken": proof, "expiresInSeconds": 300}))
}
func (h *wechatHandlers) existing(ctx context.Context, identifier, password string) (identity.Account, error) {
	if h == nil || h.identities == nil {
		return identity.Account{}, identity.ErrInvalidCredentials
	}
	var (
		account identity.Account
		err     error
	)
	switch {
	case strings.HasPrefix(identifier, "+"):
		account, err = h.identities.FindAccountByPhoneFresh(ctx, identifier)
	case strings.Contains(identifier, "@"):
		account, err = h.identities.FindAccountByEmailFresh(ctx, identifier)
	default:
		account, err = h.identities.FindAccountByUsernameFresh(ctx, identifier)
	}
	if err != nil || account.Status != identity.StatusActive {
		return identity.Account{}, identity.ErrInvalidCredentials
	}
	if err = h.identities.VerifyPassword(ctx, account.ID, password); err != nil {
		return identity.Account{}, identity.ErrInvalidCredentials
	}
	return account, nil
}
func (h *wechatHandlers) bind(w http.ResponseWriter, r *http.Request) {
	if !h.configured() {
		platformresponse.WriteError(r.Context(), w, apperrors.Internal(errors.New("WeChat provider unavailable")))
		return
	}
	if !h.allowUnauthenticated(r) {
		platformresponse.WriteError(r.Context(), w, apperrors.Forbidden("binding temporarily limited"))
		return
	}
	var req struct {
		PendingToken string `json:"pendingToken"`
		Identifier   string `json:"identifier"`
		Password     string `json:"password"`
	}
	if decodeWeChat(w, r, &req) != nil {
		platformresponse.WriteError(r.Context(), w, apperrors.InvalidParameter("invalid binding request"))
		return
	}
	account, err := h.existing(r.Context(), strings.TrimSpace(req.Identifier), req.Password)
	if err != nil {
		platformresponse.WriteError(r.Context(), w, apperrors.Unauthorized("invalid account credentials"))
		return
	}
	if err = h.binder.BindPending(r.Context(), req.PendingToken, account.ID); err != nil {
		writeBindingError(r.Context(), w, err)
		return
	}
	// The client now obtains its own session through a fresh wx.login code.
	platformresponse.WriteJSON(r.Context(), w, http.StatusOK, platformresponse.Success(r.Context(), map[string]any{"bound": true, "account": accountResponse{ID: account.ID, DisplayName: account.DisplayName}}))
}
func (h *wechatHandlers) register(w http.ResponseWriter, r *http.Request) {
	if !h.configured() {
		platformresponse.WriteError(r.Context(), w, apperrors.Internal(errors.New("WeChat provider unavailable")))
		return
	}
	if os.Getenv("APP_PUBLIC_REGISTRATION_ENABLED") != "true" {
		platformresponse.WriteError(r.Context(), w, apperrors.Forbidden("test account registration disabled"))
		return
	}
	if !h.allowUnauthenticated(r) {
		platformresponse.WriteError(r.Context(), w, apperrors.Forbidden("registration temporarily limited"))
		return
	}
	var req struct {
		PendingToken string `json:"pendingToken"`
		Username     string `json:"username"`
		Email        string `json:"email"`
		DisplayName  string `json:"displayName"`
		Password     string `json:"password"`
	}
	if decodeWeChat(w, r, &req) != nil || strings.TrimSpace(req.Email) == "" || strings.TrimSpace(req.Username) == "" {
		platformresponse.WriteError(r.Context(), w, apperrors.InvalidParameter("username and email required"))
		return
	}
	account, err := h.binder.CreateFromPending(r.Context(), req.PendingToken, identity.CreateAccountInput{
		Identity: identity.Identity{Username: req.Username, Email: req.Email}, DisplayName: req.DisplayName, Password: req.Password,
		Status: identity.StatusActive,
	})
	if err != nil {
		writeBindingError(r.Context(), w, err)
		return
	}
	platformresponse.WriteJSON(r.Context(), w, http.StatusCreated, platformresponse.Success(r.Context(), map[string]any{"bound": true, "emailVerified": false, "account": accountResponse{ID: account.ID, DisplayName: account.DisplayName}}))
}
func (h *wechatHandlers) unlink(w http.ResponseWriter, r *http.Request) {
	if !h.configured() {
		platformresponse.WriteError(r.Context(), w, apperrors.Internal(errors.New("WeChat provider unavailable")))
		return
	}
	principal, ok := AuthenticationFromContext(r.Context())
	if !ok {
		platformresponse.WriteError(r.Context(), w, apperrors.Unauthorized("authentication required"))
		return
	}
	var req struct {
		Password string `json:"password"`
	}
	if decodeWeChat(w, r, &req) != nil || req.Password == "" {
		platformresponse.WriteError(r.Context(), w, apperrors.InvalidParameter("password required"))
		return
	}
	if err := h.identities.VerifyPassword(r.Context(), principal.Principal.AccountID, req.Password); err != nil {
		platformresponse.WriteError(r.Context(), w, apperrors.Unauthorized("reauthentication required"))
		return
	}
	// Revoke existing sessions before changing the provider mapping. If
	// the session store is unavailable the binding must remain unchanged.
	if h.sessionAdmin == nil {
		platformresponse.WriteError(r.Context(), w, apperrors.Internal(errors.New("session administrator unavailable")))
		return
	}
	if _, err := h.sessionAdmin.RevokeByAccount(r.Context(), principal.Principal.AccountID); err != nil {
		platformresponse.WriteError(r.Context(), w, apperrors.Internal(err))
		return
	}
	if err := h.binder.Unlink(r.Context(), "wechat", principal.Principal.AccountID); err != nil {
		writeBindingError(r.Context(), w, err)
		return
	}
	platformresponse.WriteJSON(r.Context(), w, http.StatusOK, platformresponse.Success(r.Context(), map[string]any{"unlinked": true}))
}
func (h *wechatHandlers) relink(w http.ResponseWriter, r *http.Request) {
	if !h.configured() {
		platformresponse.WriteError(r.Context(), w, apperrors.Internal(errors.New("WeChat provider unavailable")))
		return
	}
	authenticated, ok := AuthenticationFromContext(r.Context())
	if !ok {
		platformresponse.WriteError(r.Context(), w, apperrors.Unauthorized("authentication required"))
		return
	}
	if !h.allowUnauthenticated(r) {
		platformresponse.WriteError(r.Context(), w, apperrors.Forbidden("relink temporarily limited"))
		return
	}
	var req struct {
		PendingToken     string `json:"pendingToken"`
		SourceIdentifier string `json:"sourceIdentifier"`
		SourcePassword   string `json:"sourcePassword"`
		TargetIdentifier string `json:"targetIdentifier"`
		TargetPassword   string `json:"targetPassword"`
	}
	if decodeWeChat(w, r, &req) != nil {
		platformresponse.WriteError(r.Context(), w, apperrors.InvalidParameter("invalid relink request"))
		return
	}
	source, err := h.existing(r.Context(), strings.TrimSpace(req.SourceIdentifier), req.SourcePassword)
	if err != nil {
		platformresponse.WriteError(r.Context(), w, apperrors.Unauthorized("source account verification failed"))
		return
	}
	if authenticated.Principal.AccountID != source.ID {
		platformresponse.WriteError(r.Context(), w, apperrors.Unauthorized("source account session required"))
		return
	}
	target, err := h.existing(r.Context(), strings.TrimSpace(req.TargetIdentifier), req.TargetPassword)
	if err != nil {
		platformresponse.WriteError(r.Context(), w, apperrors.Unauthorized("target account verification failed"))
		return
	}
	if h.sessionAdmin == nil {
		platformresponse.WriteError(r.Context(), w, apperrors.Internal(errors.New("session administrator unavailable")))
		return
	}
	// Revoke the two old principals first, avoiding a successful relink
	// with still-live sessions if Redis is temporarily unavailable.
	if _, err := h.sessionAdmin.RevokeByAccount(r.Context(), source.ID); err != nil {
		platformresponse.WriteError(r.Context(), w, apperrors.Internal(err))
		return
	}
	if _, err := h.sessionAdmin.RevokeByAccount(r.Context(), target.ID); err != nil {
		platformresponse.WriteError(r.Context(), w, apperrors.Internal(err))
		return
	}
	if err = h.binder.Relink(r.Context(), req.PendingToken, source.ID, target.ID); err != nil {
		writeBindingError(r.Context(), w, err)
		return
	}
	platformresponse.WriteJSON(r.Context(), w, http.StatusOK, platformresponse.Success(r.Context(), map[string]any{"relinked": true}))
}
func writeBindingError(ctx context.Context, w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, identity.ErrExternalConflict), errors.Is(err, identity.ErrIdentityConflict):
		platformresponse.WriteError(ctx, w, apperrors.InvalidParameter("identity already linked or unavailable"))
	case errors.Is(err, identity.ErrPendingInvalid), errors.Is(err, identity.ErrExternalUnbound):
		platformresponse.WriteError(ctx, w, apperrors.InvalidParameter("verification expired or unavailable"))
	case errors.Is(err, identity.ErrInvalidAccountState):
		platformresponse.WriteError(ctx, w, apperrors.Unauthorized("account unavailable"))
	default:
		platformresponse.WriteError(ctx, w, apperrors.Internal(err))
	}
}
