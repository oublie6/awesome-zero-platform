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
	FindAccountByUsername(context.Context, string) (identity.Account, error)
	FindAccountByEmail(context.Context, string) (identity.Account, error)
	FindAccountByPhone(context.Context, string) (identity.Account, error)
	GetAccountByID(context.Context, string) (identity.Account, error)
	VerifyPassword(context.Context, string, string) error
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
		account, err = p.identity.FindAccountByPhone(ctx, identifier)
	case strings.Contains(identifier, "@"):
		account, err = p.identity.FindAccountByEmail(ctx, identifier)
	default:
		account, err = p.identity.FindAccountByUsername(ctx, identifier)
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
	if err := p.identity.VerifyPassword(ctx, account.ID, password); err != nil {
		if isInvalidCredentialError(err) {
			return authn.Principal{}, authn.ErrInvalidCredentials
		}
		return authn.Principal{}, fmt.Errorf("verify identity password: %w", err)
	}

	return principal(account), nil
}

func (p *Provider) ResolveActive(ctx context.Context, accountID string) (authn.Principal, error) {
	if p == nil || p.identity == nil {
		return authn.Principal{}, authn.ErrAccountUnavailable
	}
	account, err := p.identity.GetAccountByID(ctx, accountID)
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
