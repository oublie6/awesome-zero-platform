//go:build integration

package identity

import (
    "context"
    "errors"
    "fmt"
    "os"
    "sync"
    "testing"
    "time"
)

func TestExternalIdentityBindingIntegration(t *testing.T) {
    if os.Getenv("APP_API_INTEGRATION")!="1"{t.Skip("set APP_API_INTEGRATION=1")}
    service,db:=newIntegrationService(t)
    cleanupIdentityTables(t,db)
    ctx:=context.Background()
    b,err:=NewExternalBinding(db,service,[]byte("test-only-external-subject-hmac-key-32bytes"))
    if err!=nil{t.Fatal(err)}
    makeAccount:=func(username string)Account{
        t.Helper()
        a,err:=service.CreateAccount(ctx,CreateAccountInput{
            Identity:Identity{Username:username,Email:username+"@example.test"},
            DisplayName:username,Password:"sample-password-98765",
        })
        if err!=nil{t.Fatal(err)}
        return a
    }
    a:=makeAccount("provider.accountA")
    c:=makeAccount("provider.accountB")
    proof,err:=b.BeginVerified(ctx,"wechat","unique-openid-A")
    if err!=nil{t.Fatal(err)}
    if _,err:=b.ResolveVerified(ctx,"wechat","unique-openid-A");!errors.Is(err,ErrExternalUnbound){t.Fatalf("unbound: %v",err)}
    if err=b.BindPending(ctx,proof,a.ID);err!=nil{t.Fatal(err)}
    if err=b.BindPending(ctx,proof,c.ID);!errors.Is(err,ErrPendingInvalid){t.Fatalf("replay accepted: %v",err)}
    found,err:=b.ResolveVerified(ctx,"wechat","unique-openid-A")
    if err!=nil || found.ID!=a.ID{t.Fatalf("resolve %v %v",found,err)}
    another,err:=b.BeginVerified(ctx,"wechat","unique-openid-B")
    if err!=nil{t.Fatal(err)}
    if err=b.BindPending(ctx,another,a.ID);!errors.Is(err,ErrExternalConflict){t.Fatalf("duplicate provider account link: %v",err)}
    relink,err:=b.BeginVerified(ctx,"wechat","unique-openid-A")
    if err!=nil{t.Fatal(err)}
    if err=b.Relink(ctx,relink,a.ID,c.ID);err!=nil{t.Fatal(err)}
    found,err=b.ResolveVerified(ctx,"wechat","unique-openid-A")
    if err!=nil || found.ID!=c.ID{t.Fatalf("relink %v %v",found,err)}
    if err=b.Unlink(ctx,"wechat",c.ID);err!=nil{t.Fatal(err)}
    if _,err=b.ResolveVerified(ctx,"wechat","unique-openid-A");!errors.Is(err,ErrExternalUnbound){t.Fatalf("unlink resolve: %v",err)}
    fresh,err:=b.BeginVerified(ctx,"wechat","unique-openid-A")
    if err!=nil{t.Fatal(err)}
    born,err:=b.CreateFromPending(ctx,fresh,CreateAccountInput{
        Identity:Identity{Username:"provider.new",Email:"provider.new@example.test"},
        DisplayName:"New Provider",Password:"new-password-12345",
    })
    if err!=nil{t.Fatal(err)}
    if err=b.BindPending(ctx,fresh,a.ID);!errors.Is(err,ErrPendingInvalid){t.Fatalf("create proof replay: %v",err)}
    found,err=b.ResolveVerified(ctx,"wechat","unique-openid-A")
    if err!=nil||found.ID!=born.ID{t.Fatalf("new platform account mapping %v %v",found,err)}
}

func TestConcurrentProviderSubjectClaimIntegration(t *testing.T) {
    if os.Getenv("APP_API_INTEGRATION")!="1"{t.Skip("set APP_API_INTEGRATION=1")}
    service,db:=newIntegrationService(t)
    cleanupIdentityTables(t,db)
    b,err:=NewExternalBinding(db,service,[]byte("test-only-external-subject-hmac-key-32bytes"))
    if err!=nil{t.Fatal(err)}
    ctx:=context.Background()
    ids:=make([]string,2)
    stamp:=time.Now().UnixNano()
    for i:=range ids{
        a,err:=service.CreateAccount(ctx,CreateAccountInput{
            Identity:Identity{Username:fmt.Sprintf("claim%d_%d",i,stamp),Email:fmt.Sprintf("claim%d_%d@example.test",i,stamp)},
            DisplayName:"Concurrent",Password:"password-12345678",
        })
        if err!=nil{t.Fatal(err)}
        ids[i]=a.ID
    }
    proofs:=make([]string,2)
    for i:=range proofs {
        proofs[i],err=b.BeginVerified(ctx,"wechat","same-openid")
        if err!=nil{t.Fatal(err)}
    }
    var wg sync.WaitGroup
    results:=make([]error,2)
    for i:=range proofs{
        wg.Add(1)
        go func(index int){defer wg.Done();results[index]=b.BindPending(ctx,proofs[index],ids[index])}(i)
    }
    wg.Wait()
    success:=0
    for _,err:=range results{
        if err==nil{success++}else if !errors.Is(err,ErrExternalConflict){t.Errorf("unexpected claim result: %v",err)}
    }
    if success!=1{t.Fatalf("successful claims=%d, want 1",success)}
}
