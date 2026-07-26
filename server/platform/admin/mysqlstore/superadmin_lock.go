package mysqlstore

import (
	"context"
	"fmt"
	"time"
)

const superAdminLockName = "awesome-zero-platform:admin:super-admin"

// WithSuperAdminLock serializes bootstrap, last-active-super-admin status
// changes, and role replacement across all app-api replicas. MySQL named locks
// are connection-scoped, so the dedicated connection must remain open for the
// entire callback.
func (s *Store) WithSuperAdminLock(ctx context.Context, fn func(context.Context) error) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("mysql database is required")
	}
	if fn == nil {
		return fmt.Errorf("super administrator lock callback is required")
	}

	conn, err := s.db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("reserve super administrator lock connection: %w", err)
	}
	defer conn.Close()

	var acquired int
	if err := conn.QueryRowContext(ctx, "SELECT GET_LOCK(?, ?)", superAdminLockName, 10).Scan(&acquired); err != nil {
		return fmt.Errorf("acquire super administrator lock: %w", err)
	}
	if acquired != 1 {
		return fmt.Errorf("acquire super administrator lock: timed out")
	}
	defer func() {
		releaseCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		var released int
		_ = conn.QueryRowContext(releaseCtx, "SELECT RELEASE_LOCK(?)", superAdminLockName).Scan(&released)
	}()

	return fn(ctx)
}
