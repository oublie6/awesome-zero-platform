package cache

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/redis/go-redis/v9/logging"
)

type Handle interface {
	Client() *redis.Client
	Ping(context.Context) error
	Close() error
}

type Config struct {
	Addr             string        `json:",optional"`
	Username         string        `json:",optional"`
	Password         string        `json:",optional"`
	DB               int           `json:",default=0"`
	PoolSize         int           `json:",default=10"`
	DialTimeout      time.Duration `json:",default=3s"`
	ReadTimeout      time.Duration `json:",default=3s"`
	WriteTimeout     time.Duration `json:",default=3s"`
	StartupTimeout   time.Duration `json:",default=3s"`
	ReadinessTimeout time.Duration `json:",default=2s"`
}

type Resource struct {
	client    *redis.Client
	closeFn   func() error
	pingFn    func(context.Context) error
	closeOnce sync.Once
	closeErr  error
}

type StartupError struct {
	Dependency string
	Cause      error
}

func init() {
	logging.Disable()
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
	if c.DB == 0 {
		c.DB = 0
	}
	if c.PoolSize == 0 {
		c.PoolSize = 10
	}
	if c.DialTimeout == 0 {
		c.DialTimeout = 3 * time.Second
	}
	if c.ReadTimeout == 0 {
		c.ReadTimeout = 3 * time.Second
	}
	if c.WriteTimeout == 0 {
		c.WriteTimeout = 3 * time.Second
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
		return fmt.Errorf("redis.addr must not be empty")
	}
	if c.DB < 0 {
		return fmt.Errorf("redis.db must be greater than or equal to 0")
	}
	if c.PoolSize < 1 {
		return fmt.Errorf("redis.poolSize must be greater than 0")
	}
	if c.DialTimeout <= 0 {
		return fmt.Errorf("redis.dialTimeout must be greater than 0")
	}
	if c.ReadTimeout <= 0 {
		return fmt.Errorf("redis.readTimeout must be greater than 0")
	}
	if c.WriteTimeout <= 0 {
		return fmt.Errorf("redis.writeTimeout must be greater than 0")
	}
	if c.StartupTimeout <= 0 {
		return fmt.Errorf("redis.startupTimeout must be greater than 0")
	}
	if c.ReadinessTimeout <= 0 {
		return fmt.Errorf("redis.readinessTimeout must be greater than 0")
	}

	return nil
}

func Open(ctx context.Context, cfg Config) (Handle, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         cfg.Addr,
		Username:     cfg.Username,
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	})

	resource := &Resource{
		client:  client,
		closeFn: client.Close,
		pingFn: func(ctx context.Context) error {
			return client.Ping(ctx).Err()
		},
	}

	checkCtx, cancel := context.WithTimeout(ctx, cfg.StartupTimeout)
	defer cancel()
	if err := resource.Ping(checkCtx); err != nil {
		_ = resource.Close()
		return nil, &StartupError{
			Dependency: "redis",
			Cause:      err,
		}
	}

	return resource, nil
}

func (r *Resource) Client() *redis.Client {
	if r == nil {
		return nil
	}

	return r.client
}

func (r *Resource) Ping(ctx context.Context) error {
	if r == nil || r.pingFn == nil {
		return fmt.Errorf("redis resource is not initialized")
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
