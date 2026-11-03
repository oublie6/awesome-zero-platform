package mysqlstore

import (
	"bytes"
	"errors"
	"testing"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
)

func TestValidateReplayActionRejectsMalformedJournalMetadata(t *testing.T) {
	valid := LiveAction{
		HandID: "hand-1", Version: 2, Epoch: 3,
		CommandID: "request-1", Actor: domain.AccountID("player-1"),
		ActorPosition: 1, ExpectedVersion: 1,
		Payload: []byte(`{"v":"doudizhu-live-bid-command-v1","score":1}`),
	}
	digest := bytes.Repeat([]byte{7}, 32)
	if err := validateReplayAction(valid, 2, 2, digest); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name          string
		edit          func(*LiveAction)
		digest        []byte
		previousEpoch uint64
	}{
		{name: "gap", edit: func(a *LiveAction) { a.Version = 4 }},
		{name: "wrong base", edit: func(a *LiveAction) { a.ExpectedVersion = 2 }},
		{name: "zero epoch", edit: func(a *LiveAction) { a.Epoch = 0 }},
		{name: "writer epoch moved backwards", previousEpoch: 4},
		{name: "empty request ID", edit: func(a *LiveAction) { a.CommandID = "" }},
		{name: "oversized request ID", edit: func(a *LiveAction) { a.CommandID = string(bytes.Repeat([]byte("x"), 129)) }},
		{name: "empty actor", edit: func(a *LiveAction) { a.Actor = "" }},
		{name: "forged actor position", edit: func(a *LiveAction) { a.ActorPosition = 4 }},
		{name: "empty payload", edit: func(a *LiveAction) { a.Payload = nil }},
		{name: "malformed JSON", edit: func(a *LiveAction) { a.Payload = []byte("{") }},
		{name: "oversized action", edit: func(a *LiveAction) { a.Payload = bytes.Repeat([]byte("a"), maxLiveActionPayloadBytes+1) }},
		{name: "short digest", digest: digest[:31]},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			action := valid
			if tc.edit != nil {
				tc.edit(&action)
			}
			gotDigest := digest
			if tc.digest != nil {
				gotDigest = tc.digest
			}
			if !errors.Is(validateReplayAction(action, 2, tc.previousEpoch, gotDigest), ErrLiveReplayIntegrity) {
				t.Fatal("corrupt journal action was accepted")
			}
		})
	}
}

func TestValidateReplayActionAcceptsWriterTakeover(t *testing.T) {
	act := LiveAction{
		Version: 40, Epoch: 7, CommandID: "request-40",
		Actor: "player-3", ActorPosition: 3, ExpectedVersion: 39,
		Payload: []byte(`{"v":"doudizhu-live-pass-command-v1"}`),
	}
	if err := validateReplayAction(act, 40, 6, make([]byte, 32)); err != nil {
		t.Fatalf("valid fenced writer epoch rejected: %v", err)
	}
	if err := validateReplayAction(act, 40, 7, make([]byte, 32)); err != nil {
		t.Fatalf("same owner epoch rejected: %v", err)
	}
}

func TestLiveActionHMACBindsAcceptedRequestIdentity(t *testing.T) {
	original := LiveAction{
		HandID: "hand-1", Version: 2, Epoch: 3, CommandID: "request-1",
		Actor: "player-1", ActorPosition: 1, ExpectedVersion: 1,
		ClientSeq: 91, Payload: []byte(`{"v":"bid","score":3}`),
		ResultDigest: [32]byte{8}, IntegrityVersion: 2,
	}
	key := deriveLiveActionIntegrityKey(bytes.Repeat([]byte{4}, 32))
	journal := &LiveJournal{integrityKey: key}
	tag, err := liveActionIntegrityTag(original, key)
	if err != nil || journal.verifyLiveActionIntegrity(original, tag[:]) != nil {
		t.Fatalf("valid signed request rejected: %v", err)
	}
	for _, mutate := range []func(*LiveAction){
		func(a *LiveAction) { a.ClientSeq = 92 },
		func(a *LiveAction) { a.CommandID = "different" },
		func(a *LiveAction) { a.Actor = "other-player" },
		func(a *LiveAction) { a.ActorPosition = 2 },
		func(a *LiveAction) { a.Epoch = 4 },
		func(a *LiveAction) { a.ExpectedVersion = 2 },
		func(a *LiveAction) { a.Payload = []byte(`{"v":"bid","score":2}`) },
		func(a *LiveAction) { a.ResultDigest[0] ^= 1 },
	} {
		changed := original
		mutate(&changed)
		if journal.verifyLiveActionIntegrity(changed, tag[:]) == nil {
			t.Fatalf("mutated request identity accepted: %+v", changed)
		}
	}
	// MySQL JSON reorders object keys without changing semantic command data.
	reordered := original
	reordered.Payload = []byte(`{"score":3,"v":"bid"}`)
	if err := journal.verifyLiveActionIntegrity(reordered, tag[:]); err != nil {
		t.Fatalf("normal JSON object reordering rejected: %v", err)
	}
	// Rotation permits old roots strictly as readers, not current writers.
	newKey := deriveLiveActionIntegrityKey(bytes.Repeat([]byte{5}, 32))
	if err := (&LiveJournal{integrityKey: newKey, integrityReaders: [][]byte{key}}).verifyLiveActionIntegrity(original, tag[:]); err != nil {
		t.Fatalf("retained authentication root failed: %v", err)
	}
	legacy := original
	legacy.IntegrityVersion = 1
	legacy.ClientSeq = 0
	legacyTag, err := liveActionIntegrityTag(legacy, nil)
	if err != nil || (&LiveJournal{}).verifyLiveActionIntegrity(legacy, legacyTag[:]) != nil {
		t.Fatalf("zero-sequence internal legacy failed: %v", err)
	}
	legacy.ClientSeq = 91
	if (&LiveJournal{}).verifyLiveActionIntegrity(legacy, legacyTag[:]) == nil {
		t.Fatal("public request accepted unkeyed tag")
	}
}
