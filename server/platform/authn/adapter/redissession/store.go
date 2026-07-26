package redissession

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/oublie6/awesome-zero-platform/server/platform/authn"
	"github.com/redis/go-redis/v9"
)

type Store struct {
	client *redis.Client
	prefix string
	now    func() time.Time
}

func New(client *redis.Client, prefix string) (*Store, error) {
	if client == nil {
		return nil, fmt.Errorf("redis client is required")
	}
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		prefix = "authn:session:"
	}
	return &Store{client: client, prefix: prefix, now: time.Now}, nil
}

func (s *Store) Create(ctx context.Context, session authn.Session) error {
	payload, ttl, err := s.encode(session)
	if err != nil {
		return err
	}
	created, err := s.client.SetNX(ctx, s.key(session.ID), payload, ttl).Result()
	if err != nil {
		return err
	}
	if !created {
		return authn.ErrSessionConflict
	}
	return nil
}

func (s *Store) Get(ctx context.Context, sessionID string) (authn.Session, error) {
	raw, err := s.client.Get(ctx, s.key(sessionID)).Bytes()
	if errors.Is(err, redis.Nil) {
		return authn.Session{}, authn.ErrSessionNotFound
	}
	if err != nil {
		return authn.Session{}, err
	}

	var session authn.Session
	if err := json.Unmarshal(raw, &session); err != nil {
		return authn.Session{}, fmt.Errorf("decode session: %w", err)
	}
	if session.ID != sessionID || !session.ExpiresAt.After(s.now().UTC()) {
		_ = s.client.Del(ctx, s.key(sessionID)).Err()
		return authn.Session{}, authn.ErrSessionNotFound
	}
	return session, nil
}

func (s *Store) Rotate(ctx context.Context, sessionID, currentDigest, nextDigest string, nextExpiresAt time.Time) (authn.Session, error) {
	key := s.key(sessionID)
	var updated authn.Session

	err := s.client.Watch(ctx, func(tx *redis.Tx) error {
		raw, err := tx.Get(ctx, key).Bytes()
		if errors.Is(err, redis.Nil) {
			return authn.ErrSessionNotFound
		}
		if err != nil {
			return err
		}

		var session authn.Session
		if err := json.Unmarshal(raw, &session); err != nil {
			return fmt.Errorf("decode session: %w", err)
		}
		if session.ID != sessionID || session.RefreshDigest != currentDigest || !session.ExpiresAt.After(s.now().UTC()) {
			return authn.ErrInvalidRefresh
		}

		session.RefreshDigest = nextDigest
		session.ExpiresAt = nextExpiresAt.UTC()
		payload, ttl, err := s.encode(session)
		if err != nil {
			return err
		}

		_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
			pipe.Set(ctx, key, payload, ttl)
			return nil
		})
		if err != nil {
			return err
		}
		updated = session
		return nil
	}, key)
	if errors.Is(err, redis.TxFailedErr) {
		return authn.Session{}, authn.ErrSessionConflict
	}
	if err != nil {
		return authn.Session{}, err
	}
	return updated, nil
}

func (s *Store) Revoke(ctx context.Context, sessionID string) error {
	deleted, err := s.client.Del(ctx, s.key(sessionID)).Result()
	if err != nil {
		return err
	}
	if deleted == 0 {
		return authn.ErrSessionNotFound
	}
	return nil
}

func (s *Store) encode(session authn.Session) ([]byte, time.Duration, error) {
	if strings.TrimSpace(session.ID) == "" || strings.TrimSpace(session.AccountID) == "" || strings.TrimSpace(session.RefreshDigest) == "" {
		return nil, 0, fmt.Errorf("session id, account id, and refresh digest are required")
	}
	ttl := session.ExpiresAt.Sub(s.now().UTC())
	if ttl <= 0 {
		return nil, 0, authn.ErrSessionNotFound
	}
	payload, err := json.Marshal(session)
	if err != nil {
		return nil, 0, fmt.Errorf("encode session: %w", err)
	}
	return payload, ttl, nil
}

func (s *Store) key(sessionID string) string {
	return s.prefix + strings.TrimSpace(sessionID)
}
