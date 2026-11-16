package authn

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

// A simulated provider association: the authenticator can sign a token after
// unlink has committed, but authorization must never accept that stale token.
type toggleExternalVerifier struct {
	mu    sync.RWMutex
	bound map[string]string
	fail  bool
}

func (v *toggleExternalVerifier) VerifyExternalSession(_ context.Context, accountID, provider, digest string) (bool, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()
	if v.fail {
		return false, errors.New("binding database unavailable")
	}
	return provider == "wechat" && v.bound[digest] == accountID, nil
}
func (v *toggleExternalVerifier) set(digest, id string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if id == "" {
		delete(v.bound, digest)
	} else {
		v.bound[digest] = id
	}
}
func TestExternalSessionUnlinkAndRelinkCannotReuseAccessOrRefresh(t *testing.T) {
	ctx := context.Background()
	accountID := "01984f63-ec7f-7a4a-b908-33e8ff14d465"
	otherID := "01984f63-ec7f-7a4a-b908-33e8ff14d466"
	digest := strings.Repeat("a", 64)
	identity := fakeIdentity{principal: Principal{AccountID: accountID, DisplayName: "Player"}}
	store := newMemoryStore()
	service, err := NewService(identity, &fakeCodec{}, store, Config{AccessTTL: 15 * time.Minute, RefreshTTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	verifier := &toggleExternalVerifier{bound: map[string]string{digest: accountID}}
	if err = service.ConfigureExternalBindingVerifier(verifier); err != nil {
		t.Fatal(err)
	}
	auth, tokens, err := service.IssueVerifiedExternalAccount(ctx, accountID, "wechat", digest)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = service.AuthenticateAccess(ctx, tokens.AccessToken); err != nil {
		t.Fatalf("linked token rejected: %v", err)
	}
	session, err := store.Get(ctx, auth.SessionID)
	if err != nil {
		t.Fatal(err)
	}
	if session.ExternalProvider != "wechat" || session.ExternalSubjectDigest != digest {
		t.Fatal("external session not tagged with verified provider digest")
	}
	// The same account still exists and remains active; only the WeChat
	// provider mapping changes (Unlink then Relink).
	verifier.set(digest, "")
	if _, err = service.AuthenticateAccess(ctx, tokens.AccessToken); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("unlinked access allowed: %v", err)
	}
	if _, _, err = service.Refresh(ctx, tokens.RefreshToken); !errors.Is(err, ErrInvalidRefresh) {
		t.Fatalf("unlinked refresh allowed: %v", err)
	}
	verifier.set(digest, otherID)
	if _, _, err = service.IssueVerifiedExternalAccount(ctx, accountID, "wechat", digest); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("old AccountID could issue after relink: %v", err)
	}
}
func TestExternalSessionDatabaseOutageAndMissingVerifierFailClosed(t *testing.T) {
	ctx := context.Background()
	accountID := "01984f63-ec7f-7a4a-b908-33e8ff14d465"
	digest := strings.Repeat("b", 64)
	service, err := NewService(fakeIdentity{principal: Principal{AccountID: accountID}}, &fakeCodec{}, newMemoryStore(), Config{})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err = service.IssueVerifiedExternalAccount(ctx, accountID, "wechat", digest); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("missing verifier accepted: %v", err)
	}
	verifier := &toggleExternalVerifier{bound: map[string]string{digest: accountID}}
	if err = service.ConfigureExternalBindingVerifier(verifier); err != nil {
		t.Fatal(err)
	}
	_, tokens, err := service.IssueVerifiedExternalAccount(ctx, accountID, "wechat", digest)
	if err != nil {
		t.Fatal(err)
	}
	verifier.mu.Lock()
	verifier.fail = true
	verifier.mu.Unlock()
	if _, err = service.AuthenticateAccess(ctx, tokens.AccessToken); err == nil {
		t.Fatal("database outage accepted external access")
	}
	if _, _, err = service.Refresh(ctx, tokens.RefreshToken); err == nil {
		t.Fatal("database outage accepted external refresh")
	}
}

type revokeAfterCreateStore struct {
	*memoryStore
	verifier *toggleExternalVerifier
	digest   string
}

func (s *revokeAfterCreateStore) Create(ctx context.Context, session Session) error {
	if err := s.memoryStore.Create(ctx, session); err != nil {
		return err
	}
	// Deterministically simulate commit of Unlink after Redis session
	// creation but before the new provider login handler can respond.
	s.verifier.set(s.digest, "")
	return nil
}
func TestExternalIssueConcurrentWithUnlinkReturnsNoBearer(t *testing.T) {
	ctx := context.Background()
	accountID := "01984f63-ec7f-7a4a-b908-33e8ff14d465"
	digest := strings.Repeat("c", 64)
	verifier := &toggleExternalVerifier{bound: map[string]string{digest: accountID}}
	store := &revokeAfterCreateStore{memoryStore: newMemoryStore(), verifier: verifier, digest: digest}
	service, err := NewService(fakeIdentity{principal: Principal{AccountID: accountID}}, &fakeCodec{}, store, Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err = service.ConfigureExternalBindingVerifier(verifier); err != nil {
		t.Fatal(err)
	}
	auth, tokens, err := service.IssueVerifiedExternalAccount(ctx, accountID, "wechat", digest)
	if !errors.Is(err, ErrInvalidToken) || auth.SessionID != "" || tokens.AccessToken != "" {
		t.Fatalf("racing unlink leaked credentials: %v / %#v / %#v", err, auth, tokens)
	}
	store.mu.Lock()
	n := len(store.sessions)
	store.mu.Unlock()
	if n != 0 {
		t.Fatalf("racing login left %d live sessions", n)
	}
}
func TestPasswordSessionsDoNotRequireProviderMapping(t *testing.T) {
	ctx := context.Background()
	accountID := "01984f63-ec7f-7a4a-b908-33e8ff14d465"
	service, err := NewService(fakeIdentity{principal: Principal{AccountID: accountID}}, &fakeCodec{}, newMemoryStore(), Config{})
	if err != nil {
		t.Fatal(err)
	}
	verifier := &toggleExternalVerifier{bound: map[string]string{}}
	if err = service.ConfigureExternalBindingVerifier(verifier); err != nil {
		t.Fatal(err)
	}
	_, tokens, err := service.Login(ctx, "account", "password")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = service.AuthenticateAccess(ctx, tokens.AccessToken); err != nil {
		t.Fatalf("correct platform password account lost access on WeChat unlink: %v", err)
	}
}
