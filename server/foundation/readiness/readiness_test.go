package readiness

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestChecker(t *testing.T) {
	t.Parallel()

	checker := New(10*time.Millisecond,
		stubProbe{name: "mysql"},
		stubProbe{name: "redis", err: errors.New("down")},
	)

	status := checker.Check(context.Background())
	if status.Ready {
		t.Fatal("expected checker to be unready")
	}
	if len(status.Unavailable) != 1 || status.Unavailable[0] != "redis" {
		t.Fatalf("unexpected unavailable probes: %#v", status.Unavailable)
	}
}

type stubProbe struct {
	name string
	err  error
}

func (s stubProbe) Name() string               { return s.name }
func (s stubProbe) Ping(context.Context) error { return s.err }
