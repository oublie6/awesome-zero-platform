package identityprovider

import (
	"context"
	"strings"

	"github.com/oublie6/awesome-zero-platform/server/platform/authn"
	"github.com/oublie6/awesome-zero-platform/server/platform/identity"
)

type Provider struct {
	identity *identity.Service
}

func New(service *identity.Service) *Provider {
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
	if err != nil || account.Status != identity.StatusActive {
		return authn.Principal{}, authn.ErrInvalidCredentials
	}
	if err := p.identity.VerifyPassword(ctx, account.ID, password); err != nil {
		return authn.Principal{}, authn.ErrInvalidCredentials
	}

	return principal(account), nil
}

func (p *Provider) ResolveActive(ctx context.Context, accountID string) (authn.Principal, error) {
	if p == nil || p.identity == nil {
		return authn.Principal{}, authn.ErrAccountUnavailable
	}
	account, err := p.identity.GetAccountByID(ctx, accountID)
	if err != nil || account.Status != identity.StatusActive {
		return authn.Principal{}, authn.ErrAccountUnavailable
	}
	return principal(account), nil
}

func principal(account identity.Account) authn.Principal {
	return authn.Principal{
		AccountID:   account.ID,
		DisplayName: account.DisplayName,
	}
}
