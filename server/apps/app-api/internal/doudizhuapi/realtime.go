package doudizhuapi

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/application"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/bidding"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/livehand"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/playing"
	"github.com/oublie6/awesome-zero-platform/server/business/gamecore"
	"github.com/oublie6/awesome-zero-platform/server/business/gamecore/infrastructure/mysqlarchive"
	"github.com/oublie6/awesome-zero-platform/server/platform/realtime"
)

const (
	RealtimeCommandV1  = "doudizhu.command"
	RealtimeSyncV1     = "doudizhu.hand.sync"
	RealtimeResultV1   = "doudizhu.command.result"
	RealtimeChangedV1  = "doudizhu.hand.changed"
	RealtimeSnapshotV1 = "doudizhu.hand.snapshot"
	RealtimeEvidenceV1 = "doudizhu.hand.evidence"
	RealtimeErrorV1    = "doudizhu.error"

	RealtimeEventVersionV1 = "doudizhu-realtime-event-v1"
	RealtimeSyncRequestV1  = "doudizhu-realtime-sync-v1"
)

type RealtimeHub interface {
	RegisterHandler(string, realtime.Handler) error
	SendAccount(string, realtime.Envelope) (int, error)
}

type RealtimeDispatcher interface {
	Execute(context.Context, domain.AccountID, CommandRequest) (CommandResponse, error)
	PrivateView(context.Context, domain.AccountID, domain.HandID) (ViewResult, error)
	FinalEvidence(context.Context, domain.AccountID, domain.HandID) (application.FinalEvidenceResult, error)
}

type HandAudience interface {
	LoadHand(context.Context, domain.HandID) (domain.HandSnapshot, error)
}

type RealtimeBridge struct {
	hub        RealtimeHub
	dispatcher RealtimeDispatcher
	audience   HandAudience
}

type RealtimeSyncRequest struct {
	Version      string `json:"v"`
	HandID       string `json:"handId"`
	KnownVersion uint64 `json:"knownVersion,omitempty"`
}

type RealtimeChangedEvent struct {
	Version  string          `json:"v"`
	HandID   string          `json:"handId"`
	Response CommandResponse `json:"response"`
}

type RealtimeSnapshotEvent struct {
	Version      string      `json:"v"`
	HandID       string      `json:"handId"`
	KnownVersion uint64      `json:"knownVersion,omitempty"`
	NotModified  bool        `json:"notModified"`
	View         *ViewResult `json:"view,omitempty"`
}

type RealtimeEvidenceEvent struct {
	Version  string                          `json:"v"`
	HandID   string                          `json:"handId"`
	Evidence application.FinalEvidenceResult `json:"evidence"`
}

type RealtimeErrorEvent struct {
	Version   string `json:"v"`
	RequestID string `json:"requestId,omitempty"`
	Code      string `json:"code"`
	Message   string `json:"message"`
}

func RegisterRealtime(hub RealtimeHub, dispatcher RealtimeDispatcher, audience HandAudience) (*RealtimeBridge, error) {
	if hub == nil || dispatcher == nil || audience == nil {
		return nil, ErrInvalidRequest
	}
	bridge := &RealtimeBridge{hub: hub, dispatcher: dispatcher, audience: audience}
	if err := hub.RegisterHandler(RealtimeCommandV1, bridge.handleCommand); err != nil {
		return nil, err
	}
	if err := hub.RegisterHandler(RealtimeSyncV1, bridge.handleSync); err != nil {
		return nil, err
	}
	return bridge, nil
}

func (b *RealtimeBridge) handleCommand(ctx context.Context, connection realtime.ConnectionContext, envelope realtime.Envelope) (*realtime.Envelope, error) {
	actor := domain.AccountID(strings.TrimSpace(connection.AccountID))
	var request CommandRequest
	if actor == "" || decodePayload(envelope.Payload, &request) != nil {
		return realtimeErrorEnvelope(envelope.ID, ErrInvalidRequest), nil
	}
	response, err := b.dispatcher.Execute(ctx, actor, request)
	if err != nil {
		return realtimeErrorEnvelope(envelope.ID, err), nil
	}
	if handID, ok := commandHandID(request); ok && commandApplied(response) {
		b.broadcastHand(ctx, handID, response)
	}
	return realtimeJSONEnvelope(envelope.ID, RealtimeResultV1, response), nil
}

func (b *RealtimeBridge) handleSync(ctx context.Context, connection realtime.ConnectionContext, envelope realtime.Envelope) (*realtime.Envelope, error) {
	actor := domain.AccountID(strings.TrimSpace(connection.AccountID))
	var request RealtimeSyncRequest
	if actor == "" || decodePayload(envelope.Payload, &request) != nil ||
		request.Version != RealtimeSyncRequestV1 || request.HandID == "" || request.HandID != strings.TrimSpace(request.HandID) || len(request.HandID) > 128 {
		return realtimeErrorEnvelope(envelope.ID, ErrInvalidRequest), nil
	}
	handID := domain.HandID(request.HandID)
	view, err := b.dispatcher.PrivateView(ctx, actor, handID)
	if err == nil {
		event := RealtimeSnapshotEvent{
			Version:      RealtimeEventVersionV1,
			HandID:       request.HandID,
			KnownVersion: request.KnownVersion,
			NotModified:  request.KnownVersion != 0 && request.KnownVersion == view.Version,
		}
		if !event.NotModified {
			copy := view
			event.View = &copy
		}
		return realtimeJSONEnvelope(envelope.ID, RealtimeSnapshotV1, event), nil
	}
	evidence, evidenceErr := b.dispatcher.FinalEvidence(ctx, actor, handID)
	if evidenceErr == nil {
		return realtimeJSONEnvelope(envelope.ID, RealtimeEvidenceV1, RealtimeEvidenceEvent{
			Version: RealtimeEventVersionV1, HandID: request.HandID, Evidence: evidence,
		}), nil
	}
	return realtimeErrorEnvelope(envelope.ID, preferredSyncError(err, evidenceErr)), nil
}

func (b *RealtimeBridge) broadcastHand(ctx context.Context, handID domain.HandID, response CommandResponse) {
	hand, err := b.audience.LoadHand(ctx, handID)
	if err != nil {
		return
	}
	changed := RealtimeChangedEvent{Version: RealtimeEventVersionV1, HandID: string(handID), Response: cloneResponse(response)}
	for _, seat := range hand.Seats {
		accountID := strings.TrimSpace(string(seat.AccountID))
		if accountID == "" {
			continue
		}
		_, _ = b.hub.SendAccount(accountID, *realtimeJSONEnvelope("", RealtimeChangedV1, changed))
		view, viewErr := b.dispatcher.PrivateView(ctx, seat.AccountID, handID)
		if viewErr == nil {
			_, _ = b.hub.SendAccount(accountID, *realtimeJSONEnvelope("", RealtimeSnapshotV1, RealtimeSnapshotEvent{
				Version: RealtimeEventVersionV1, HandID: string(handID), View: &view,
			}))
			continue
		}
		evidence, evidenceErr := b.dispatcher.FinalEvidence(ctx, seat.AccountID, handID)
		if evidenceErr == nil {
			_, _ = b.hub.SendAccount(accountID, *realtimeJSONEnvelope("", RealtimeEvidenceV1, RealtimeEvidenceEvent{
				Version: RealtimeEventVersionV1, HandID: string(handID), Evidence: evidence,
			}))
		}
	}
}

func commandHandID(request CommandRequest) (domain.HandID, bool) {
	if strings.HasPrefix(request.Type, "hand.") {
		return domain.HandID(request.AggregateID), true
	}
	if request.Type != TypeRoomStart {
		return "", false
	}
	var payload struct {
		HandID string `json:"handId"`
	}
	if decodePayload(request.Payload, &payload) != nil || payload.HandID == "" || payload.HandID != strings.TrimSpace(payload.HandID) {
		return "", false
	}
	return domain.HandID(payload.HandID), true
}

func commandApplied(response CommandResponse) bool {
	if response.Live != nil || response.Termination != nil {
		return true
	}
	return response.Durable != nil && response.Durable.Accepted
}

func realtimeJSONEnvelope(id, messageType string, value any) *realtime.Envelope {
	payload, err := json.Marshal(value)
	if err != nil {
		return realtimeErrorEnvelope(id, err)
	}
	return &realtime.Envelope{ID: id, Type: messageType, Payload: payload}
}

func realtimeErrorEnvelope(requestID string, err error) *realtime.Envelope {
	code, message := realtimeError(err)
	payload, _ := json.Marshal(RealtimeErrorEvent{
		Version: RealtimeEventVersionV1, RequestID: requestID, Code: code, Message: message,
	})
	return &realtime.Envelope{ID: requestID, Type: RealtimeErrorV1, Payload: payload}
}

func realtimeError(err error) (string, string) {
	switch {
	case errors.Is(err, ErrInvalidRequest), errors.Is(err, application.ErrInvalidCommand), errors.Is(err, domain.ErrInvalidArgument),
		errors.Is(err, livehand.ErrMalformedCommand), errors.Is(err, livehand.ErrCardNotHeld),
		errors.Is(err, bidding.ErrInvalidScore), errors.Is(err, playing.ErrInvalidPattern), errors.Is(err, playing.ErrInvalidPatternValue):
		return "INVALID_REQUEST", "Doudizhu request is invalid"
	case errors.Is(err, ErrReplayConflict), errors.Is(err, application.ErrOptimisticConflict), errors.Is(err, application.ErrSequenceConflict),
		errors.Is(err, domain.ErrVersionConflict), errors.Is(err, livehand.ErrVersionConflict), errors.Is(err, gamecore.ErrInstanceExists),
		errors.Is(err, gamecore.ErrFinalizationPending), errors.Is(err, bidding.ErrWrongTurn), errors.Is(err, bidding.ErrBidNotHigher),
		errors.Is(err, bidding.ErrBiddingComplete), errors.Is(err, playing.ErrWrongTurn), errors.Is(err, playing.ErrCannotPass),
		errors.Is(err, playing.ErrDoesNotBeat), errors.Is(err, playing.ErrPlayingComplete):
		return "CONFLICT", "Doudizhu state conflict"
	case errors.Is(err, domain.ErrNotSeated), errors.Is(err, domain.ErrForbidden), errors.Is(err, application.ErrFinalEvidenceForbidden), errors.Is(err, livehand.ErrViewerNotSeated):
		return "FORBIDDEN", "Doudizhu access denied"
	case errors.Is(err, application.ErrNotFound), errors.Is(err, gamecore.ErrInstanceNotFound), errors.Is(err, mysqlarchive.ErrArchiveNotFound):
		return "NOT_FOUND", "Doudizhu resource not found"
	default:
		return "INTERNAL", "Doudizhu request failed"
	}
}

func preferredSyncError(viewErr, evidenceErr error) error {
	if errors.Is(viewErr, domain.ErrNotSeated) || errors.Is(viewErr, domain.ErrForbidden) || errors.Is(viewErr, livehand.ErrViewerNotSeated) ||
		errors.Is(evidenceErr, application.ErrFinalEvidenceForbidden) {
		return domain.ErrForbidden
	}
	if !errors.Is(evidenceErr, mysqlarchive.ErrArchiveNotFound) && !errors.Is(evidenceErr, application.ErrNotFound) {
		return evidenceErr
	}
	return viewErr
}
