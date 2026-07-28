//go:build integration

package mysqlstore_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"hash"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	mysql "github.com/go-sql-driver/mysql"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/application"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/infrastructure/mysqlstore"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/infrastructure/protection"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/infrastructure/textnormalization"
	"github.com/oublie6/awesome-zero-platform/server/business/gamecore"
)

func TestApplicationPersistenceWithRealMySQL(t *testing.T) {
	if os.Getenv("APP_API_INTEGRATION") != "1" {
		t.Skip("APP_API_INTEGRATION is not enabled")
	}
	db := openIntegrationDB(t)
	prefix := fmt.Sprintf("g21-%d", time.Now().UnixNano())
	actor := domain.AccountID(prefix + "-actor")
	roomID := prefix + "-room"
	handID := domain.HandID(prefix + "-hand")
	cleanupIntegrationRows(t, db, prefix)
	defer cleanupIntegrationRows(t, db, prefix)

	store, err := mysqlstore.New(sqlDatabase{db})
	if err != nil {
		t.Fatal(err)
	}
	clock := &integrationClock{now: time.Now().UTC().Truncate(time.Millisecond)}
	ids := &integrationIDs{prefix: prefix}
	opener := &integrationOpener{}
	keyring, _ := protection.NewStaticKeyring("storage-key-1", map[string][]byte{"storage-key-1": bytes.Repeat([]byte{7}, 32)})
	protector, _ := protection.New(keyring)
	service, err := application.NewService(store, clock, ids, integrationSetup{}, integrationBeaconVerifier{}, integrationLiveRuntime{}, opener, protector, textnormalization.NFKC{}, application.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}

	create := integrationCommand(clock.Now(), application.CommandRoomCreate, domain.AggregateRoom, roomID, prefix+"-create", 1, 0)
	first, err := service.CreateRoom(context.Background(), actor, create)
	if err != nil || !first.Accepted {
		t.Fatalf("create=%#v err=%v", first, err)
	}
	duplicate, err := service.CreateRoom(context.Background(), actor, create)
	if err != nil || !duplicate.Accepted || !duplicate.Duplicate {
		t.Fatalf("duplicate=%#v err=%v", duplicate, err)
	}
	assertCount(t, db, "SELECT COUNT(*) FROM doudizhu_command_results WHERE command_id = ?", 1, create.CommandID)
	assertCount(t, db, "SELECT COUNT(*) FROM doudizhu_outbox_events WHERE causation_command_id = ?", 1, create.CommandID)

	hand, plaintext, expectedDigest := revealingIntegrationHand(t, handID, domain.RoomID(roomID), actor)
	if err := store.WithinCommand(context.Background(), actor, prefix+"-seed", func(ctx context.Context, tx application.Transaction) error {
		return tx.InsertHand(ctx, hand, clock.Now())
	}); err != nil {
		t.Fatal(err)
	}
	opener.plaintext = plaintext
	reveal := integrationCommand(clock.Now(), application.CommandHandRevealSubmit, domain.AggregateHand, string(handID), prefix+"-reveal", 1, hand.Version)
	revealResult, err := service.SubmitHandReveal(context.Background(), actor, reveal, application.SubmitRevealInput{Envelope: application.SecureEnvelope{KeyID: "reveal-key-1"}})
	if err != nil || !revealResult.Accepted {
		t.Fatalf("reveal=%#v err=%v", revealResult, err)
	}
	var ciphertext []byte
	var storedDigest []byte
	if err := db.QueryRow("SELECT ciphertext, contribution_digest FROM doudizhu_contribution_records WHERE hand_id = ?", handID).Scan(&ciphertext, &storedDigest); err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(ciphertext, []byte("integration phrase")) || bytes.Contains(ciphertext, bytes.Repeat([]byte{0x2a}, 32)) {
		t.Fatal("stored contribution contains plaintext")
	}
	if !bytes.Equal(storedDigest, expectedDigest[:]) {
		t.Fatalf("stored digest=%x want=%x", storedDigest, expectedDigest)
	}
	clock.Set(reveal.ExpiresAt.Add(time.Hour))
	replay, err := service.SubmitHandReveal(context.Background(), actor, reveal, application.SubmitRevealInput{Envelope: application.SecureEnvelope{KeyID: "reveal-key-1"}})
	if err != nil || !replay.Accepted || !replay.Duplicate || opener.calls.Load() != 1 {
		t.Fatalf("replay=%#v calls=%d err=%v", replay, opener.calls.Load(), err)
	}
	assertCount(t, db, "SELECT COUNT(*) FROM doudizhu_contribution_records WHERE hand_id = ?", 1, handID)
}

func TestConcurrentDuplicateCommandExecutesOnceWithRealMySQL(t *testing.T) {
	if os.Getenv("APP_API_INTEGRATION") != "1" {
		t.Skip("APP_API_INTEGRATION is not enabled")
	}
	db := openIntegrationDB(t)
	prefix := fmt.Sprintf("g21-dup-%d", time.Now().UnixNano())
	actor := domain.AccountID(prefix + "-actor")
	roomID := prefix + "-room"
	cleanupIntegrationRows(t, db, prefix)
	defer cleanupIntegrationRows(t, db, prefix)

	service, clock := newIntegrationService(t, db, prefix)
	command := integrationCommand(clock.Now(), application.CommandRoomCreate, domain.AggregateRoom, roomID, prefix+"-command", 1, 0)

	const workers = 16
	results := make([]application.CommandResult, workers)
	errorsSeen := make([]error, workers)
	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(workers)
	for index := 0; index < workers; index++ {
		go func(index int) {
			defer wg.Done()
			<-start
			results[index], errorsSeen[index] = service.CreateRoom(context.Background(), actor, command)
		}(index)
	}
	close(start)
	wg.Wait()

	originals := 0
	duplicates := 0
	for index := range results {
		if errorsSeen[index] != nil {
			t.Fatalf("worker %d error: %v", index, errorsSeen[index])
		}
		if !results[index].Accepted {
			t.Fatalf("worker %d result=%#v", index, results[index])
		}
		if results[index].Duplicate {
			duplicates++
		} else {
			originals++
		}
	}
	if originals != 1 || duplicates != workers-1 {
		t.Fatalf("originals=%d duplicates=%d", originals, duplicates)
	}
	assertCount(t, db, "SELECT COUNT(*) FROM doudizhu_rooms WHERE room_id = ?", 1, roomID)
	assertCount(t, db, "SELECT COUNT(*) FROM doudizhu_command_results WHERE command_id = ?", 1, command.CommandID)
	assertCount(t, db, "SELECT COUNT(*) FROM doudizhu_outbox_events WHERE causation_command_id = ?", 1, command.CommandID)
}

func TestConcurrentSameSequenceIsSerializedWithRealMySQL(t *testing.T) {
	if os.Getenv("APP_API_INTEGRATION") != "1" {
		t.Skip("APP_API_INTEGRATION is not enabled")
	}
	db := openIntegrationDB(t)
	prefix := fmt.Sprintf("g21-seq-%d", time.Now().UnixNano())
	actor := domain.AccountID(prefix + "-actor")
	roomID := prefix + "-room"
	cleanupIntegrationRows(t, db, prefix)
	defer cleanupIntegrationRows(t, db, prefix)

	service, clock := newIntegrationService(t, db, prefix)
	create := integrationCommand(clock.Now(), application.CommandRoomCreate, domain.AggregateRoom, roomID, prefix+"-create", 1, 0)
	created, err := service.CreateRoom(context.Background(), actor, create)
	if err != nil || !created.Accepted {
		t.Fatalf("create=%#v err=%v", created, err)
	}

	commands := []application.Command{
		integrationCommand(clock.Now(), application.CommandRoomReadySet, domain.AggregateRoom, roomID, prefix+"-ready", 2, 1),
		integrationCommand(clock.Now(), application.CommandRoomLeave, domain.AggregateRoom, roomID, prefix+"-leave", 2, 1),
	}
	results := make([]application.CommandResult, 2)
	errorsSeen := make([]error, 2)
	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-start
		results[0], errorsSeen[0] = service.SetRoomReady(context.Background(), actor, commands[0], application.SetReadyInput{Ready: true})
	}()
	go func() {
		defer wg.Done()
		<-start
		results[1], errorsSeen[1] = service.LeaveRoom(context.Background(), actor, commands[1])
	}()
	close(start)
	wg.Wait()

	accepted := 0
	sequenceRejected := 0
	for index := range results {
		if errorsSeen[index] != nil {
			t.Fatalf("worker %d error: %v", index, errorsSeen[index])
		}
		if results[index].Accepted {
			accepted++
		} else if results[index].Failure != nil && results[index].Failure.Code == application.CodeSequenceConflict {
			sequenceRejected++
		} else {
			t.Fatalf("worker %d result=%#v", index, results[index])
		}
	}
	if accepted != 1 || sequenceRejected != 1 {
		t.Fatalf("accepted=%d sequenceRejected=%d results=%#v", accepted, sequenceRejected, results)
	}
	var sequence uint64
	if err := db.QueryRow(`
SELECT last_sequence FROM doudizhu_client_sequences
WHERE aggregate_type = ? AND aggregate_id = ? AND actor_account_id = ?`,
		domain.AggregateRoom, roomID, actor).Scan(&sequence); err != nil {
		t.Fatal(err)
	}
	if sequence != 2 {
		t.Fatalf("sequence=%d want=2", sequence)
	}
	assertCount(t, db, "SELECT COUNT(*) FROM doudizhu_command_results WHERE aggregate_id = ?", 3, roomID)
	assertCount(t, db, "SELECT COUNT(*) FROM doudizhu_outbox_events WHERE aggregate_id = ?", 2, roomID)
}

func TestConcurrentRoomJoinsPreventLostUpdatesWithRealMySQL(t *testing.T) {
	if os.Getenv("APP_API_INTEGRATION") != "1" {
		t.Skip("APP_API_INTEGRATION is not enabled")
	}
	db := openIntegrationDB(t)
	prefix := fmt.Sprintf("g21-join-%d", time.Now().UnixNano())
	owner := domain.AccountID(prefix + "-owner")
	actors := []domain.AccountID{
		domain.AccountID(prefix + "-actor-2"),
		domain.AccountID(prefix + "-actor-3"),
	}
	roomID := prefix + "-room"
	cleanupIntegrationRows(t, db, prefix)
	defer cleanupIntegrationRows(t, db, prefix)

	service, clock := newIntegrationService(t, db, prefix)
	create := integrationCommand(clock.Now(), application.CommandRoomCreate, domain.AggregateRoom, roomID, prefix+"-create", 1, 0)
	created, err := service.CreateRoom(context.Background(), owner, create)
	if err != nil || !created.Accepted {
		t.Fatalf("create=%#v err=%v", created, err)
	}

	results := make([]application.CommandResult, 2)
	errorsSeen := make([]error, 2)
	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)
	for index := range actors {
		go func(index int) {
			defer wg.Done()
			<-start
			command := integrationCommand(clock.Now(), application.CommandRoomJoin, domain.AggregateRoom, roomID, fmt.Sprintf("%s-join-%d", prefix, index), 1, 1)
			results[index], errorsSeen[index] = service.JoinRoom(context.Background(), actors[index], command)
		}(index)
	}
	close(start)
	wg.Wait()

	winner := -1
	loser := -1
	for index := range results {
		if errorsSeen[index] != nil {
			t.Fatalf("worker %d error: %v", index, errorsSeen[index])
		}
		if results[index].Accepted {
			winner = index
		} else if results[index].Failure != nil && results[index].Failure.Code == application.CodeVersionConflict {
			loser = index
		} else {
			t.Fatalf("worker %d result=%#v", index, results[index])
		}
	}
	if winner < 0 || loser < 0 || winner == loser {
		t.Fatalf("winner=%d loser=%d results=%#v", winner, loser, results)
	}

	retry := integrationCommand(clock.Now(), application.CommandRoomJoin, domain.AggregateRoom, roomID, prefix+"-join-retry", 2, 2)
	retried, err := service.JoinRoom(context.Background(), actors[loser], retry)
	if err != nil || !retried.Accepted || retried.AggregateVersion != 3 {
		t.Fatalf("retry=%#v err=%v", retried, err)
	}

	var encoded []byte
	if err := db.QueryRow("SELECT snapshot_json FROM doudizhu_rooms WHERE room_id = ?", roomID).Scan(&encoded); err != nil {
		t.Fatal(err)
	}
	var snapshot domain.RoomSnapshot
	if err := json.Unmarshal(encoded, &snapshot); err != nil {
		t.Fatal(err)
	}
	seen := map[domain.AccountID]bool{}
	for _, seat := range snapshot.Seats {
		if seat.AccountID != "" {
			seen[seat.AccountID] = true
		}
	}
	if !seen[owner] || !seen[actors[0]] || !seen[actors[1]] || snapshot.Version != 3 {
		t.Fatalf("snapshot=%#v", snapshot)
	}
	assertCount(t, db, "SELECT COUNT(*) FROM doudizhu_outbox_events WHERE aggregate_id = ?", 3, roomID)
}

func newIntegrationService(t *testing.T, db *sql.DB, prefix string) (*application.Service, *integrationClock) {
	t.Helper()
	store, err := mysqlstore.New(sqlDatabase{db})
	if err != nil {
		t.Fatal(err)
	}
	clock := &integrationClock{now: time.Now().UTC().Truncate(time.Millisecond)}
	keyring, err := protection.NewStaticKeyring("storage-key-1", map[string][]byte{"storage-key-1": bytes.Repeat([]byte{7}, 32)})
	if err != nil {
		t.Fatal(err)
	}
	protector, err := protection.New(keyring)
	if err != nil {
		t.Fatal(err)
	}
	service, err := application.NewService(
		store, clock, &integrationIDs{prefix: prefix}, integrationSetup{},
		integrationBeaconVerifier{}, integrationLiveRuntime{},
		&integrationOpener{}, protector, textnormalization.NFKC{}, application.DefaultConfig(),
	)
	if err != nil {
		t.Fatal(err)
	}
	return service, clock
}

type sqlDatabase struct{ db *sql.DB }

func (d sqlDatabase) DB() *sql.DB { return d.db }

type integrationClock struct {
	mu  sync.RWMutex
	now time.Time
}

func (c *integrationClock) Now() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.now
}

func (c *integrationClock) Set(value time.Time) {
	c.mu.Lock()
	c.now = value
	c.mu.Unlock()
}

type integrationIDs struct {
	mu     sync.Mutex
	prefix string
	next   int
}

func (g *integrationIDs) NewID() (string, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.next++
	return fmt.Sprintf("%s-id-%d", g.prefix, g.next), nil
}

type integrationSetup struct{}

func (integrationSetup) PrepareHand(context.Context, domain.RoomSnapshot, domain.HandID) (application.HandSetup, error) {
	return application.HandSetup{}, fmt.Errorf("not used")
}

func (integrationSetup) ReleaseHand(context.Context, domain.HandID) error { return nil }

type integrationBeaconVerifier struct{}

func (integrationBeaconVerifier) Verify(_ context.Context, _ domain.BeaconPlan, value domain.BeaconValue) (domain.BeaconValue, error) {
	return value, nil
}

type integrationLiveRuntime struct{}

func (integrationLiveRuntime) Start(context.Context, domain.HandSnapshot) error {
	return fmt.Errorf("not used")
}
func (integrationLiveRuntime) RollbackStart(context.Context, domain.HandID) error   { return nil }
func (integrationLiveRuntime) ReleasePrepared(context.Context, domain.HandID) error { return nil }
func (integrationLiveRuntime) PublicView(context.Context, domain.HandID, domain.AccountID) (application.LiveHandView, error) {
	return application.LiveHandView{}, fmt.Errorf("not used")
}
func (integrationLiveRuntime) PrivateView(context.Context, domain.HandID, domain.AccountID) (application.LiveHandView, error) {
	return application.LiveHandView{}, fmt.Errorf("not used")
}
func (integrationLiveRuntime) Abort(context.Context, domain.HandID, string) (gamecore.FinalRecord, error) {
	return gamecore.FinalRecord{}, fmt.Errorf("not used")
}
func (integrationLiveRuntime) RetryArchive(context.Context, domain.HandID) (gamecore.FinalRecord, error) {
	return gamecore.FinalRecord{}, fmt.Errorf("not used")
}
func (integrationLiveRuntime) Contains(domain.HandID) bool { return false }

type integrationOpener struct {
	plaintext []byte
	calls     atomic.Int32
}

func (o *integrationOpener) Open(context.Context, application.SecureEnvelope, []byte, application.RevealKeyContext) ([]byte, error) {
	o.calls.Add(1)
	return append([]byte(nil), o.plaintext...), nil
}

func integrationCommand(now time.Time, name string, aggregateType domain.AggregateType, aggregateID, commandID string, sequence, version uint64) application.Command {
	return application.Command{Version: application.CommandProtocolV1, Name: name, CommandID: commandID, AggregateType: aggregateType, AggregateID: aggregateID, ClientSeq: sequence, ExpectedVersion: version, IssuedAt: now, ExpiresAt: now.Add(time.Minute)}
}

func revealingIntegrationHand(t *testing.T, handID domain.HandID, roomID domain.RoomID, actor domain.AccountID) (domain.HandSnapshot, []byte, domain.ContributionDigest) {
	t.Helper()
	seats := [3]domain.HandSeat{{Seat: 1, AccountID: actor}, {Seat: 2, AccountID: "integration-actor-2"}, {Seat: 3, AccountID: "integration-actor-3"}}
	var server domain.ServerCommitment
	server[0] = 1
	hand, _, err := domain.NewHand(handID, roomID, seats, server, "reveal-key-1", domain.BeaconPlan{Provider: "beacon", Round: "round"})
	if err != nil {
		t.Fatal(err)
	}
	random := bytes.Repeat([]byte{0x2a}, 32)
	phrase := "integration phrase"
	phraseHash := sha256.Sum256([]byte(phrase))
	digest := integrationContributionDigest(handID, domain.SeatOne, random, phraseHash)
	digests := []domain.ContributionDigest{digest, {2}, {3}}
	for index, current := range seats {
		commitment := domain.ComputeClientCommitment(handID, current.Seat, digests[index])
		if _, err := hand.SubmitCommit(current.AccountID, commitment, hand.Snapshot().Version); err != nil {
			t.Fatal(err)
		}
	}
	payload, _ := json.Marshal(map[string]any{"v": application.RevealPlaintextV1, "handId": handID, "seat": 1, "secureRandom": base64.RawURLEncoding.EncodeToString(random), "phrase": phrase, "normalization": "NFKC-v1"})
	return hand.Snapshot(), payload, digest
}

func integrationContributionDigest(handID domain.HandID, seat domain.Seat, random []byte, phraseHash [32]byte) domain.ContributionDigest {
	h := sha256.New()
	h.Write([]byte(application.ContributionV1))
	h.Write([]byte{0})
	writeIntegrationLength(h, []byte(handID))
	h.Write([]byte{byte(seat)})
	h.Write(random)
	h.Write(phraseHash[:])
	var result domain.ContributionDigest
	copy(result[:], h.Sum(nil))
	return result
}
func writeIntegrationLength(dst hash.Hash, value []byte) {
	var size [4]byte
	binary.BigEndian.PutUint32(size[:], uint32(len(value)))
	dst.Write(size[:])
	dst.Write(value)
}

func openIntegrationDB(t *testing.T) *sql.DB {
	t.Helper()
	config := mysql.Config{User: envOr("APP_MYSQL_USER", "app_local"), Passwd: envOr("APP_MYSQL_PASSWORD", "local-dev-only-mysql-password"), Net: "tcp", Addr: envOr("APP_MYSQL_ADDR", "127.0.0.1:3306"), DBName: envOr("APP_MYSQL_DATABASE", "awesome_zero_platform"), ParseTime: true, Loc: time.UTC, AllowNativePasswords: true, Params: map[string]string{"charset": "utf8mb4", "time_zone": "'+00:00'"}}
	db, err := sql.Open("mysql", config.FormatDSN())
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(24)
	db.SetMaxIdleConns(24)
	if err := db.Ping(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}
func envOr(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
func cleanupIntegrationRows(t *testing.T, db *sql.DB, prefix string) {
	t.Helper()
	like := prefix + "%"
	for _, statement := range []string{
		"DELETE FROM doudizhu_outbox_events WHERE event_id LIKE ?",
		"DELETE FROM doudizhu_contribution_records WHERE record_id LIKE ? OR hand_id LIKE ?",
		"DELETE FROM doudizhu_command_results WHERE command_id LIKE ? OR aggregate_id LIKE ?",
		"DELETE FROM doudizhu_client_sequences WHERE aggregate_id LIKE ?",
		"DELETE FROM doudizhu_hands WHERE hand_id LIKE ? OR room_id LIKE ?",
		"DELETE FROM doudizhu_rooms WHERE room_id LIKE ?",
	} {
		count := strings.Count(statement, "?")
		args := make([]any, count)
		for i := range args {
			args[i] = like
		}
		if _, err := db.Exec(statement, args...); err != nil {
			t.Fatalf("cleanup %q: %v", statement, err)
		}
	}
}
func assertCount(t *testing.T, db *sql.DB, query string, expected int, args ...any) {
	t.Helper()
	var count int
	if err := db.QueryRow(query, args...).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != expected {
		t.Fatalf("count=%d want=%d for %s", count, expected, query)
	}
}

var _ = url.QueryEscape
