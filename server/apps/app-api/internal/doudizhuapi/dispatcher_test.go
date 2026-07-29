package doudizhuapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/application"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/application/lifecycle"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/bidding"
)

func TestLiveRequestReplayAndConflict(t *testing.T) {
	backend := &backendStub{}
	life := &lifecycleStub{}
	dispatcher := newDispatcherForTest(t, backend, life)
	request := CommandRequest{Version: CommandRequestV1, RequestID: "req-1", Type: TypeHandBid, AggregateID: "hand-1", ClientSeq: 1, ExpectedVersion: 2, Payload: json.RawMessage(`{"score":1}`)}

	first, err := dispatcher.Execute(context.Background(), "account-1", request)
	if err != nil || first.Live == nil || first.Live.Version != 3 || backend.bidCalls != 1 || life.bidCalls != 1 {
		t.Fatalf("first=%#v bidCalls=%d lifecycle=%d err=%v", first, backend.bidCalls, life.bidCalls, err)
	}
	encoded, err := json.Marshal(first)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(encoded, []byte(`"payload":{"ok":true}`)) || bytes.Contains(encoded, []byte("eyJvayI6dHJ1ZX0=")) {
		t.Fatalf("live payload was not emitted as raw JSON: %s", encoded)
	}
	first.Live.Payload[0] ^= 0xff
	second, err := dispatcher.Execute(context.Background(), "account-1", request)
	if err != nil || second.Live == nil || string(second.Live.Payload) != `{"ok":true}` || backend.bidCalls != 1 || life.bidCalls != 1 {
		t.Fatalf("second=%#v bidCalls=%d lifecycle=%d err=%v", second, backend.bidCalls, life.bidCalls, err)
	}

	changed := request
	changed.Payload = json.RawMessage(`{"score":2}`)
	if _, err := dispatcher.Execute(context.Background(), "account-1", changed); !errors.Is(err, ErrReplayConflict) {
		t.Fatalf("conflict error=%v", err)
	}
}

func TestDurableCommandBindingAndLifecycleTracking(t *testing.T) {
	backend := &backendStub{}
	life := &lifecycleStub{}
	dispatcher := newDispatcherForTest(t, backend, life)
	request := CommandRequest{Version: CommandRequestV1, RequestID: "req-dealt", Type: TypeHandDealt, AggregateID: "hand-1", ClientSeq: 7, ExpectedVersion: 8}
	result, err := dispatcher.Execute(context.Background(), "account-1", request)
	if err != nil || result.Durable == nil || !result.Durable.Accepted || life.trackCalls != 1 {
		t.Fatalf("result=%#v track=%d err=%v", result, life.trackCalls, err)
	}
	if backend.lastCommand.Name != application.CommandHandDealt || backend.lastCommand.AggregateType != domain.AggregateHand || backend.lastCommand.CommandID != "req-dealt" || backend.lastCommand.ClientSeq != 7 || backend.lastCommand.ExpectedVersion != 8 || backend.lastCommand.IssuedAt.IsZero() || backend.lastCommand.ExpiresAt.Sub(backend.lastCommand.IssuedAt) != time.Minute {
		t.Fatalf("command=%#v", backend.lastCommand)
	}
}

func TestConcurrentReplayExecutesOnce(t *testing.T) {
	backend := &backendStub{block: make(chan struct{})}
	dispatcher := newDispatcherForTest(t, backend, &lifecycleStub{})
	request := CommandRequest{Version: CommandRequestV1, RequestID: "req-pass", Type: TypeHandPass, AggregateID: "hand-1", ClientSeq: 1, ExpectedVersion: 2}
	var wg sync.WaitGroup
	wg.Add(2)
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() {
			defer wg.Done()
			_, err := dispatcher.Execute(context.Background(), "account-1", request)
			errs <- err
		}()
	}
	for backend.passCount() == 0 {
		time.Sleep(time.Millisecond)
	}
	close(backend.block)
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if backend.passCount() != 1 {
		t.Fatalf("pass calls=%d", backend.passCount())
	}
}

func newDispatcherForTest(t *testing.T, backend Backend, life Lifecycle) *Dispatcher {
	t.Helper()
	dispatcher, err := NewDispatcher(backend, evidenceStub{}, life, fixedClock{now: time.UnixMilli(1000).UTC()}, DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	return dispatcher
}

type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time { return c.now }

type evidenceStub struct{}

func (evidenceStub) Get(context.Context, domain.AccountID, domain.HandID) (application.FinalEvidenceResult, error) {
	return application.FinalEvidenceResult{}, nil
}

type lifecycleStub struct{ trackCalls, bidCalls, playCalls, cancelCalls int }

func (s *lifecycleStub) TrackBidding(context.Context, domain.AccountID, domain.HandID) error {
	s.trackCalls++
	return nil
}
func (s *lifecycleStub) AfterBid(context.Context, domain.AccountID, domain.HandID, application.LiveHandCommandResult) (lifecycle.Outcome, error) {
	s.bidCalls++
	return lifecycle.Outcome{}, nil
}
func (s *lifecycleStub) AfterPlay(context.Context, domain.AccountID, domain.HandID, application.LiveHandCommandResult) error {
	s.playCalls++
	return nil
}
func (s *lifecycleStub) Cancel(context.Context, domain.AccountID, domain.HandID) (lifecycle.Outcome, error) {
	s.cancelCalls++
	return lifecycle.Outcome{Triggered: true}, nil
}

func (b *backendStub) passCount() int { b.mu.Lock(); defer b.mu.Unlock(); return b.passCalls }

type backendStub struct {
	mu                  sync.Mutex
	bidCalls, passCalls int
	lastCommand         application.Command
	block               chan struct{}
	publicView          application.LiveHandView
}

func (b *backendStub) durable(command application.Command) (application.CommandResult, error) {
	b.mu.Lock()
	b.lastCommand = command
	b.mu.Unlock()
	return application.CommandResult{Version: application.CommandResultV1, CommandID: command.CommandID, Accepted: true, AggregateType: command.AggregateType, AggregateID: command.AggregateID, AggregateVersion: command.ExpectedVersion + 1, Events: []application.EventRef{}}, nil
}
func (b *backendStub) CreateRoom(_ context.Context, _ domain.AccountID, c application.Command) (application.CommandResult, error) {
	return b.durable(c)
}
func (b *backendStub) JoinRoom(_ context.Context, _ domain.AccountID, c application.Command) (application.CommandResult, error) {
	return b.durable(c)
}
func (b *backendStub) LeaveRoom(_ context.Context, _ domain.AccountID, c application.Command) (application.CommandResult, error) {
	return b.durable(c)
}
func (b *backendStub) SetRoomReady(_ context.Context, _ domain.AccountID, c application.Command, _ application.SetReadyInput) (application.CommandResult, error) {
	return b.durable(c)
}
func (b *backendStub) StartRoomHand(_ context.Context, _ domain.AccountID, c application.Command, _ application.StartHandInput) (application.CommandResult, error) {
	return b.durable(c)
}
func (b *backendStub) SubmitHandCommit(_ context.Context, _ domain.AccountID, c application.Command, _ application.SubmitCommitInput) (application.CommandResult, error) {
	return b.durable(c)
}
func (b *backendStub) SubmitHandReveal(_ context.Context, _ domain.AccountID, c application.Command, _ application.SubmitRevealInput) (application.CommandResult, error) {
	return b.durable(c)
}
func (b *backendStub) LockHandBeacon(_ context.Context, _ domain.AccountID, c application.Command, _ application.LockBeaconInput) (application.CommandResult, error) {
	return b.durable(c)
}
func (b *backendStub) MarkHandDealt(_ context.Context, _ domain.AccountID, c application.Command) (application.CommandResult, error) {
	return b.durable(c)
}
func (b *backendStub) SubmitLiveHandBid(context.Context, domain.AccountID, domain.HandID, uint64, bidding.Score) (application.LiveHandCommandResult, error) {
	b.mu.Lock()
	b.bidCalls++
	b.mu.Unlock()
	return application.LiveHandCommandResult{Version: 3, Payload: []byte(`{"ok":true}`)}, nil
}
func (b *backendStub) SubmitLiveHandPlay(context.Context, domain.AccountID, domain.HandID, uint64, []string) (application.LiveHandCommandResult, error) {
	return application.LiveHandCommandResult{Version: 3, Payload: []byte(`{"ok":true}`)}, nil
}
func (b *backendStub) SubmitLiveHandPass(context.Context, domain.AccountID, domain.HandID, uint64) (application.LiveHandCommandResult, error) {
	b.mu.Lock()
	b.passCalls++
	block := b.block
	b.mu.Unlock()
	if block != nil {
		<-block
	}
	return application.LiveHandCommandResult{Version: 3, Payload: []byte(`{"ok":true}`)}, nil
}
func (b *backendStub) GetHandPublicView(context.Context, domain.AccountID, domain.HandID) (application.LiveHandView, error) {
	return b.publicView, nil
}
func (b *backendStub) GetHandPrivateView(context.Context, domain.AccountID, domain.HandID) (application.LiveHandView, error) {
	return application.LiveHandView{}, nil
}

func TestViewsExposeRawJSONAndCopyPayload(t *testing.T) {
	backend := &backendStub{publicView: application.LiveHandView{Version: 4, Payload: []byte(`{"phase":"PLAYING"}`)}}
	dispatcher := newDispatcherForTest(t, backend, &lifecycleStub{})
	view, err := dispatcher.PublicView(context.Background(), "account-1", "hand-1")
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(view)
	if err != nil {
		t.Fatal(err)
	}
	if string(encoded) != `{"version":4,"payload":{"phase":"PLAYING"}}` {
		t.Fatalf("encoded view=%s", encoded)
	}
	view.Payload[0] ^= 0xff
	if string(backend.publicView.Payload) != `{"phase":"PLAYING"}` {
		t.Fatal("dispatcher leaked mutable backend payload")
	}
}
