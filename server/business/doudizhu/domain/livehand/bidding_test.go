package livehand

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/bidding"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/carddeck"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/randomizedsetup"
	"github.com/oublie6/awesome-zero-platform/server/business/gamecore"
)

func TestLiveBidScoreThreeSelectsLandlordAndRevealsBottomCards(t *testing.T) {
	game := newDirectBiddingGame(t, 1)
	outcome, err := game.Apply(bidCommand(t, 1, 1, bidding.ScoreThree))
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Version != 2 || outcome.Terminal {
		t.Fatalf("outcome=%#v", outcome)
	}
	var result BidResult
	if err := json.Unmarshal(outcome.Payload, &result); err != nil {
		t.Fatal(err)
	}
	if result.Phase != PhasePlaying || result.LandlordSeat != 1 || result.WinningScore != bidding.ScoreThree || result.PlayingSeat != 1 || result.RequiresTermination {
		t.Fatalf("result=%#v", result)
	}
	if len(game.current[0]) != 20 || len(game.current[1]) != 17 || len(game.current[2]) != 17 {
		t.Fatalf("hand sizes=%d/%d/%d", len(game.current[0]), len(game.current[1]), len(game.current[2]))
	}
	wantBottom := []carddeck.Card{51, 52, 53}
	if !reflect.DeepEqual(game.current[0][17:], wantBottom) {
		t.Fatalf("landlord cards=%v want=%v", game.current[0][17:], wantBottom)
	}

	public := readPublicView(t, game)
	if public.Phase != PhasePlaying || public.LandlordSeat != 1 || public.PlayingSeat != 1 || len(public.LandlordCards) != 3 {
		t.Fatalf("public=%#v", public)
	}
	if public.Seats[0].RemainingCards != 20 || public.Seats[1].RemainingCards != 17 || public.Seats[2].RemainingCards != 17 {
		t.Fatalf("public seats=%#v", public.Seats)
	}
	privateLandlord := readPrivateView(t, game, 1)
	privateFarmer := readPrivateView(t, game, 2)
	if len(privateLandlord.Cards) != 20 || len(privateFarmer.Cards) != 17 {
		t.Fatalf("private sizes=%d/%d", len(privateLandlord.Cards), len(privateFarmer.Cards))
	}
	if _, err := game.Apply(bidCommand(t, 1, 2, bidding.ScorePass)); !errors.Is(err, ErrUnsupportedCommand) {
		t.Fatalf("post-bidding error=%v", err)
	}
	if len(game.current[0]) != 20 {
		t.Fatalf("landlord cards appended twice: %d", len(game.current[0]))
	}
}

func TestLiveBidPublicViewHidesBottomCardsBeforeSelection(t *testing.T) {
	game := newDirectBiddingGame(t, 2)
	public := readPublicView(t, game)
	if public.Phase != PhaseBidding || public.Bidding.FirstBidder != 2 || len(public.LandlordCards) != 0 || public.LandlordSeat != 0 {
		t.Fatalf("public=%#v", public)
	}
	for _, seat := range public.Seats {
		if seat.RemainingCards != 17 {
			t.Fatalf("seat=%#v", seat)
		}
	}
}

func TestLiveBidThreePassesRequireTerminationWithoutReveal(t *testing.T) {
	game := newDirectBiddingGame(t, 3)
	if _, err := game.Apply(bidCommand(t, 3, 1, bidding.ScorePass)); err != nil {
		t.Fatal(err)
	}
	if _, err := game.Apply(bidCommand(t, 1, 2, bidding.ScorePass)); err != nil {
		t.Fatal(err)
	}
	outcome, err := game.Apply(bidCommand(t, 2, 3, bidding.ScorePass))
	if err != nil {
		t.Fatal(err)
	}
	var result BidResult
	if err := json.Unmarshal(outcome.Payload, &result); err != nil {
		t.Fatal(err)
	}
	if result.Phase != PhaseNoLandlord || !result.RequiresTermination || !result.Bidding.NoLandlord || result.LandlordSeat != 0 {
		t.Fatalf("result=%#v", result)
	}
	public := readPublicView(t, game)
	if public.Phase != PhaseNoLandlord || len(public.LandlordCards) != 0 {
		t.Fatalf("public=%#v", public)
	}
	for index := range game.current {
		if len(game.current[index]) != 17 || public.Seats[index].RemainingCards != 17 {
			t.Fatalf("seat %d sizes=%d/%d", index+1, len(game.current[index]), public.Seats[index].RemainingCards)
		}
	}
	if _, err := game.Apply(bidCommand(t, 3, 4, bidding.ScoreThree)); !errors.Is(err, ErrUnsupportedCommand) {
		t.Fatalf("post-no-landlord error=%v", err)
	}
}

func TestLiveBidRejectsInvalidCommandsWithoutMutation(t *testing.T) {
	tests := []struct {
		name    string
		command gamecore.Command
		want    error
	}{
		{name: "stale version", command: gamecore.Command{ActorPosition: 1, ExpectedVersion: 2, Payload: mustJSON(t, BidCommand{Version: BidCommandVersion, Score: bidding.ScorePass})}, want: ErrVersionConflict},
		{name: "wrong actor", command: gamecore.Command{ActorPosition: 2, ExpectedVersion: 1, Payload: mustJSON(t, BidCommand{Version: BidCommandVersion, Score: bidding.ScorePass})}, want: bidding.ErrWrongTurn},
		{name: "empty", command: gamecore.Command{ActorPosition: 1, ExpectedVersion: 1}, want: ErrMalformedCommand},
		{name: "unknown field", command: gamecore.Command{ActorPosition: 1, ExpectedVersion: 1, Payload: []byte(`{"v":"doudizhu-live-bid-command-v1","score":0,"seat":1}`)}, want: ErrMalformedCommand},
		{name: "trailing json", command: gamecore.Command{ActorPosition: 1, ExpectedVersion: 1, Payload: []byte(`{"v":"doudizhu-live-bid-command-v1","score":0}{}`)}, want: ErrMalformedCommand},
		{name: "unsupported version", command: gamecore.Command{ActorPosition: 1, ExpectedVersion: 1, Payload: mustJSON(t, BidCommand{Version: "future", Score: bidding.ScorePass})}, want: gamecore.ErrUnsupportedVersion},
		{name: "invalid score", command: gamecore.Command{ActorPosition: 1, ExpectedVersion: 1, Payload: mustJSON(t, BidCommand{Version: BidCommandVersion, Score: bidding.Score(4)})}, want: bidding.ErrInvalidScore},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			game := newDirectBiddingGame(t, 1)
			before := directGameState(game)
			if _, err := game.Apply(test.command); !errors.Is(err, test.want) {
				t.Fatalf("error=%v want=%v", err, test.want)
			}
			if after := directGameState(game); !reflect.DeepEqual(after, before) {
				t.Fatalf("game mutated: before=%#v after=%#v", before, after)
			}
		})
	}
}

func TestLiveBidReturnedViewsAreCopyIsolated(t *testing.T) {
	game := newDirectBiddingGame(t, 1)
	view, err := game.View(gamecore.ViewRequest{ViewerPosition: 1})
	if err != nil {
		t.Fatal(err)
	}
	for index := range view.Payload {
		view.Payload[index] = 'x'
	}
	second := readPrivateView(t, game, 1)
	if len(second.Cards) != 17 || second.Public.Bidding.FirstBidder != 1 {
		t.Fatalf("second=%#v", second)
	}
}

func newDirectBiddingGame(t *testing.T, first uint8) *Game {
	t.Helper()
	dealDigest := dealDigestForFirst(t, first)
	auction, err := bidding.New(dealDigest)
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := gamecore.NewSetupArtifact(randomizedsetup.Descriptor(), randomizedsetup.ArtifactVersion, []byte("direct-livehand-test"))
	if err != nil {
		t.Fatal(err)
	}
	var current [3][]carddeck.Card
	card := carddeck.Card(0)
	for seat := range current {
		current[seat] = make([]carddeck.Card, carddeck.CardsPerSeat)
		for index := range current[seat] {
			current[seat][index] = card
			card++
		}
	}
	return &Game{
		id:       "direct-bidding-hand",
		seats:    [3]domain.HandSeat{{Seat: 1, AccountID: "player-1"}, {Seat: 2, AccountID: "player-2"}, {Seat: 3, AccountID: "player-3"}},
		phase:    PhaseBidding,
		version:  1,
		artifact: artifact,
		setup: randomizedsetup.Setup{
			DealDigest:    dealDigest,
			LandlordCards: [carddeck.LandlordCardCount]carddeck.Card{51, 52, 53},
		},
		auction: auction,
		current: current,
	}
}

type directState struct {
	Phase   string
	Version uint64
	Auction bidding.Snapshot
	Cards   [3][]carddeck.Card
}

func directGameState(game *Game) directState {
	state := directState{Phase: game.phase, Version: game.version, Auction: game.auction.Snapshot()}
	for index := range game.current {
		state.Cards[index] = append([]carddeck.Card(nil), game.current[index]...)
	}
	return state
}

func dealDigestForFirst(t *testing.T, first uint8) carddeck.DealDigest {
	t.Helper()
	for value := uint64(1); ; value++ {
		var encoded [8]byte
		binary.BigEndian.PutUint64(encoded[:], value)
		sum := sha256.Sum256(encoded[:])
		digest := carddeck.DealDigest(sum)
		position, err := bidding.DeriveFirstBidder(digest)
		if err != nil {
			t.Fatal(err)
		}
		if position == first {
			return digest
		}
	}
}

func bidCommand(t *testing.T, position uint8, version uint64, score bidding.Score) gamecore.Command {
	t.Helper()
	return gamecore.Command{ActorPosition: position, ExpectedVersion: version, Payload: mustJSON(t, BidCommand{Version: BidCommandVersion, Score: score})}
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	payload, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return payload
}

func readPublicView(t *testing.T, game *Game) PublicView {
	t.Helper()
	view, err := game.View(gamecore.ViewRequest{PublicOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	var result PublicView
	if err := json.Unmarshal(view.Payload, &result); err != nil {
		t.Fatal(err)
	}
	return result
}

func readPrivateView(t *testing.T, game *Game, position uint8) PrivateView {
	t.Helper()
	view, err := game.View(gamecore.ViewRequest{ViewerPosition: position})
	if err != nil {
		t.Fatal(err)
	}
	var result PrivateView
	if err := json.Unmarshal(view.Payload, &result); err != nil {
		t.Fatal(err)
	}
	return result
}
