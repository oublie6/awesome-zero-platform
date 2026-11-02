package wechatcode

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCodeVerificationUsesOnlyProviderOpenID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("appid") != "test-app" || r.URL.Query().Get("secret") != "server-secret-123456789" ||
			r.URL.Query().Get("js_code") != "valid-one-time-code" || r.URL.Query().Get("grant_type") != "authorization_code" {
			t.Errorf("invalid provider query")
		}
		if r.Method != http.MethodGet {
			t.Error("invalid provider method")
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"openid": "opaque-wechat-subject", "session_key": "never-visible-session-key",
		})
	}))
	defer server.Close()
	v, err := New("test-app", "server-secret-123456789")
	if err != nil {
		t.Fatal(err)
	}
	v.endpoint = server.URL
	got, err := v.Exchange(context.Background(), "valid-one-time-code")
	if err != nil || got != "opaque-wechat-subject" {
		t.Fatalf("verified subject=%q err=%v", got, err)
	}
	if _, err := v.Exchange(context.Background(), " "); !errors.Is(err, ErrInvalidCode) {
		t.Fatalf("blank code accepted: %v", err)
	}
}
func TestCodeVerificationFailsClosedOnProviderErrorAndRedirect(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("js_code") == "redirect-code" {
			http.Redirect(w, r, "https://untrusted.example/steal", http.StatusFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"errcode":40029,"errmsg":"invalid code","openid":"forged-openid"}`))
	}))
	defer server.Close()
	v, err := New("test-app", "server-secret-123456789")
	if err != nil {
		t.Fatal(err)
	}
	v.endpoint = server.URL
	for _, code := range []string{"invalid-code", "redirect-code"} {
		if got, err := v.Exchange(context.Background(), code); !errors.Is(err, ErrInvalidCode) || got != "" {
			t.Fatalf("provider error accepted: got=%q err=%v", got, err)
		}
	}
}
