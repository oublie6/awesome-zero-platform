package gamecore_test

import (
	"bytes"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/oublie6/awesome-zero-platform/server/business/gamecore"
)

type memoryArchive struct {
	mu        sync.Mutex
	records   []gamecore.FinalRecord
	failCount int
}

func (a *memoryArchive) Archive(record gamecore.FinalRecord) error {
	if err := record.Validate(); err != nil {
		return err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.records = append(a.records, record)
	if a.failCount > 0 {
		a.failCount--
		return errors.New("archive unavailable")
	}
	return nil
}

func (a *memoryArchive) Calls() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return len(a.records)
}

func (a *memoryArchive) Records() []gamecore.FinalRecord {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]gamecore.FinalRecord(nil), a.records...)
}

type fakeLiveGame struct {
	descriptor gamecore.Descriptor
	id         gamecore.InstanceID

	mu           sync.Mutex
	version      uint64
	applyCalls   int
	abortCalls   int
	lastCommand  []byte
	publicState  []byte
	privateState map[uint8][]byte

	activeApply int32
	maxApply    int32
	delay       time.Duration
	entered     chan<- struct{}
	release     <-chan struct{}
}

func newFakeLiveGame(t *testing.T, id string, participants uint8) *fakeLiveGame {
	t.Helper()
	return &fakeLiveGame{
		descriptor:  testDescriptor(t, participants),
		id:          gamecore.InstanceID(id),
		publicState: []byte("public"),
		privateState: map[uint8][]byte{
			1: []byte("private-1"),
			2: []byte("private-2"),
			3: []byte("private-3"),
			4: []byte("private-4"),
		},
	}
}

func (g *fakeLiveGame) Descriptor() gamecore.Descriptor { return g.descriptor }
func (g *fakeLiveGame) InstanceID() gamecore.InstanceID { return g.id }

func (g *fakeLiveGame) Apply(command gamecore.Command) (gamecore.CommandOutcome, error) {
	active := atomic.AddInt32(&g.activeApply, 1)
	for {
		current := atomic.LoadInt32(&g.maxApply)
		if active <= current || atomic.CompareAndSwapInt32(&g.maxApply, current, active) {
			break
		}
	}
	defer atomic.AddInt32(&g.activeApply, -1)
	if g.entered != nil {
		g.entered <- struct{}{}
	}
	if g.release != nil {
		<-g.release
	}
	if g.delay > 0 {
		time.Sleep(g.delay)
	}

	g.mu.Lock()
	defer g.mu.Unlock()
	g.applyCalls++
	g.version++
	g.lastCommand = append([]byte(nil), command.Payload...)
	terminal := bytes.Equal(command.Payload, []byte("finish"))
	outcome := gamecore.CommandOutcome{
		Version:  g.version,
		Payload:  []byte(fmt.Sprintf("ack-%d", g.version)),
		Terminal: terminal,
	}
	if terminal {
		outcome.FinalPayload = []byte("completed-record")
	}
	return outcome, nil
}

func (g *fakeLiveGame) View(request gamecore.ViewRequest) (gamecore.GameView, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	payload := g.publicState
	if !request.PublicOnly {
		payload = g.privateState[request.ViewerPosition]
	}
	return gamecore.GameView{Version: g.version, Payload: payload}, nil
}

func (g *fakeLiveGame) Abort(reason string) (gamecore.AbortOutcome, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.abortCalls++
	g.version++
	return gamecore.AbortOutcome{Version: g.version, FinalPayload: []byte("aborted:" + reason)}, nil
}

func (g *fakeLiveGame) ApplyCalls() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.applyCalls
}

func (g *fakeLiveGame) AbortCalls() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.abortCalls
}

func (g *fakeLiveGame) LastCommand() []byte {
	g.mu.Lock()
	defer g.mu.Unlock()
	return append([]byte(nil), g.lastCommand...)
}

func TestLiveDirectoryRoutesViewsCopiesPayloadsAndSkipsArchiveForOrdinaryCommands(t *testing.T) {
	archive := &memoryArchive{}
	directory, err := gamecore.NewLiveDirectory(archive)
	if err != nil {
		t.Fatal(err)
	}
	game := newFakeLiveGame(t, "live-1", 4)
	if err := directory.Add(game.descriptor, game); err != nil {
		t.Fatal(err)
	}
	registeredDescriptor := game.descriptor
	mutatedDescriptor, err := gamecore.NewDescriptor("mutated", "mutated-rules-v1", "mutated-module-v1", "mutated-fair-v1", 1)
	if err != nil {
		t.Fatal(err)
	}
	game.descriptor = mutatedDescriptor
	defer func() { game.descriptor = registeredDescriptor }()
	if directory.Count() != 1 || !directory.Contains("live-1") {
		t.Fatal("live instance was not retained in memory")
	}
	if err := directory.Add(game.descriptor, game); !errors.Is(err, gamecore.ErrInstanceExists) {
		t.Fatalf("duplicate add: %v", err)
	}
	otherDescriptor, err := gamecore.NewDescriptor("other", "rules-v1", "module-v1", "fair-v1", 4)
	if err != nil {
		t.Fatal(err)
	}
	otherGame := newFakeLiveGame(t, "live-2", 4)
	if err := directory.Add(otherDescriptor, otherGame); !errors.Is(err, gamecore.ErrInvalidArgument) {
		t.Fatalf("descriptor mismatch: %v", err)
	}

	payload := []byte("move")
	outcome, err := directory.Apply("live-1", gamecore.Command{ActorPosition: 2, ExpectedVersion: 0, Payload: payload})
	if err != nil {
		t.Fatal(err)
	}
	payload[0] = 'X'
	if got := string(game.LastCommand()); got != "move" {
		t.Fatalf("command payload was not copied: %q", got)
	}
	outcome.Payload[0] = 'X'
	if archive.Calls() != 0 {
		t.Fatalf("ordinary command archived %d records", archive.Calls())
	}

	publicView, err := directory.View("live-1", gamecore.ViewRequest{PublicOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	privateView, err := directory.View("live-1", gamecore.ViewRequest{ViewerPosition: 2})
	if err != nil {
		t.Fatal(err)
	}
	if string(publicView.Payload) != "public" || string(privateView.Payload) != "private-2" {
		t.Fatalf("unexpected views public=%q private=%q", publicView.Payload, privateView.Payload)
	}
	publicView.Payload[0] = 'X'
	again, err := directory.View("live-1", gamecore.ViewRequest{PublicOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	if string(again.Payload) != "public" {
		t.Fatal("view payload accessor shared game-owned storage")
	}
	if _, err := directory.View("live-1", gamecore.ViewRequest{ViewerPosition: 5}); !errors.Is(err, gamecore.ErrInvalidArgument) {
		t.Fatalf("invalid viewer accepted: %v", err)
	}
}

func TestLiveDirectorySerializesOneInstanceWithoutGlobalGameplayLock(t *testing.T) {
	archive := &memoryArchive{}
	directory, err := gamecore.NewLiveDirectory(archive)
	if err != nil {
		t.Fatal(err)
	}
	game := newFakeLiveGame(t, "serial", 4)
	game.delay = 5 * time.Millisecond
	if err := directory.Add(game.descriptor, game); err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	for index := 0; index < 12; index++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := directory.Apply("serial", gamecore.Command{ActorPosition: 1, Payload: []byte("move")}); err != nil {
				t.Errorf("apply: %v", err)
			}
		}()
	}
	wg.Wait()
	if got := atomic.LoadInt32(&game.maxApply); got != 1 {
		t.Fatalf("same-instance commands overlapped: max=%d", got)
	}
	if game.ApplyCalls() != 12 {
		t.Fatalf("apply calls=%d", game.ApplyCalls())
	}

	entered := make(chan struct{}, 2)
	release := make(chan struct{})
	first := newFakeLiveGame(t, "parallel-1", 4)
	second := newFakeLiveGame(t, "parallel-2", 4)
	first.entered, first.release = entered, release
	second.entered, second.release = entered, release
	if err := directory.Add(first.descriptor, first); err != nil {
		t.Fatal(err)
	}
	if err := directory.Add(second.descriptor, second); err != nil {
		t.Fatal(err)
	}
	result := make(chan error, 2)
	go func() {
		_, err := directory.Apply("parallel-1", gamecore.Command{ActorPosition: 1, Payload: []byte("move")})
		result <- err
	}()
	go func() {
		_, err := directory.Apply("parallel-2", gamecore.Command{ActorPosition: 1, Payload: []byte("move")})
		result <- err
	}()
	for index := 0; index < 2; index++ {
		select {
		case <-entered:
		case <-time.After(time.Second):
			close(release)
			t.Fatal("different instances were forced through a global gameplay lock")
		}
	}
	close(release)
	for index := 0; index < 2; index++ {
		if err := <-result; err != nil {
			t.Fatal(err)
		}
	}
}

func TestTerminalArchiveFailureRetainsPendingRecordWithoutReapplying(t *testing.T) {
	archive := &memoryArchive{failCount: 1}
	directory, err := gamecore.NewLiveDirectory(archive)
	if err != nil {
		t.Fatal(err)
	}
	game := newFakeLiveGame(t, "terminal", 4)
	if err := directory.Add(game.descriptor, game); err != nil {
		t.Fatal(err)
	}
	outcome, err := directory.Apply("terminal", gamecore.Command{ActorPosition: 1, Payload: []byte("finish")})
	if !errors.Is(err, gamecore.ErrArchiveFailed) {
		t.Fatalf("terminal archive failure: %v", err)
	}
	if !outcome.Terminal || game.ApplyCalls() != 1 || !directory.Contains("terminal") {
		t.Fatalf("unexpected terminal failure state: outcome=%+v calls=%d contains=%v", outcome, game.ApplyCalls(), directory.Contains("terminal"))
	}
	pending, exists, err := directory.PendingFinalRecord("terminal")
	if err != nil || !exists {
		t.Fatalf("pending record: exists=%v err=%v", exists, err)
	}
	if pending.Status() != gamecore.FinalStatusCompleted || string(pending.Payload()) != "completed-record" {
		t.Fatalf("unexpected pending record: status=%s payload=%q", pending.Status(), pending.Payload())
	}
	if _, err := directory.Apply("terminal", gamecore.Command{ActorPosition: 1, Payload: []byte("finish")}); !errors.Is(err, gamecore.ErrFinalizationPending) {
		t.Fatalf("reapply while pending: %v", err)
	}
	if game.ApplyCalls() != 1 {
		t.Fatalf("terminal command reapplied: calls=%d", game.ApplyCalls())
	}
	retried, err := directory.RetryArchive("terminal")
	if err != nil {
		t.Fatal(err)
	}
	if retried.Digest() != pending.Digest() {
		t.Fatal("retry changed logical final record")
	}
	if directory.Contains("terminal") || directory.Count() != 0 {
		t.Fatal("successfully archived terminal instance remained active")
	}
	if archive.Calls() != 2 {
		t.Fatalf("archive attempts=%d", archive.Calls())
	}
	if _, err := directory.RetryArchive("terminal"); !errors.Is(err, gamecore.ErrInstanceNotFound) {
		t.Fatalf("repeated retry: %v", err)
	}
	for _, record := range archive.Records() {
		if record.Digest() != pending.Digest() {
			t.Fatal("archive retry delivered a different record")
		}
	}
}

func TestAbortArchivesOnceAndRemovesOnlyAfterSuccess(t *testing.T) {
	archive := &memoryArchive{failCount: 1}
	directory, err := gamecore.NewLiveDirectory(archive)
	if err != nil {
		t.Fatal(err)
	}
	game := newFakeLiveGame(t, "abort", 4)
	if err := directory.Add(game.descriptor, game); err != nil {
		t.Fatal(err)
	}
	record, err := directory.Abort("abort", "server shutdown")
	if !errors.Is(err, gamecore.ErrArchiveFailed) {
		t.Fatalf("abort archive failure: %v", err)
	}
	if record.Status() != gamecore.FinalStatusAborted || game.AbortCalls() != 1 || !directory.Contains("abort") {
		t.Fatalf("unexpected abort state: status=%s calls=%d contains=%v", record.Status(), game.AbortCalls(), directory.Contains("abort"))
	}
	if _, err := directory.Abort("abort", "again"); !errors.Is(err, gamecore.ErrFinalizationPending) {
		t.Fatalf("second abort: %v", err)
	}
	if game.AbortCalls() != 1 {
		t.Fatalf("abort reapplied: %d", game.AbortCalls())
	}
	if _, err := directory.RetryArchive("abort"); err != nil {
		t.Fatal(err)
	}
	if directory.Contains("abort") {
		t.Fatal("archived abort remained active")
	}
}

func TestConcurrentTerminalCommandsCannotUseRemovedEntry(t *testing.T) {
	archive := &memoryArchive{}
	directory, err := gamecore.NewLiveDirectory(archive)
	if err != nil {
		t.Fatal(err)
	}
	game := newFakeLiveGame(t, "terminal-race", 4)
	game.delay = 10 * time.Millisecond
	if err := directory.Add(game.descriptor, game); err != nil {
		t.Fatal(err)
	}

	results := make(chan error, 2)
	for index := 0; index < 2; index++ {
		go func() {
			_, err := directory.Apply("terminal-race", gamecore.Command{ActorPosition: 1, Payload: []byte("finish")})
			results <- err
		}()
	}
	var success, notFound int
	for index := 0; index < 2; index++ {
		err := <-results
		switch {
		case err == nil:
			success++
		case errors.Is(err, gamecore.ErrInstanceNotFound):
			notFound++
		default:
			t.Fatalf("unexpected error: %v", err)
		}
	}
	if success != 1 || notFound != 1 || game.ApplyCalls() != 1 || archive.Calls() != 1 {
		t.Fatalf("success=%d notFound=%d apply=%d archive=%d", success, notFound, game.ApplyCalls(), archive.Calls())
	}
}
