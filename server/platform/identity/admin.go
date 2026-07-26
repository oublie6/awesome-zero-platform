package identity

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

type AccountQuery struct {
	Search   string        `json:"search"`
	Status   AccountStatus `json:"status"`
	Page     int           `json:"page"`
	PageSize int           `json:"pageSize"`
}

type AccountPage struct {
	Items    []Account `json:"items"`
	Total    int64     `json:"total"`
	Page     int       `json:"page"`
	PageSize int       `json:"pageSize"`
}

type accountLister interface {
	ListAccounts(context.Context, AccountQuery) (AccountPage, error)
}

func (s *Service) ListAccounts(ctx context.Context, query AccountQuery) (AccountPage, error) {
	if s == nil {
		return AccountPage{}, fmt.Errorf("identity service is unavailable")
	}
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PageSize <= 0 {
		query.PageSize = 20
	}
	if query.PageSize > 100 {
		query.PageSize = 100
	}
	query.Search = strings.TrimSpace(query.Search)
	if query.Status != "" {
		if err := query.Status.Validate(); err != nil {
			return AccountPage{}, err
		}
	}
	lister, ok := s.accounts.(accountLister)
	if !ok {
		return AccountPage{}, fmt.Errorf("identity account listing is unavailable")
	}
	return lister.ListAccounts(ctx, query)
}

func (s *Service) ResetPassword(ctx context.Context, accountID, newPassword string) error {
	normalizedID, err := normalizeAccountID(accountID)
	if err != nil {
		return err
	}
	if err := validatePasswordInput(newPassword); err != nil {
		return err
	}
	passwordHash, err := s.hasher.Hash(newPassword)
	if err != nil {
		return wrapIdentityError(ErrPersistence, err)
	}
	now := s.now().UTC()
	return s.transactions.WithinTransaction(ctx, func(txCtx context.Context, tx *sql.Tx) error {
		if _, err := s.accounts.GetAccountByIDTx(txCtx, tx, normalizedID); err != nil {
			return err
		}
		credential, err := s.credentials.GetPasswordCredentialByAccountIDTx(txCtx, tx, normalizedID)
		if err != nil {
			return err
		}
		credential.Hash = passwordHash
		credential.PasswordChangedAt = now
		credential.UpdatedAt = now
		return s.credentials.UpdatePasswordCredentialTx(txCtx, tx, credential)
	})
}
