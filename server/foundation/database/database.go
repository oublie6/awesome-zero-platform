package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	mysql "github.com/go-sql-driver/mysql"
)

type Handle interface {
	DB() *sql.DB
	Ping(context.Context) error
	Close() error
	WithinTransaction(context.Context, func(context.Context, *sql.Tx) error) error
}

type Config struct {
	Addr             string        `json:",default=127.0.0.1:3306"`
	Database         string        `json:",optional"`
	User             string        `json:",optional"`
	Password         string        `json:",optional"`
	Charset          string        `json:",default=utf8mb4"`
	ParseTime        bool          `json:",default=true"`
	Location         string        `json:",default=UTC"`
	TimeZone         string        `json:",default=+00:00"`
	Timeout          time.Duration `json:",default=3s"`
	MaxOpenConns     int           `json:",default=4"`
	MaxIdleConns     int           `json:",default=2"`
	ConnMaxLifetime  time.Duration `json:",default=5m"`
	StartupTimeout   time.Duration `json:",default=3s"`
	ReadinessTimeout time.Duration `json:",default=2s"`
}

type Resource struct {
	db        *sql.DB
	closeFn   func() error
	pingFn    func(context.Context) error
	beginTxFn func(context.Context, *sql.TxOptions) (*sql.Tx, error)

	closeOnce sync.Once
	closeErr  error
}

type StartupError struct {
	Dependency string
	Cause      error
}

func init() {
	_ = mysql.SetLogger(&mysql.NopLogger{})
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
	if c.Addr == "" {
		c.Addr = "127.0.0.1:3306"
	}
	if c.Charset == "" {
		c.Charset = "utf8mb4"
	}
	if !c.ParseTime {
		c.ParseTime = true
	}
	if c.Location == "" {
		c.Location = "UTC"
	}
	if c.TimeZone == "" {
		c.TimeZone = "+00:00"
	}
	if c.Timeout == 0 {
		c.Timeout = 3 * time.Second
	}
	if c.MaxOpenConns == 0 {
		c.MaxOpenConns = 4
	}
	if c.MaxIdleConns == 0 {
		c.MaxIdleConns = 2
	}
	if c.ConnMaxLifetime == 0 {
		c.ConnMaxLifetime = 5 * time.Minute
	}
	if c.StartupTimeout == 0 {
		c.StartupTimeout = 3 * time.Second
	}
	if c.ReadinessTimeout == 0 {
		c.ReadinessTimeout = 2 * time.Second
	}
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.Addr) == "" {
		return fmt.Errorf("mysql.addr must not be empty")
	}
	if _, _, err := net.SplitHostPort(c.Addr); err != nil {
		return fmt.Errorf("mysql.addr must be in host:port format")
	}
	if strings.TrimSpace(c.Database) == "" {
		return fmt.Errorf("mysql.database must not be empty")
	}
	if strings.TrimSpace(c.User) == "" {
		return fmt.Errorf("mysql.user must not be empty")
	}
	if strings.TrimSpace(c.Password) == "" {
		return fmt.Errorf("mysql.password must not be empty")
	}
	if c.Charset != "utf8mb4" {
		return fmt.Errorf("mysql.charset must be utf8mb4")
	}
	if !c.ParseTime {
		return fmt.Errorf("mysql.parseTime must be true")
	}
	if strings.TrimSpace(c.Location) == "" {
		return fmt.Errorf("mysql.location must not be empty")
	}
	if _, err := time.LoadLocation(c.Location); err != nil {
		return fmt.Errorf("mysql.location must be a valid time zone")
	}
	if strings.TrimSpace(c.TimeZone) == "" {
		return fmt.Errorf("mysql.timeZone must not be empty")
	}
	if c.Timeout <= 0 {
		return fmt.Errorf("mysql.timeout must be greater than 0")
	}
	if c.MaxOpenConns < 1 {
		return fmt.Errorf("mysql.maxOpenConns must be greater than 0")
	}
	if c.MaxIdleConns < 0 || c.MaxIdleConns > c.MaxOpenConns {
		return fmt.Errorf("mysql.maxIdleConns must be between 0 and mysql.maxOpenConns")
	}
	if c.ConnMaxLifetime <= 0 {
		return fmt.Errorf("mysql.connMaxLifetime must be greater than 0")
	}
	if c.StartupTimeout <= 0 {
		return fmt.Errorf("mysql.startupTimeout must be greater than 0")
	}
	if c.ReadinessTimeout <= 0 {
		return fmt.Errorf("mysql.readinessTimeout must be greater than 0")
	}

	return nil
}

func Open(ctx context.Context, cfg Config) (Handle, error) {
	loc, err := time.LoadLocation(cfg.Location)
	if err != nil {
		return nil, fmt.Errorf("mysql configuration is invalid")
	}

	driverConfig := mysql.Config{
		User:      cfg.User,
		Passwd:    cfg.Password,
		Net:       "tcp",
		Addr:      cfg.Addr,
		DBName:    cfg.Database,
		Collation: "utf8mb4_unicode_ci",
		ParseTime: cfg.ParseTime,
		Loc:       loc,
		Timeout:   cfg.Timeout,
		Params: map[string]string{
			"charset":   cfg.Charset,
			"time_zone": quoteTimeZone(cfg.TimeZone),
		},
	}

	db, err := sql.Open("mysql", driverConfig.FormatDSN())
	if err != nil {
		return nil, fmt.Errorf("mysql configuration is invalid")
	}

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	resource := &Resource{
		db:      db,
		closeFn: db.Close,
		pingFn: func(ctx context.Context) error {
			return db.PingContext(ctx)
		},
		beginTxFn: db.BeginTx,
	}

	checkCtx, cancel := context.WithTimeout(ctx, cfg.StartupTimeout)
	defer cancel()
	if err := resource.Ping(checkCtx); err != nil {
		_ = resource.Close()
		return nil, &StartupError{
			Dependency: "mysql",
			Cause:      err,
		}
	}

	return resource, nil
}

func (r *Resource) DB() *sql.DB {
	if r == nil {
		return nil
	}

	return r.db
}

func (r *Resource) Ping(ctx context.Context) error {
	if r == nil || r.pingFn == nil {
		return errors.New("mysql resource is not initialized")
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

func (r *Resource) WithinTransaction(ctx context.Context, fn func(context.Context, *sql.Tx) error) (err error) {
	if r == nil || r.beginTxFn == nil {
		return errors.New("mysql resource is not initialized")
	}

	tx, err := r.beginTxFn(ctx, nil)
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

		if commitErr := tx.Commit(); commitErr != nil {
			err = fmt.Errorf("commit transaction: %w", commitErr)
		}
	}()

	err = fn(ctx, tx)
	return err
}

func rollback(ctx context.Context, tx *sql.Tx) error {
	if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
		return fmt.Errorf("rollback transaction: %w", err)
	}

	return nil
}

func quoteTimeZone(value string) string {
	if strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'") {
		return value
	}

	return "'" + value + "'"
}
