package cache

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

var errModelNotFound = errors.New("model not found")

func TestDisabledModelCacheInvokesLoader(t *testing.T) {
	modelCache, err := NewModelCache(Config{}, "identity", errModelNotFound)
	if err != nil {
		t.Fatalf("NewModelCache() error = %v", err)
	}

	loads := 0
	var value string
	err = modelCache.TakeCtx(context.Background(), &value, modelCache.Key("id", "1"), func(target any) error {
		loads++
		*target.(*string) = "database"
		return nil
	})
	if err != nil {
		t.Fatalf("TakeCtx() error = %v", err)
	}
	if loads != 1 || value != "database" {
		t.Fatalf("loads=%d value=%q, want 1/database", loads, value)
	}
	if err := modelCache.DelCtx(context.Background(), "unused"); err != nil {
		t.Fatalf("DelCtx() disabled cache error = %v", err)
	}
}

func TestModelCacheDelegatesAndDeduplicatesInvalidation(t *testing.T) {
	backend := &fakeModelCache{notFound: errModelNotFound}
	modelCache, err := NewModelCacheWithBackend(backend, "platform:model", errModelNotFound)
	if err != nil {
		t.Fatalf("NewModelCacheWithBackend() error = %v", err)
	}

	key := modelCache.Key("username", "张 三@example.com")
	if key != "platform:model:username:%E5%BC%A0+%E4%B8%89%40example.com" {
		t.Fatalf("Key() = %q", key)
	}

	var value string
	if err := modelCache.TakeCtx(context.Background(), &value, key, func(target any) error {
		*target.(*string) = "loaded"
		return nil
	}); err != nil {
		t.Fatalf("TakeCtx() error = %v", err)
	}
	if value != "loaded" || backend.takeKey != key {
		t.Fatalf("value=%q takeKey=%q", value, backend.takeKey)
	}

	if err := modelCache.DelCtx(context.Background(), "a", "", "a", "b"); err != nil {
		t.Fatalf("DelCtx() error = %v", err)
	}
	if !reflect.DeepEqual(backend.deleted, []string{"a", "b"}) {
		t.Fatalf("deleted=%v, want [a b]", backend.deleted)
	}
	if !modelCache.IsNotFound(errModelNotFound) {
		t.Fatal("IsNotFound() = false, want true")
	}
}

type fakeModelCache struct {
	notFound error
	takeKey  string
	deleted  []string
}

func (f *fakeModelCache) Del(keys ...string) error {
	return f.DelCtx(context.Background(), keys...)
}
func (f *fakeModelCache) DelCtx(_ context.Context, keys ...string) error {
	f.deleted = append([]string(nil), keys...)
	return nil
}
func (f *fakeModelCache) Get(string, any) error { return f.notFound }
func (f *fakeModelCache) GetCtx(context.Context, string, any) error {
	return f.notFound
}
func (f *fakeModelCache) IsNotFound(err error) bool { return errors.Is(err, f.notFound) }
func (f *fakeModelCache) Set(string, any) error     { return nil }
func (f *fakeModelCache) SetCtx(context.Context, string, any) error {
	return nil
}
func (f *fakeModelCache) SetWithExpire(string, any, time.Duration) error { return nil }
func (f *fakeModelCache) SetWithExpireCtx(context.Context, string, any, time.Duration) error {
	return nil
}
func (f *fakeModelCache) Take(value any, key string, query func(any) error) error {
	return f.TakeCtx(context.Background(), value, key, query)
}
func (f *fakeModelCache) TakeCtx(_ context.Context, value any, key string, query func(any) error) error {
	f.takeKey = key
	return query(value)
}
func (f *fakeModelCache) TakeWithExpire(value any, key string, query func(any, time.Duration) error) error {
	return f.TakeWithExpireCtx(context.Background(), value, key, query)
}
func (f *fakeModelCache) TakeWithExpireCtx(_ context.Context, value any, key string, query func(any, time.Duration) error) error {
	f.takeKey = key
	return query(value, time.Minute)
}
