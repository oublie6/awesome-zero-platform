package livehand

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/bidding"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/carddeck"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/playing"
	"github.com/oublie6/awesome-zero-platform/server/business/gamecore"
)

func TestLivePlayRemovesHeldCardAndAdvancesTurn(t *testing.T) {
	game := newDirectPlayingGame(t)
	played := game.current[0][0]
	outcome, err := game.Apply(livePlayCommand(t, 1, 2, played))
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Version != 3 || outcome.Terminal {
		t.Fatalf("outcome=%#v", outcome)
	}
	var result PlayResult
	if err := json.Unmarshal(outcome.Payload, &result); err != nil {
		t.Fatal(err)
	}
	if result.Version != PlayResultVersion || result.Phase != PhasePlaying || result.Playing.CurrentSeat != 2 || result.Playing.LeadingSeat != 1 || result.WinnerSeat != 0 {
		t.Fatalf("result=%#v", result)
	}
	if len(game.current[0]) != 19 || containsCard(game.current[0], played) {
		t.Fatalf("remaining=%v", game.current[0])
	}
	public := readPublicView(t, game)
	if public.Playing == nil || public.Playing.CurrentSeat != 2 || public.Seats[0].RemainingCards != 19 || public.WinnerSeat != 0 {
		t.Fatalf("public=%#v", public)
	}
	private := readPrivateView(t, game, 1)
	if len(private.Cards) != 19 {
		t.Fatalf("private=%#v", private)
	}
}

func TestLivePlayRejectsWrongTurnAndCardsNotHeldWithoutMutation(t *testing.T) {
	game := newDirectPlayingGame(t)
	tests := []struct {
		name    string
		command gamecore.Command
		want    error
	}{
		{
			name:    "wrong turn",
			command: livePlayCommand(t, 2, 2, game.current[1][0]),
			want:    playing.ErrWrongTurn,
		},
		{
			name:    "card not held",
			command: livePlayCommand(t, 1, 2, game.current[1][0]),
			want:    ErrCardNotHeld,
		},
		{
			name:    "stale version",
			command: livePlayCommand(t, 1, 1, game.current[0][0]),
			want:    ErrVersionConflict,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			before := capturePlayingGame(game)
			if _, err := game.Apply(test.command); !errors.Is(err, test.want) {
				t.Fatalf("error=%v want=%v", err, test.want)
			}
			if after := capturePlayingGame(game); !reflect.DeepEqual(after, before) {
				t.Fatalf("game mutated: before=%#v after=%#v", before, after)
			}
		})
	}
}

func TestLivePlayMustBeatAndTwoPassesResetTrick(t *testing.T) {
	game := newDirectPlayingGame(t)
	if _, err := game.Apply(livePlayCommand(t, 1, 2, carddeck.BigJoker)); err != nil {
		t.Fatal(err)
	}
	before := capturePlayingGame(game)
	if _, err := game.Apply(livePlayCommand(t, 2, 3, game.current[1][0])); !errors.Is(err, playing.ErrDoesNotBeat) {
		t.Fatalf("error=%v", err)
	}
	if after := capturePlayingGame(game); !reflect.DeepEqual(after, before) {
		t.Fatalf("failed play mutated game: before=%#v after=%#v", before, after)
	}
	if _, err := game.Apply(livePassCommand(t, 2, 3)); err != nil {
		t.Fatal(err)
	}
	outcome, err := game.Apply(livePassCommand(t, 3, 4))
	if err != nil {
		t.Fatal(err)
	}
	var result PlayResult
	if err := json.Unmarshal(outcome.Payload, &result); err != nil {
		t.Fatal(err)
	}
	if outcome.Version != 5 || result.Playing.CurrentSeat != 1 || result.Playing.LeadingSeat != 0 || result.Playing.LeadingPattern != nil || result.Playing.PassCount != 0 {
		t.Fatalf("result=%#v", result)
	}
}

func TestLivePassRejectsEmptyTrickWithoutMutation(t *testing.T) {
	game := newDirectPlayingGame(t)
	before := capturePlayingGame(game)
	if _, err := game.Apply(livePassCommand(t, 1, 2)); !errors.Is(err, playing.ErrCannotPass) {
		t.Fatalf("error=%v", err)
	}
	if after := capturePlayingGame(game); !reflect.DeepEqual(after, before) {
		t.Fatalf("game mutated: before=%#v after=%#v", before, after)
	}
}

func TestLiveWinningPlayCompletesGameplay(t *testing.T) {
	game := newDirectPlayingGame(t)
	last := game.current[0][0]
	game.current[0] = []carddeck.Card{last}
	outcome, err := game.Apply(livePlayCommand(t, 1, 2, last))
	if err != nil {
		t.Fatal(err)
	}
	var result PlayResult
	if err := json.Unmarshal(outcome.Payload, &result); err != nil {
		t.Fatal(err)
	}
	if result.Phase != PhaseGameplayComplete || result.WinnerSeat != 1 || !result.Playing.Complete || result.Playing.WinnerSeat != 1 || len(game.current[0]) != 0 {
		t.Fatalf("result=%#v current=%v", result, game.current[0])
	}
	public := readPublicView(t, game)
	if public.Phase != PhaseGameplayComplete || public.WinnerSeat != 1 || public.Playing == nil || !public.Playing.Complete {
		t.Fatalf("public=%#v", public)
	}
	if _, err := game.Apply(livePassCommand(t, 1, 3)); !errors.Is(err, ErrUnsupportedCommand) {
		t.Fatalf("post-win error=%v", err)
	}
}

func TestLivePlayingCommandsRejectMalformedPayloadsWithoutMutation(t *testing.T) {
	game := newDirectPlayingGame(t)
	tests := []struct {
		name    string
		payload []byte
		want    error
	}{
		{name: "empty", want: ErrMalformedCommand},
		{name: "unknown play field", payload: []byte(`{"v":"doudizhu-live-play-command-v1","cards":["C3"],"seat":1}`), want: ErrMalformedCommand},
		{name: "trailing json", payload: []byte(`{"v":"doudizhu-live-pass-command-v1"}{}`), want: ErrMalformedCommand},
		{name: "unsupported", payload: []byte(`{"v":"future"}`), want: gamecore.ErrUnsupportedVersion},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			before := capturePlayingGame(game)
			command := gamecore.Command{ActorPosition: 1, ExpectedVersion: 2, Payload: test.payload}
			if _, err := game.Apply(command); !errors.Is(err, test.want) {
				t.Fatalf("error=%v want=%v", err, test.want)
			}
			if after := capturePlayingGame(game); !reflect.DeepEqual(after, before) {
				t.Fatalf("game mutated: before=%#v after=%#v", before, after)
			}
		})
	}
}

func newDirectPlayingGame(t *testing.T) *Game {
	t.Helper()
	game := newDirectBiddingGame(t, 1)
	if _, err := game.Apply(bidCommand(t, 1, 1, bidding.ScoreThree)); err != nil {
		t.Fatal(err)
	}
	if game.phase != PhasePlaying || game.play == nil || game.version != 2 {
		t.Fatalf("game=%#v", game)
	}
	return game
}

func livePlayCommand(t *testing.T, seat uint8, version uint64, cards ...carddeck.Card) gamecore.Command {
	t.Helper()
	codes, err := cardCodes(cards)
	if err != nil {
		t.Fatal(err)
	}
	return gamecore.Command{
		ActorPosition:   seat,
		ExpectedVersion: version,
		Payload:         mustJSON(t, PlayCommand{Version: PlayCommandVersion, Cards: codes}),
	}
}

func livePassCommand(t *testing.T, seat uint8, version uint64) gamecore.Command {
	t.Helper()
	return gamecore.Command{
		ActorPosition:   seat,
		ExpectedVersion: version,
		Payload:         mustJSON(t, PassCommand{Version: PassCommandVersion}),
	}
}

type capturedPlayingGame struct {
	Phase       string
	Version     uint64
	PlayingSeat uint8
	WinnerSeat  uint8
	Cards       [3][]carddeck.Card
	Playing     playing.Snapshot
}

func capturePlayingGame(game *Game) capturedPlayingGame {
	result := capturedPlayingGame{
		Phase:       game.phase,
		Version:     game.version,
		PlayingSeat: game.playingSeat,
		WinnerSeat:  game.winnerSeat,
	}
	for index := range game.current {
		result.Cards[index] = append([]carddeck.Card(nil), game.current[index]...)
	}
	if game.play != nil {
		result.Playing = game.play.Snapshot()
	}
	return result
}

func containsCard(cards []carddeck.Card, wanted carddeck.Card) bool {
	for _, card := range cards {
		if card == wanted {
			return true
		}
	}
	return false
}
