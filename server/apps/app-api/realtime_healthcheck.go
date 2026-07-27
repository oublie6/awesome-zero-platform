package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/oublie6/awesome-zero-platform/server/platform/realtime"
)

func runRealtimeHealthcheck(endpoint, token string, browserProtocol, insecureTLS bool) error {
	endpoint = strings.TrimSpace(endpoint)
	token = strings.TrimSpace(token)
	if endpoint == "" {
		return fmt.Errorf("realtime URL is required")
	}
	if token == "" {
		return fmt.Errorf("APP_REALTIME_HEALTHCHECK_TOKEN is required")
	}

	dialer := websocket.Dialer{
		HandshakeTimeout: 5 * time.Second,
		TLSClientConfig: &tls.Config{
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: insecureTLS, // #nosec G402 -- explicit probe-only flag for ephemeral self-signed CI certificates.
		},
	}
	headers := http.Header{}
	if browserProtocol {
		dialer.Subprotocols = []string{"bearer", token}
	} else {
		headers.Set("Authorization", "Bearer "+token)
	}

	connection, response, err := dialer.Dial(endpoint, headers)
	if err != nil {
		if response != nil {
			return fmt.Errorf("dial realtime endpoint: HTTP %d: %w", response.StatusCode, err)
		}
		return fmt.Errorf("dial realtime endpoint: %w", err)
	}
	defer connection.Close()
	if browserProtocol && connection.Subprotocol() != "bearer" {
		return fmt.Errorf("unexpected websocket subprotocol %q", connection.Subprotocol())
	}

	if err := connection.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return fmt.Errorf("set realtime read deadline: %w", err)
	}
	var hello realtime.Envelope
	if err := connection.ReadJSON(&hello); err != nil {
		return fmt.Errorf("read realtime hello: %w", err)
	}
	if hello.Type != realtime.TypeSystemHello {
		return fmt.Errorf("expected %s, got %s", realtime.TypeSystemHello, hello.Type)
	}
	var helloPayload realtime.HelloPayload
	if err := json.Unmarshal(hello.Payload, &helloPayload); err != nil {
		return fmt.Errorf("decode realtime hello: %w", err)
	}
	if helloPayload.ProtocolVersion != realtime.ProtocolVersion || helloPayload.ConnectionID == "" || helloPayload.AccountID == "" {
		return fmt.Errorf("realtime hello is incomplete")
	}

	requestID := "realtime-healthcheck"
	if err := connection.SetWriteDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return fmt.Errorf("set realtime write deadline: %w", err)
	}
	if err := connection.WriteJSON(realtime.Envelope{ID: requestID, Type: realtime.TypeSystemPing}); err != nil {
		return fmt.Errorf("send realtime ping: %w", err)
	}
	if err := connection.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return fmt.Errorf("reset realtime read deadline: %w", err)
	}
	for {
		var responseEnvelope realtime.Envelope
		if err := connection.ReadJSON(&responseEnvelope); err != nil {
			return fmt.Errorf("read realtime pong: %w", err)
		}
		if responseEnvelope.Type == realtime.TypeSystemPong && responseEnvelope.ID == requestID {
			return nil
		}
	}
}
