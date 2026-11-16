package identityprovider

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/oublie6/awesome-zero-platform/server/platform/authn"
	"github.com/oublie6/awesome-zero-platform/server/platform/identity"
)

type accountService interface {
	FindAccountByUsernameFresh(context.Context, string) (identity.Account, error)
	FindAccountByEmailFresh(context.Context, string) (identity.Account, error)
	FindAccountByPhoneFresh(context.Context, string) (identity.Account, error)
	GetAccountByIDFresh(context.Context, string) (identity.Account, error)
	VerifyPasswordWithFingerprint(context.Context, string, string) (string, error)
	CredentialFingerprint(context.Context, string) (string, error)
}

type Provider struct {
	identity accountService
}

func New(service accountService) *Provider {
	return &Provider{identity: service}
}

func (p *Provider) Authenticate(ctx context.Context, identifier, password string) (authn.Principal, error) {
	if p == nil || p.identity == nil {
		return authn.Principal{}, authn.ErrAccountUnavailable
	}

	identifier = strings.TrimSpace(identifier)
	var (
		account identity.Account
		err     error
	)
	switch {
	case strings.HasPrefix(identifier, "+"):
		account, err = p.identity.FindAccountByPhoneFresh(ctx, identifier)
	case strings.Contains(identifier, "@"):
		account, err = p.identity.FindAccountByEmailFresh(ctx, identifier)
	default:
		account, err = p.identity.FindAccountByUsernameFresh(ctx, identifier)
	}
	if err != nil {
		if isInvalidCredentialError(err) {
			return authn.Principal{}, authn.ErrInvalidCredentials
		}
		return authn.Principal{}, fmt.Errorf("find identity account: %w", err)
	}
	if account.Status != identity.StatusActive {
		return authn.Principal{}, authn.ErrInvalidCredentials
	}
	fingerprint, err := p.identity.VerifyPasswordWithFingerprint(ctx, account.ID, password)
	if err != nil {
		if isInvalidCredentialError(err) {
			return authn.Principal{}, authn.ErrInvalidCredentials
		}
		return authn.Principal{}, fmt.Errorf("verify identity password: %w", err)
	}

	result := principal(account)
	result.PasswordFingerprint = fingerprint
	return result, nil
}
func (p *Provider) CredentialFingerprint(ctx context.Context, id string) (string, error) {
	if p == nil || p.identity == nil {
		return "", authn.ErrAccountUnavailable
	}
	return p.identity.CredentialFingerprint(ctx, id)
}
// CheckRealtimeAuthority never reuses the provider's account model cache.
// Missing support fails closed instead of reverting to separate lookups.
func (p *Provider) CheckRealtimeAuthority(ctx context.Context, accountID, provider, subjectDigest, passwordFingerprint string)(bool,error){
    if p==nil || p.identity==nil{return false,authn.ErrAccountUnavailable}
    checker,ok:=p.identity.(interface{
        CheckRealtimeAuthority(context.Context,string,string,string,string)(bool,error)
    })
    if !ok{return false,fmt.Errorf("identity authority query is unavailable")}
    return checker.CheckRealtimeAuthority(ctx,accountID,provider,subjectDigest,passwordFingerprint)
}

func (p *Provider) ResolveActive(ctx context.Context, accountID string) (authn.Principal, error) {
	if p == nil || p.identity == nil {
		return authn.Principal{}, authn.ErrAccountUnavailable
	}
	account, err := p.identity.GetAccountByIDFresh(ctx, accountID)
	if err != nil {
		if errors.Is(err, identity.ErrAccountNotFound) || errors.Is(err, identity.ErrInvalidAccountState) {
			return authn.Principal{}, authn.ErrAccountUnavailable
		}
		return authn.Principal{}, fmt.Errorf("get identity account: %w", err)
	}
	if account.Status != identity.StatusActive {
		return authn.Principal{}, authn.ErrAccountUnavailable
	}
	return principal(account), nil
}

func isInvalidCredentialError(err error) bool {
	return errors.Is(err, identity.ErrAccountNotFound) ||
		errors.Is(err, identity.ErrInvalidCredentials) ||
		errors.Is(err, identity.ErrInvalidAccountState)
}

func principal(account identity.Account) authn.Principal {
	return authn.Principal{
		AccountID:   account.ID,
		DisplayName: account.DisplayName,
	}
}
