package cache

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	zerocache "github.com/zeromicro/go-zero/core/stores/cache"
	zeroredis "github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/syncx"
)

// ModelCache provides the cache-aside behavior used by entity persistence
// adapters. It deliberately exposes only key construction, cached reads and
// invalidation so Redis details do not escape into application services.
type ModelCache struct {
	backend     zerocache.Cache
	prefix      string
	errNotFound error
}

// NewModelCache creates a go-zero Redis cache for one persistence namespace.
// When caching is disabled the returned object transparently invokes database
// loaders and treats invalidation as a no-op.
func NewModelCache(cfg Config, namespace string, errNotFound error) (*ModelCache, error) {
	if errNotFound == nil {
		return nil, fmt.Errorf("model cache not-found error is required")
	}
	if !cfg.Model.Enabled {
		return &ModelCache{errNotFound: errNotFound}, nil
	}
	namespace = strings.Trim(strings.TrimSpace(namespace), ":")
	if namespace == "" {
		return nil, fmt.Errorf("model cache namespace is required")
	}

	redisConf := zeroredis.RedisConf{
		Host:        cfg.Addr,
		Type:        zeroredis.NodeType,
		User:        cfg.Username,
		Pass:        cfg.Password,
		PingTimeout: cfg.StartupTimeout,
	}
	if err := redisConf.Validate(); err != nil {
		return nil, fmt.Errorf("validate model cache redis: %w", err)
	}

	cluster := zerocache.CacheConf{{RedisConf: redisConf, Weight: 100}}
	backend := zerocache.New(
		cluster,
		syncx.NewSingleFlight(),
		zerocache.NewStat("model_"+strings.ReplaceAll(namespace, ":", "_")),
		errNotFound,
		zerocache.WithExpiry(cfg.Model.TTL),
		zerocache.WithNotFoundExpiry(cfg.Model.NotFoundTTL),
	)

	prefix := strings.Trim(strings.TrimSpace(cfg.Model.KeyPrefix), ":")
	if prefix == "" {
		prefix = "awesome-zero-platform:model"
	}
	return &ModelCache{
		backend:     backend,
		prefix:      prefix + ":" + namespace,
		errNotFound: errNotFound,
	}, nil
}

// NewModelCacheWithBackend supports deterministic persistence-adapter tests.
func NewModelCacheWithBackend(backend zerocache.Cache, prefix string, errNotFound error) (*ModelCache, error) {
	if backend == nil {
		return nil, fmt.Errorf("model cache backend is required")
	}
	if errNotFound == nil {
		return nil, fmt.Errorf("model cache not-found error is required")
	}
	prefix = strings.Trim(strings.TrimSpace(prefix), ":")
	if prefix == "" {
		return nil, fmt.Errorf("model cache prefix is required")
	}
	return &ModelCache{backend: backend, prefix: prefix, errNotFound: errNotFound}, nil
}

// Key returns a deterministic, namespaced cache key.
func (c *ModelCache) Key(parts ...string) string {
	encoded := make([]string, 0, len(parts)+1)
	if c != nil && c.prefix != "" {
		encoded = append(encoded, c.prefix)
	}
	for _, part := range parts {
		encoded = append(encoded, url.QueryEscape(strings.TrimSpace(part)))
	}
	return strings.Join(encoded, ":")
}

// TakeCtx reads through go-zero cache. When caching is disabled it invokes the
// database loader directly.
func (c *ModelCache) TakeCtx(ctx context.Context, value any, key string, loader func(any) error) error {
	if loader == nil {
		return fmt.Errorf("model cache loader is required")
	}
	if c == nil || c.backend == nil {
		return loader(value)
	}
	return c.backend.TakeCtx(ctx, value, key, loader)
}

// DelCtx invalidates all supplied keys after a successful database write.
func (c *ModelCache) DelCtx(ctx context.Context, keys ...string) error {
	if c == nil || c.backend == nil || len(keys) == 0 {
		return nil
	}
	return c.backend.DelCtx(ctx, uniqueNonEmpty(keys)...)
}

// IsNotFound reports whether err matches the cache's configured not-found
// sentinel, including when caching is disabled.
func (c *ModelCache) IsNotFound(err error) bool {
	if err == nil || c == nil {
		return false
	}
	if c.backend != nil && c.backend.IsNotFound(err) {
		return true
	}
	return c.errNotFound != nil && errors.Is(err, c.errNotFound)
}

func uniqueNonEmpty(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}