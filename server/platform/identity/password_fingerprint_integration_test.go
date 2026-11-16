//go:build integration

package identity

import (
    "context"
    "os"
    "testing"
)
func TestCredentialFingerprintChangesOnResetAndPasswordChangeIntegration(t *testing.T){
    if os.Getenv("APP_API_INTEGRATION")!="1"{t.Skip("requires isolated MySQL integration environment")}
    service,db:=newIntegrationService(t)
    cleanupIdentityTables(t,db)
    ctx:=context.Background()
    account,err:=service.CreateAccount(ctx,CreateAccountInput{
        Identity:Identity{Username:"fingerprint.test",Email:"fingerprint.test@example.test"},
        DisplayName:"Fingerprint",Password:"initial-password-1234",
    })
    if err!=nil{t.Fatal(err)}
    original,err:=service.VerifyPasswordWithFingerprint(ctx,account.ID,"initial-password-1234")
    if err!=nil||len(original)!=64{t.Fatalf("verified credential fingerprint: %v",err)}
    current,err:=service.CredentialFingerprint(ctx,account.ID)
    if err!=nil||current!=original{t.Fatal("fingerprint does not match verified credential")}
    if err=service.ChangePassword(ctx,account.ID,"initial-password-1234","updated-password-1234");err!=nil{t.Fatal(err)}
    changed,err:=service.CredentialFingerprint(ctx,account.ID)
    if err!=nil||changed==original{t.Fatal("password change did not fence old session fingerprint")}
    if _,err=service.VerifyPasswordWithFingerprint(ctx,account.ID,"initial-password-1234");err==nil{t.Fatal("old password accepted")}
    if err=service.ResetPassword(ctx,account.ID,"reset-password-1234");err!=nil{t.Fatal(err)}
    reset,err:=service.CredentialFingerprint(ctx,account.ID)
    if err!=nil||reset==changed||reset==original{t.Fatal("password reset did not change credential fingerprint")}
}
