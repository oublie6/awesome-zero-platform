package authn

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidToken       = errors.New("invalid access token")
	ErrInvalidRefresh     = errors.New("invalid refresh token")
	ErrSessionNotFound    = errors.New("session not found")
	ErrSessionConflict    = errors.New("session conflict")
	ErrAccountUnavailable = errors.New("account unavailable")
)

type Principal struct {
	AccountID   string
	DisplayName string
}

type AccessClaims struct {
	Subject   string
	SessionID string
	IssuedAt  time.Time
	ExpiresAt time.Time
}

type Session struct {
	ID            string    `json:"id"`
	AccountID     string    `json:"accountId"`
	RefreshDigest string    `json:"refreshDigest"`
	CreatedAt     time.Time `json:"createdAt"`
	ExpiresAt     time.Time `json:"expiresAt"`
}

type TokenPair struct {
	AccessToken      string
	RefreshToken     string
	AccessExpiresAt  time.Time
	RefreshExpiresAt time.Time
}

type Authentication struct {
	Principal Principal
	SessionID string
	ExpiresAt time.Time
}

type IdentityProvider interface {
	Authenticate(context.Context, string, string) (Principal, error)
	ResolveActive(context.Context, string) (Principal, error)
}

type AccessTokenCodec interface {
	Issue(AccessClaims) (string, error)
	Parse(string) (AccessClaims, error)
}

type SessionStore interface {
	Create(context.Context, Session) error
	Get(context.Context, string) (Session, error)
	Rotate(context.Context, string, string, string, time.Time) (Session, error)
	Revoke(context.Context, string) error
}

type Config struct {
	AccessTTL  time.Duration
	RefreshTTL time.Duration
}

func (c *Config) Prepare() {
	if c.AccessTTL == 0 {
		c.AccessTTL = 15 * time.Minute
	}
	if c.RefreshTTL == 0 {
		c.RefreshTTL = 30 * 24 * time.Hour
	}
}

func (c Config) Validate() error {
	if c.AccessTTL <= 0 {
		return fmt.Errorf("access token ttl must be greater than zero")
	}
	if c.RefreshTTL <= c.AccessTTL {
		return fmt.Errorf("refresh token ttl must be greater than access token ttl")
	}
	return nil
}

type Service struct {
	identity IdentityProvider
	codec    AccessTokenCodec
	sessions SessionStore
	config   Config
	now      func() time.Time
}

func NewService(identity IdentityProvider, codec AccessTokenCodec, sessions SessionStore, cfg Config) (*Service, error) {
	cfg.Prepare()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if identity == nil {
		return nil, fmt.Errorf("identity provider is required")
	}
	if codec == nil {
		return nil, fmt.Errorf("access token codec is required")
	}
	if sessions == nil {
		return nil, fmt.Errorf("session store is required")
	}

	return &Service{
		identity: identity,
		codec:    codec,
		sessions: sessions,
		config:   cfg,
		now:      time.Now,
	}, nil
}

func (s *Service) Login(ctx context.Context, identifier, password string) (Authentication, TokenPair, error) {
	principal, err := s.identity.Authenticate(ctx, strings.TrimSpace(identifier), password)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) || errors.Is(err, ErrAccountUnavailable) {
			return Authentication{}, TokenPair{}, ErrInvalidCredentials
		}
		return Authentication{}, TokenPair{}, fmt.Errorf("authenticate identity: %w", err)
	}
	if err := validatePrincipal(principal); err != nil {
		return Authentication{}, TokenPair{}, err
	}

	sessionID, err := newSessionID()
	if err != nil {
		return Authentication{}, TokenPair{}, fmt.Errorf("generate session id: %w", err)
	}

	now := s.now().UTC()
	refreshToken, refreshDigest, err := newRefreshToken(sessionID)
	if err != nil {
		return Authentication{}, TokenPair{}, fmt.Errorf("generate refresh token: %w", err)
	}

	session := Session{
		ID:            sessionID,
		AccountID:     principal.AccountID,
		RefreshDigest: refreshDigest,
		CreatedAt:     now,
		ExpiresAt:     now.Add(s.config.RefreshTTL),
	}
	if err := s.sessions.Create(ctx, session); err != nil {
		return Authentication{}, TokenPair{}, fmt.Errorf("create session: %w", err)
	}

	accessExpiresAt := now.Add(s.config.AccessTTL)
	accessToken, err := s.codec.Issue(AccessClaims{
		Subject:   principal.AccountID,
		SessionID: sessionID,
		IssuedAt:  now,
		ExpiresAt: accessExpiresAt,
	})
	if err != nil {
		_ = s.sessions.Revoke(ctx, sessionID)
		return Authentication{}, TokenPair{}, fmt.Errorf("issue access token: %w", err)
	}

	authentication := Authentication{
		Principal: principal,
		SessionID: sessionID,
		ExpiresAt: accessExpiresAt,
	}
	tokens := TokenPair{
		AccessToken:      accessToken,
		RefreshToken:     refreshToken,
		AccessExpiresAt:  accessExpiresAt,
		RefreshExpiresAt: session.ExpiresAt,
	}
	return authentication, tokens, nil
}

func (s *Service) Refresh(ctx context.Context, refreshToken string) (Authentication, TokenPair, error) {
	sessionID, currentDigest, err := parseRefreshToken(refreshToken)
	if err != nil {
		return Authentication{}, TokenPair{}, ErrInvalidRefresh
	}

	session, err := s.sessions.Get(ctx, sessionID)
	if err != nil {
		if errors.Is(err, ErrSessionNotFound) {
			return Authentication{}, TokenPair{}, ErrInvalidRefresh
		}
		return Authentication{}, TokenPair{}, fmt.Errorf("load session: %w", err)
	}
	if subtleStringMismatch(session.RefreshDigest, currentDigest) {
		return Authentication{}, TokenPair{}, ErrInvalidRefresh
	}

	principal, err := s.identity.ResolveActive(ctx, session.AccountID)
	if err != nil {
		if errors.Is(err, ErrAccountUnavailable) {
			_ = s.sessions.Revoke(ctx, sessionID)
			return Authentication{}, TokenPair{}, ErrAccountUnavailable
		}
		return Authentication{}, TokenPair{}, fmt.Errorf("resolve active account: %w", err)
	}
	if err := validatePrincipal(principal); err != nil {
		return Authentication{}, TokenPair{}, err
	}

	now := s.now().UTC()
	nextRefreshToken, nextDigest, err := newRefreshToken(sessionID)
	if err != nil {
		return Authentication{}, TokenPair{}, fmt.Errorf("generate refresh token: %w", err)
	}
	accessExpiresAt := now.Add(s.config.AccessTTL)
	accessToken, err := s.codec.Issue(AccessClaims{
		Subject:   principal.AccountID,
		SessionID: sessionID,
		IssuedAt:  now,
		ExpiresAt: accessExpiresAt,
	})
	if err != nil {
		return Authentication{}, TokenPair{}, fmt.Errorf("issue access token: %w", err)
	}

	nextSession, err := s.sessions.Rotate(ctx, sessionID, currentDigest, nextDigest, now.Add(s.config.RefreshTTL))
	if err != nil {
		if errors.Is(err, ErrInvalidRefresh) || errors.Is(err, ErrSessionNotFound) || errors.Is(err, ErrSessionConflict) {
			return Authentication{}, TokenPair{}, ErrInvalidRefresh
		}
		return Authentication{}, TokenPair{}, fmt.Errorf("rotate session: %w", err)
	}

	authentication := Authentication{
		Principal: principal,
		SessionID: sessionID,
		ExpiresAt: accessExpiresAt,
	}
	tokens := TokenPair{
		AccessToken:      accessToken,
		RefreshToken:     nextRefreshToken,
		AccessExpiresAt:  accessExpiresAt,
		RefreshExpiresAt: nextSession.ExpiresAt,
	}
	return authentication, tokens, nil
}

func (s *Service) AuthenticateAccess(ctx context.Context, rawToken string) (Authentication, error) {
	claims, err := s.codec.Parse(strings.TrimSpace(rawToken))
	if err != nil {
		return Authentication{}, ErrInvalidToken
	}
	now := s.now().UTC()
	if strings.TrimSpace(claims.Subject) == "" || strings.TrimSpace(claims.SessionID) == "" || !claims.ExpiresAt.After(now) {
		return Authentication{}, ErrInvalidToken
	}

	session, err := s.sessions.Get(ctx, claims.SessionID)
	if err != nil {
		if errors.Is(err, ErrSessionNotFound) {
			return Authentication{}, ErrInvalidToken
		}
		return Authentication{}, fmt.Errorf("load session: %w", err)
	}
	if session.AccountID != claims.Subject || !session.ExpiresAt.After(now) {
		return Authentication{}, ErrInvalidToken
	}

	principal, err := s.identity.ResolveActive(ctx, claims.Subject)
	if err != nil {
		if errors.Is(err, ErrAccountUnavailable) {
			_ = s.sessions.Revoke(ctx, claims.SessionID)
			return Authentication{}, ErrAccountUnavailable
		}
		return Authentication{}, fmt.Errorf("resolve active account: %w", err)
	}
	if err := validatePrincipal(principal); err != nil {
		return Authentication{}, err
	}

	return Authentication{
		Principal: principal,
		SessionID: claims.SessionID,
		ExpiresAt: claims.ExpiresAt,
	}, nil
}

func (s *Service) Logout(ctx context.Context, rawToken string) error {
	authentication, err := s.AuthenticateAccess(ctx, rawToken)
	if err != nil {
		return err
	}
	if err := s.sessions.Revoke(ctx, authentication.SessionID); err != nil && !errors.Is(err, ErrSessionNotFound) {
		return fmt.Errorf("revoke session: %w", err)
	}
	return nil
}

func validatePrincipal(principal Principal) error {
	if strings.TrimSpace(principal.AccountID) == "" {
		return fmt.Errorf("identity provider returned an empty account id")
	}
	return nil
}

func newSessionID() (string, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", err
	}
	return id.String(), nil
}

func newRefreshToken(sessionID string) (string, string, error) {
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return "", "", err
	}
	plain := sessionID + "." + base64.RawURLEncoding.EncodeToString(secret)
	return plain, digest(plain), nil
}

func parseRefreshToken(token string) (string, string, error) {
	token = strings.TrimSpace(token)
	sessionID, secret, ok := strings.Cut(token, ".")
	if !ok || sessionID == "" || secret == "" {
		return "", "", ErrInvalidRefresh
	}
	if _, err := uuid.Parse(sessionID); err != nil {
		return "", "", ErrInvalidRefresh
	}
	decoded, err := base64.RawURLEncoding.DecodeString(secret)
	if err != nil || len(decoded) != 32 {
		return "", "", ErrInvalidRefresh
	}
	return sessionID, digest(token), nil
}

func digest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func subtleStringMismatch(left, right string) bool {
	return subtle.ConstantTimeCompare([]byte(left), []byte(right)) != 1
}
