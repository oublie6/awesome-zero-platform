package application

import (
	"context"
	"crypto/sha256"
	"errors"
	"testing"
	"time"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
)

type managedTestSetup struct {
	provider string
	released int
}

func (s *managedTestSetup) PrepareHand(_ context.Context, _ domain.RoomSnapshot, id domain.HandID) (HandSetup, error) {
	return HandSetup{
		HandID: id, ServerCommitment: domain.ServerCommitment{1},
		RevealKeyID: "reveal-test", RevealPublicKeySHA256: domain.RevealPublicKeyHash{1},
		RevealKeyBoundAt: time.Now().UTC(),
		BeaconPlan: domain.BeaconPlan{Provider: s.provider, Round: "future-round"},
	}, nil
}
func (s *managedTestSetup) ReleaseHand(context.Context, domain.HandID) error {
	s.released++
	return nil
}

type managedTestProtector struct {
	fail bool
	calls int
}
func (p *managedTestProtector) Seal(_ context.Context, plaintext, aad []byte) (ProtectedPayload, error) {
	p.calls++
	if p.fail {
		return ProtectedPayload{}, errors.New("injected encryption failure")
	}
	if len(plaintext) == 0 || len(aad) == 0 {
		return ProtectedPayload{}, errors.New("missing plaintext or aad")
	}
	// Unit-test fake: only an irreversible digest is stored, never plaintext.
	digest := sha256.Sum256(append(append([]byte(nil), aad...), plaintext...))
	return ProtectedPayload{
		KeyID: "test-root", Nonce: []byte{1, 2, 3},
		Ciphertext: digest[:], AADDigest: sha256.Sum256(aad),
	}, nil
}

func managedStartFixture(t *testing.T, beacon string, protector *managedTestProtector) (*Service, *testStore, *testClock, *managedTestSetup, uint64) {
	t.Helper()
	store := newTestStore()
	clock := &testClock{now: time.Now().UTC().Truncate(time.Millisecond)}
	setup := &managedTestSetup{provider: beacon}
	service, err := NewService(store, clock, &testIDs{}, setup, testBeaconVerifier{},
		testLiveRuntime{}, testOpener{}, protector, testNormalizer{}, DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	room, _, err := domain.NewRoom("room-managed", "seat-a")
	if err != nil {
		t.Fatal(err)
	}
	for _, actor := range []domain.AccountID{"seat-b", "seat-c"} {
		if _, err := room.Join(actor, room.Snapshot().Version); err != nil {
			t.Fatal(err)
		}
	}
	for _, actor := range []domain.AccountID{"seat-a", "seat-b", "seat-c"} {
		if _, err := room.SetReady(actor, true, room.Snapshot().Version); err != nil {
			t.Fatal(err)
		}
	}
	store.state.rooms["room-managed"] = room.Snapshot()
	return service, store, clock, setup, room.Snapshot().Version
}

func TestServerManagedStartAtomicAndExactReplay(t *testing.T) {
	protector := &managedTestProtector{}
	svc, store, clock, setup, version := managedStartFixture(t, "drand-quicknet", protector)
	command := testCommand(clock.Now(), CommandRoomHandStart, domain.AggregateRoom, "room-managed", "start-1", 1, version)
	input := StartHandInput{HandID: "hand-managed", FairnessMode: domain.FairnessModeServerManaged}
	first, err := svc.StartRoomHand(context.Background(), "seat-a", command, input)
	if err != nil || !first.Accepted || first.Duplicate {
		t.Fatalf("managed start: result=%+v err=%v", first, err)
	}
	if len(store.state.records) != 3 || protector.calls != 3 || setup.released != 0 {
		t.Fatalf("protected records=%d seal=%d release=%d", len(store.state.records), protector.calls, setup.released)
	}
	stored := store.state.hands["hand-managed"]
	if stored.FairnessMode != domain.FairnessModeServerManaged ||
		stored.Phase != domain.HandWaitingPublicBeacon {
		t.Fatalf("wrong durable managed hand: mode=%q phase=%s", stored.FairnessMode, stored.Phase)
	}
	for i, seat := range stored.Seats {
		own := stored.Contributions[i]
		if !own.Committed || !own.Revealed || own.RecordRef == "" ||
			own.Commitment != domain.ComputeClientCommitment(stored.ID, seat.Seat, own.Digest) {
			t.Fatalf("bad sealed contribution seat=%d: %+v", seat.Seat, own)
		}
		found := false
		for _, record := range store.state.records {
			if record.RecordID == own.RecordRef && record.Seat == seat.Seat &&
				record.ActorAccountID == seat.AccountID && record.CommandID == command.CommandID &&
				record.ContributionDigest == own.Digest &&
				record.ProtectionKeyID == "test-root" && len(record.Ciphertext) > 0 {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing protected seat record %d", seat.Seat)
		}
	}
	if _, err := domain.RestoreHand(stored); err != nil {
		t.Fatalf("managed snapshot unrecoverable: %v", err)
	}
	outboxSize := len(store.state.outbox)
	for _, event := range store.state.outbox {
		if event.ActorAccountID != "seat-a" {
			t.Fatalf("server-managed event misattributed: %s", event.ActorAccountID)
		}
	}
	replayed, err := svc.StartRoomHand(context.Background(), "seat-a", command, input)
	if err != nil || !replayed.Accepted || !replayed.Duplicate ||
		len(store.state.outbox) != outboxSize || len(store.state.records) != 3 ||
		protector.calls != 3 {
		t.Fatalf("duplicate regenerated contributions: result=%+v err=%v", replayed, err)
	}
	changed, err := svc.StartRoomHand(context.Background(), "seat-a", command, StartHandInput{HandID: "hand-managed"})
	if err != nil || changed.Accepted || changed.Failure == nil || changed.Failure.Code != CodeConflict {
		t.Fatalf("request ID did not bind fairness mode: result=%+v err=%v", changed, err)
	}
}

func TestServerManagedStartRollsBackAfterSealFailure(t *testing.T) {
	protector := &managedTestProtector{fail: true}
	svc, store, clock, setup, version := managedStartFixture(t, "drand-quicknet", protector)
	command := testCommand(clock.Now(), CommandRoomHandStart, domain.AggregateRoom, "room-managed", "start-fail", 1, version)
	_, err := svc.StartRoomHand(context.Background(), "seat-a", command,
		StartHandInput{HandID: "hand-fail", FairnessMode: domain.FairnessModeServerManaged})
	if err == nil || len(store.state.records) != 0 || len(store.state.hands) != 0 ||
		len(store.state.outbox) != 0 || setup.released != 1 {
		t.Fatalf("partial managed hand persisted: err=%v records=%d hands=%d release=%d", err,
			len(store.state.records), len(store.state.hands), setup.released)
	}
}

func TestServerManagedStartRequiresPublicBeacon(t *testing.T) {
	protector := &managedTestProtector{}
	svc, store, clock, setup, version := managedStartFixture(t, "hmac", protector)
	command := testCommand(clock.Now(), CommandRoomHandStart, domain.AggregateRoom, "room-managed", "start-hmac", 1, version)
	_, err := svc.StartRoomHand(context.Background(), "seat-a", command,
		StartHandInput{HandID: "hand-hmac", FairnessMode: domain.FairnessModeServerManaged})
	if err == nil || protector.calls != 0 || len(store.state.hands) != 0 || setup.released != 1 {
		t.Fatalf("untrusted managed beacon accepted: err=%v calls=%d release=%d", err, protector.calls, setup.released)
	}
}
