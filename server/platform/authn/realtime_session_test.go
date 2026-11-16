package authn

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestValidateRealtimeSessionTracksActiveSessionLifetime(t *testing.T) {
	ctx := context.Background()
	accountID := "01984f63-ec7f-7a4a-b908-33e8ff14d465"
	store := newMemoryStore()
	svc, err := NewService(fakeIdentity{principal: Principal{AccountID: accountID}}, &fakeCodec{}, store, Config{
		AccessTTL: 5 * time.Minute, RefreshTTL: time.Hour,
	})
	if err != nil {
		t.Fatal(err)
	}
	auth, tokens, err := svc.Login(ctx, "alice", "password")
	if err != nil {
		t.Fatal(err)
	}
	if err = svc.ValidateRealtimeSession(ctx, auth.SessionID, accountID); err != nil {
		t.Fatal(err)
	}
	if err = svc.ValidateRealtimeSession(ctx, auth.SessionID, "different"); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("another account accepted this session: %v", err)
	}
	if err = svc.ValidateRealtimeSession(ctx, "unknown", accountID); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("unknown session: %v", err)
	}
	// Access JWT expiry does not disconnect an otherwise valid realtime
	// session; it is still bounded by the current server-side session TTL.
	svc.now = func() time.Time { return auth.ExpiresAt.Add(time.Second) }
	if err = svc.ValidateRealtimeSession(ctx, auth.SessionID, accountID); err != nil {
		t.Fatalf("realtime session improperly tied to access JWT TTL: %v", err)
	}
	svc.now = time.Now
	if err = svc.Logout(ctx, tokens.AccessToken); err != nil {
		t.Fatal(err)
	}
	if err = svc.ValidateRealtimeSession(ctx, auth.SessionID, accountID); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("logged out session remained valid: %v", err)
	}
}

func TestValidateRealtimeSessionDetectsProviderCutover(t *testing.T) {
	ctx := context.Background()
	accountID := "01984f63-ec7f-7a4a-b908-33e8ff14d465"
	digest := strings.Repeat("d", 64)
	verifier := &toggleExternalVerifier{bound: map[string]string{digest: accountID}}
	svc, err := NewService(fakeIdentity{principal: Principal{AccountID: accountID}}, &fakeCodec{}, newMemoryStore(), Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err = svc.ConfigureExternalBindingVerifier(verifier); err != nil {
		t.Fatal(err)
	}
	auth, _, err := svc.IssueVerifiedExternalAccount(ctx, accountID, "wechat", digest)
	if err != nil {
		t.Fatal(err)
	}
	if err = svc.ValidateRealtimeSession(ctx, auth.SessionID, accountID); err != nil {
		t.Fatal(err)
	}
	verifier.set(digest, "")
	if err = svc.ValidateRealtimeSession(ctx, auth.SessionID, accountID); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("unlinked provider still authorizes websocket: %v", err)
	}
}

func TestValidateRealtimeSessionFailsClosedWithoutBindingDatabase(t *testing.T) {
	ctx := context.Background()
	accountID := "01984f63-ec7f-7a4a-b908-33e8ff14d465"
	digest := strings.Repeat("f", 64)
	verifier := &toggleExternalVerifier{bound: map[string]string{digest: accountID}}
	svc, err := NewService(fakeIdentity{principal: Principal{AccountID: accountID}}, &fakeCodec{}, newMemoryStore(), Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err = svc.ConfigureExternalBindingVerifier(verifier); err != nil {
		t.Fatal(err)
	}
	auth, _, err := svc.IssueVerifiedExternalAccount(ctx, accountID, "wechat", digest)
	if err != nil {
		t.Fatal(err)
	}
	verifier.mu.Lock()
	verifier.fail = true
	verifier.mu.Unlock()
	if err = svc.ValidateRealtimeSession(ctx, auth.SessionID, accountID); err == nil {
		t.Fatal("unavailable provider binding data authorized websocket")
	}
}

func TestValidateRealtimeSessionRejectsOldPasswordCredential(t *testing.T) {
	ctx := context.Background()
	id := &credentialIdentity{version: "verified-password-version"}
	svc, err := NewService(id, &fakeCodec{}, newMemoryStore(), Config{})
	if err != nil {
		t.Fatal(err)
	}
	auth, _, err := svc.Login(ctx, "alice", "current-password")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.ValidateRealtimeSession(ctx, auth.SessionID, auth.Principal.AccountID); err != nil {
		t.Fatal(err)
	}
	id.mu.Lock()
	id.version = "rotated-password-version"
	id.mu.Unlock()
	if err := svc.ValidateRealtimeSession(ctx, auth.SessionID, auth.Principal.AccountID); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("password rotation left old WebSocket session valid: %v", err)
	}
}
func TestValidateRealtimeSessionRejectsDisabledAccount(t *testing.T) {
	ctx := context.Background()
	id := "01984f63-ec7f-7a4a-b908-33e8ff14d465"
	store := newMemoryStore()
	svc, err := NewService(fakeIdentity{principal: Principal{AccountID: id}, resolveErr: ErrAccountUnavailable},
		&fakeCodec{}, store, Config{})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	session := Session{ID: "disabled-session", AccountID: id, CreatedAt: now, ExpiresAt: now.Add(time.Hour)}
	if err := store.Create(ctx, session); err != nil {
		t.Fatal(err)
	}
	if err := svc.ValidateRealtimeSession(ctx, session.ID, id); !errors.Is(err, ErrAccountUnavailable) {
		t.Fatalf("disabled account authorized WebSocket: %v", err)
	}
}

type fastRealtimeIdentity struct{
    fakeIdentity
    calls int
    allow bool
    fail error
}
func (f *fastRealtimeIdentity) CheckRealtimeAuthority(_ context.Context,accountID,provider,digest,credential string)(bool,error){
    f.calls++
    if f.fail!=nil{return false,f.fail}
    return f.allow,nil
}
type fastRealtimeStore struct{
    *memoryStore
    optimizedCalls int
}
func(s *fastRealtimeStore) GetForAccount(ctx context.Context,sessionID,accountID string)(Session,error){
    s.optimizedCalls++
    session,err:=s.memoryStore.Get(ctx,sessionID)
    if err!=nil{return Session{},err}
    if session.AccountID!=accountID{return Session{},ErrSessionNotFound}
    return session,nil
}
func TestRealtimeUsesCombinedAuthorityAndSessionRead(t *testing.T){
    ctx:=context.Background()
    id:="01984f63-ec7f-7a4a-b908-33e8ff14d465"
    identity:=&fastRealtimeIdentity{fakeIdentity:fakeIdentity{principal:Principal{AccountID:id}},allow:true}
    store:=&fastRealtimeStore{memoryStore:newMemoryStore()}
    svc,err:=NewService(identity,&fakeCodec{},store,Config{})
    if err!=nil{t.Fatal(err)}
    auth,_,err:=svc.Login(ctx,"alice","password")
    if err!=nil{t.Fatal(err)}
    if err=svc.ValidateRealtimeSession(ctx,auth.SessionID,id);err!=nil{t.Fatal(err)}
    if store.optimizedCalls!=1||identity.calls!=1{t.Fatalf("fast paths not used session=%d authority=%d",store.optimizedCalls,identity.calls)}
    identity.allow=false
    if err=svc.ValidateRealtimeSession(ctx,auth.SessionID,id);!errors.Is(err,ErrInvalidToken){
        t.Fatalf("revoked combined authority accepted: %v",err)
    }
    identity.allow=true
    identity.fail=errors.New("mysql unavailable")
    if err=svc.ValidateRealtimeSession(ctx,auth.SessionID,id);err==nil{
        t.Fatal("combined authority failure allowed the session")
    }
}
