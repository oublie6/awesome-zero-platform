package realtime

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

type mutableSessionAuthenticator struct {
	mu          sync.RWMutex
	valid       bool
	unavailable bool
}

func (a *mutableSessionAuthenticator) Authenticate(_ context.Context, token string) (Identity, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if token != "test-platform-token" || !a.valid || a.unavailable {
		return Identity{}, ErrAuthenticationFailed
	}
	return Identity{AccountID: "test-account", SessionID: "test-session", DisplayName: "Test"}, nil
}
func (a *mutableSessionAuthenticator) ValidateSession(_ context.Context, id Identity) error {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.unavailable {
		return errors.New("session store unavailable")
	}
	if !a.valid || id.AccountID != "test-account" || id.SessionID != "test-session" {
		return ErrAuthenticationFailed
	}
	return nil
}
func (a *mutableSessionAuthenticator) expire() {
	a.mu.Lock()
	a.valid = false
	a.mu.Unlock()
}
func (a *mutableSessionAuthenticator) unavailableNow() {
	a.mu.Lock()
	a.unavailable = true
	a.mu.Unlock()
}
func newRevalidationTestHub(t *testing.T, auth *mutableSessionAuthenticator, interval time.Duration) (*Hub, string) {
	t.Helper()
	config := Config{Enabled: true}
	config.Prepare()
	config.PingInterval = interval
	config.PongTimeout = interval * 4
	hub, err := New(config, auth, newTestMetrics())
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(hub.Handler())
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = hub.Close(ctx)
		server.Close()
	})
	return hub, "ws" + strings.TrimPrefix(server.URL, "http")
}
func connectRevalidated(t *testing.T, endpoint string) *websocket.Conn {
	t.Helper()
	headers := http.Header{}
	headers.Set("Authorization", "Bearer test-platform-token")
	conn, resp, err := websocket.DefaultDialer.Dial(endpoint, headers)
	if err != nil {
		t.Fatalf("dial: %v, response: %v", err, resp)
	}
	t.Cleanup(func() { _ = conn.Close() })
	if hello := readEnvelope(t, conn); hello.Type != TypeSystemHello {
		t.Fatalf("unexpected hello: %q", hello.Type)
	}
	return conn
}
func assertSocketClosed(t *testing.T, c *websocket.Conn) {
	t.Helper()
	_ = c.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, payload, err := c.ReadMessage()
	if err == nil {
		t.Fatalf("revoked socket still received message: %s", payload)
	}
	if ne, ok := err.(interface{ Timeout() bool }); ok && ne.Timeout() {
		t.Fatalf("revoked socket was not terminated: %v", err)
	}
}
func awaitNoClients(t *testing.T, h *Hub) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for h.Snapshot().ActiveConnections != 0 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	got := h.Snapshot()
	if got.ActiveConnections != 0 || got.Accounts != 0 || got.Topics != 0 {
		t.Fatalf("revoked connection still registered: %+v", got)
	}
}

func TestExistingSocketCannotDispatchAfterRevocation(t *testing.T) {
	for _, cause := range []string{"logout", "wechat-unlink", "wechat-relink", "password-reset", "account-disabled", "session-epoch-changed", "redis-unavailable"} {
		t.Run(cause, func(t *testing.T) {
			auth := &mutableSessionAuthenticator{valid: true}
			hub, endpoint := newRevalidationTestHub(t, auth, time.Second)
			var dispatched atomic.Int64
			if err := hub.RegisterHandler("game.command", func(_ context.Context, id ConnectionContext, envelope Envelope) (*Envelope, error) {
				if id.AccountID != "test-account" {
					t.Errorf("unexpected actor: %q", id.AccountID)
				}
				dispatched.Add(1)
				return &Envelope{Type: "game.result", ID: envelope.ID}, nil
			}); err != nil {
				t.Fatal(err)
			}
			conn := connectRevalidated(t, endpoint)
			if err := conn.WriteJSON(Envelope{ID: "before", Type: "game.command"}); err != nil {
				t.Fatal(err)
			}
			if msg := readEnvelope(t, conn); msg.Type != "game.result" {
				t.Fatalf("pre-revoke request: %+v", msg)
			}
			if cause == "redis-unavailable" {
				auth.unavailableNow()
			} else {
				auth.expire()
			}
			if err := conn.WriteJSON(Envelope{ID: "after", Type: "game.command"}); err != nil {
				t.Fatal(err)
			}
			assertSocketClosed(t, conn)
			if dispatched.Load() != 1 {
				t.Fatalf("revoked session dispatched application handler %d times", dispatched.Load())
			}
			awaitNoClients(t, hub)
		})
	}
}

func TestRevokedSocketCannotReceiveQueuedAccountOrTopicEvents(t *testing.T) {
	for _, route := range []string{"account", "topic"} {
		t.Run(route, func(t *testing.T) {
			auth := &mutableSessionAuthenticator{valid: true}
			hub, endpoint := newRevalidationTestHub(t, auth, time.Second)
			conn := connectRevalidated(t, endpoint)
			if route == "topic" {
				if err := conn.WriteJSON(Envelope{ID: "sub", Type: TypeTopicSubscribe, Payload: []byte(`{"topic":"table.demo"}`)}); err != nil {
					t.Fatal(err)
				}
				if got := readEnvelope(t, conn); got.Type != TypeSystemAck {
					t.Fatalf("subscription failed: %+v", got)
				}
			}
			auth.expire()
			var err error
			if route == "account" {
				_, err = hub.SendAccount("test-account", Envelope{Type: "game.private"})
			} else {
				_, err = hub.Publish("table.demo", Envelope{Type: "game.event"})
			}
			if err != nil {
				t.Fatal(err)
			}
			assertSocketClosed(t, conn)
			awaitNoClients(t, hub)
		})
	}
}

func TestIdleSocketClosesAfterSessionExpires(t *testing.T) {
	auth := &mutableSessionAuthenticator{valid: true}
	hub, endpoint := newRevalidationTestHub(t, auth, 25*time.Millisecond)
	conn := connectRevalidated(t, endpoint)
	auth.expire()
	assertSocketClosed(t, conn)
	awaitNoClients(t, hub)
}

func TestRevokedSocketCannotReconnectWithOldToken(t *testing.T) {
	auth := &mutableSessionAuthenticator{valid: true}
	_, endpoint := newRevalidationTestHub(t, auth, time.Second)
	conn := connectRevalidated(t, endpoint)
	auth.expire()
	assertSocketClosed(t, conn)
	headers := http.Header{}
	headers.Set("Authorization", "Bearer test-platform-token")
	other, response, err := websocket.DefaultDialer.Dial(endpoint, headers)
	if other != nil {
		_ = other.Close()
	}
	if err == nil || response == nil || response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("old credentials reconnected: status=%v, error=%v", response, err)
	}
	_ = response.Body.Close()
}
