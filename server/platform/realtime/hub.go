package realtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"
)

var topicPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9:._-]{0,127}$`)

var builtInTypes = map[string]struct{}{
	TypeSystemHello:      {},
	TypeSystemPing:       {},
	TypeSystemPong:       {},
	TypeSystemAck:        {},
	TypeSystemError:      {},
	TypeTopicSubscribe:   {},
	TypeTopicUnsubscribe: {},
}

type Hub struct {
	config        Config
	authenticator Authenticator
	metrics       Metrics
	now           func() time.Time

	mu       sync.RWMutex
	clients  map[string]*client
	accounts map[string]map[string]*client
	topics   map[string]map[string]*client
	handlers map[string]Handler
	closing  bool
}

func New(config Config, authenticator Authenticator, metrics Metrics) (*Hub, error) {
	config.Prepare()
	if err := config.Validate(); err != nil {
		return nil, err
	}
	if !config.Enabled {
		return nil, fmt.Errorf("realtime transport is disabled")
	}
	if authenticator == nil {
		return nil, fmt.Errorf("realtime authenticator is required")
	}
	if metrics == nil {
		metrics = noopMetrics{}
	}
	return &Hub{
		config:        config,
		authenticator: authenticator,
		metrics:       metrics,
		now:           time.Now,
		clients:       make(map[string]*client),
		accounts:      make(map[string]map[string]*client),
		topics:        make(map[string]map[string]*client),
		handlers:      make(map[string]Handler),
	}, nil
}

func (h *Hub) Config() Config {
	return h.config
}

func (h *Hub) RegisterHandler(messageType string, handler Handler) error {
	messageType = strings.TrimSpace(messageType)
	if messageType == "" || len(messageType) > 128 {
		return ErrInvalidEnvelope
	}
	if _, reserved := builtInTypes[messageType]; reserved {
		return fmt.Errorf("realtime message type %q is reserved", messageType)
	}
	if handler == nil {
		return fmt.Errorf("realtime handler is required")
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closing {
		return ErrClosed
	}
	if _, exists := h.handlers[messageType]; exists {
		return fmt.Errorf("realtime handler %q is already registered", messageType)
	}
	h.handlers[messageType] = handler
	return nil
}

func (h *Hub) Publish(topic string, envelope Envelope) (int, error) {
	if err := validateTopic(topic); err != nil {
		return 0, err
	}
	envelope.Topic = topic
	payload, err := h.encode(envelope)
	if err != nil {
		return 0, err
	}

	h.mu.RLock()
	subscribers := h.topics[topic]
	clients := make([]*client, 0, len(subscribers))
	for _, connection := range subscribers {
		clients = append(clients, connection)
	}
	h.mu.RUnlock()

	delivered := 0
	for _, connection := range clients {
		if connection.enqueue(payload) {
			delivered++
		}
	}
	return delivered, nil
}

func (h *Hub) SendAccount(accountID string, envelope Envelope) (int, error) {
	accountID = strings.TrimSpace(accountID)
	if accountID == "" {
		return 0, fmt.Errorf("account id is required")
	}
	payload, err := h.encode(envelope)
	if err != nil {
		return 0, err
	}

	h.mu.RLock()
	connections := h.accounts[accountID]
	clients := make([]*client, 0, len(connections))
	for _, connection := range connections {
		clients = append(clients, connection)
	}
	h.mu.RUnlock()

	delivered := 0
	for _, connection := range clients {
		if connection.enqueue(payload) {
			delivered++
		}
	}
	return delivered, nil
}

func (h *Hub) Snapshot() Snapshot {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return Snapshot{
		ActiveConnections: len(h.clients),
		Accounts:          len(h.accounts),
		Topics:            len(h.topics),
		Handlers:          len(h.handlers),
		Closing:           h.closing,
	}
}

func (h *Hub) Close(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}

	h.mu.Lock()
	if h.closing {
		h.mu.Unlock()
		return nil
	}
	h.closing = true
	clients := make([]*client, 0, len(h.clients))
	for _, connection := range h.clients {
		clients = append(clients, connection)
	}
	h.mu.Unlock()

	for _, connection := range clients {
		connection.terminate(closeServiceRestart, "server shutting down")
	}
	for _, connection := range clients {
		select {
		case <-connection.finished:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

func (h *Hub) register(connection *client) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closing {
		return ErrClosed
	}
	if len(h.clients) >= h.config.MaxConnections {
		return ErrConnectionLimit
	}
	accountConnections := h.accounts[connection.identity.AccountID]
	if len(accountConnections) >= h.config.MaxConnectionsPerAccount {
		return ErrAccountLimit
	}
	if accountConnections == nil {
		accountConnections = make(map[string]*client)
		h.accounts[connection.identity.AccountID] = accountConnections
	}
	h.clients[connection.id] = connection
	accountConnections[connection.id] = connection
	h.metrics.ConnectionAccepted()
	return nil
}

func (h *Hub) unregister(connection *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, exists := h.clients[connection.id]; !exists {
		return
	}
	delete(h.clients, connection.id)
	if accountConnections := h.accounts[connection.identity.AccountID]; accountConnections != nil {
		delete(accountConnections, connection.id)
		if len(accountConnections) == 0 {
			delete(h.accounts, connection.identity.AccountID)
		}
	}
	for topic := range connection.topics {
		if subscribers := h.topics[topic]; subscribers != nil {
			delete(subscribers, connection.id)
			if len(subscribers) == 0 {
				delete(h.topics, topic)
			}
		}
	}
	h.metrics.ConnectionClosed()
}

func (h *Hub) subscribe(connection *client, topic string) error {
	if err := validateTopic(topic); err != nil {
		return err
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, exists := h.clients[connection.id]; !exists {
		return ErrClosed
	}
	if connection.topics == nil {
		connection.topics = make(map[string]struct{})
	}
	if _, exists := connection.topics[topic]; exists {
		return nil
	}
	connection.topics[topic] = struct{}{}
	subscribers := h.topics[topic]
	if subscribers == nil {
		subscribers = make(map[string]*client)
		h.topics[topic] = subscribers
	}
	subscribers[connection.id] = connection
	return nil
}

func (h *Hub) unsubscribe(connection *client, topic string) error {
	if err := validateTopic(topic); err != nil {
		return err
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(connection.topics, topic)
	if subscribers := h.topics[topic]; subscribers != nil {
		delete(subscribers, connection.id)
		if len(subscribers) == 0 {
			delete(h.topics, topic)
		}
	}
	return nil
}

func (h *Hub) handle(ctx context.Context, connection *client, envelope Envelope) {
	if err := validateEnvelope(envelope); err != nil {
		connection.sendError(envelope.ID, "INVALID_ENVELOPE", "message envelope is invalid")
		return
	}

	switch envelope.Type {
	case TypeSystemPing:
		connection.sendEnvelope(Envelope{ID: envelope.ID, Type: TypeSystemPong, Payload: envelope.Payload})
		return
	case TypeTopicSubscribe, TypeTopicUnsubscribe:
		var request TopicPayload
		if err := json.Unmarshal(envelope.Payload, &request); err != nil {
			connection.sendError(envelope.ID, "INVALID_TOPIC", "topic payload is invalid")
			return
		}
		var err error
		if envelope.Type == TypeTopicSubscribe {
			err = h.subscribe(connection, request.Topic)
		} else {
			err = h.unsubscribe(connection, request.Topic)
		}
		if err != nil {
			connection.sendError(envelope.ID, "INVALID_TOPIC", "topic is invalid")
			return
		}
		connection.sendEnvelope(Envelope{
			ID:   envelope.ID,
			Type: TypeSystemAck,
			Payload: mustJSON(AckPayload{
				RequestID: envelope.ID,
				Action:    envelope.Type,
				Topic:     request.Topic,
			}),
		})
		return
	}

	h.mu.RLock()
	handler := h.handlers[envelope.Type]
	h.mu.RUnlock()
	if handler == nil {
		connection.sendError(envelope.ID, "UNKNOWN_TYPE", ErrUnknownMessageType.Error())
		return
	}

	response, err := handler(ctx, connection.context(), envelope)
	if err != nil {
		connection.sendError(envelope.ID, "HANDLER_ERROR", "message handling failed")
		return
	}
	if response != nil {
		if response.ID == "" {
			response.ID = envelope.ID
		}
		connection.sendEnvelope(*response)
	}
}

func (h *Hub) encode(envelope Envelope) ([]byte, error) {
	if err := validateEnvelope(envelope); err != nil {
		return nil, err
	}
	if envelope.SentAt.IsZero() {
		envelope.SentAt = h.now().UTC()
	}
	return json.Marshal(envelope)
}

func validateTopic(topic string) error {
	if topic == "" || topic != strings.TrimSpace(topic) || !topicPattern.MatchString(topic) {
		return ErrInvalidTopic
	}
	return nil
}

func mustJSON(value any) json.RawMessage {
	payload, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return payload
}

func rejectionReason(err error) string {
	switch {
	case errors.Is(err, ErrAuthenticationFailed):
		return "authentication"
	case errors.Is(err, ErrConnectionLimit):
		return "global_limit"
	case errors.Is(err, ErrAccountLimit):
		return "account_limit"
	case errors.Is(err, ErrClosed):
		return "closing"
	default:
		return "upgrade"
	}
}
