package mysqlstore

import (
	"context"
	"time"

	"github.com/oublie6/awesome-zero-platform/server/foundation/database"
	"github.com/oublie6/awesome-zero-platform/server/platform/authz"
)

// WithSuperAdminLock serializes bootstrap, last-active-super-admin status
// changes, and policy membership changes across all app-api replicas.
func (s *Store) WithSuperAdminLock(ctx context.Context, fn func(context.Context) error) error {
	return database.WithAdvisoryLock(ctx, s.db, authz.SuperAdminMutationLock, 10*time.Second, fn)
}
