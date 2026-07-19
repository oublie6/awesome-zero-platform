package database

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Handle interface {
	Pool() *pgxpool.Pool
	Ping(context.Context) error
	Close() error
	WithinTransaction(context.Context, func(context.Context, pgx.Tx) error) error
}

type Config struct {
	Host              string        `json:",optional"`
	Port              int           `json:",default=5432"`
	Database          string        `json:",optional"`
	User              string        `json:",optional"`
	Password          string        `json:",optional"`
	SSLMode           string        `json:",default=disable"`
	MaxConns          int32         `json:",default=4"`
	MinConns          int32         `json:",default=0"`
	ConnectTimeout    time.Duration `json:",default=3s"`
	StartupTimeout    time.Duration `json:",default=3s"`
	ReadinessTimeout  time.Duration `json:",default=2s"`
	HealthCheckPeriod time.Duration `json:",default=30s"`
}

type Resource struct {
	pool      *pgxpool.Pool
	closeFn   func() error
	pingFn    func(context.Context) error
	beginTxFn func(context.Context, pgx.TxOptions) (pgx.Tx, error)

	closeOnce sync.Once
	closeErr  error
}

type StartupError struct {
	Dependency string
	Cause      error
}

func (e *StartupError) Error() string {
	return fmt.Sprintf("%s startup connectivity check failed", e.Dependency)
}

func (e *StartupError) Unwrap() error {
	if e == nil {
		return nil
	}

	return e.Cause
}

func (c *Config) Prepare() {
	if c.Port == 0 {
		c.Port = 5432
	}
	if c.SSLMode == "" {
		c.SSLMode = "disable"
	}
	if c.MaxConns == 0 {
		c.MaxConns = 4
	}
	if c.ConnectTimeout == 0 {
		c.ConnectTimeout = 3 * time.Second
	}
	if c.StartupTimeout == 0 {
		c.StartupTimeout = 3 * time.Second
	}
	if c.ReadinessTimeout == 0 {
		c.ReadinessTimeout = 2 * time.Second
	}
	if c.HealthCheckPeriod == 0 {
		c.HealthCheckPeriod = 30 * time.Second
	}
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.Host) == "" {
		return fmt.Errorf("postgres.host must not be empty")
	}
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("postgres.port must be between 1 and 65535")
	}
	if strings.TrimSpace(c.Database) == "" {
		return fmt.Errorf("postgres.database must not be empty")
	}
	if strings.TrimSpace(c.User) == "" {
		return fmt.Errorf("postgres.user must not be empty")
	}
	if strings.TrimSpace(c.Password) == "" {
		return fmt.Errorf("postgres.password must not be empty")
	}
	if strings.TrimSpace(c.SSLMode) == "" {
		return fmt.Errorf("postgres.sslMode must not be empty")
	}
	if c.MaxConns < 1 {
		return fmt.Errorf("postgres.maxConns must be greater than 0")
	}
	if c.MinConns < 0 || c.MinConns > c.MaxConns {
		return fmt.Errorf("postgres.minConns must be between 0 and postgres.maxConns")
	}
	if c.ConnectTimeout <= 0 {
		return fmt.Errorf("postgres.connectTimeout must be greater than 0")
	}
	if c.StartupTimeout <= 0 {
		return fmt.Errorf("postgres.startupTimeout must be greater than 0")
	}
	if c.ReadinessTimeout <= 0 {
		return fmt.Errorf("postgres.readinessTimeout must be greater than 0")
	}
	if c.HealthCheckPeriod <= 0 {
		return fmt.Errorf("postgres.healthCheckPeriod must be greater than 0")
	}

	return nil
}

func Open(ctx context.Context, cfg Config) (Handle, error) {
	config, err := pgxpool.ParseConfig(cfg.connectionString())
	if err != nil {
		return nil, fmt.Errorf("postgres configuration is invalid")
	}

	config.MaxConns = cfg.MaxConns
	config.MinConns = cfg.MinConns
	config.HealthCheckPeriod = cfg.HealthCheckPeriod
	config.ConnConfig.ConnectTimeout = cfg.ConnectTimeout

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("postgres resource initialization failed")
	}

	resource := &Resource{
		pool: pool,
		closeFn: func() error {
			pool.Close()
			return nil
		},
		pingFn: pool.Ping,
		beginTxFn: func(ctx context.Context, options pgx.TxOptions) (pgx.Tx, error) {
			return pool.BeginTx(ctx, options)
		},
	}

	checkCtx, cancel := context.WithTimeout(ctx, cfg.StartupTimeout)
	defer cancel()
	if err := resource.Ping(checkCtx); err != nil {
		_ = resource.Close()
		return nil, &StartupError{
			Dependency: "postgres",
			Cause:      err,
		}
	}

	return resource, nil
}

func (r *Resource) Pool() *pgxpool.Pool {
	if r == nil {
		return nil
	}

	return r.pool
}

func (r *Resource) Ping(ctx context.Context) error {
	if r == nil || r.pingFn == nil {
		return errors.New("postgres resource is not initialized")
	}

	return r.pingFn(ctx)
}

func (r *Resource) Close() error {
	if r == nil {
		return nil
	}

	r.closeOnce.Do(func() {
		if r.closeFn != nil {
			r.closeErr = r.closeFn()
		}
	})

	return r.closeErr
}

func (r *Resource) WithinTransaction(ctx context.Context, fn func(context.Context, pgx.Tx) error) (err error) {
	if r == nil || r.beginTxFn == nil {
		return errors.New("postgres resource is not initialized")
	}

	tx, err := r.beginTxFn(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	defer func() {
		if recovered := recover(); recovered != nil {
			if rollbackErr := rollback(ctx, tx); rollbackErr != nil {
				panic(fmt.Errorf("rollback transaction after panic: %w", rollbackErr))
			}
			panic(recovered)
		}

		if err != nil {
			if rollbackErr := rollback(ctx, tx); rollbackErr != nil {
				err = errors.Join(err, rollbackErr)
			}
			return
		}

		if commitErr := tx.Commit(ctx); commitErr != nil {
			err = fmt.Errorf("commit transaction: %w", commitErr)
		}
	}()

	err = fn(ctx, tx)
	return err
}

func rollback(ctx context.Context, tx pgx.Tx) error {
	if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
		return fmt.Errorf("rollback transaction: %w", err)
	}

	return nil
}

func (c Config) connectionString() string {
	query := url.Values{}
	query.Set("sslmode", c.SSLMode)

	u := &url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(c.User, c.Password),
		Host:     net.JoinHostPort(c.Host, strconv.Itoa(c.Port)),
		Path:     c.Database,
		RawQuery: query.Encode(),
	}

	return u.String()
}
