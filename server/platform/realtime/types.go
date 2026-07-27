package realtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	ProtocolVersion = "1"

	TypeSystemHello      = "system.hello"
	TypeSystemPing       = "system.ping"
	TypeSystemPong       = "system.pong"
	TypeSystemAck        = "system.ack"
	TypeSystemError      = "system.error"
	TypeTopicSubscribe   = "topic.subscribe"
	TypeTopicUnsubscribe = "topic.unsubscribe"
)

var (
	ErrClosed               = errors.New("realtime hub is closed")
	ErrConnectionLimit      = errors.New("realtime connection limit reached")
	ErrAccountLimit         = errors.New("realtime account connection limit reached")
	ErrUnknownMessageType   = errors.New("unknown realtime message type")
	ErrInvalidEnvelope      = errors.New("invalid realtime envelope")
	ErrInvalidTopic         = errors.New("invalid realtime topic")
	ErrSlowConsumer         = errors.New("realtime client send queue is full")
	ErrAuthenticationFailed = errors.New("realtime authentication failed")
)

type Config struct {
	Enabled                  bool
	Path                     string        `json:",default=/ws"`
	AllowedOrigins           []string      `json:",optional"`
	MaxConnections           int           `json:",default=4096"`
	MaxConnectionsPerAccount int           `json:",default=4"`
	SendQueueSize            int           `json:",default=64"`
	MaxMessageBytes          int64         `json:",default=65536"`
	HandshakeTimeout         time.Duration `json:",default=5s"`
	WriteTimeout             time.Duration `json:",default=10s"`
	PongTimeout              time.Duration `json:",default=60s"`
	PingInterval             time.Duration `json:",default=25s"`
}

func (c *Config) Prepare() {
	if strings.TrimSpace(c.Path) == "" {
		c.Path = "/ws"
	}
	if c.MaxConnections == 0 {
		c.MaxConnections = 4096
	}
	if c.MaxConnectionsPerAccount == 0 {
		c.MaxConnectionsPerAccount = 4
	}
	if c.SendQueueSize == 0 {
		c.SendQueueSize = 64
	}
	if c.MaxMessageBytes == 0 {
		c.MaxMessageBytes = 64 * 1024
	}
	if c.HandshakeTimeout == 0 {
		c.HandshakeTimeout = 5 * time.Second
	}
	if c.WriteTimeout == 0 {
		c.WriteTimeout = 10 * time.Second
	}
	if c.PongTimeout == 0 {
		c.PongTimeout = 60 * time.Second
	}
	if c.PingInterval == 0 {
		c.PingInterval = 25 * time.Second
	}
	for index := range c.AllowedOrigins {
		c.AllowedOrigins[index] = strings.TrimSpace(c.AllowedOrigins[index])
	}
}

func (c Config) Validate() error {
	if !c.Enabled {
		return nil
	}
	if !strings.HasPrefix(c.Path, "/") || strings.ContainsAny(c.Path, "?#") {
		return fmt.Errorf("realtime path must be an absolute HTTP path")
	}
	if c.MaxConnections <= 0 {
		return fmt.Errorf("realtime max connections must be greater than zero")
	}
	if c.MaxConnectionsPerAccount <= 0 || c.MaxConnectionsPerAccount > c.MaxConnections {
		return fmt.Errorf("realtime per-account connection limit must be between one and max connections")
	}
	if c.SendQueueSize <= 0 {
		return fmt.Errorf("realtime send queue size must be greater than zero")
	}
	if c.MaxMessageBytes < 1024 {
		return fmt.Errorf("realtime max message bytes must be at least 1024")
	}
	if c.HandshakeTimeout <= 0 || c.WriteTimeout <= 0 || c.PongTimeout <= 0 || c.PingInterval <= 0 {
		return fmt.Errorf("realtime timeouts must be greater than zero")
	}
	if c.PingInterval >= c.PongTimeout {
		return fmt.Errorf("realtime ping interval must be shorter than pong timeout")
	}
	for _, origin := range c.AllowedOrigins {
		if origin == "" {
			continue
		}
		if origin == "*" {
			return fmt.Errorf("realtime allowed origins must not contain wildcard origins")
		}
		if !strings.HasPrefix(origin, "http://") && !strings.HasPrefix(origin, "https://") {
			return fmt.Errorf("realtime allowed origin %q must include http or https scheme", origin)
		}
	}
	return nil
}

type Identity struct {
	AccountID   string
	SessionID   string
	DisplayName string
}

type Authenticator interface {
	Authenticate(context.Context, string) (Identity, error)
}

type Envelope struct {
	ID      string          `json:"id,omitempty"`
	Type    string          `json:"type"`
	Topic   string          `json:"topic,omitempty"`
	Payload json.RawMessage `json:"payload,omitempty"`
	SentAt  time.Time       `json:"sentAt,omitempty"`
}

type ConnectionContext struct {
	ConnectionID string
	AccountID    string
	SessionID    string
	DisplayName  string
}

type Handler func(context.Context, ConnectionContext, Envelope) (*Envelope, error)

type Snapshot struct {
	ActiveConnections int  `json:"activeConnections"`
	Accounts          int  `json:"accounts"`
	Topics            int  `json:"topics"`
	Handlers          int  `json:"handlers"`
	Closing           bool `json:"closing"`
}

type HelloPayload struct {
	ProtocolVersion string    `json:"protocolVersion"`
	ConnectionID    string    `json:"connectionId"`
	AccountID       string    `json:"accountId"`
	DisplayName     string    `json:"displayName,omitempty"`
	ServerTime      time.Time `json:"serverTime"`
}

type TopicPayload struct {
	Topic string `json:"topic"`
}

type AckPayload struct {
	RequestID string `json:"requestId,omitempty"`
	Action    string `json:"action"`
	Topic     string `json:"topic,omitempty"`
}

type ErrorPayload struct {
	RequestID string `json:"requestId,omitempty"`
	Code      string `json:"code"`
	Message   string `json:"message"`
}

func validateEnvelope(envelope Envelope) error {
	if envelope.Type == "" || envelope.Type != strings.TrimSpace(envelope.Type) || len(envelope.Type) > 128 {
		return ErrInvalidEnvelope
	}
	if len(envelope.ID) > 128 || len(envelope.Topic) > 128 {
		return ErrInvalidEnvelope
	}
	return nil
}
