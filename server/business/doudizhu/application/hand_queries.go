package application

import (
	"context"
	"fmt"
	"strings"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
)

func (s *Service) GetHandPublicView(ctx context.Context, actor domain.AccountID, handID domain.HandID) (LiveHandView, error) {
	snapshot, err := s.authorizeHandViewer(ctx, actor, handID)
	if err != nil {
		return LiveHandView{}, err
	}
	return s.liveHands.PublicView(ctx, snapshot.ID, actor)
}

func (s *Service) GetHandPrivateView(ctx context.Context, actor domain.AccountID, handID domain.HandID) (LiveHandView, error) {
	snapshot, err := s.authorizeHandViewer(ctx, actor, handID)
	if err != nil {
		return LiveHandView{}, err
	}
	return s.liveHands.PrivateView(ctx, snapshot.ID, actor)
}

func (s *Service) authorizeHandViewer(ctx context.Context, actor domain.AccountID, handID domain.HandID) (domain.HandSnapshot, error) {
	if s == nil || s.store == nil || s.liveHands == nil {
		return domain.HandSnapshot{}, fmt.Errorf("%w: live hand query dependencies", ErrInvalidCommand)
	}
	if strings.TrimSpace(string(actor)) == "" || len(actor) > 128 || strings.TrimSpace(string(handID)) == "" || len(handID) > 128 {
		return domain.HandSnapshot{}, fmt.Errorf("%w: actor or hand ID", ErrInvalidCommand)
	}
	snapshot, err := s.store.LoadHand(ctx, handID)
	if err != nil {
		return domain.HandSnapshot{}, err
	}
	for _, seat := range snapshot.Seats {
		if seat.AccountID == actor {
			return snapshot, nil
		}
	}
	return domain.HandSnapshot{}, domain.ErrNotSeated
}
