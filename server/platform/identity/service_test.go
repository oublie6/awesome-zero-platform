package identity

import (
	"context"
	"testing"
	"time"
)

type countingHasher struct {
	hashCalls int
}

func (h *countingHasher) Hash(string) (string, error) {
	h.hashCalls++
	return "$argon2id$v=19$m=8192,t=1,p=1$ZmFrZXNhbHQ$ZmFrZWhhc2g", nil
}

func (h *countingHasher) Verify(string, string) (bool, error) {
	return true, nil
}

func (h *countingHasher) Params() PasswordParams {
	return TestPasswordParams
}

func TestCreateAccountRejectsInvalidPasswordBeforeHashing(t *testing.T) {
	t.Parallel()

	hasher := &countingHasher{}
	service := newService(serviceDependencies{
		hasher: hasher,
		now:    func() time.Time { return time.Unix(0, 0) },
	})

	_, err := service.CreateAccount(context.Background(), CreateAccountInput{
		Identity: Identity{
			Email: "alice@example.com",
		},
		DisplayName: "Alice",
		Password:    "short",
	})
	if err == nil {
		t.Fatal("expected password validation error, got nil")
	}
	if hasher.hashCalls != 0 {
		t.Fatalf("hashCalls = %d, want 0", hasher.hashCalls)
	}
}
