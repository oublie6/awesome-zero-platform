package application

import (
	"context"
	"fmt"
	"strings"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
)

// RoomView is an authenticated, server-authoritative lobby projection.
// Non-members can see only a limited invite-code preview, never player identities.
type RoomView struct {
	RoomID       string           `json:"roomId"`
	Version      uint64           `json:"version"`
	Status       domain.RoomStatus `json:"status"`
	Occupied     int              `json:"occupied"`
	Member       bool             `json:"member"`
	Seats        []RoomSeatView   `json:"seats,omitempty"`
	ActiveHandID string           `json:"activeHandId,omitempty"`
}
type RoomSeatView struct {
	Position uint8  `json:"position"`
	AccountID string `json:"accountId,omitempty"`
	Ready bool `json:"ready"`
}

// GetRoomView is intentionally a read path: it cannot resolve ambiguous writes
// by guessing. The caller must be an authenticated actor; possession of a room
// ID does not authorize viewing other players' identities or active hand IDs.
func (s *Service) GetRoomView(ctx context.Context, actor domain.AccountID, roomID domain.RoomID) (RoomView, error) {
	if s == nil || s.store == nil {
		return RoomView{}, fmt.Errorf("%w: room query unavailable", ErrInvalidCommand)
	}
	if strings.TrimSpace(string(actor)) == "" || len(actor) > 128 ||
		strings.TrimSpace(string(roomID)) == "" || string(roomID) != strings.TrimSpace(string(roomID)) || len(roomID) > 128 {
		return RoomView{}, fmt.Errorf("%w: invalid room query", ErrInvalidCommand)
	}
	reader, ok := s.store.(interface {
		LoadRoom(context.Context, domain.RoomID) (domain.RoomSnapshot, error)
	})
	if !ok {
		return RoomView{}, fmt.Errorf("%w: room query store unavailable", ErrInvalidCommand)
	}
	snapshot, err := reader.LoadRoom(ctx, roomID)
	if err != nil { return RoomView{}, err }
	if snapshot.ID != roomID {
		return RoomView{}, fmt.Errorf("%w: room identity mismatch", ErrInvalidCommand)
	}
	if _, err := domain.RestoreRoom(snapshot); err != nil {
		return RoomView{}, fmt.Errorf("restore queried room: %w", err)
	}
	view := RoomView{RoomID: string(roomID), Version: snapshot.Version, Status: snapshot.Status}
	for _, seat := range snapshot.Seats {
		if seat.AccountID != "" {
			view.Occupied++
			if seat.AccountID == actor { view.Member = true }
		}
	}
	if view.Member {
		for _, seat := range snapshot.Seats {
			view.Seats = append(view.Seats, RoomSeatView{Position: uint8(seat.Seat), AccountID: string(seat.AccountID), Ready: seat.Ready})
		}
		view.ActiveHandID = string(snapshot.ActiveHandID)
	}
	return view, nil
}
