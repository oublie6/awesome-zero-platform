package database

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"time"
)

type advisoryLocksContextKey struct{}

// WithAdvisoryLock executes fn while holding a connection-scoped MySQL named
// lock. The lock is reentrant within the returned callback context, allowing
// an application-level operation to call a persistence adapter that protects
// the same invariant without deadlocking on another database connection.
func WithAdvisoryLock(
	ctx context.Context,
	db *sql.DB,
	name string,
	timeout time.Duration,
	fn func(context.Context) error,
) error {
	if db == nil {
		return fmt.Errorf("mysql database is required")
	}
	if name == "" {
		return fmt.Errorf("mysql advisory lock name is required")
	}
	if fn == nil {
		return fmt.Errorf("mysql advisory lock callback is required")
	}
	if heldAdvisoryLock(ctx, name) {
		return fn(ctx)
	}

	conn, err := db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("reserve mysql advisory lock connection: %w", err)
	}
	defer conn.Close()

	seconds := int64(math.Ceil(timeout.Seconds()))
	if seconds < 1 {
		seconds = 1
	}
	var acquired sql.NullInt64
	if err := conn.QueryRowContext(ctx, "SELECT GET_LOCK(?, ?)", name, seconds).Scan(&acquired); err != nil {
		return fmt.Errorf("acquire mysql advisory lock %q: %w", name, err)
	}
	if !acquired.Valid || acquired.Int64 != 1 {
		return fmt.Errorf("acquire mysql advisory lock %q: timed out", name)
	}

	defer func() {
		releaseCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		var released sql.NullInt64
		_ = conn.QueryRowContext(releaseCtx, "SELECT RELEASE_LOCK(?)", name).Scan(&released)
	}()

	return fn(withAdvisoryLock(ctx, name))
}

func heldAdvisoryLock(ctx context.Context, name string) bool {
	locks, _ := ctx.Value(advisoryLocksContextKey{}).(map[string]struct{})
	_, ok := locks[name]
	return ok
}

func withAdvisoryLock(ctx context.Context, name string) context.Context {
	previous, _ := ctx.Value(advisoryLocksContextKey{}).(map[string]struct{})
	locks := make(map[string]struct{}, len(previous)+1)
	for lock := range previous {
		locks[lock] = struct{}{}
	}
	locks[name] = struct{}{}
	return context.WithValue(ctx, advisoryLocksContextKey{}, locks)
}
