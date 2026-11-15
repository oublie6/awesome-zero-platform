package securityhttp

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/oublie6/awesome-zero-platform/server/platform/authn"
	"github.com/oublie6/awesome-zero-platform/server/platform/identity"
)

type stubWeChatVerifier struct {
	subject string
	err     error
}

func (s stubWeChatVerifier) Exchange(_ context.Context, _ string) (string, error) {
	return s.subject, s.err
}

type stubWeChatBinder struct {
	account identity.Account
	err     error
	pending string
	started int
}

func (s *stubWeChatBinder) ResolveVerified(context.Context, string, string) (identity.Account, error) {
	return s.account, s.err
}
func (s *stubWeChatBinder) BeginVerified(context.Context, string, string) (string, error) {
	s.started++
	return s.pending, nil
}
func (s *stubWeChatBinder) BindPending(context.Context, string, string) error { return nil }
func (s *stubWeChatBinder) CreateFromPending(context.Context, string, identity.CreateAccountInput) (identity.Account, error) {
	return identity.Account{}, nil
}
func (s *stubWeChatBinder) Unlink(context.Context, string, string) error         { return nil }
func (s *stubWeChatBinder) Relink(context.Context, string, string, string) error { return nil }

type stubWeChatIssuer struct{ issued int }

func (s *stubWeChatIssuer) IssueVerifiedAccount(_ context.Context, id string) (authn.Authentication, authn.TokenPair, error) {
	s.issued++
	return authn.Authentication{Principal: authn.Principal{AccountID: id, DisplayName: "Player"}},
		authn.TokenPair{AccessToken: "server-token", RefreshToken: "refresh-token"}, nil
}
func postWeChat(handler http.HandlerFunc, path, body string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	w := httptest.NewRecorder()
	handler(w, r)
	return w
}
func TestWechatUnboundCodeRequiresAccountChoice(t *testing.T) {
	binder := &stubWeChatBinder{err: identity.ErrExternalUnbound, pending: strings.Repeat("p", 43)}
	issuer := &stubWeChatIssuer{}
	h := &wechatHandlers{verifier: stubWeChatVerifier{subject: "verified-openid"}, binder: binder, issuer: issuer, identities: &identity.Service{}}
	w := postWeChat(h.code, "/auth/wechat/code", `{"code":"verified-code"}`)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "binding_required") {
		t.Fatalf("unbound response: %d %s", w.Code, w.Body)
	}
	if strings.Contains(w.Body.String(), "accessToken") {
		t.Fatal("unbound provider identity got access token")
	}
	if binder.started != 1 || issuer.issued != 0 {
		t.Fatal("unbound must create only pending proof")
	}
}
func TestWechatBoundCodePreservesStableAccountID(t *testing.T) {
	binder := &stubWeChatBinder{account: identity.Account{ID: "server-account-id", Status: identity.StatusActive}}
	issuer := &stubWeChatIssuer{}
	h := &wechatHandlers{verifier: stubWeChatVerifier{subject: "verified-openid"}, binder: binder, issuer: issuer, identities: &identity.Service{}}
	w := postWeChat(h.code, "/auth/wechat/code", `{"code":"verified-code"}`)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "server-token") {
		t.Fatalf("bound response: %d %s", w.Code, w.Body)
	}
	if issuer.issued != 1 || binder.started != 0 {
		t.Fatal("existing binding must login without new account")
	}
}
func TestWechatRejectedCodeNeverBeginsBinding(t *testing.T) {
	binder := &stubWeChatBinder{err: identity.ErrExternalUnbound}
	issuer := &stubWeChatIssuer{}
	h := &wechatHandlers{verifier: stubWeChatVerifier{err: errors.New("invalid code")}, binder: binder, issuer: issuer, identities: &identity.Service{}}
	w := postWeChat(h.code, "/auth/wechat/code", `{"code":"expired-code"}`)
	if w.Code != http.StatusUnauthorized || issuer.issued != 0 || binder.started != 0 {
		t.Fatalf("verification bypass: %d", w.Code)
	}
}
func TestWechatNewAccountRegistrationDisabledByDefault(t *testing.T) {
	t.Setenv("APP_PUBLIC_REGISTRATION_ENABLED", "false")
	h := &wechatHandlers{verifier: stubWeChatVerifier{}, binder: &stubWeChatBinder{}, issuer: &stubWeChatIssuer{}, identities: &identity.Service{}}
	w := postWeChat(h.register, "/auth/wechat/register", `{"pendingToken":"xyz"}`)
	if w.Code != http.StatusForbidden {
		t.Fatalf("public unverified registration should fail: %d", w.Code)
	}
}

func TestWechatCodeRejectsEmbeddedNewline(t *testing.T) {
	binder := &stubWeChatBinder{err: identity.ErrExternalUnbound}
	issuer := &stubWeChatIssuer{}
	h := &wechatHandlers{verifier: stubWeChatVerifier{subject: "verified-openid"}, binder: binder, issuer: issuer, identities: &identity.Service{}}
	w := postWeChat(h.code, "/auth/wechat/code", `{"code":"bad\ncode"}`)
	if w.Code != http.StatusBadRequest || issuer.issued != 0 || binder.started != 0 {
		t.Fatalf("embedded newline code was accepted: status=%d", w.Code)
	}
}
