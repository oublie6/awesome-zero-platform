package identity

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

// ExternalBinding resolves a verified provider subject to a platform account.
// It accepts only a verified subject from a provider adapter, never arbitrary
// client-supplied IDs. HMAC hides OpenID at rest and namespaces credentials.
type ExternalBinding struct {
	db *sql.DB
	identity *Service
	key []byte
}
func NewExternalBinding(db *sql.DB,service *Service,subjectHMACKey []byte)(*ExternalBinding,error){
	if db==nil || service==nil || len(subjectHMACKey)<32{
		return nil,fmt.Errorf("verified identity binding requires DB and >=256-bit subject HMAC key")
	}
	return &ExternalBinding{db:db,identity:service,key:append([]byte(nil),subjectHMACKey...)},nil
}
func (b *ExternalBinding) ResolveVerified(ctx context.Context,provider,subject string)(Account,error){
	if b==nil || b.db==nil || b.identity==nil || provider!="wechat" ||
		len(subject)<4 || len(subject)>256 || strings.TrimSpace(subject)!=subject {
		return Account{},fmt.Errorf("invalid verified external identity")
	}
	mac:=hmac.New(sha256.New,b.key)
	mac.Write([]byte(provider))
	mac.Write([]byte{0})
	mac.Write([]byte(subject))
	digest:=mac.Sum(nil)
	username:="wx_"+hex.EncodeToString(digest[:12])
	var accountID string
	err:=b.db.QueryRowContext(ctx,`SELECT account_id FROM identity_external_subjects
WHERE provider=? AND subject_hmac=?`,provider,digest).Scan(&accountID)
	if err==nil{return b.identity.GetAccountByIDFresh(ctx,accountID)}
	if !errors.Is(err,sql.ErrNoRows){return Account{},fmt.Errorf("find external identity link: %w",err)}
	account,err:=b.identity.FindAccountByUsernameFresh(ctx,username)
	if errors.Is(err,ErrAccountNotFound) {
		password:=make([]byte,48)
		if _,err=rand.Read(password);err!=nil{return Account{},fmt.Errorf("external account entropy: %w",err)}
		secret:=base64.RawURLEncoding.EncodeToString(password)
		clear(password)
		account,err=b.identity.CreateAccount(ctx,CreateAccountInput{
			Identity:Identity{Username:username},DisplayName:"微信玩家",Status:StatusActive,Password:secret,
		})
		if err!=nil {
			// Concurrent verified exchanges can race on the unique derived
			// username; reread the winner instead of reusing an invalid account.
			account,err=b.identity.FindAccountByUsernameFresh(ctx,username)
			if err!=nil{return Account{},fmt.Errorf("create verified external account: %w",err)}
		}
	} else if err!=nil {return Account{},fmt.Errorf("resolve external account: %w",err)}
	if account.Status!=StatusActive{return Account{},fmt.Errorf("external account disabled")}
	_,err=b.db.ExecContext(ctx,`INSERT INTO identity_external_subjects(
provider,subject_hmac,account_id,created_at) VALUES(?,?,?,UTC_TIMESTAMP(6))
ON DUPLICATE KEY UPDATE subject_hmac=VALUES(subject_hmac)`,provider,digest,account.ID)
	if err!=nil{return Account{},fmt.Errorf("bind external identity: %w",err)}
	// In a race return the account that actually owns the unique subject link.
	if err:=b.db.QueryRowContext(ctx,`SELECT account_id FROM identity_external_subjects
WHERE provider=? AND subject_hmac=?`,provider,digest).Scan(&accountID);err!=nil {
		return Account{},fmt.Errorf("read verified external identity: %w",err)
	}
	return b.identity.GetAccountByIDFresh(ctx,accountID)
}
