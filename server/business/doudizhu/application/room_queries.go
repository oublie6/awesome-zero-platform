package application

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
)

// RoomView is an authenticated, server-authoritative lobby projection.
// Non-members can see only a limited invite-code preview, never player identities.
type RoomView struct {
	RoomID             string            `json:"roomId"`
	Version            uint64            `json:"version"`
	Status             domain.RoomStatus `json:"status"`
	Occupied           int               `json:"occupied"`
	Member             bool              `json:"member"`
	LastClientSequence uint64            `json:"lastClientSequence"`
	Seats              []RoomSeatView    `json:"seats,omitempty"`
	ActiveHandID       string            `json:"activeHandId,omitempty"`
	LastHandID         string            `json:"lastHandId,omitempty"`
}
type RoomSeatView struct {
	Position  uint8  `json:"position"`
	AccountID string `json:"accountId,omitempty"`
	Ready     bool   `json:"ready"`
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
	if err != nil {
		return RoomView{}, err
	}
	if snapshot.ID != roomID {
		return RoomView{}, fmt.Errorf("%w: room identity mismatch", ErrInvalidCommand)
	}
	if _, err := domain.RestoreRoom(snapshot); err != nil {
		return RoomView{}, fmt.Errorf("restore queried room: %w", err)
	}
	view := RoomView{RoomID: string(roomID), Version: snapshot.Version, Status: snapshot.Status}
	seqReader, ok := s.store.(interface {
		RoomClientSequence(context.Context, domain.RoomID, domain.AccountID) (uint64, error)
	})
	if !ok {
		return RoomView{}, fmt.Errorf("%w: room sequence query unavailable", ErrInvalidCommand)
	}
	view.LastClientSequence, err = seqReader.RoomClientSequence(ctx, roomID, actor)
	if err != nil {
		return RoomView{}, err
	}
	for _, seat := range snapshot.Seats {
		if seat.AccountID != "" {
			view.Occupied++
			if seat.AccountID == actor {
				view.Member = true
			}
		}
	}
	if view.Member {
		for _, seat := range snapshot.Seats {
			view.Seats = append(view.Seats, RoomSeatView{Position: uint8(seat.Seat), AccountID: string(seat.AccountID), Ready: seat.Ready})
		}
		view.ActiveHandID = string(snapshot.ActiveHandID)
		// Room membership does not confer authorization to a previous hand's
		// archive. A replacement player may join after the old hand finishes:
		// giving them lastHandId would make reconnect request inaccessible
		// evidence and fail with 403. Preserve strict archive permissions.
		if snapshot.LastHandID != "" {
			lastHand, err := s.store.LoadHand(ctx, snapshot.LastHandID)
			switch {
			case errors.Is(err, ErrNotFound):
				// A pruned/not-retained historical hand cannot be restored.
			case err != nil:
				return RoomView{}, fmt.Errorf("load last room hand: %w", err)
			default:
				if lastHand.ID != snapshot.LastHandID || lastHand.RoomID != roomID {
					return RoomView{}, fmt.Errorf("%w: last room hand identity mismatch", ErrInvalidCommand)
				}
				if _, err := domain.RestoreHand(lastHand); err != nil {
					return RoomView{}, fmt.Errorf("restore last room hand: %w", err)
				}
				if lastHand.Phase.Terminal() {
					for _, historicalSeat := range lastHand.Seats {
						if historicalSeat.AccountID == actor {
							view.LastHandID = string(snapshot.LastHandID)
							break
						}
					}
				}
			}
		}
	}
	return view, nil
}

// GetRecoverableRooms exposes only current, nonclosed room memberships for
// this authenticated account. A discovered ID is merely a locator; the
// ordinary room read rechecks membership and filters historical hand access.
func (s *Service) GetRecoverableRooms(ctx context.Context, actor domain.AccountID) ([]RoomView, error) {
	if s == nil || s.store == nil || strings.TrimSpace(string(actor)) == "" ||
		string(actor) != strings.TrimSpace(string(actor)) || len(actor) > 128 {
		return nil, fmt.Errorf("%w: invalid authenticated room discovery", ErrInvalidCommand)
	}
	reader, ok := s.store.(interface {
		FindRecoverableRoomIDs(context.Context, domain.AccountID) ([]domain.RoomID, error)
	})
	if !ok {
		return nil, fmt.Errorf("%w: recoverable room query unavailable", ErrInvalidCommand)
	}
	ids, err := reader.FindRecoverableRoomIDs(ctx, actor)
	if err != nil {
		return nil, err
	}
	if len(ids) > 32 {
		return nil, fmt.Errorf("%w: recoverable room limit", ErrInvalidCommand)
	}
	views := make([]RoomView, 0, len(ids))
	seen := make(map[domain.RoomID]bool, len(ids))
	for _, id := range ids {
		if id == "" || seen[id] {
			return nil, fmt.Errorf("%w: invalid or repeated recoverable room ID", ErrInvalidCommand)
		}
		seen[id] = true
		view, queryErr := s.GetRoomView(ctx, actor, id)
		switch {
		case errors.Is(queryErr, ErrNotFound):
			// A concurrently closed/removed membership cannot be resurrected
			// by a previously generated discovery locator.
			continue
		case queryErr != nil:
			return nil, queryErr
		case !view.Member || view.Status == domain.RoomClosed:
			continue
		default:
			views = append(views, view)
		}
	}
	return views, nil
}
