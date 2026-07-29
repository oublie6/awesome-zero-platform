package lifecycle

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/application"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/livehand"
)

func TestNoLandlordTerminationRetriesTheSameCommand(t *testing.T) {
	clock := &testClock{now: time.UnixMilli(1_000).UTC()}
	hands := &testHands{snapshots: map[domain.HandID]domain.HandSnapshot{
		"hand-1": testHand("hand-1", "seat-1", domain.HandBidding, 8),
	}}
	terminal := &testTerminal{abortFailures: 1}
	ids := &testIDs{prefix: "lifecycle"}
	supervisor := newTestSupervisor(t, hands, terminal, clock, ids)

	result := bidResult(t, "hand-1", 4, livehand.PhaseNoLandlord, true)
	outcome, err := supervisor.AfterBid(context.Background(), "seat-1", "hand-1", result)
	if err == nil || !outcome.Triggered || !supervisor.Pending("hand-1") {
		t.Fatalf("outcome=%#v pending=%v err=%v", outcome, supervisor.Pending("hand-1"), err)
	}
	first := terminal.abortCalls[0]
	if first.actor != SystemActor || first.command.Name != application.CommandHandAbort ||
		first.command.ExpectedVersion != 8 || first.command.ClientSeq != 1 || first.reason != ReasonNoLandlord {
		t.Fatalf("first=%#v", first)
	}

	retried, err := supervisor.Retry(context.Background(), "hand-1")
	if err != nil || !retried.Triggered || supervisor.Pending("hand-1") {
		t.Fatalf("retried=%#v pending=%v err=%v", retried, supervisor.Pending("hand-1"), err)
	}
	if len(terminal.abortCalls) != 2 || terminal.abortCalls[1].command != first.command {
		t.Fatalf("abort calls=%#v", terminal.abortCalls)
	}
	if ids.calls != 1 {
		t.Fatalf("id calls=%d want=1", ids.calls)
	}
}

func TestBiddingAndPlayingDeadlinesUseUnifiedExpiry(t *testing.T) {
	clock := &testClock{now: time.UnixMilli(2_000).UTC()}
	hands := &testHands{snapshots: map[domain.HandID]domain.HandSnapshot{
		"bidding": testHand("bidding", "seat-1", domain.HandBidding, 5),
		"playing": testHand("playing", "seat-2", domain.HandBidding, 7),
	}}
	terminal := &testTerminal{}
	config := DefaultConfig()
	config.BiddingTimeout = 5 * time.Second
	config.PlayingTimeout = 10 * time.Second
	supervisor, err := New(hands, terminal, clock, &testIDs{prefix: "timeout"}, config)
	if err != nil {
		t.Fatal(err)
	}

	if err := supervisor.TrackBidding(context.Background(), "seat-1", "bidding"); err != nil {
		t.Fatal(err)
	}
	if _, err := supervisor.AfterBid(context.Background(), "seat-2", "playing", bidResult(t, "playing", 9, livehand.PhasePlaying, false)); err != nil {
		t.Fatal(err)
	}
	clock.Advance(5 * time.Second)
	terminated, err := supervisor.Sweep(context.Background())
	if err != nil || len(terminated) != 1 || terminated[0] != "bidding" {
		t.Fatalf("terminated=%v err=%v", terminated, err)
	}
	if len(terminal.expireCalls) != 1 || terminal.expireCalls[0].reason != ReasonBiddingTimeout {
		t.Fatalf("expire calls=%#v", terminal.expireCalls)
	}

	clock.Advance(5 * time.Second)
	terminated, err = supervisor.Sweep(context.Background())
	if err != nil || len(terminated) != 1 || terminated[0] != "playing" {
		t.Fatalf("terminated=%v err=%v", terminated, err)
	}
	if len(terminal.expireCalls) != 2 || terminal.expireCalls[1].reason != ReasonPlayingTimeout {
		t.Fatalf("expire calls=%#v", terminal.expireCalls)
	}
}

func TestParticipantCancellationRequiresASeat(t *testing.T) {
	clock := &testClock{now: time.UnixMilli(3_000).UTC()}
	hands := &testHands{snapshots: map[domain.HandID]domain.HandSnapshot{
		"hand-1": testHand("hand-1", "seat-1", domain.HandBidding, 4),
	}}
	terminal := &testTerminal{}
	supervisor := newTestSupervisor(t, hands, terminal, clock, &testIDs{prefix: "cancel"})

	if _, err := supervisor.Cancel(context.Background(), "outsider", "hand-1"); !errors.Is(err, domain.ErrNotSeated) {
		t.Fatalf("outsider error=%v", err)
	}
	if len(terminal.abortCalls) != 0 {
		t.Fatalf("unexpected abort calls=%#v", terminal.abortCalls)
	}
	outcome, err := supervisor.Cancel(context.Background(), "seat-1", "hand-1")
	if err != nil || !outcome.Triggered || len(terminal.abortCalls) != 1 || terminal.abortCalls[0].reason != ReasonParticipantCancel {
		t.Fatalf("outcome=%#v calls=%#v err=%v", outcome, terminal.abortCalls, err)
	}
}

func TestCompletedPlayForgetsTheDeadline(t *testing.T) {
	clock := &testClock{now: time.UnixMilli(4_000).UTC()}
	hands := &testHands{snapshots: map[domain.HandID]domain.HandSnapshot{
		"hand-1": testHand("hand-1", "seat-1", domain.HandBidding, 6),
	}}
	terminal := &testTerminal{}
	supervisor := newTestSupervisor(t, hands, terminal, clock, &testIDs{prefix: "complete"})
	if err := supervisor.TrackBidding(context.Background(), "seat-1", "hand-1"); err != nil {
		t.Fatal(err)
	}
	payload, _ := json.Marshal(livehand.PlayResult{
		Version: livehand.PlayResultVersion, HandID: "hand-1", StateVersion: 12, Phase: livehand.PhaseCompleted,
	})
	if err := supervisor.AfterPlay(context.Background(), "seat-1", "hand-1", application.LiveHandCommandResult{Version: 12, Payload: payload}); err != nil {
		t.Fatal(err)
	}
	clock.Advance(time.Hour)
	terminated, err := supervisor.Sweep(context.Background())
	if err != nil || len(terminated) != 0 || len(terminal.expireCalls) != 0 {
		t.Fatalf("terminated=%v calls=%#v err=%v", terminated, terminal.expireCalls, err)
	}
}

func TestLifecycleResultDecodingIsStrict(t *testing.T) {
	supervisor := newTestSupervisor(t,
		&testHands{snapshots: map[domain.HandID]domain.HandSnapshot{"hand-1": testHand("hand-1", "seat-1", domain.HandBidding, 2)}},
		&testTerminal{}, &testClock{now: time.UnixMilli(5_000).UTC()}, &testIDs{prefix: "strict"},
	)
	_, err := supervisor.AfterBid(context.Background(), "seat-1", "hand-1", application.LiveHandCommandResult{
		Version: 2, Payload: []byte(`{"v":"doudizhu-live-bid-result-v1","handId":"hand-1","stateVersion":2,"phase":"BIDDING","bidding":{},"requiresTermination":false,"unknown":true}`),
	})
	if err == nil {
		t.Fatal("unknown JSON field should fail")
	}
}

func newTestSupervisor(t *testing.T, hands HandReader, terminal TerminalService, clock application.Clock, ids application.IDGenerator) *Supervisor {
	t.Helper()
	config := DefaultConfig()
	config.BiddingTimeout = 5 * time.Second
	config.PlayingTimeout = 10 * time.Second
	supervisor, err := New(hands, terminal, clock, ids, config)
	if err != nil {
		t.Fatal(err)
	}
	return supervisor
}

func testHand(id domain.HandID, participant domain.AccountID, value domain.HandPhase, version uint64) domain.HandSnapshot {
	return domain.HandSnapshot{
		ID: id, RoomID: domain.RoomID(string(id) + "-room"), Phase: value, Version: version,
		Seats: [3]domain.HandSeat{
			{Seat: domain.SeatOne, AccountID: participant},
			{Seat: domain.SeatTwo, AccountID: domain.AccountID(string(id) + "-seat-2")},
			{Seat: domain.SeatThree, AccountID: domain.AccountID(string(id) + "-seat-3")},
		},
	}
}

func bidResult(t *testing.T, handID domain.HandID, version uint64, value string, terminal bool) application.LiveHandCommandResult {
	t.Helper()
	payload, err := json.Marshal(livehand.BidResult{
		Version: livehand.BidResultVersion, HandID: string(handID), StateVersion: version,
		Phase: value, RequiresTermination: terminal,
	})
	if err != nil {
		t.Fatal(err)
	}
	return application.LiveHandCommandResult{Version: version, Payload: payload, RequiresTermination: terminal}
}

type testHands struct {
	snapshots map[domain.HandID]domain.HandSnapshot
}

func (h *testHands) LoadHand(_ context.Context, handID domain.HandID) (domain.HandSnapshot, error) {
	snapshot, ok := h.snapshots[handID]
	if !ok {
		return domain.HandSnapshot{}, fmt.Errorf("missing hand %s", handID)
	}
	return snapshot, nil
}

type terminalCall struct {
	actor   domain.AccountID
	command application.Command
	reason  string
}

type testTerminal struct {
	mu            sync.Mutex
	abortCalls    []terminalCall
	expireCalls   []terminalCall
	abortFailures int
	expireFailure int
}

func (t *testTerminal) AbortHand(_ context.Context, actor domain.AccountID, command application.Command, input application.TerminateHandInput) (application.CommandResult, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.abortCalls = append(t.abortCalls, terminalCall{actor: actor, command: command, reason: input.ReasonCode})
	result := accepted(command)
	if t.abortFailures > 0 {
		t.abortFailures--
		return result, fmt.Errorf("archive unavailable")
	}
	return result, nil
}

func (t *testTerminal) ExpireHand(_ context.Context, actor domain.AccountID, command application.Command, input application.TerminateHandInput) (application.CommandResult, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.expireCalls = append(t.expireCalls, terminalCall{actor: actor, command: command, reason: input.ReasonCode})
	result := accepted(command)
	if t.expireFailure > 0 {
		t.expireFailure--
		return result, fmt.Errorf("archive unavailable")
	}
	return result, nil
}

func accepted(command application.Command) application.CommandResult {
	return application.CommandResult{
		Version: application.CommandResultV1, CommandID: command.CommandID, Accepted: true,
		AggregateType: domain.AggregateHand, AggregateID: command.AggregateID,
		AggregateVersion: command.ExpectedVersion + 2, Events: []application.EventRef{},
	}
}

type testClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *testClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *testClock) Advance(value time.Duration) {
	c.mu.Lock()
	c.now = c.now.Add(value)
	c.mu.Unlock()
}

type testIDs struct {
	mu     sync.Mutex
	prefix string
	calls  int
}

func (g *testIDs) NewID() (string, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.calls++
	return fmt.Sprintf("%s-%d", g.prefix, g.calls), nil
}
