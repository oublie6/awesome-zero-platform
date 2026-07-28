from pathlib import Path

path = Path("server/business/doudizhu/domain/livehand/playing_test.go")
text = path.read_text(encoding="utf-8")
old = '''func TestLiveWinningPlayCompletesGameplay(t *testing.T) {
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
'''
new = '''func TestLiveWinningPlayProducesCompletedOutcome(t *testing.T) {
	game := newDirectPlayingGame(t)
	game.transcript = validTranscriptForLiveTest(t, string(game.id))
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
	if !outcome.Terminal || len(outcome.FinalPayload) == 0 || result.Phase != PhaseCompleted || result.WinnerSeat != 1 || !result.Playing.Complete || result.Playing.WinnerSeat != 1 || result.Settlement == nil || len(game.current[0]) != 0 {
		t.Fatalf("outcome=%#v result=%#v current=%v", outcome, result, game.current[0])
	}
	if result.Settlement.Version != settlement.RulesVersion || !result.Settlement.LandlordWon || !result.Settlement.Spring {
		t.Fatalf("settlement=%#v", result.Settlement)
	}
	if _, err := game.Apply(livePassCommand(t, 1, 3)); !errors.Is(err, gamecore.ErrInstanceNotFound) {
		t.Fatalf("post-win error=%v", err)
	}
}
'''
if text.count(old) != 1:
    raise SystemExit(f"expected winning test once, found {text.count(old)}")
text = text.replace(old, new, 1)
old_import = '\t"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/playing"\n'
new_import = old_import + '\t"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/settlement"\n'
if text.count(old_import) != 1:
    raise SystemExit("playing import mismatch")
text = text.replace(old_import, new_import, 1)
path.write_text(text, encoding="utf-8")
