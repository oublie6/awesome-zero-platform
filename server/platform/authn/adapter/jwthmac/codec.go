package jwthmac

import (
	"fmt"
	"strings"

	"github.com/golang-jwt/jwt/v4"
	"github.com/oublie6/awesome-zero-platform/server/platform/authn"
)

type Codec struct {
	secret []byte
	issuer string
	parser *jwt.Parser
}

type claims struct {
	SessionID string `json:"sid"`
	jwt.RegisteredClaims
}

func New(secret, issuer string) (*Codec, error) {
	secret = strings.TrimSpace(secret)
	issuer = strings.TrimSpace(issuer)
	if len(secret) < 32 {
		return nil, fmt.Errorf("access token secret must contain at least 32 characters")
	}
	if issuer == "" {
		return nil, fmt.Errorf("access token issuer must not be empty")
	}

	return &Codec{
		secret: []byte(secret),
		issuer: issuer,
		parser: &jwt.Parser{ValidMethods: []string{jwt.SigningMethodHS256.Alg()}},
	}, nil
}

func (c *Codec) Issue(input authn.AccessClaims) (string, error) {
	if strings.TrimSpace(input.Subject) == "" || strings.TrimSpace(input.SessionID) == "" {
		return "", fmt.Errorf("access token subject and session id are required")
	}
	if !input.ExpiresAt.After(input.IssuedAt) {
		return "", fmt.Errorf("access token expiry must be after issue time")
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims{
		SessionID: input.SessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    c.issuer,
			Subject:   input.Subject,
			IssuedAt:  jwt.NewNumericDate(input.IssuedAt.UTC()),
			ExpiresAt: jwt.NewNumericDate(input.ExpiresAt.UTC()),
		},
	})
	return token.SignedString(c.secret)
}

func (c *Codec) Parse(raw string) (authn.AccessClaims, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return authn.AccessClaims{}, authn.ErrInvalidToken
	}

	parsed := &claims{}
	token, err := c.parser.ParseWithClaims(raw, parsed, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, authn.ErrInvalidToken
		}
		return c.secret, nil
	})
	if err != nil || token == nil || !token.Valid {
		return authn.AccessClaims{}, authn.ErrInvalidToken
	}
	if parsed.Issuer != c.issuer || parsed.Subject == "" || parsed.SessionID == "" || parsed.IssuedAt == nil || parsed.ExpiresAt == nil {
		return authn.AccessClaims{}, authn.ErrInvalidToken
	}

	return authn.AccessClaims{
		Subject:   parsed.Subject,
		SessionID: parsed.SessionID,
		IssuedAt:  parsed.IssuedAt.Time.UTC(),
		ExpiresAt: parsed.ExpiresAt.Time.UTC(),
	}, nil
}
