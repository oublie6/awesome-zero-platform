package livehand

import (
	"bytes"
	"testing"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/bidding"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/carddeck"
	"github.com/oublie6/awesome-zero-platform/server/business/gamecore"
)

// A speculative bid must not leak a landlord selection, bottom cards or
// auction history into the authority until its MySQL commit succeeds.
func TestCandidateForkBidIsFullyIsolated(t *testing.T) {
	authority := newDirectBiddingGame(t, 1)
	before := directGameState(authority)
	candidate, err := authority.Fork()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := candidate.Apply(bidCommand(t, 1, 1, bidding.ScoreThree)); err != nil {
		t.Fatal(err)
	}
	if authority.version != 1 || authority.phase != PhaseBidding || candidate.phase != PhasePlaying {
		t.Fatalf("candidate mutated authoritative phase: %s/%s", authority.phase, candidate.phase)
	}
	if len(authority.current[0]) != 17 || len(candidate.current[0]) != 20 {
		t.Fatal("candidate exposed bottom cards before commit")
	}
	if got := directGameState(authority); got.Version != before.Version || got.Phase != before.Phase ||
		len(got.Auction.Actions) != len(before.Auction.Actions) {
		t.Fatal("speculative auction leaked into live state")
	}
	if len(authority.history) != 0 || len(candidate.history) != 1 {
		t.Fatal("command history aliased")
	}
}

func TestCandidateForkPlayAndNestedHistoryAreFullyIsolated(t *testing.T) {
	authority := newDirectPlayingGame(t)
	before, err := authority.View(gamecore.ViewRequest{PublicOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	candidate, err := authority.Fork()
	if err != nil {
		t.Fatal(err)
	}
	card := authority.current[0][0]
	outcome, err := candidate.Apply(livePlayCommand(t, 1, authority.version, card))
	if err != nil || outcome.Version != authority.version+1 {
		t.Fatalf("candidate play: %+v %v", outcome, err)
	}
	after, err := authority.View(gamecore.ViewRequest{PublicOnly: true})
	if err != nil || !bytes.Equal(before.Payload, after.Payload) {
		t.Fatal("speculative play mutated authoritative public state")
	}
	if candidate.play.Snapshot().Revision <= authority.play.Snapshot().Revision ||
		len(candidate.current[0]) != len(authority.current[0])-1 {
		t.Fatal("play snapshot or cards leaked between candidates")
	}
	candidate.current[0][0] = carddeck.Card(53)
	if bytes.Equal([]byte{byte(candidate.current[0][0])}, []byte{byte(authority.current[0][0])}) {
		t.Fatal("private hand backing arrays were shared")
	}
}
