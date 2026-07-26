package authn

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestServiceLoginRefreshAndLogout(t *testing.T) {
	now := time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC)
	identity := fakeIdentity{principal: Principal{AccountID: "01984f63-ec7f-7a4a-b908-33e8ff14d465", DisplayName: "Test User"}}
	codec := &fakeCodec{}
	store := newMemoryStore()
	service, err := NewService(identity, codec, store, Config{AccessTTL: 15 * time.Minute, RefreshTTL: time.Hour})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	service.now = func() time.Time { return now }

	authentication, tokens, err := service.Login(context.Background(), "tester", "correct-password")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if authentication.Principal.AccountID != identity.principal.AccountID || tokens.AccessToken == "" || tokens.RefreshToken == "" {
		t.Fatalf("unexpected login result: %#v %#v", authentication, tokens)
	}

	oldRefresh := tokens.RefreshToken
	_, refreshed, err := service.Refresh(context.Background(), oldRefresh)
	if err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}
	if refreshed.RefreshToken == oldRefresh {
		t.Fatal("refresh token was not rotated")
	}
	if _, _, err := service.Refresh(context.Background(), oldRefresh); !errors.Is(err, ErrInvalidRefresh) {
		t.Fatalf("reusing refresh token error = %v, want ErrInvalidRefresh", err)
	}

	if err := service.Logout(context.Background(), refreshed.AccessToken); err != nil {
		t.Fatalf("Logout() error = %v", err)
	}
	if _, err := service.AuthenticateAccess(context.Background(), refreshed.AccessToken); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("AuthenticateAccess() after logout error = %v, want ErrInvalidToken", err)
	}
}

func TestServiceRejectsInvalidCredentials(t *testing.T) {
	service, err := NewService(fakeIdentity{authenticateErr: ErrInvalidCredentials}, &fakeCodec{}, newMemoryStore(), Config{AccessTTL: time.Minute, RefreshTTL: time.Hour})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	if _, _, err := service.Login(context.Background(), "missing", "wrong"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("Login() error = %v, want ErrInvalidCredentials", err)
	}
}

type fakeIdentity struct {
	principal       Principal
	authenticateErr error
	resolveErr      error
}

func (f fakeIdentity) Authenticate(context.Context, string, string) (Principal, error) {
	if f.authenticateErr != nil {
		return Principal{}, f.authenticateErr
	}
	return f.principal, nil
}

func (f fakeIdentity) ResolveActive(context.Context, string) (Principal, error) {
	if f.resolveErr != nil {
		return Principal{}, f.resolveErr
	}
	return f.principal, nil
}

type fakeCodec struct {
	mu     sync.Mutex
	claims map[string]AccessClaims
	next   int
}

func (f *fakeCodec) Issue(claims AccessClaims) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.claims == nil {
		f.claims = make(map[string]AccessClaims)
	}
	f.next++
	token := "access-token-" + time.Unix(int64(f.next), 0).UTC().Format("150405")
	f.claims[token] = claims
	return token, nil
}

func (f *fakeCodec) Parse(token string) (AccessClaims, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	claims, ok := f.claims[token]
	if !ok {
		return AccessClaims{}, ErrInvalidToken
	}
	return claims, nil
}

type memoryStore struct {
	mu       sync.Mutex
	sessions map[string]Session
}

func newMemoryStore() *memoryStore {
	return &memoryStore{sessions: make(map[string]Session)}
}

func (s *memoryStore) Create(_ context.Context, session Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.sessions[session.ID]; exists {
		return ErrSessionConflict
	}
	s.sessions[session.ID] = session
	return nil
}

func (s *memoryStore) Get(_ context.Context, id string) (Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	session, ok := s.sessions[id]
	if !ok {
		return Session{}, ErrSessionNotFound
	}
	return session, nil
}

func (s *memoryStore) Rotate(_ context.Context, id, currentDigest, nextDigest string, expiresAt time.Time) (Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	session, ok := s.sessions[id]
	if !ok || session.RefreshDigest != currentDigest {
		return Session{}, ErrInvalidRefresh
	}
	session.RefreshDigest = nextDigest
	session.ExpiresAt = expiresAt
	s.sessions[id] = session
	return session, nil
}

func (s *memoryStore) Revoke(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.sessions[id]; !ok {
		return ErrSessionNotFound
	}
	delete(s.sessions, id)
	return nil
}
