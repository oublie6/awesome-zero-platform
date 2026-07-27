package realtime

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

type testAuthenticator struct{}

func (testAuthenticator) Authenticate(_ context.Context, token string) (Identity, error) {
	if token != "valid-token" {
		return Identity{}, ErrAuthenticationFailed
	}
	return Identity{AccountID: "account-1", SessionID: "session-1", DisplayName: "Player One"}, nil
}

type testMetrics struct {
	mu            sync.Mutex
	accepted      int
	closed        int
	rejected      map[string]int
	received      int
	sent          int
	slowConsumers int
}

func newTestMetrics() *testMetrics {
	return &testMetrics{rejected: make(map[string]int)}
}

func (m *testMetrics) ConnectionAccepted() {
	m.mu.Lock()
	m.accepted++
	m.mu.Unlock()
}

func (m *testMetrics) ConnectionRejected(reason string) {
	m.mu.Lock()
	m.rejected[reason]++
	m.mu.Unlock()
}

func (m *testMetrics) ConnectionClosed() {
	m.mu.Lock()
	m.closed++
	m.mu.Unlock()
}

func (m *testMetrics) MessageReceived() {
	m.mu.Lock()
	m.received++
	m.mu.Unlock()
}

func (m *testMetrics) MessageSent() {
	m.mu.Lock()
	m.sent++
	m.mu.Unlock()
}

func (m *testMetrics) SlowConsumer() {
	m.mu.Lock()
	m.slowConsumers++
	m.mu.Unlock()
}

func newWebSocketTestServer(t *testing.T, mutate func(*Config)) (*Hub, *testMetrics, *httptest.Server, string) {
	t.Helper()
	config := Config{Enabled: true}
	config.Prepare()
	config.PingInterval = time.Hour
	config.PongTimeout = 2 * time.Hour
	if mutate != nil {
		mutate(&config)
	}
	metrics := newTestMetrics()
	hub, err := New(config, testAuthenticator{}, metrics)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	server := httptest.NewServer(hub.Handler())
	endpoint := "ws" + strings.TrimPrefix(server.URL, "http")
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = hub.Close(ctx)
		server.Close()
	})
	return hub, metrics, server, endpoint
}

func dialNative(t *testing.T, endpoint, token string) (*websocket.Conn, *http.Response, error) {
	t.Helper()
	headers := http.Header{}
	if token != "" {
		headers.Set("Authorization", "Bearer "+token)
	}
	return websocket.DefaultDialer.Dial(endpoint, headers)
}

func readEnvelope(t *testing.T, connection *websocket.Conn) Envelope {
	t.Helper()
	if err := connection.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("SetReadDeadline() error = %v", err)
	}
	var envelope Envelope
	if err := connection.ReadJSON(&envelope); err != nil {
		t.Fatalf("ReadJSON() error = %v", err)
	}
	return envelope
}

func TestAuthenticatedWebSocketHelloAndPing(t *testing.T) {
	hub, metrics, _, endpoint := newWebSocketTestServer(t, nil)
	connection, response, err := dialNative(t, endpoint, "valid-token")
	if err != nil {
		t.Fatalf("Dial() error = %v, response = %#v", err, response)
	}
	defer connection.Close()

	hello := readEnvelope(t, connection)
	if hello.Type != TypeSystemHello {
		t.Fatalf("hello.Type = %q, want %q", hello.Type, TypeSystemHello)
	}
	var payload HelloPayload
	if err := json.Unmarshal(hello.Payload, &payload); err != nil {
		t.Fatalf("decode hello: %v", err)
	}
	if payload.ProtocolVersion != ProtocolVersion || payload.ConnectionID == "" || payload.AccountID != "account-1" {
		t.Fatalf("unexpected hello payload: %+v", payload)
	}

	if err := connection.WriteJSON(Envelope{ID: "ping-1", Type: TypeSystemPing}); err != nil {
		t.Fatalf("WriteJSON() error = %v", err)
	}
	pong := readEnvelope(t, connection)
	if pong.Type != TypeSystemPong || pong.ID != "ping-1" {
		t.Fatalf("pong = %+v", pong)
	}
	if snapshot := hub.Snapshot(); snapshot.ActiveConnections != 1 || snapshot.Accounts != 1 {
		t.Fatalf("Snapshot() = %+v", snapshot)
	}
	metrics.mu.Lock()
	defer metrics.mu.Unlock()
	if metrics.accepted != 1 || metrics.received != 1 || metrics.sent < 2 {
		t.Fatalf("metrics = %+v", metrics)
	}
}

func TestBrowserSubprotocolAuthenticatesWithoutEchoingToken(t *testing.T) {
	_, _, _, endpoint := newWebSocketTestServer(t, nil)
	dialer := websocket.Dialer{Subprotocols: []string{"bearer", "valid-token"}, HandshakeTimeout: 2 * time.Second}
	connection, response, err := dialer.Dial(endpoint, nil)
	if err != nil {
		t.Fatalf("Dial() error = %v, response = %#v", err, response)
	}
	defer connection.Close()
	if connection.Subprotocol() != "bearer" {
		t.Fatalf("Subprotocol() = %q, want bearer", connection.Subprotocol())
	}
	if connection.Subprotocol() == "valid-token" {
		t.Fatal("server echoed the access token as a subprotocol")
	}
	if hello := readEnvelope(t, connection); hello.Type != TypeSystemHello {
		t.Fatalf("hello.Type = %q", hello.Type)
	}
}

func TestHandshakeRejectsMissingInvalidAndQueryTokens(t *testing.T) {
	_, metrics, _, endpoint := newWebSocketTestServer(t, nil)
	tests := []struct {
		name       string
		endpoint   string
		token      string
		wantStatus int
	}{
		{name: "missing", endpoint: endpoint, wantStatus: http.StatusUnauthorized},
		{name: "invalid", endpoint: endpoint, token: "wrong", wantStatus: http.StatusUnauthorized},
		{name: "query access token", endpoint: endpoint + "?access_token=valid-token", wantStatus: http.StatusBadRequest},
		{name: "query token", endpoint: endpoint + "?token=valid-token", wantStatus: http.StatusBadRequest},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			connection, response, err := dialNative(t, test.endpoint, test.token)
			if connection != nil {
				connection.Close()
			}
			if err == nil {
				t.Fatal("Dial() error = nil, want failed handshake")
			}
			if response == nil || response.StatusCode != test.wantStatus {
				t.Fatalf("response = %#v, want HTTP %d", response, test.wantStatus)
			}
			response.Body.Close()
		})
	}

	metrics.mu.Lock()
	defer metrics.mu.Unlock()
	if metrics.rejected["authentication"] != 2 || metrics.rejected["query_token"] != 2 {
		t.Fatalf("rejected metrics = %+v", metrics.rejected)
	}
}

func TestTopicPublishAccountSendAndCustomHandler(t *testing.T) {
	hub, _, _, endpoint := newWebSocketTestServer(t, nil)
	if err := hub.RegisterHandler("game.echo", func(_ context.Context, connection ConnectionContext, envelope Envelope) (*Envelope, error) {
		if connection.AccountID != "account-1" {
			return nil, errors.New("unexpected account")
		}
		return &Envelope{Type: "game.echoed", Payload: envelope.Payload}, nil
	}); err != nil {
		t.Fatalf("RegisterHandler() error = %v", err)
	}

	connection, _, err := dialNative(t, endpoint, "valid-token")
	if err != nil {
		t.Fatalf("Dial() error = %v", err)
	}
	defer connection.Close()
	_ = readEnvelope(t, connection)

	subscribePayload, _ := json.Marshal(TopicPayload{Topic: "game:table-42"})
	if err := connection.WriteJSON(Envelope{ID: "subscribe-1", Type: TypeTopicSubscribe, Payload: subscribePayload}); err != nil {
		t.Fatalf("subscribe WriteJSON() error = %v", err)
	}
	if ack := readEnvelope(t, connection); ack.Type != TypeSystemAck || ack.ID != "subscribe-1" {
		t.Fatalf("subscribe ack = %+v", ack)
	}

	delivered, err := hub.Publish("game:table-42", Envelope{Type: "game.state", Payload: json.RawMessage(`{"turn":2}`)})
	if err != nil || delivered != 1 {
		t.Fatalf("Publish() = %d, %v", delivered, err)
	}
	if event := readEnvelope(t, connection); event.Type != "game.state" || event.Topic != "game:table-42" {
		t.Fatalf("topic event = %+v", event)
	}

	delivered, err = hub.SendAccount("account-1", Envelope{Type: "account.notice"})
	if err != nil || delivered != 1 {
		t.Fatalf("SendAccount() = %d, %v", delivered, err)
	}
	if event := readEnvelope(t, connection); event.Type != "account.notice" {
		t.Fatalf("account event = %+v", event)
	}

	if err := connection.WriteJSON(Envelope{ID: "echo-1", Type: "game.echo", Payload: json.RawMessage(`{"value":7}`)}); err != nil {
		t.Fatalf("echo WriteJSON() error = %v", err)
	}
	if response := readEnvelope(t, connection); response.Type != "game.echoed" || response.ID != "echo-1" {
		t.Fatalf("handler response = %+v", response)
	}
}

func TestPerAccountLimitAndSlowConsumerBound(t *testing.T) {
	hub, metrics, _, endpoint := newWebSocketTestServer(t, func(config *Config) {
		config.MaxConnections = 2
		config.MaxConnectionsPerAccount = 1
		config.SendQueueSize = 1
	})
	first, _, err := dialNative(t, endpoint, "valid-token")
	if err != nil {
		t.Fatalf("first Dial() error = %v", err)
	}
	defer first.Close()
	_ = readEnvelope(t, first)

	second, _, err := dialNative(t, endpoint, "valid-token")
	if err != nil {
		t.Fatalf("second Dial() unexpectedly failed before close frame: %v", err)
	}
	defer second.Close()
	_ = second.SetReadDeadline(time.Now().Add(2 * time.Second))
	if _, _, err := second.ReadMessage(); err == nil {
		t.Fatal("second connection remained open despite per-account limit")
	}
	if snapshot := hub.Snapshot(); snapshot.ActiveConnections != 1 {
		t.Fatalf("Snapshot() = %+v", snapshot)
	}

	bounded := newClient(hub, nil, "bounded", Identity{AccountID: "account-2", SessionID: "session-2"})
	if err := hub.register(bounded); err != nil {
		t.Fatalf("register bounded client: %v", err)
	}
	if !bounded.enqueue([]byte(`{"type":"first"}`)) {
		t.Fatal("first bounded enqueue failed")
	}
	if bounded.enqueue([]byte(`{"type":"second"}`)) {
		t.Fatal("second bounded enqueue succeeded despite full queue")
	}
	metrics.mu.Lock()
	defer metrics.mu.Unlock()
	if metrics.rejected["account_limit"] != 1 || metrics.slowConsumers != 1 {
		t.Fatalf("metrics = %+v", metrics)
	}
}
