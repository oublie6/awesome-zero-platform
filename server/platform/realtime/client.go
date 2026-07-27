package realtime

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const closeServiceRestart = websocket.CloseServiceRestart

type client struct {
	hub      *Hub
	conn     *websocket.Conn
	id       string
	identity Identity
	ctx      context.Context
	cancel   context.CancelFunc

	send      chan []byte
	done      chan struct{}
	finished  chan struct{}
	closeOne  sync.Once
	closeCode int
	closeText string

	topics map[string]struct{}
}

func newClient(hub *Hub, connection *websocket.Conn, id string, identity Identity) *client {
	ctx, cancel := context.WithCancel(context.Background())
	return &client{
		hub:      hub,
		conn:     connection,
		id:       id,
		identity: identity,
		ctx:      ctx,
		cancel:   cancel,
		send:     make(chan []byte, hub.config.SendQueueSize),
		done:     make(chan struct{}),
		finished: make(chan struct{}),
		topics:   make(map[string]struct{}),
	}
}

func (c *client) start() {
	go c.writePump()
	go c.readPump()
	c.sendEnvelope(Envelope{
		Type: TypeSystemHello,
		Payload: mustJSON(HelloPayload{
			ProtocolVersion: ProtocolVersion,
			ConnectionID:    c.id,
			AccountID:       c.identity.AccountID,
			DisplayName:     c.identity.DisplayName,
			ServerTime:      c.hub.now().UTC(),
		}),
	})
}

func (c *client) context() ConnectionContext {
	return ConnectionContext{
		ConnectionID: c.id,
		AccountID:    c.identity.AccountID,
		SessionID:    c.identity.SessionID,
		DisplayName:  c.identity.DisplayName,
	}
}

func (c *client) enqueue(payload []byte) bool {
	select {
	case <-c.done:
		return false
	case c.send <- payload:
		c.hub.metrics.MessageSent()
		return true
	default:
		c.hub.metrics.SlowConsumer()
		c.terminate(websocket.CloseTryAgainLater, "slow consumer")
		return false
	}
}

func (c *client) sendEnvelope(envelope Envelope) bool {
	payload, err := c.hub.encode(envelope)
	if err != nil {
		return false
	}
	return c.enqueue(payload)
}

func (c *client) sendError(requestID, code, message string) {
	c.sendEnvelope(Envelope{
		ID:   requestID,
		Type: TypeSystemError,
		Payload: mustJSON(ErrorPayload{
			RequestID: requestID,
			Code:      code,
			Message:   message,
		}),
	})
}

func (c *client) terminate(code int, text string) {
	c.closeOne.Do(func() {
		c.closeCode = code
		c.closeText = text
		c.cancel()
		close(c.done)
		c.hub.unregister(c)
	})
}

func (c *client) readPump() {
	defer c.terminate(websocket.CloseNormalClosure, "connection closed")

	c.conn.SetReadLimit(c.hub.config.MaxMessageBytes)
	_ = c.conn.SetReadDeadline(time.Now().Add(c.hub.config.PongTimeout))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(c.hub.config.PongTimeout))
	})

	for {
		messageType, payload, err := c.conn.ReadMessage()
		if err != nil {
			return
		}
		if messageType != websocket.TextMessage {
			c.terminate(websocket.CloseUnsupportedData, "text messages required")
			return
		}

		var envelope Envelope
		if err := json.Unmarshal(payload, &envelope); err != nil {
			c.sendError("", "INVALID_JSON", "message must be valid JSON")
			continue
		}
		c.hub.metrics.MessageReceived()
		c.hub.handle(c.ctx, c, envelope)
	}
}

func (c *client) writePump() {
	ticker := time.NewTicker(c.hub.config.PingInterval)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close()
		close(c.finished)
	}()

	for {
		select {
		case <-c.done:
			_ = c.conn.SetWriteDeadline(time.Now().Add(c.hub.config.WriteTimeout))
			_ = c.conn.WriteControl(
				websocket.CloseMessage,
				websocket.FormatCloseMessage(c.closeCode, c.closeText),
				time.Now().Add(c.hub.config.WriteTimeout),
			)
			return
		case payload := <-c.send:
			if err := c.conn.SetWriteDeadline(time.Now().Add(c.hub.config.WriteTimeout)); err != nil {
				c.terminate(websocket.CloseInternalServerErr, "write deadline failed")
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, payload); err != nil {
				c.terminate(websocket.CloseAbnormalClosure, "write failed")
				return
			}
		case <-ticker.C:
			deadline := time.Now().Add(c.hub.config.WriteTimeout)
			if err := c.conn.WriteControl(websocket.PingMessage, nil, deadline); err != nil {
				c.terminate(websocket.CloseAbnormalClosure, "heartbeat failed")
				return
			}
		}
	}
}
