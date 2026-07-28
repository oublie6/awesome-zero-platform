package mysqlstore

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/application"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
)

func TestWithinCommandClaimsAndCommits(t *testing.T) {
	command := application.Command{
		Version: application.CommandProtocolV1, Name: application.CommandRoomCreate,
		CommandID: "command", AggregateType: domain.AggregateRoom, AggregateID: "room-1",
		ClientSeq: 1, IssuedAt: time.Now().UTC().Truncate(time.Millisecond),
	}
	command.ExpiresAt = command.IssuedAt.Add(time.Minute)
	commandJSON, _ := json.Marshal(command)
	result := application.CommandResult{
		Version: application.CommandResultV1, CommandID: command.CommandID,
		Accepted: true, AggregateType: command.AggregateType, AggregateID: command.AggregateID,
		AggregateVersion: 1, Events: []application.EventRef{},
	}
	db, script := newScriptDB(t,
		beginStep(),
		execStep("INSERT INTO doudizhu_command_results", driver.RowsAffected(1)),
		queryStep("SELECT command_json, result_json, completed_at", []string{"command_json", "result_json", "completed_at"}, [][]driver.Value{{commandJSON, nil, nil}}),
		execStep("UPDATE doudizhu_command_results", driver.RowsAffected(1)),
		commitStep(),
	)
	store, err := New(scriptDatabase{db})
	if err != nil {
		t.Fatal(err)
	}
	err = store.WithinCommand(context.Background(), "actor", command.CommandID, func(ctx context.Context, tx application.Transaction) error {
		stored, completed, err := tx.ClaimCommand(ctx, "actor", command, command.IssuedAt)
		if err != nil {
			return err
		}
		if completed || stored.Command.CommandID != command.CommandID {
			t.Fatalf("claim=%#v completed=%v", stored, completed)
		}
		return tx.CompleteCommand(ctx, "actor", command.CommandID, result, command.IssuedAt)
	})
	if err != nil {
		t.Fatal(err)
	}
	script.assertDone(t)
}

func TestClaimCommandReturnsCompletedDuplicate(t *testing.T) {
	command := application.Command{
		Version: application.CommandProtocolV1, Name: application.CommandRoomCreate,
		CommandID: "command", AggregateType: domain.AggregateRoom, AggregateID: "room-1",
		ClientSeq: 1, IssuedAt: time.Now().UTC().Truncate(time.Millisecond),
	}
	command.ExpiresAt = command.IssuedAt.Add(time.Minute)
	result := application.CommandResult{
		Version: application.CommandResultV1, CommandID: command.CommandID,
		Accepted: true, AggregateType: command.AggregateType, AggregateID: command.AggregateID,
		AggregateVersion: 1, Events: []application.EventRef{},
	}
	commandJSON, _ := json.Marshal(command)
	resultJSON, _ := json.Marshal(result)
	db, script := newScriptDB(t,
		beginStep(),
		execStep("INSERT INTO doudizhu_command_results", driver.RowsAffected(0)),
		queryStep("SELECT command_json, result_json, completed_at", []string{"command_json", "result_json", "completed_at"}, [][]driver.Value{{commandJSON, resultJSON, command.IssuedAt}}),
		commitStep(),
	)
	store, _ := New(scriptDatabase{db})
	err := store.WithinCommand(context.Background(), "actor", command.CommandID, func(ctx context.Context, tx application.Transaction) error {
		stored, completed, err := tx.ClaimCommand(ctx, "actor", command, command.IssuedAt)
		if err != nil {
			return err
		}
		if !completed || !stored.Result.Accepted {
			t.Fatalf("stored=%#v completed=%v", stored, completed)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	script.assertDone(t)
}

func TestOptimisticConflictRollsBack(t *testing.T) {
	db, script := newScriptDB(t,
		beginStep(),
		execStep("UPDATE doudizhu_rooms", driver.RowsAffected(0)),
		rollbackStep(),
	)
	store, _ := New(scriptDatabase{db})
	err := store.WithinCommand(context.Background(), "actor", "command", func(ctx context.Context, tx application.Transaction) error {
		return tx.UpdateRoom(ctx, validRoomSnapshot(), 1, time.Now().UTC())
	})
	if !errors.Is(err, application.ErrOptimisticConflict) {
		t.Fatalf("error=%v", err)
	}
	script.assertDone(t)
}

func TestDeadlockIsMarkedRetryable(t *testing.T) {
	db, script := newScriptDB(t,
		beginStep(),
		scriptStep{kind: "exec", contains: "UPDATE doudizhu_rooms", err: errors.New("Error 1213 (40001): Deadlock found when trying to get lock")},
		rollbackStep(),
	)
	store, _ := New(scriptDatabase{db})
	err := store.WithinCommand(context.Background(), "actor", "command", func(ctx context.Context, tx application.Transaction) error {
		return tx.UpdateRoom(ctx, validRoomSnapshot(), 1, time.Now().UTC())
	})
	if !errors.Is(err, application.ErrRetryableTransaction) {
		t.Fatalf("error=%v, want retryable transaction", err)
	}
	script.assertDone(t)
}

func TestContributionInsertReceivesOnlyProtectedBytes(t *testing.T) {
	plaintext := []byte("original secret phrase")
	ciphertext := []byte{9, 8, 7, 6}
	step := execStep("INSERT INTO doudizhu_contribution_records", driver.RowsAffected(1))
	step.check = func(arguments []driver.NamedValue) error {
		for _, argument := range arguments {
			switch value := argument.Value.(type) {
			case string:
				if strings.Contains(value, string(plaintext)) {
					return fmt.Errorf("plaintext string reached SQL adapter")
				}
			case []byte:
				if strings.Contains(string(value), string(plaintext)) {
					return fmt.Errorf("plaintext bytes reached SQL adapter")
				}
			}
		}
		return nil
	}
	db, script := newScriptDB(t,
		beginStep(), step, commitStep(),
	)
	store, _ := New(scriptDatabase{db})
	err := store.WithinCommand(context.Background(), "actor", "command", func(ctx context.Context, tx application.Transaction) error {
		return tx.InsertContributionRecord(ctx, application.ProtectedContributionRecord{
			RecordID: "record-1", HandID: "hand-1", Seat: domain.SeatOne,
			ActorAccountID: "actor", CommandID: "command", ContributionDigest: domain.ContributionDigest{1},
			ProtectionKeyID: "key-1", Nonce: []byte{1, 2, 3}, Ciphertext: ciphertext, AADDigest: [32]byte{2}, CreatedAt: time.Now().UTC(),
		})
	})
	if err != nil {
		t.Fatal(err)
	}
	script.assertDone(t)
}

func validRoomSnapshot() domain.RoomSnapshot {
	return domain.RoomSnapshot{
		ID: "room-1", OwnerID: "actor", Status: domain.RoomWaitingPlayers,
		Seats:   [3]domain.SeatSnapshot{{Seat: domain.SeatOne, AccountID: "actor"}, {Seat: domain.SeatTwo}, {Seat: domain.SeatThree}},
		Version: 2,
	}
}

type scriptDatabase struct{ db *sql.DB }

func (d scriptDatabase) DB() *sql.DB { return d.db }

type scriptStep struct {
	kind     string
	contains string
	columns  []string
	rows     [][]driver.Value
	result   driver.Result
	err      error
	check    func([]driver.NamedValue) error
}

func queryStep(contains string, columns []string, rows [][]driver.Value) scriptStep {
	return scriptStep{kind: "query", contains: contains, columns: columns, rows: rows}
}
func execStep(contains string, result driver.Result) scriptStep {
	return scriptStep{kind: "exec", contains: contains, result: result}
}
func beginStep() scriptStep    { return scriptStep{kind: "begin"} }
func commitStep() scriptStep   { return scriptStep{kind: "commit"} }
func rollbackStep() scriptStep { return scriptStep{kind: "rollback"} }

type script struct {
	mu    sync.Mutex
	steps []scriptStep
}

func (s *script) pop(kind, query string, args []driver.NamedValue) (scriptStep, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.steps) == 0 {
		return scriptStep{}, fmt.Errorf("unexpected %s: %s", kind, query)
	}
	step := s.steps[0]
	s.steps = s.steps[1:]
	if step.kind != kind {
		return scriptStep{}, fmt.Errorf("got %s, want %s for %s", kind, step.kind, query)
	}
	if step.contains != "" && !strings.Contains(strings.Join(strings.Fields(query), " "), step.contains) {
		return scriptStep{}, fmt.Errorf("query %q does not contain %q", query, step.contains)
	}
	if step.check != nil {
		if err := step.check(args); err != nil {
			return scriptStep{}, err
		}
	}
	return step, step.err
}
func (s *script) assertDone(t *testing.T) {
	t.Helper()
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.steps) != 0 {
		t.Fatalf("%d scripted SQL steps remain: %#v", len(s.steps), s.steps)
	}
}

var driverSequence atomic.Uint64
var scriptRegistry sync.Map

type scriptDriver struct{ id string }

func (d scriptDriver) Open(string) (driver.Conn, error) {
	value, _ := scriptRegistry.Load(d.id)
	return &scriptConn{script: value.(*script)}, nil
}

func newScriptDB(t *testing.T, steps ...scriptStep) (*sql.DB, *script) {
	t.Helper()
	id := fmt.Sprintf("ddz-script-%d", driverSequence.Add(1))
	s := &script{steps: append([]scriptStep(nil), steps...)}
	scriptRegistry.Store(id, s)
	sql.Register(id, scriptDriver{id: id})
	db, err := sql.Open(id, "")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close(); scriptRegistry.Delete(id) })
	return db, s
}

type scriptConn struct{ script *script }

func (c *scriptConn) Prepare(string) (driver.Stmt, error) { return nil, driver.ErrSkip }
func (c *scriptConn) Close() error                        { return nil }
func (c *scriptConn) Begin() (driver.Tx, error) {
	return c.BeginTx(context.Background(), driver.TxOptions{})
}
func (c *scriptConn) BeginTx(_ context.Context, options driver.TxOptions) (driver.Tx, error) {
	if options.Isolation != driver.IsolationLevel(sql.LevelReadCommitted) {
		return nil, fmt.Errorf("isolation=%d, want READ COMMITTED", options.Isolation)
	}
	if _, err := c.script.pop("begin", "", nil); err != nil {
		return nil, err
	}
	return &scriptTx{script: c.script}, nil
}
func (c *scriptConn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	step, err := c.script.pop("query", query, args)
	if err != nil {
		return nil, err
	}
	return &scriptRows{columns: step.columns, rows: step.rows}, nil
}
func (c *scriptConn) ExecContext(_ context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	step, err := c.script.pop("exec", query, args)
	if err != nil {
		return nil, err
	}
	if step.result == nil {
		return driver.RowsAffected(1), nil
	}
	return step.result, nil
}
func (c *scriptConn) CheckNamedValue(*driver.NamedValue) error { return nil }

type scriptTx struct{ script *script }

func (t *scriptTx) Commit() error   { _, err := t.script.pop("commit", "", nil); return err }
func (t *scriptTx) Rollback() error { _, err := t.script.pop("rollback", "", nil); return err }

type scriptRows struct {
	columns []string
	rows    [][]driver.Value
	index   int
}

func (r *scriptRows) Columns() []string { return r.columns }
func (r *scriptRows) Close() error      { return nil }
func (r *scriptRows) Next(dest []driver.Value) error {
	if r.index >= len(r.rows) {
		return io.EOF
	}
	copy(dest, r.rows[r.index])
	r.index++
	return nil
}
