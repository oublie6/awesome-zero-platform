package realtime

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

func (h *Hub) Handler() http.Handler {
	upgrader := websocket.Upgrader{
		HandshakeTimeout: h.config.HandshakeTimeout,
		Subprotocols:     []string{"bearer"},
		CheckOrigin:      h.checkOrigin,
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			writeHandshakeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "WebSocket handshake requires GET")
			return
		}
		if r.URL.Query().Has("access_token") || r.URL.Query().Has("token") {
			h.metrics.ConnectionRejected("query_token")
			writeHandshakeError(w, http.StatusBadRequest, "QUERY_TOKEN_REJECTED", "access tokens are not accepted in URLs")
			return
		}

		token, err := handshakeToken(r)
		if err != nil {
			h.metrics.ConnectionRejected("authentication")
			writeHandshakeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "WebSocket authentication is required")
			return
		}
		identity, err := h.authenticator.Authenticate(r.Context(), token)
		if err != nil || strings.TrimSpace(identity.AccountID) == "" || strings.TrimSpace(identity.SessionID) == "" {
			h.metrics.ConnectionRejected("authentication")
			writeHandshakeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "WebSocket authentication failed")
			return
		}

		connection, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			h.metrics.ConnectionRejected("upgrade")
			return
		}

		connectionID, err := uuid.NewV7()
		if err != nil {
			_ = connection.WriteControl(
				websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseInternalServerErr, "connection id failed"),
				h.now().Add(h.config.WriteTimeout),
			)
			_ = connection.Close()
			return
		}
		client := newClient(h, connection, connectionID.String(), identity)
		if err := h.register(client); err != nil {
			h.metrics.ConnectionRejected(rejectionReason(err))
			_ = connection.WriteControl(
				websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseTryAgainLater, "connection unavailable"),
				h.now().Add(h.config.WriteTimeout),
			)
			_ = connection.Close()
			return
		}
		client.start()
	})
}

func handshakeToken(r *http.Request) (string, error) {
	authorization := strings.TrimSpace(r.Header.Get("Authorization"))
	if authorization != "" {
		scheme, token, found := strings.Cut(authorization, " ")
		if !found || !strings.EqualFold(strings.TrimSpace(scheme), "Bearer") || strings.TrimSpace(token) == "" {
			return "", ErrAuthenticationFailed
		}
		return strings.TrimSpace(token), nil
	}

	protocols := websocket.Subprotocols(r)
	if len(protocols) != 2 || !strings.EqualFold(strings.TrimSpace(protocols[0]), "bearer") {
		return "", ErrAuthenticationFailed
	}
	token := strings.TrimSpace(protocols[1])
	if token == "" {
		return "", ErrAuthenticationFailed
	}
	return token, nil
}

func (h *Hub) checkOrigin(r *http.Request) bool {
	rawOrigin := strings.TrimSpace(r.Header.Get("Origin"))
	if rawOrigin == "" {
		return true
	}
	origin, err := url.Parse(rawOrigin)
	if err != nil || origin.Scheme == "" || origin.Host == "" {
		return false
	}
	if len(h.config.AllowedOrigins) == 0 {
		return strings.EqualFold(origin.Host, r.Host)
	}
	normalized := strings.TrimSuffix(origin.Scheme+"://"+origin.Host, "/")
	for _, allowed := range h.config.AllowedOrigins {
		if strings.EqualFold(normalized, strings.TrimSuffix(strings.TrimSpace(allowed), "/")) {
			return true
		}
	}
	return false
}

func writeHandshakeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"code":    code,
		"message": message,
	})
}

func IsAuthenticationError(err error) bool {
	return errors.Is(err, ErrAuthenticationFailed)
}
