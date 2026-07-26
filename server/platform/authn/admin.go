package authn

import (
	"context"
	"time"
)

type SessionView struct {
	ID        string    `json:"id"`
	AccountID string    `json:"accountId"`
	CreatedAt time.Time `json:"createdAt"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// SessionAdministrator is deliberately separate from SessionStore. Normal
// authentication does not need list or account-wide revoke capabilities.
type SessionAdministrator interface {
	ListByAccount(context.Context, string) ([]SessionView, error)
	RevokeByAccount(context.Context, string) (int64, error)
}
