package mysqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/application"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
)

type Database interface {
	DB() *sql.DB
}

type Store struct {
	database Database
}

func New(database Database) (*Store, error) {
	if database == nil || database.DB() == nil {
		return nil, fmt.Errorf("mysql doudizhu store configuration is invalid")
	}
	return &Store{database: database}, nil
}

// WithinCommand executes one command in a READ COMMITTED transaction. Command
// deduplication is serialized by the command-result primary key inside the
// transaction instead of a global or connection-scoped named lock. Independent
// commands therefore use independent pooled connections and unrelated
// aggregates can progress concurrently.
func (s *Store) WithinCommand(
	ctx context.Context,
	actor domain.AccountID,
	commandID string,
	fn func(context.Context, application.Transaction) error,
) (err error) {
	if fn == nil || actor == "" || commandID == "" {
		return fmt.Errorf("mysql doudizhu command transaction is invalid")
	}
	tx, err := s.database.DB().BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return translateTransactionError(fmt.Errorf("begin doudizhu transaction: %w", err))
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			_ = tx.Rollback()
			panic(recovered)
		}
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
				err = errors.Join(err, fmt.Errorf("rollback doudizhu transaction: %w", rollbackErr))
			}
			err = translateTransactionError(err)
			return
		}
		if commitErr := tx.Commit(); commitErr != nil {
			err = translateTransactionError(fmt.Errorf("commit doudizhu transaction: %w", commitErr))
		}
	}()

	err = fn(ctx, &transaction{tx: tx})
	return err
}

func translateTransactionError(err error) error {
	if err == nil || errors.Is(err, application.ErrRetryableTransaction) {
		return err
	}
	message := strings.ToLower(err.Error())
	if strings.Contains(message, "error 1205") || strings.Contains(message, "error 1213") ||
		strings.Contains(message, "deadlock found") || strings.Contains(message, "lock wait timeout") {
		return fmt.Errorf("%w: mysql concurrency conflict", application.ErrRetryableTransaction)
	}
	return err
}
