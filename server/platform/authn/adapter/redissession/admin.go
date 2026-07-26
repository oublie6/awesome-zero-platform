package redissession

import (
	"context"
	"encoding/json"
	"sort"
	"strings"

	"github.com/oublie6/awesome-zero-platform/server/platform/authn"
)

func (s *Store) ListByAccount(ctx context.Context, accountID string) ([]authn.SessionView, error) {
	accountID = strings.TrimSpace(accountID)
	if accountID == "" {
		return nil, nil
	}
	var cursor uint64
	views := make([]authn.SessionView, 0)
	for {
		keys, next, err := s.client.Scan(ctx, cursor, s.prefix+"*", 100).Result()
		if err != nil {
			return nil, err
		}
		for _, key := range keys {
			raw, err := s.client.Get(ctx, key).Bytes()
			if err != nil {
				continue
			}
			var session authn.Session
			if err := json.Unmarshal(raw, &session); err != nil || session.AccountID != accountID {
				continue
			}
			views = append(views, authn.SessionView{
				ID:        session.ID,
				AccountID: session.AccountID,
				CreatedAt: session.CreatedAt,
				ExpiresAt: session.ExpiresAt,
			})
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}
	sort.Slice(views, func(i, j int) bool { return views[i].CreatedAt.After(views[j].CreatedAt) })
	return views, nil
}

func (s *Store) RevokeByAccount(ctx context.Context, accountID string) (int64, error) {
	views, err := s.ListByAccount(ctx, accountID)
	if err != nil {
		return 0, err
	}
	if len(views) == 0 {
		return 0, nil
	}
	keys := make([]string, 0, len(views))
	for _, view := range views {
		keys = append(keys, s.key(view.ID))
	}
	return s.client.Del(ctx, keys...).Result()
}
