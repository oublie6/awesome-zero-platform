package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/application"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/application/lifecycle"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/application/turntimer"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/bidding"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/carddeck"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/livehand"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/playing"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/infrastructure/mysqlstore"
)

type dqStoreFixture struct {
	turn    mysqlstore.TurnDeadline
	now     time.Time
	missing bool
	marked  bool
}

func (x *dqStoreFixture) GetTurnDeadline(_ context.Context, id domain.HandID) (mysqlstore.TurnDeadline, time.Time, error) {
	if x.missing {
		return mysqlstore.TurnDeadline{}, x.now, errors.New("unavailable")
	}
	return x.turn, x.now, nil
}
func (x *dqStoreFixture) PendingTurnDeliveries(_ context.Context, limit int) ([]mysqlstore.TurnDeadline, error) {
	if limit != 64 {
		return nil, errors.New("invalid bound")
	}
	return []mysqlstore.TurnDeadline{x.turn}, nil
}
func (x *dqStoreFixture) MarkTurnDelivery(_ context.Context, item mysqlstore.TurnDeadline, ok bool) error {
	if item != x.turn || !ok {
		return errors.New("unexpected delivery")
	}
	x.marked = true
	return nil
}

type dqReaderFixture struct{ phase domain.HandPhase }

func (x *dqReaderFixture) LoadHand(_ context.Context, id domain.HandID) (domain.HandSnapshot, error) {
	if id != "hand-1" {
		return domain.HandSnapshot{}, errors.New("wrong hand")
	}
	return domain.HandSnapshot{Phase: x.phase,
		Seats: [3]domain.HandSeat{{Seat: 1, AccountID: "one"}, {Seat: 2, AccountID: "two"}, {Seat: 3, AccountID: "three"}}}, nil
}

type dqGameFixture struct {
	view     livehand.PrivateView
	bidScore int
	playCard string
	passed   int
}

func (x *dqGameFixture) GetHandPrivateView(_ context.Context, a domain.AccountID, id domain.HandID) (application.LiveHandView, error) {
	if a != "one" || id != "hand-1" {
		return application.LiveHandView{}, errors.New("actor mismatch")
	}
	body, err := json.Marshal(x.view)
	return application.LiveHandView{Version: x.view.Public.StateVersion, Payload: body}, err
}
func (x *dqGameFixture) SubmitLiveHandBidWithID(_ context.Context, a domain.AccountID, id domain.HandID, command string, version uint64, score bidding.Score) (application.LiveHandCommandResult, error) {
	if command == "" || version != 3 || a != "one" {
		return application.LiveHandCommandResult{}, errors.New("invalid bid")
	}
	x.bidScore = int(score)
	return application.LiveHandCommandResult{Version: 4}, nil
}
func (x *dqGameFixture) SubmitLiveHandPlayWithID(_ context.Context, a domain.AccountID, id domain.HandID, command string, version uint64, cards []string) (application.LiveHandCommandResult, error) {
	if command == "" || version != 3 || a != "one" || len(cards) != 1 {
		return application.LiveHandCommandResult{}, errors.New("invalid play")
	}
	x.playCard = cards[0]
	return application.LiveHandCommandResult{Version: 4}, nil
}
func (x *dqGameFixture) SubmitLiveHandPassWithID(_ context.Context, a domain.AccountID, id domain.HandID, command string, version uint64) (application.LiveHandCommandResult, error) {
	if command == "" || version != 3 || a != "one" {
		return application.LiveHandCommandResult{}, errors.New("invalid pass")
	}
	x.passed++
	return application.LiveHandCommandResult{Version: 4}, nil
}

type dqLifecycleFixture struct{ bids, plays int }

func (x *dqLifecycleFixture) AfterBid(_ context.Context, a domain.AccountID, id domain.HandID, result application.LiveHandCommandResult) (lifecycle.Outcome, error) {
	x.bids++
	return lifecycle.Outcome{}, nil
}
func (x *dqLifecycleFixture) AfterPlay(_ context.Context, a domain.AccountID, id domain.HandID, result application.LiveHandCommandResult) error {
	x.plays++
	return nil
}

type dqPublisherFixture struct{ enqueued int }

func (x *dqPublisherFixture) At(_ context.Context, body []byte, at time.Time) (string, error) {
	t, err := turntimer.Decode(body)
	if err != nil {
		return "", err
	}
	if t.Deadline() != at {
		return "", errors.New("deadline mismatch")
	}
	x.enqueued++
	return "job-1", nil
}
func (x *dqPublisherFixture) Close() error { return nil }

type dqIDFixture struct{}

func (dqIDFixture) NewID() (string, error) { return "delivery-1", nil }

func dqWorkerFixture(phase string) (*turnDqWorker, *dqStoreFixture, *dqGameFixture, *dqLifecycleFixture) {
	deadline := time.UnixMilli(2_000_000_000_000).UTC()
	store := &dqStoreFixture{turn: mysqlstore.TurnDeadline{HandID: "hand-1", Version: 3, Phase: phase, Seat: 1, Deadline: deadline}, now: deadline.Add(time.Second)}
	game := &dqGameFixture{view: livehand.PrivateView{Version: livehand.PrivateViewVersion, Position: 1, Cards: []string{"H5", "D3", "YJ"},
		Public: livehand.PublicView{Version: livehand.PublicViewVersion, HandID: "hand-1", Phase: phase, StateVersion: 3,
			Bidding: bidding.Snapshot{CurrentBidder: 1}, PlayingSeat: 1,
			Playing: &playing.Snapshot{Version: playing.StateVersion, CurrentSeat: 1}}}}
	life := &dqLifecycleFixture{}
	readerPhase := domain.HandBidding
	if phase == livehand.PhasePlaying {
		readerPhase = domain.HandPlaying
	}
	worker := &turnDqWorker{store: store, reader: &dqReaderFixture{phase: readerPhase}, game: game, lifecycle: life, ids: dqIDFixture{}, publisher: &dqPublisherFixture{}}
	return worker, store, game, life
}
func dqTicket(store *dqStoreFixture) turntimer.Ticket {
	t := store.turn
	return turntimer.Ticket{HandID: string(t.HandID), LiveVersion: t.Version, Seat: t.Seat, DeadlineMillis: t.Deadline.UnixMilli(), DeliveryID: "job-1"}
}
func TestTurnDQTimeoutBiddingUsesExistingCommittedBid(t *testing.T) {
	w, store, game, life := dqWorkerFixture(livehand.PhaseBidding)
	if err := w.process(context.Background(), dqTicket(store)); err != nil {
		t.Fatal(err)
	}
	if game.bidScore != 0 || life.bids != 1 {
		t.Fatalf("bid=%d callbacks=%d", game.bidScore, life.bids)
	}
}
func TestTurnDQTimeoutCanPassOrLeadMinimumSingle(t *testing.T) {
	w, store, game, life := dqWorkerFixture(livehand.PhasePlaying)
	game.view.Public.Playing.LeadingCards = []carddeck.Card{0}
	if err := w.process(context.Background(), dqTicket(store)); err != nil {
		t.Fatal(err)
	}
	if game.passed != 1 || life.plays != 1 {
		t.Fatalf("pass=%d after=%d", game.passed, life.plays)
	}
	game.view.Public.Playing.LeadingCards = nil
	if err := w.process(context.Background(), dqTicket(store)); err != nil {
		t.Fatal(err)
	}
	if game.playCard != "D3" || life.plays != 2 {
		t.Fatalf("lead=%s after=%d", game.playCard, life.plays)
	}
}
func TestTurnDQIgnoresPrematureAndStaleTickets(t *testing.T) {
	w, store, game, life := dqWorkerFixture(livehand.PhasePlaying)
	store.now = store.turn.Deadline.Add(-time.Millisecond)
	if err := w.process(context.Background(), dqTicket(store)); err != nil {
		t.Fatal(err)
	}
	store.now = store.turn.Deadline.Add(time.Second)
	stale := dqTicket(store)
	stale.LiveVersion--
	if err := w.process(context.Background(), stale); err != nil {
		t.Fatal(err)
	}
	if game.playCard != "" || life.plays != 0 {
		t.Fatal("stale timeout altered game")
	}
}
func TestTurnDQBoundedDispatch(t *testing.T) {
	w, store, _, _ := dqWorkerFixture(livehand.PhasePlaying)
	publisher := &dqPublisherFixture{}
	w.publisher = publisher
	if err := w.dispatch(context.Background(), 64); err != nil {
		t.Fatal(err)
	}
	if publisher.enqueued != 1 || !store.marked {
		t.Fatal("delivery was not recorded")
	}
}
func TestDeterministicTimeoutCommandID(t *testing.T) {
	_, store, _, _ := dqWorkerFixture(livehand.PhasePlaying)
	a := turnTimeoutCommandID(store.turn)
	b := turnTimeoutCommandID(store.turn)
	if a != b || len(a) > 128 {
		t.Fatalf("unstable timeout ID")
	}
}

type dqBatchStoreFixture struct {
	rows    []mysqlstore.TurnDeadline
	seen    bool
	results map[domain.HandID]bool
}

func (x *dqBatchStoreFixture) GetTurnDeadline(context.Context, domain.HandID) (mysqlstore.TurnDeadline, time.Time, error) {
	return mysqlstore.TurnDeadline{}, time.Time{}, errors.New("not used by dispatch")
}
func (x *dqBatchStoreFixture) PendingTurnDeliveries(_ context.Context, limit int) ([]mysqlstore.TurnDeadline, error) {
	if limit != 64 {
		return nil, errors.New("invalid batch limit")
	}
	if x.seen {
		return nil, nil
	}
	x.seen = true
	return append([]mysqlstore.TurnDeadline(nil), x.rows...), nil
}
func (x *dqBatchStoreFixture) MarkTurnDelivery(_ context.Context, item mysqlstore.TurnDeadline, ok bool) error {
	if x.results == nil {
		x.results = map[domain.HandID]bool{}
	}
	x.results[item.HandID] = ok
	return nil
}

type dqPartialFailurePublisher struct{ delivered []domain.HandID }

func (x *dqPartialFailurePublisher) At(_ context.Context, body []byte, deadline time.Time) (string, error) {
	ticket, err := turntimer.Decode(body)
	if err != nil || !ticket.Deadline().Equal(deadline) {
		return "", errors.New("invalid queue ticket")
	}
	if ticket.HandID == "broken" {
		return "", errors.New("one beanstalkd publisher unavailable")
	}
	x.delivered = append(x.delivered, domain.HandID(ticket.HandID))
	return "accepted", nil
}
func (*dqPartialFailurePublisher) Close() error { return nil }

func TestTurnDQDispatchContinuesAfterOnePublisherFailure(t *testing.T) {
	deadline := time.UnixMilli(2_000_000_000_000).UTC()
	store := &dqBatchStoreFixture{rows: []mysqlstore.TurnDeadline{
		{HandID: "broken", Version: 1, Phase: livehand.PhaseBidding, Seat: 1, Deadline: deadline},
		{HandID: "healthy", Version: 1, Phase: livehand.PhaseBidding, Seat: 2, Deadline: deadline},
	}}
	publisher := &dqPartialFailurePublisher{}
	worker := &turnDqWorker{store: store, ids: dqIDFixture{}, publisher: publisher}
	if err := worker.dispatch(context.Background(), 64); err == nil {
		t.Fatal("an enqueue failure must be surfaced to repair/observability")
	}
	if !store.seen || store.results["broken"] || !store.results["healthy"] ||
		len(publisher.delivered) != 1 || publisher.delivered[0] != "healthy" {
		t.Fatalf("failed hand starved healthy delivery: results=%v published=%v", store.results, publisher.delivered)
	}
}
