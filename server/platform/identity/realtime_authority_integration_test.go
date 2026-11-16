//go:build integration

package identity

import (
    "context"
    "os"
    "strings"
    "testing"
)

func TestSingleStatementRealtimeAuthorityIntegration(t *testing.T){
    if os.Getenv("APP_API_INTEGRATION")!="1"{t.Skip("isolated MySQL integration required")}
    service,db:=newIntegrationService(t)
    cleanupIdentityTables(t,db)
    ctx:=context.Background()
    account,err:=service.CreateAccount(ctx,CreateAccountInput{
        Identity:Identity{Username:"authority.account",Email:"authority.account@example.test"},
        DisplayName:"Authority",Password:"account-password-12345",
    })
    if err!=nil{t.Fatal(err)}
    fingerprint,err:=service.CredentialFingerprint(ctx,account.ID)
    if err!=nil{t.Fatal(err)}
    cases:=[]struct{
        name,accountID,provider,subject,password string
        allowed bool
    }{
        {"password",account.ID,"","",fingerprint,true},
        {"wrong password",account.ID,"","",strings.Repeat("0",64),false},
        {"active legacy",account.ID,"","","",true},
        {"wrong account","01984f63-ec7f-7a4a-b908-33e8ff14d465","","",fingerprint,false},
        {"missing binding",account.ID,"wechat",strings.Repeat("1",64),"",false},
        {"bad provider",account.ID,"other",strings.Repeat("1",64),"",false},
        {"half-linked",account.ID,"wechat","","",false},
    }
    for _,test:=range cases{
        t.Run(test.name,func(t *testing.T){
            allowed,err:=service.CheckRealtimeAuthority(ctx,test.accountID,test.provider,test.subject,test.password)
            if err!=nil || allowed!=test.allowed{t.Fatalf("allowed=%v err=%v want=%v",allowed,err,test.allowed)}
        })
    }
    binder,err:=NewExternalBinding(db,service,[]byte("test-only-external-subject-hmac-key-32bytes"))
    if err!=nil{t.Fatal(err)}
    proof,err:=binder.BeginVerified(ctx,"wechat","realtime-verified-openid")
    if err!=nil{t.Fatal(err)}
    if err:=binder.BindPending(ctx,proof,account.ID);err!=nil{t.Fatal(err)}
    digest,err:=binder.VerifiedSubjectDigest("wechat","realtime-verified-openid")
    if err!=nil{t.Fatal(err)}
    if allowed,err:=service.CheckRealtimeAuthority(ctx,account.ID,"wechat",digest,"");err!=nil || !allowed{
        t.Fatalf("linked external session rejected %v %v",allowed,err)
    }
    if allowed,err:=service.CheckRealtimeAuthority(ctx,account.ID,"wechat",digest,strings.Repeat("0",64));err!=nil || allowed{
        t.Fatalf("invalid combined proof accepted %v %v",allowed,err)
    }
    if err:=service.ResetPassword(ctx,account.ID,"new-account-password-12345");err!=nil{t.Fatal(err)}
    if allowed,err:=service.CheckRealtimeAuthority(ctx,account.ID,"","",fingerprint);err!=nil || allowed{
        t.Fatalf("old password fingerprint accepted %v %v",allowed,err)
    }
    next,err:=service.CredentialFingerprint(ctx,account.ID)
    if err!=nil{t.Fatal(err)}
    if allowed,err:=service.CheckRealtimeAuthority(ctx,account.ID,"","",next);err!=nil || !allowed{
        t.Fatalf("new password fingerprint rejected %v %v",allowed,err)
    }
    if err:=binder.Unlink(ctx,"wechat",account.ID);err!=nil{t.Fatal(err)}
    if allowed,err:=service.CheckRealtimeAuthority(ctx,account.ID,"wechat",digest,"");err!=nil || allowed{
        t.Fatalf("unlinked provider accepted %v %v",allowed,err)
    }
    if _,err:=service.DisableAccount(ctx,account.ID);err!=nil{t.Fatal(err)}
    if allowed,err:=service.CheckRealtimeAuthority(ctx,account.ID,"","",next);err!=nil || allowed{
        t.Fatalf("disabled account accepted %v %v",allowed,err)
    }
}
