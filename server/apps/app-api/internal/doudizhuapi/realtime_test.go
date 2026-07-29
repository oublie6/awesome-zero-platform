package doudizhuapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/application"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
	"github.com/oublie6/awesome-zero-platform/server/business/gamecore"
	"github.com/oublie6/awesome-zero-platform/server/business/gamecore/infrastructure/mysqlarchive"
	"github.com/oublie6/awesome-zero-platform/server/platform/realtime"
)

type realtimeTestHub struct {
	handlers map[string]realtime.Handler
	sent     map[string][]realtime.Envelope
	sendErr  error
}

func (h *realtimeTestHub) RegisterHandler(messageType string, handler realtime.Handler) error {
	if h.handlers == nil {
		h.handlers = make(map[string]realtime.Handler)
	}
	if h.handlers[messageType] != nil {
		return errors.New("duplicate handler")
	}
	h.handlers[messageType] = handler
	return nil
}

func (h *realtimeTestHub) SendAccount(accountID string, envelope realtime.Envelope) (int, error) {
	if h.sent == nil {
		h.sent = make(map[string][]realtime.Envelope)
	}
	h.sent[accountID] = append(h.sent[accountID], envelope)
	if h.sendErr != nil {
		return 0, h.sendErr
	}
	return 1, nil
}

type realtimeTestDispatcher struct {
	execute  func(context.Context, domain.AccountID, CommandRequest) (CommandResponse, error)
	view     func(context.Context, domain.AccountID, domain.HandID) (ViewResult, error)
	evidence func(context.Context, domain.AccountID, domain.HandID) (application.FinalEvidenceResult, error)
}

func (d realtimeTestDispatcher) Execute(ctx context.Context, actor domain.AccountID, request CommandRequest) (CommandResponse, error) {
	return d.execute(ctx, actor, request)
}

func (d realtimeTestDispatcher) PrivateView(ctx context.Context, actor domain.AccountID, handID domain.HandID) (ViewResult, error) {
	return d.view(ctx, actor, handID)
}

func (d realtimeTestDispatcher) FinalEvidence(ctx context.Context, actor domain.AccountID, handID domain.HandID) (application.FinalEvidenceResult, error) {
	return d.evidence(ctx, actor, handID)
}

type realtimeTestAudience struct {
	hand domain.HandSnapshot
	err  error
}

func (a realtimeTestAudience) LoadHand(context.Context, domain.HandID) (domain.HandSnapshot, error) {
	return a.hand, a.err
}

func TestRealtimeCommandUsesConnectionIdentityAndBroadcastsPrivateSnapshots(t *testing.T) {
	hub := &realtimeTestHub{}
	hand := domain.HandSnapshot{ID: "hand-1", Seats: [3]domain.HandSeat{
		{Seat: domain.SeatOne, AccountID: "account-1"},
		{Seat: domain.SeatTwo, AccountID: "account-2"},
		{Seat: domain.SeatThree, AccountID: "account-3"},
	}}
	var executedActor domain.AccountID
	dispatcher := realtimeTestDispatcher{
		execute: func(_ context.Context, actor domain.AccountID, request CommandRequest) (CommandResponse, error) {
			executedActor = actor
			if request.AggregateID != "hand-1" || request.Type != TypeHandPass {
				t.Fatalf("request = %#v", request)
			}
			return CommandResponse{Version: CommandResponseV1, RequestID: request.RequestID, Type: request.Type, Live: &LiveResult{
				Version: 8, Payload: json.RawMessage(`{"stateVersion":8}`),
			}}, nil
		},
		view: func(_ context.Context, actor domain.AccountID, handID domain.HandID) (ViewResult, error) {
			return ViewResult{Version: 8, Payload: json.RawMessage(fmt.Sprintf(`{"handId":%q,"accountId":%q}`, handID, actor))}, nil
		},
		evidence: func(context.Context, domain.AccountID, domain.HandID) (application.FinalEvidenceResult, error) {
			return application.FinalEvidenceResult{}, mysqlarchive.ErrArchiveNotFound
		},
	}
	if _, err := RegisterRealtime(hub, dispatcher, realtimeTestAudience{hand: hand}); err != nil {
		t.Fatal(err)
	}
	request := CommandRequest{Version: CommandRequestV1, RequestID: "request-1", Type: TypeHandPass, AggregateID: "hand-1", ClientSeq: 4, ExpectedVersion: 7}
	payload, _ := json.Marshal(request)
	response, err := hub.handlers[RealtimeCommandV1](context.Background(), realtime.ConnectionContext{AccountID: "account-2"}, realtime.Envelope{ID: "message-1", Type: RealtimeCommandV1, Payload: payload})
	if err != nil || response == nil || response.Type != RealtimeResultV1 || response.ID != "message-1" {
		t.Fatalf("response=%#v err=%v", response, err)
	}
	if executedActor != "account-2" {
		t.Fatalf("executed actor = %q", executedActor)
	}
	for _, accountID := range []string{"account-1", "account-2", "account-3"} {
		events := hub.sent[accountID]
		if len(events) != 2 || events[0].Type != RealtimeChangedV1 || events[1].Type != RealtimeSnapshotV1 {
			t.Fatalf("events[%s] = %#v", accountID, events)
		}
		var snapshot RealtimeSnapshotEvent
		if err := json.Unmarshal(events[1].Payload, &snapshot); err != nil {
			t.Fatal(err)
		}
		if snapshot.View == nil || !json.Valid(snapshot.View.Payload) || !containsJSONText(snapshot.View.Payload, accountID) {
			t.Fatalf("snapshot[%s] = %#v", accountID, snapshot)
		}
	}
}

func TestRealtimeAcceptedCommandIsNotRolledBackByBroadcastFailure(t *testing.T) {
	hub := &realtimeTestHub{sendErr: errors.New("slow consumer")}
	hand := domain.HandSnapshot{ID: "hand-1", Seats: [3]domain.HandSeat{
		{Seat: domain.SeatOne, AccountID: "account-1"},
		{Seat: domain.SeatTwo, AccountID: "account-2"},
		{Seat: domain.SeatThree, AccountID: "account-3"},
	}}
	dispatcher := realtimeTestDispatcher{
		execute: func(_ context.Context, _ domain.AccountID, request CommandRequest) (CommandResponse, error) {
			return CommandResponse{Version: CommandResponseV1, RequestID: request.RequestID, Type: request.Type, Live: &LiveResult{Version: 3, Payload: json.RawMessage(`{"accepted":true}`)}}, nil
		},
		view: func(context.Context, domain.AccountID, domain.HandID) (ViewResult, error) {
			return ViewResult{Version: 3, Payload: json.RawMessage(`{"private":true}`)}, nil
		},
		evidence: func(context.Context, domain.AccountID, domain.HandID) (application.FinalEvidenceResult, error) {
			return application.FinalEvidenceResult{}, mysqlarchive.ErrArchiveNotFound
		},
	}
	if _, err := RegisterRealtime(hub, dispatcher, realtimeTestAudience{hand: hand}); err != nil {
		t.Fatal(err)
	}
	request := CommandRequest{Version: CommandRequestV1, RequestID: "request-1", Type: TypeHandPass, AggregateID: "hand-1", ExpectedVersion: 2}
	payload, _ := json.Marshal(request)
	response, err := hub.handlers[RealtimeCommandV1](context.Background(), realtime.ConnectionContext{AccountID: "account-1"}, realtime.Envelope{ID: "message-1", Payload: payload})
	if err != nil || response == nil || response.Type != RealtimeResultV1 {
		t.Fatalf("response=%#v err=%v", response, err)
	}
}

func TestRealtimeSyncReturnsSnapshotNotModifiedOrFinalEvidence(t *testing.T) {
	hub := &realtimeTestHub{}
	active := true
	dispatcher := realtimeTestDispatcher{
		execute: func(context.Context, domain.AccountID, CommandRequest) (CommandResponse, error) {
			return CommandResponse{}, errors.New("unexpected execute")
		},
		view: func(_ context.Context, actor domain.AccountID, handID domain.HandID) (ViewResult, error) {
			if actor != "account-1" || handID != "hand-1" {
				return ViewResult{}, domain.ErrNotSeated
			}
			if !active {
				return ViewResult{}, gamecore.ErrInstanceNotFound
			}
			return ViewResult{Version: 9, Payload: json.RawMessage(`{"private":true}`)}, nil
		},
		evidence: func(_ context.Context, actor domain.AccountID, handID domain.HandID) (application.FinalEvidenceResult, error) {
			if actor != "account-1" || handID != "hand-1" {
				return application.FinalEvidenceResult{}, application.ErrFinalEvidenceForbidden
			}
			return application.FinalEvidenceResult{Version: application.FinalEvidenceResultV1, HandID: handID}, nil
		},
	}
	if _, err := RegisterRealtime(hub, dispatcher, realtimeTestAudience{}); err != nil {
		t.Fatal(err)
	}
	sync := func(known uint64) *realtime.Envelope {
		payload, _ := json.Marshal(RealtimeSyncRequest{Version: RealtimeSyncRequestV1, HandID: "hand-1", KnownVersion: known})
		response, err := hub.handlers[RealtimeSyncV1](context.Background(), realtime.ConnectionContext{AccountID: "account-1"}, realtime.Envelope{ID: "sync-1", Type: RealtimeSyncV1, Payload: payload})
		if err != nil {
			t.Fatal(err)
		}
		return response
	}
	response := sync(9)
	if response.Type != RealtimeSnapshotV1 {
		t.Fatalf("active response = %#v", response)
	}
	var snapshot RealtimeSnapshotEvent
	if err := json.Unmarshal(response.Payload, &snapshot); err != nil {
		t.Fatal(err)
	}
	if !snapshot.NotModified || snapshot.View != nil {
		t.Fatalf("snapshot = %#v", snapshot)
	}
	response = sync(8)
	if err := json.Unmarshal(response.Payload, &snapshot); err != nil || snapshot.NotModified || snapshot.View == nil {
		t.Fatalf("stale snapshot=%#v err=%v", snapshot, err)
	}
	active = false
	response = sync(9)
	if response.Type != RealtimeEvidenceV1 {
		t.Fatalf("terminal response = %#v", response)
	}
	var evidence RealtimeEvidenceEvent
	if err := json.Unmarshal(response.Payload, &evidence); err != nil || evidence.Evidence.HandID != "hand-1" {
		t.Fatalf("evidence=%#v err=%v", evidence, err)
	}
}

func TestRealtimeRejectsMalformedPayloadAndForbiddenSync(t *testing.T) {
	hub := &realtimeTestHub{}
	dispatcher := realtimeTestDispatcher{
		execute: func(context.Context, domain.AccountID, CommandRequest) (CommandResponse, error) {
			return CommandResponse{}, errors.New("unexpected execute")
		},
		view: func(context.Context, domain.AccountID, domain.HandID) (ViewResult, error) {
			return ViewResult{}, domain.ErrNotSeated
		},
		evidence: func(context.Context, domain.AccountID, domain.HandID) (application.FinalEvidenceResult, error) {
			return application.FinalEvidenceResult{}, application.ErrFinalEvidenceForbidden
		},
	}
	if _, err := RegisterRealtime(hub, dispatcher, realtimeTestAudience{}); err != nil {
		t.Fatal(err)
	}
	response, err := hub.handlers[RealtimeCommandV1](context.Background(), realtime.ConnectionContext{AccountID: "account-1"}, realtime.Envelope{ID: "bad", Payload: json.RawMessage(`{"v":"bad","unknown":true}`)})
	if err != nil || response.Type != RealtimeErrorV1 {
		t.Fatalf("malformed response=%#v err=%v", response, err)
	}
	assertRealtimeErrorCode(t, response, "INVALID_REQUEST")

	payload, _ := json.Marshal(RealtimeSyncRequest{Version: RealtimeSyncRequestV1, HandID: "hand-1"})
	response, err = hub.handlers[RealtimeSyncV1](context.Background(), realtime.ConnectionContext{AccountID: "outsider"}, realtime.Envelope{ID: "sync", Payload: payload})
	if err != nil || response.Type != RealtimeErrorV1 {
		t.Fatalf("forbidden response=%#v err=%v", response, err)
	}
	assertRealtimeErrorCode(t, response, "FORBIDDEN")
}

func assertRealtimeErrorCode(t *testing.T, envelope *realtime.Envelope, expected string) {
	t.Helper()
	var event RealtimeErrorEvent
	if err := json.Unmarshal(envelope.Payload, &event); err != nil {
		t.Fatal(err)
	}
	if event.Code != expected {
		t.Fatalf("error event = %#v", event)
	}
}

func containsJSONText(payload json.RawMessage, value string) bool {
	var decoded map[string]any
	if json.Unmarshal(payload, &decoded) != nil {
		return false
	}
	return decoded["accountId"] == value
}
