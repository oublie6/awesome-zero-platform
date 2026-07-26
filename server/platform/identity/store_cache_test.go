package identity

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	foundationcache "github.com/oublie6/awesome-zero-platform/server/foundation/cache"
)

func TestInvalidateAccountsDeletesPrimaryAndOldNewUniqueKeys(t *testing.T) {
	backend := &recordingIdentityCache{notFound: ErrAccountNotFound}
	modelCache, err := foundationcache.NewModelCacheWithBackend(backend, "platform:identity", ErrAccountNotFound)
	if err != nil {
		t.Fatalf("NewModelCacheWithBackend() error = %v", err)
	}
	store := NewCachedMySQLStore(nil, modelCache)

	previous := accountRecord{
		ID:          "account-1",
		UsernameKey: "old-user",
		EmailKey:    "old@example.com",
		PhoneKey:    "+8613800000000",
	}
	next := accountRecord{
		ID:          "account-1",
		UsernameKey: "new-user",
		EmailKey:    "new@example.com",
		PhoneKey:    "+8613900000000",
	}
	if err := store.invalidateAccounts(context.Background(), previous, next); err != nil {
		t.Fatalf("invalidateAccounts() error = %v", err)
	}

	want := []string{
		"platform:identity:id:account-1",
		"platform:identity:username:old-user",
		"platform:identity:email:old%40example.com",
		"platform:identity:phone:%2B8613800000000",
		"platform:identity:username:new-user",
		"platform:identity:email:new%40example.com",
		"platform:identity:phone:%2B8613900000000",
	}
	if !reflect.DeepEqual(backend.deleted, want) {
		t.Fatalf("deleted keys = %#v, want %#v", backend.deleted, want)
	}
}

func TestCachedAccountReadUsesLoaderThroughCache(t *testing.T) {
	backend := &recordingIdentityCache{notFound: ErrAccountNotFound}
	modelCache, err := foundationcache.NewModelCacheWithBackend(backend, "platform:identity", ErrAccountNotFound)
	if err != nil {
		t.Fatalf("NewModelCacheWithBackend() error = %v", err)
	}
	store := NewCachedMySQLStore(nil, modelCache)

	loads := 0
	var account Account
	err = store.takeAccount(context.Background(), &account, store.accountIDKey("account-1"), func() (Account, error) {
		loads++
		return Account{ID: "account-1", DisplayName: "Cached User"}, nil
	})
	if err != nil {
		t.Fatalf("takeAccount() error = %v", err)
	}
	if loads != 1 || account.ID != "account-1" || backend.takeKey != "platform:identity:id:account-1" {
		t.Fatalf("loads=%d account=%#v takeKey=%q", loads, account, backend.takeKey)
	}
}

type recordingIdentityCache struct {
	notFound error
	takeKey  string
	deleted  []string
}

func (f *recordingIdentityCache) Del(keys ...string) error {
	return f.DelCtx(context.Background(), keys...)
}
func (f *recordingIdentityCache) DelCtx(_ context.Context, keys ...string) error {
	f.deleted = append([]string(nil), keys...)
	return nil
}
func (f *recordingIdentityCache) Get(string, any) error { return f.notFound }
func (f *recordingIdentityCache) GetCtx(context.Context, string, any) error {
	return f.notFound
}
func (f *recordingIdentityCache) IsNotFound(err error) bool { return errors.Is(err, f.notFound) }
func (f *recordingIdentityCache) Set(string, any) error      { return nil }
func (f *recordingIdentityCache) SetCtx(context.Context, string, any) error {
	return nil
}
func (f *recordingIdentityCache) SetWithExpire(string, any, time.Duration) error { return nil }
func (f *recordingIdentityCache) SetWithExpireCtx(context.Context, string, any, time.Duration) error {
	return nil
}
func (f *recordingIdentityCache) Take(value any, key string, query func(any) error) error {
	return f.TakeCtx(context.Background(), value, key, query)
}
func (f *recordingIdentityCache) TakeCtx(_ context.Context, value any, key string, query func(any) error) error {
	f.takeKey = key
	return query(value)
}
func (f *recordingIdentityCache) TakeWithExpire(value any, key string, query func(any, time.Duration) error) error {
	return f.TakeWithExpireCtx(context.Background(), value, key, query)
}
func (f *recordingIdentityCache) TakeWithExpireCtx(_ context.Context, value any, key string, query func(any, time.Duration) error) error {
	f.takeKey = key
	return query(value, time.Minute)
}
