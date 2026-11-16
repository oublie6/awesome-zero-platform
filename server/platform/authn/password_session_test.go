package authn

import (
    "context"
    "errors"
    "sync"
    "testing"
    "time"
)

type credentialIdentity struct{
    mu sync.Mutex
    version string
}
func(f *credentialIdentity) Authenticate(context.Context,string,string)(Principal,error){
    f.mu.Lock();defer f.mu.Unlock()
    return Principal{AccountID:"01984f63-ec7f-7a4a-b908-33e8ff14d465",PasswordFingerprint:f.version},nil
}
func(f *credentialIdentity) ResolveActive(context.Context,string)(Principal,error){
    return Principal{AccountID:"01984f63-ec7f-7a4a-b908-33e8ff14d465"},nil
}
func(f *credentialIdentity) CredentialFingerprint(context.Context,string)(string,error){
    f.mu.Lock();defer f.mu.Unlock();return f.version,nil
}
func TestPasswordSessionCannotOutliveCredentialReset(t *testing.T) {
    ctx:=context.Background()
    creds:=&credentialIdentity{version:"argon-hash-version-old"}
    service,err:=NewService(creds,&fakeCodec{},newMemoryStore(),Config{AccessTTL:time.Minute,RefreshTTL:time.Hour})
    if err!=nil{t.Fatal(err)}
    _,tokens,err:=service.Login(ctx,"alice","original-password")
    if err!=nil{t.Fatal(err)}
    if _,err=service.AuthenticateAccess(ctx,tokens.AccessToken);err!=nil{t.Fatal(err)}
    creds.mu.Lock();creds.version="argon-hash-version-new";creds.mu.Unlock()
    if _,err=service.AuthenticateAccess(ctx,tokens.AccessToken);!errors.Is(err,ErrInvalidToken){
        t.Fatalf("old credential access survived password reset: %v",err)
    }
    if _,_,err=service.Refresh(ctx,tokens.RefreshToken);!errors.Is(err,ErrInvalidRefresh){
        t.Fatalf("old credential refresh survived password reset: %v",err)
    }
}
type resetDuringIssueStore struct{
    *memoryStore
    creds *credentialIdentity
}
func(s *resetDuringIssueStore)Create(ctx context.Context,session Session)error{
    if err:=s.memoryStore.Create(ctx,session);err!=nil{return err}
    s.creds.mu.Lock();s.creds.version="new-after-auth-before-session";s.creds.mu.Unlock()
    return nil
}
func TestPasswordSessionIssuedAfterResetStillCarriesOldProof(t *testing.T){
    ctx:=context.Background()
    creds:=&credentialIdentity{version:"original-proof"}
    store:=&resetDuringIssueStore{memoryStore:newMemoryStore(),creds:creds}
    service,err:=NewService(creds,&fakeCodec{},store,Config{})
    if err!=nil{t.Fatal(err)}
    auth,tokens,err:=service.Login(ctx,"alice","original-password")
    if !errors.Is(err,ErrInvalidToken) || auth.SessionID!="" || tokens.AccessToken!=""{
        t.Fatalf("password reset during issuance returned credentials: %v",err)
    }
    store.mu.Lock();n:=len(store.sessions);store.mu.Unlock()
    if n!=0{t.Fatalf("password reset race left %d live sessions",n)}
}
