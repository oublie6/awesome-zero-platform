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
		first.command.ExpectedVersion != 8 || first.command.ClientSeq != 8 || first.reason != ReasonNoLandlord {
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

func TestRepeatedDealtTrackingPreservesOriginalDeadline(t *testing.T) {
	clock := &testClock{now: time.UnixMilli(2_400).UTC()}
	hands := &testHands{snapshots: map[domain.HandID]domain.HandSnapshot{
		"hand-dealt": testHand("hand-dealt", "seat-1", domain.HandBidding, 5),
	}}
	terminal := &testTerminal{}
	supervisor := newTestSupervisor(t, hands, terminal, clock, &testIDs{prefix: "dealt-idempotent"})
	if err := supervisor.TrackBidding(context.Background(), "seat-1", "hand-dealt"); err != nil {
		t.Fatal(err)
	}
	clock.Advance(4 * time.Second)
	if err := supervisor.TrackBidding(context.Background(), "seat-1", "hand-dealt"); err != nil {
		t.Fatal(err)
	}
	clock.Advance(2 * time.Second)
	terminated, err := supervisor.Sweep(context.Background())
	if err != nil || len(terminated) != 1 || terminated[0] != "hand-dealt" {
		t.Fatalf("duplicate dealt extended bidding timeout: terminated=%v err=%v", terminated, err)
	}
	if len(terminal.expireCalls) != 1 || terminal.expireCalls[0].reason != ReasonBiddingTimeout {
		t.Fatalf("unexpected expiry after duplicate dealt: %#v", terminal.expireCalls)
	}
}

func TestLateDealtTrackingCannotRegressPlayingDeadline(t *testing.T) {
	clock := &testClock{now: time.UnixMilli(3_500).UTC()}
	hands := &testHands{snapshots: map[domain.HandID]domain.HandSnapshot{
		"hand-playing": testHand("hand-playing", "seat-1", domain.HandBidding, 5),
	}}
	terminal := &testTerminal{}
	supervisor := newTestSupervisor(t, hands, terminal, clock, &testIDs{prefix: "dealt-phase"})
	if err := supervisor.TrackBidding(context.Background(), "seat-1", "hand-playing"); err != nil {
		t.Fatal(err)
	}
	if _, err := supervisor.AfterBid(context.Background(), "seat-1", "hand-playing", bidResult(t, "hand-playing", 6, livehand.PhasePlaying, false)); err != nil {
		t.Fatal(err)
	}
	playing := hands.snapshots["hand-playing"]
	playing.Phase = domain.HandPlaying
	hands.snapshots["hand-playing"] = playing
	clock.Advance(4 * time.Second)
	if err := supervisor.TrackBidding(context.Background(), "seat-1", "hand-playing"); err != nil {
		t.Fatal(err)
	}
	clock.Advance(6 * time.Second)
	terminated, err := supervisor.Sweep(context.Background())
	if err != nil || len(terminated) != 1 || terminated[0] != "hand-playing" {
		t.Fatalf("late dealt tracking replaced playing timer: terminated=%v err=%v", terminated, err)
	}
	if len(terminal.expireCalls) != 1 || terminal.expireCalls[0].reason != ReasonPlayingTimeout {
		t.Fatalf("wrong phase timeout after dealt replay: %#v", terminal.expireCalls)
	}
}

// A previously committed low bid can return after a later winning bid.
// Callback order must not undo the authoritative PLAYING deadline.
func TestFreshBidCallbacksUseCommitVersionOrder(t *testing.T) {
	clock := &testClock{now: time.Unix(1800000030, 0)}
	hands := &testHands{snapshots: map[domain.HandID]domain.HandSnapshot{
		"hand-order": testHand("hand-order", "seat-1", domain.HandBidding, 9),
	}}
	terminal := &phaseTestTerminal{testTerminal: &testTerminal{}, hands: hands}
	s := newTestSupervisor(t, hands, terminal, clock, &testIDs{prefix: "bid-order"})
	if err := s.TrackBidding(context.Background(), "seat-1", "hand-order"); err != nil {
		t.Fatal(err)
	}
	clock.Advance(time.Second)
	// Winning bid v3 has committed and its callback finishes before the
	// previously committed low bid v2 returns from the transport.
	if _, err := s.AfterBid(context.Background(), "seat-1", "hand-order",
		bidResult(t, "hand-order", 3, livehand.PhasePlaying, false)); err != nil {
		t.Fatal(err)
	}
	if hands.snapshots["hand-order"].Phase != domain.HandPlaying {
		t.Fatal("winning bid did not durably advance hand phase")
	}
	clock.Advance(4 * time.Second)
	if _, err := s.AfterBid(context.Background(), "seat-1", "hand-order",
		bidResult(t, "hand-order", 2, livehand.PhaseBidding, false)); err != nil {
		t.Fatal(err)
	}
	if current := s.entries["hand-order"]; current.phase != phasePlaying || current.liveVersion != 3 {
		t.Fatalf("late callback regressed timer: %+v", current)
	}
	clock.Advance(2 * time.Second) // Past the old 5s bidding deadline.
	if terminated, err := s.Sweep(context.Background()); err != nil || len(terminated) != 0 {
		t.Fatalf("old bidding response ended active playing hand: %v %v", terminated, err)
	}
	clock.Advance(4 * time.Second) // The original 10s playing deadline.
	if terminated, err := s.Sweep(context.Background()); err != nil || len(terminated) != 1 ||
		terminated[0] != "hand-order" {
		t.Fatalf("playing deadline not honored: %v %v", terminated, err)
	}
	if len(terminal.expireCalls) != 1 || terminal.expireCalls[0].reason != ReasonPlayingTimeout {
		t.Fatalf("timeout reason changed: %+v", terminal.expireCalls)
	}
}

func TestFreshPlayingCallbacksOnlyRenewForHigherVersions(t *testing.T) {
	clock := &testClock{now: time.Unix(1800000040, 0)}
	hands := &testHands{snapshots: map[domain.HandID]domain.HandSnapshot{
		"hand-play": testHand("hand-play", "seat-1", domain.HandPlaying, 11),
	}}
	terminal := &testTerminal{}
	s := newTestSupervisor(t, hands, terminal, clock, &testIDs{prefix: "play-order"})
	makePlay := func(version uint64) application.LiveHandCommandResult {
		payload, err := json.Marshal(livehand.PlayResult{Version: livehand.PlayResultVersion,
			HandID: "hand-play", StateVersion: version, Phase: livehand.PhasePlaying})
		if err != nil {
			t.Fatal(err)
		}
		return application.LiveHandCommandResult{Version: version, Payload: payload}
	}
	if err := s.AfterPlay(context.Background(), "seat-1", "hand-play", makePlay(12)); err != nil {
		t.Fatal(err)
	}
	clock.Advance(time.Second)
	if err := s.AfterPlay(context.Background(), "seat-1", "hand-play", makePlay(13)); err != nil {
		t.Fatal(err)
	}
	clock.Advance(8 * time.Second)
	if err := s.AfterPlay(context.Background(), "seat-1", "hand-play", makePlay(12)); err != nil {
		t.Fatal(err)
	}
	if err := s.AfterPlay(context.Background(), "seat-1", "hand-play", makePlay(13)); err != nil {
		t.Fatal(err)
	}
	clock.Advance(2 * time.Second)
	terminated, err := s.Sweep(context.Background())
	if err != nil || len(terminated) != 1 || terminated[0] != "hand-play" {
		t.Fatalf("stale play callbacks extended deadline: %v %v", terminated, err)
	}
	if len(terminal.expireCalls) != 1 || terminal.expireCalls[0].reason != ReasonPlayingTimeout {
		t.Fatalf("unexpected playing timeout: %+v", terminal.expireCalls)
	}
}

// A sweep may observe an overdue timer, then block reading the hand while a
// newer accepted bid callback extends the deadline. It must not expire the
// hand using the stale timer snapshot.
func TestSweepDoesNotExpireRenewedDeadline(t *testing.T) {
	clock := &testClock{now: time.Unix(1800000050, 0)}
	base := &testHands{snapshots: map[domain.HandID]domain.HandSnapshot{
		"hand-sweep": testHand("hand-sweep", "seat-1", domain.HandBidding, 9),
	}}
	terminal := &testTerminal{}
	s := newTestSupervisor(t, base, terminal, clock, &testIDs{prefix: "sweep-version"})
	if err := s.TrackBidding(context.Background(), "seat-1", "hand-sweep"); err != nil {
		t.Fatal(err)
	}
	gate := &gatedHandReader{
		delegate: base,
		entered:  make(chan struct{}),
		release:  make(chan struct{}),
	}
	s.hands = gate
	clock.Advance(6 * time.Second)
	completed := make(chan struct {
		ids []domain.HandID
		err error
	}, 1)
	go func() {
		ids, err := s.Sweep(context.Background())
		completed <- struct {
			ids []domain.HandID
			err error
		}{ids, err}
	}()
	<-gate.entered
	if _, err := s.AfterBid(context.Background(), "seat-1", "hand-sweep",
		bidResult(t, "hand-sweep", 3, livehand.PhaseBidding, false)); err != nil {
		t.Fatal(err)
	}
	close(gate.release)
	result := <-completed
	if result.err != nil || len(result.ids) != 0 || len(terminal.expireCalls) != 0 {
		t.Fatalf("renewed deadline expired by a stale sweep: %+v expireCalls=%+v", result, terminal.expireCalls)
	}
	clock.Advance(5 * time.Second)
	terminated, err := s.Sweep(context.Background())
	if err != nil || len(terminated) != 1 || terminated[0] != "hand-sweep" {
		t.Fatalf("renewed deadline never expired: %v %v", terminated, err)
	}
}

type gatedHandReader struct {
	delegate HandReader
	entered  chan struct{}
	release  chan struct{}
	once     sync.Once
}

func (g *gatedHandReader) LoadHand(ctx context.Context, id domain.HandID) (domain.HandSnapshot, error) {
	g.once.Do(func() {
		close(g.entered)
		<-g.release
	})
	return g.delegate.LoadHand(ctx, id)
}

// A low bid may commit at t4 while the dispatcher callback is held until
// after the original t5 deadline. The authoritative committed live view must
// be consulted before the old timer can initiate expiry.
type committedViewTerminal struct {
	*testTerminal
	version uint64
	phase   string
}

func (v *committedViewTerminal) GetHandPublicView(_ context.Context, _ domain.AccountID, id domain.HandID) (application.LiveHandView, error) {
	payload, err := json.Marshal(struct {
		HandID       string `json:"handId"`
		StateVersion uint64 `json:"stateVersion"`
		Phase        string `json:"phase"`
	}{HandID: string(id), StateVersion: v.version, Phase: v.phase})
	if err != nil {
		return application.LiveHandView{}, err
	}
	return application.LiveHandView{Version: v.version, Payload: payload}, nil
}
func TestCommittedBeforeCallbackDoesNotExpire(t *testing.T) {
	clock := &testClock{now: time.Unix(1800000070, 0)}
	hands := &testHands{snapshots: map[domain.HandID]domain.HandSnapshot{
		"committed-callback-gap": testHand("committed-callback-gap", "seat-1", domain.HandBidding, 9),
	}}
	terminal := &committedViewTerminal{testTerminal: &testTerminal{}, version: 1, phase: livehand.PhaseBidding}
	s := newTestSupervisor(t, hands, terminal, clock, &testIDs{prefix: "committed"})
	if err := s.TrackBidding(context.Background(), "seat-1", "committed-callback-gap"); err != nil {
		t.Fatal(err)
	}
	clock.Advance(4 * time.Second)
	// The real runtime publishes a committed candidate before it returns.
	terminal.version = 2 // deliberately delay AfterBid callback
	clock.Advance(2 * time.Second)
	terminated, err := s.Sweep(context.Background())
	if err != nil || len(terminated) != 0 || len(terminal.expireCalls) != 0 {
		t.Fatalf("committed bid expired under old deadline: %v %v %+v", terminated, err, terminal.expireCalls)
	}
	// Late response is now stale and must not reset the already-refreshed timer.
	if _, err := s.AfterBid(context.Background(), "seat-1", "committed-callback-gap",
		bidResult(t, "committed-callback-gap", 2, livehand.PhaseBidding, false)); err != nil {
		t.Fatal(err)
	}
	clock.Advance(4 * time.Second)
	terminated, err = s.Sweep(context.Background())
	if err != nil || len(terminated) != 1 || terminated[0] != "committed-callback-gap" {
		t.Fatalf("committed deadline never expired: %v %v", terminated, err)
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
