package application

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
)

func TestDuplicateReturnsOriginalBeforeExpiry(t *testing.T) {
	service, store, clock := newCommandTestService(t)
	command := testCommand(clock.Now(), CommandRoomCreate, domain.AggregateRoom, "room-1", "create-1", 1, 0)
	first, err := service.CreateRoom(context.Background(), "actor-1", command)
	if err != nil || !first.Accepted || first.Duplicate {
		t.Fatalf("first=%#v err=%v", first, err)
	}
	outboxCount := len(store.state.outbox)
	clock.Set(command.ExpiresAt.Add(time.Hour))
	duplicate, err := service.CreateRoom(context.Background(), "actor-1", command)
	if err != nil || !duplicate.Accepted || !duplicate.Duplicate {
		t.Fatalf("duplicate=%#v err=%v", duplicate, err)
	}
	if duplicate.AggregateVersion != first.AggregateVersion || len(store.state.outbox) != outboxCount {
		t.Fatalf("duplicate changed state: first=%#v duplicate=%#v outbox=%d", first, duplicate, len(store.state.outbox))
	}
}

func TestCommandIDBindsPayload(t *testing.T) {
	service, _, clock := newCommandTestService(t)
	create := testCommand(clock.Now(), CommandRoomCreate, domain.AggregateRoom, "room-1", "create-1", 1, 0)
	if result, err := service.CreateRoom(context.Background(), "actor-1", create); err != nil || !result.Accepted {
		t.Fatalf("create=%#v err=%v", result, err)
	}
	ready := testCommand(clock.Now(), CommandRoomReadySet, domain.AggregateRoom, "room-1", "ready-1", 2, 1)
	first, err := service.SetRoomReady(context.Background(), "actor-1", ready, SetReadyInput{Ready: true})
	if err != nil || !first.Accepted {
		t.Fatalf("ready=%#v err=%v", first, err)
	}
	changed, err := service.SetRoomReady(context.Background(), "actor-1", ready, SetReadyInput{Ready: false})
	if err != nil {
		t.Fatal(err)
	}
	if changed.Accepted || changed.Failure == nil || changed.Failure.Code != CodeConflict {
		t.Fatalf("changed payload result=%#v", changed)
	}
}

func TestConcurrentSameSequenceAdmitsOnlyOne(t *testing.T) {
	service, store, clock := newCommandTestService(t)
	create := testCommand(clock.Now(), CommandRoomCreate, domain.AggregateRoom, "room-1", "create-1", 1, 0)
	if result, err := service.CreateRoom(context.Background(), "actor-1", create); err != nil || !result.Accepted {
		t.Fatalf("create=%#v err=%v", result, err)
	}
	ready := testCommand(clock.Now(), CommandRoomReadySet, domain.AggregateRoom, "room-1", "ready-2", 2, 1)
	leave := testCommand(clock.Now(), CommandRoomLeave, domain.AggregateRoom, "room-1", "leave-2", 2, 1)

	start := make(chan struct{})
	results := make(chan CommandResult, 2)
	errorsSeen := make(chan error, 2)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-start
		result, err := service.SetRoomReady(context.Background(), "actor-1", ready, SetReadyInput{Ready: true})
		results <- result
		errorsSeen <- err
	}()
	go func() {
		defer wg.Done()
		<-start
		result, err := service.LeaveRoom(context.Background(), "actor-1", leave)
		results <- result
		errorsSeen <- err
	}()
	close(start)
	wg.Wait()
	close(results)
	close(errorsSeen)
	for err := range errorsSeen {
		if err != nil {
			t.Fatal(err)
		}
	}
	accepted, rejected := 0, 0
	for result := range results {
		if result.Accepted {
			accepted++
		} else if result.Failure != nil && result.Failure.Code == CodeSequenceConflict {
			rejected++
		} else {
			t.Fatalf("unexpected result=%#v", result)
		}
	}
	if accepted != 1 || rejected != 1 {
		t.Fatalf("accepted=%d rejected=%d", accepted, rejected)
	}
	if got := store.state.sequences[testSequenceKey(domain.AggregateRoom, "room-1", "actor-1")]; got != 2 {
		t.Fatalf("stored sequence=%d", got)
	}
}

func TestBusinessRejectionConsumesSequenceAndReplays(t *testing.T) {
	service, store, clock := newCommandTestService(t)
	create := testCommand(clock.Now(), CommandRoomCreate, domain.AggregateRoom, "room-1", "create-1", 1, 0)
	if result, err := service.CreateRoom(context.Background(), "actor-1", create); err != nil || !result.Accepted {
		t.Fatalf("create=%#v err=%v", result, err)
	}
	join := testCommand(clock.Now(), CommandRoomJoin, domain.AggregateRoom, "room-1", "join-2", 2, 1)
	first, err := service.JoinRoom(context.Background(), "actor-1", join)
	if err != nil || first.Accepted || first.Failure == nil {
		t.Fatalf("first=%#v err=%v", first, err)
	}
	if got := store.state.sequences[testSequenceKey(domain.AggregateRoom, "room-1", "actor-1")]; got != 2 {
		t.Fatalf("stored sequence=%d", got)
	}
	replay, err := service.JoinRoom(context.Background(), "actor-1", join)
	if err != nil || replay.Accepted || !replay.Duplicate || replay.Failure.Code != first.Failure.Code {
		t.Fatalf("replay=%#v err=%v", replay, err)
	}
}

func newCommandTestService(t *testing.T) (*Service, *testStore, *testClock) {
	t.Helper()
	store := newTestStore()
	clock := &testClock{now: time.Now().UTC().Truncate(time.Millisecond)}
	service, err := NewService(store, clock, &testIDs{}, testSetup{}, testOpener{}, testProtector{}, testNormalizer{}, DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	return service, store, clock
}

func testCommand(now time.Time, name string, aggregateType domain.AggregateType, aggregateID, commandID string, sequence, version uint64) Command {
	return Command{Version: CommandProtocolV1, Name: name, CommandID: commandID, AggregateType: aggregateType, AggregateID: aggregateID, ClientSeq: sequence, ExpectedVersion: version, IssuedAt: now, ExpiresAt: now.Add(time.Minute)}
}

type testClock struct {
	mu  sync.RWMutex
	now time.Time
}

func (c *testClock) Now() time.Time      { c.mu.RLock(); defer c.mu.RUnlock(); return c.now }
func (c *testClock) Set(value time.Time) { c.mu.Lock(); c.now = value; c.mu.Unlock() }

type testIDs struct {
	mu sync.Mutex
	n  int
}

func (g *testIDs) NewID() (string, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.n++
	return fmt.Sprintf("id-%d", g.n), nil
}

type testSetup struct{}

func (testSetup) PrepareHand(context.Context, domain.RoomSnapshot, domain.HandID) (HandSetup, error) {
	return HandSetup{}, errors.New("not used")
}

type testOpener struct{}

func (testOpener) Open(context.Context, SecureEnvelope, []byte, RevealKeyContext) ([]byte, error) {
	return nil, errors.New("not used")
}

type testProtector struct{}

func (testProtector) Seal(context.Context, []byte, []byte) (ProtectedPayload, error) {
	return ProtectedPayload{}, errors.New("not used")
}

type testNormalizer struct{}

func (testNormalizer) Normalize(value string) (string, error) { return value, nil }

type testState struct {
	commands  map[string]StoredCommandResult
	sequences map[string]uint64
	rooms     map[string]domain.RoomSnapshot
	hands     map[string]domain.HandSnapshot
	records   map[string]ProtectedContributionRecord
	outbox    []OutboxEvent
}

type testStore struct {
	mu    sync.Mutex
	state testState
}

func newTestStore() *testStore {
	return &testStore{state: testState{commands: map[string]StoredCommandResult{}, sequences: map[string]uint64{}, rooms: map[string]domain.RoomSnapshot{}, hands: map[string]domain.HandSnapshot{}, records: map[string]ProtectedContributionRecord{}}}
}

func (s *testStore) WithinCommand(ctx context.Context, _ domain.AccountID, _ string, fn func(context.Context, Transaction) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	copyState := s.state.clone()
	if err := fn(ctx, &testTx{state: &copyState}); err != nil {
		return err
	}
	s.state = copyState
	return nil
}

func (s testState) clone() testState {
	copyState := testState{commands: map[string]StoredCommandResult{}, sequences: map[string]uint64{}, rooms: map[string]domain.RoomSnapshot{}, hands: map[string]domain.HandSnapshot{}, records: map[string]ProtectedContributionRecord{}, outbox: append([]OutboxEvent(nil), s.outbox...)}
	for key, value := range s.commands {
		copyState.commands[key] = StoredCommandResult{Command: value.Command, Result: cloneResult(value.Result)}
	}
	for key, value := range s.sequences {
		copyState.sequences[key] = value
	}
	for key, value := range s.rooms {
		copyState.rooms[key] = value
	}
	for key, value := range s.hands {
		copyState.hands[key] = value
	}
	for key, value := range s.records {
		copyState.records[key] = value
	}
	return copyState
}

type testTx struct{ state *testState }

func testCommandKey(actor domain.AccountID, commandID string) string {
	return string(actor) + "|" + commandID
}
func testSequenceKey(aggregateType domain.AggregateType, aggregateID string, actor domain.AccountID) string {
	return string(aggregateType) + "|" + aggregateID + "|" + string(actor)
}

func (t *testTx) ClaimCommand(_ context.Context, actor domain.AccountID, command Command, _ time.Time) (StoredCommandResult, bool, error) {
	key := testCommandKey(actor, command.CommandID)
	if stored, ok := t.state.commands[key]; ok {
		return stored, stored.Result.Version != "", nil
	}
	stored := StoredCommandResult{Command: command}
	t.state.commands[key] = stored
	return stored, false, nil
}
func (t *testTx) CompleteCommand(_ context.Context, actor domain.AccountID, commandID string, result CommandResult, _ time.Time) error {
	key := testCommandKey(actor, commandID)
	stored := t.state.commands[key]
	stored.Result = cloneResult(result)
	t.state.commands[key] = stored
	return nil
}
func (t *testTx) LockClientSequence(_ context.Context, aggregateType domain.AggregateType, aggregateID string, actor domain.AccountID) (uint64, error) {
	return t.state.sequences[testSequenceKey(aggregateType, aggregateID, actor)], nil
}
func (t *testTx) SaveClientSequence(_ context.Context, aggregateType domain.AggregateType, aggregateID string, actor domain.AccountID, sequence uint64, _ time.Time) error {
	key := testSequenceKey(aggregateType, aggregateID, actor)
	if sequence <= t.state.sequences[key] {
		return ErrSequenceConflict
	}
	t.state.sequences[key] = sequence
	return nil
}
func (t *testTx) InsertRoom(_ context.Context, snapshot domain.RoomSnapshot, _ time.Time) error {
	if _, exists := t.state.rooms[string(snapshot.ID)]; exists {
		return ErrAlreadyExists
	}
	t.state.rooms[string(snapshot.ID)] = snapshot
	return nil
}
func (t *testTx) LoadRoomForUpdate(_ context.Context, id domain.RoomID) (domain.RoomSnapshot, error) {
	snapshot, ok := t.state.rooms[string(id)]
	if !ok {
		return domain.RoomSnapshot{}, ErrNotFound
	}
	return snapshot, nil
}
func (t *testTx) UpdateRoom(_ context.Context, snapshot domain.RoomSnapshot, previousVersion uint64, _ time.Time) error {
	current, ok := t.state.rooms[string(snapshot.ID)]
	if !ok {
		return ErrNotFound
	}
	if current.Version != previousVersion {
		return ErrOptimisticConflict
	}
	t.state.rooms[string(snapshot.ID)] = snapshot
	return nil
}
func (t *testTx) InsertHand(_ context.Context, snapshot domain.HandSnapshot, _ time.Time) error {
	t.state.hands[string(snapshot.ID)] = snapshot
	return nil
}
func (t *testTx) LoadHandForUpdate(_ context.Context, id domain.HandID) (domain.HandSnapshot, error) {
	snapshot, ok := t.state.hands[string(id)]
	if !ok {
		return domain.HandSnapshot{}, ErrNotFound
	}
	return snapshot, nil
}
func (t *testTx) UpdateHand(_ context.Context, snapshot domain.HandSnapshot, previousVersion uint64, _ time.Time) error {
	current, ok := t.state.hands[string(snapshot.ID)]
	if !ok {
		return ErrNotFound
	}
	if current.Version != previousVersion {
		return ErrOptimisticConflict
	}
	t.state.hands[string(snapshot.ID)] = snapshot
	return nil
}
func (t *testTx) InsertContributionRecord(_ context.Context, record ProtectedContributionRecord) error {
	t.state.records[record.RecordID] = record
	return nil
}
func (t *testTx) AppendOutbox(_ context.Context, events []OutboxEvent) error {
	t.state.outbox = append(t.state.outbox, events...)
	return nil
}

var _ = sha256.Sum256
