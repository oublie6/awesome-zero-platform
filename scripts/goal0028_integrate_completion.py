from pathlib import Path

state_path = Path("server/business/doudizhu/domain/playing/state.go")
state = state_path.read_text(encoding="utf-8")
state_anchor = '''func (s *State) Snapshot() Snapshot {
'''
clone_method = '''func (s *State) Clone() *State {
	if s == nil {
		return nil
	}
	clone := &State{
		revision:       s.revision,
		currentSeat:    s.currentSeat,
		leadingSeat:    s.leadingSeat,
		leadingCards:   append([]carddeck.Card(nil), s.leadingCards...),
		passCount:      s.passCount,
		complete:       s.complete,
		winnerSeat:     s.winnerSeat,
		history:        make([]Action, len(s.history)),
	}
	if s.leadingPattern != nil {
		pattern := *s.leadingPattern
		clone.leadingPattern = &pattern
	}
	for index, action := range s.history {
		clone.history[index] = cloneAction(action)
	}
	return clone
}

'''
if state.count(state_anchor) != 1:
    raise SystemExit("state Snapshot anchor mismatch")
state = state.replace(state_anchor, clone_method + state_anchor, 1)
state_path.write_text(state, encoding="utf-8")

game_path = Path("server/business/doudizhu/domain/livehand/game.go")
text = game_path.read_text(encoding="utf-8")


def replace_once(old: str, new: str) -> None:
    global text
    if text.count(old) != 1:
        raise SystemExit(f"expected one match, found {text.count(old)} for {old[:160]!r}")
    text = text.replace(old, new, 1)

replace_once(
    '\t"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/randomizedsetup"\n',
    '\t"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/randomizedsetup"\n'
    '\t"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/settlement"\n',
)
replace_once(
    '\tTerminalPayloadV1     = "doudizhu-live-terminal-v1"\n',
    '\tTerminalPayloadV1     = "doudizhu-live-terminal-v1"\n'
    '\tCompletedPayloadV1    = "doudizhu-live-completed-v1"\n',
)
replace_once(
    '\tPhaseGameplayComplete = "GAMEPLAY_COMPLETE"\n\tPhaseAborted          = "ABORTED"\n',
    '\tPhaseGameplayComplete = "GAMEPLAY_COMPLETE"\n'
    '\tPhaseCompleted        = "COMPLETED"\n'
    '\tPhaseAborted          = "ABORTED"\n',
)
replace_once(
    '\twinnerSeat   uint8\n\tterminal     bool\n',
    '\twinnerSeat   uint8\n\tsettlement   *settlement.Result\n\tterminal     bool\n',
)
replace_once(
    '''type PlayResult struct {
	Version      string           `json:"v"`
	HandID       string           `json:"handId"`
	StateVersion uint64           `json:"stateVersion"`
	Phase        string           `json:"phase"`
	Playing      playing.Snapshot `json:"playing"`
	WinnerSeat   uint8            `json:"winnerSeat,omitempty"`
}
''',
    '''type PlayResult struct {
	Version      string             `json:"v"`
	HandID       string             `json:"handId"`
	StateVersion uint64             `json:"stateVersion"`
	Phase        string             `json:"phase"`
	Playing      playing.Snapshot   `json:"playing"`
	WinnerSeat   uint8              `json:"winnerSeat,omitempty"`
	Settlement   *settlement.Result `json:"settlement,omitempty"`
}
''',
)
replace_once(
    '\tWinnerSeat    uint8             `json:"winnerSeat,omitempty"`\n',
    '\tWinnerSeat    uint8              `json:"winnerSeat,omitempty"`\n'
    '\tSettlement    *settlement.Result `json:"settlement,omitempty"`\n',
)
replace_once(
    '''type TerminalPayload struct {
	Version          string      `json:"v"`
	HandID           string      `json:"handId"`
	Status           string      `json:"status"`
	Reason           string      `json:"reason"`
	StateVersion     uint64      `json:"stateVersion"`
	SetupArtifact    string      `json:"setupArtifact"`
	SetupDigest      string      `json:"setupDigest"`
	Transcript       string      `json:"transcript"`
	TranscriptDigest string      `json:"transcriptDigest"`
	CurrentHands     [3][]string `json:"currentHands"`
	LandlordCards    []string    `json:"landlordCards"`
}
''',
    '''type TerminalPayload struct {
	Version          string      `json:"v"`
	HandID           string      `json:"handId"`
	Status           string      `json:"status"`
	Reason           string      `json:"reason"`
	StateVersion     uint64      `json:"stateVersion"`
	SetupArtifact    string      `json:"setupArtifact"`
	SetupDigest      string      `json:"setupDigest"`
	Transcript       string      `json:"transcript"`
	TranscriptDigest string      `json:"transcriptDigest"`
	CurrentHands     [3][]string `json:"currentHands"`
	LandlordCards    []string    `json:"landlordCards"`
}

type CompletedPayload struct {
	Version          string             `json:"v"`
	HandID           string             `json:"handId"`
	Status           string             `json:"status"`
	StateVersion     uint64             `json:"stateVersion"`
	SetupArtifact    string             `json:"setupArtifact"`
	SetupDigest      string             `json:"setupDigest"`
	Transcript       string             `json:"transcript"`
	TranscriptDigest string             `json:"transcriptDigest"`
	Bidding          bidding.Snapshot   `json:"bidding"`
	Playing          playing.Snapshot   `json:"playing"`
	Settlement       settlement.Result  `json:"settlement"`
	FinalHands       [3][]string        `json:"finalHands"`
	LandlordCards    []string           `json:"landlordCards"`
	LandlordSeat     uint8              `json:"landlordSeat"`
	WinningScore     bidding.Score      `json:"winningScore"`
	WinnerSeat       uint8              `json:"winnerSeat"`
}
''',
)
start = text.index('func (g *Game) applyPlaying(command gamecore.Command)')
end = text.index('\nfunc (g *Game) View(', start)
new_apply = '''func (g *Game) applyPlaying(command gamecore.Command) (gamecore.CommandOutcome, error) {
	if g.play == nil {
		return gamecore.CommandOutcome{}, fmt.Errorf("%w: missing playing state", ErrUnsupportedCommand)
	}
	version, err := commandPayloadVersion(command.Payload)
	if err != nil {
		return gamecore.CommandOutcome{}, err
	}
	current := g.play.Snapshot()
	if command.ActorPosition != current.CurrentSeat {
		return gamecore.CommandOutcome{}, fmt.Errorf("%w: got %d want %d", playing.ErrWrongTurn, command.ActorPosition, current.CurrentSeat)
	}

	candidatePlay := g.play.Clone()
	candidateHands := cloneHands(g.current)
	var snapshot playing.Snapshot
	switch version {
	case PlayCommandVersion:
		_, cards, err := decodePlayCommand(command.Payload)
		if err != nil {
			return gamecore.CommandOutcome{}, err
		}
		remaining, err := removeHeldCards(candidateHands[command.ActorPosition-1], cards)
		if err != nil {
			return gamecore.CommandOutcome{}, err
		}
		snapshot, err = candidatePlay.Play(command.ActorPosition, cards, len(remaining) == 0)
		if err != nil {
			return gamecore.CommandOutcome{}, err
		}
		candidateHands[command.ActorPosition-1] = remaining
	case PassCommandVersion:
		if _, err := decodePassCommand(command.Payload); err != nil {
			return gamecore.CommandOutcome{}, err
		}
		snapshot, err = candidatePlay.Pass(command.ActorPosition)
		if err != nil {
			return gamecore.CommandOutcome{}, err
		}
	case BidCommandVersion:
		return gamecore.CommandOutcome{}, fmt.Errorf("%w: bidding already completed", ErrUnsupportedCommand)
	default:
		return gamecore.CommandOutcome{}, fmt.Errorf("%w: playing command %q", gamecore.ErrUnsupportedVersion, version)
	}

	nextVersion := g.version + 1
	phase := PhasePlaying
	var settlementResult *settlement.Result
	var finalPayload []byte
	if snapshot.Complete {
		calculated, err := settlement.Calculate(settlement.Input{
			LandlordSeat: g.landlordSeat,
			WinningScore: g.winningScore,
			Playing:      snapshot,
		})
		if err != nil {
			return gamecore.CommandOutcome{}, err
		}
		settlementResult = &calculated
		phase = PhaseCompleted
		finalPayload, err = g.buildCompletedPayload(nextVersion, snapshot, calculated, candidateHands)
		if err != nil {
			return gamecore.CommandOutcome{}, err
		}
	}
	resultPayload, err := json.Marshal(PlayResult{
		Version:      PlayResultVersion,
		HandID:       string(g.id),
		StateVersion: nextVersion,
		Phase:        phase,
		Playing:      snapshot,
		WinnerSeat:   snapshot.WinnerSeat,
		Settlement:   settlementResult,
	})
	if err != nil {
		return gamecore.CommandOutcome{}, err
	}

	g.play = candidatePlay
	g.current = candidateHands
	g.version = nextVersion
	g.playingSeat = snapshot.CurrentSeat
	if snapshot.Complete {
		g.phase = PhaseCompleted
		g.winnerSeat = snapshot.WinnerSeat
		settled := *settlementResult
		g.settlement = &settled
		g.terminal = true
		clearMaterial(&g.material)
		return gamecore.CommandOutcome{
			Version:      g.version,
			Payload:      resultPayload,
			Terminal:     true,
			FinalPayload: finalPayload,
		}, nil
	}
	return gamecore.CommandOutcome{Version: g.version, Payload: resultPayload}, nil
}

func (g *Game) buildCompletedPayload(version uint64, play playing.Snapshot, result settlement.Result, hands [3][]carddeck.Card) ([]byte, error) {
	transcriptBytes, err := g.transcript.CanonicalBytes()
	if err != nil {
		return nil, err
	}
	var finalHands [3][]string
	for index := range hands {
		finalHands[index], err = cardCodes(hands[index])
		if err != nil {
			return nil, err
		}
	}
	landlordCards, err := cardCodes(g.setup.LandlordCards[:])
	if err != nil {
		return nil, err
	}
	return json.Marshal(CompletedPayload{
		Version:          CompletedPayloadV1,
		HandID:           string(g.id),
		Status:           string(gamecore.FinalStatusCompleted),
		StateVersion:     version,
		SetupArtifact:    base64.RawURLEncoding.EncodeToString(g.artifact.Payload()),
		SetupDigest:      hexDigest(g.artifact.Digest()),
		Transcript:       base64.RawURLEncoding.EncodeToString(transcriptBytes),
		TranscriptDigest: hex.EncodeToString(g.transcript.TranscriptDigest[:]),
		Bidding:          g.auction.Snapshot(),
		Playing:          play,
		Settlement:       result,
		FinalHands:       finalHands,
		LandlordCards:    landlordCards,
		LandlordSeat:     g.landlordSeat,
		WinningScore:     g.winningScore,
		WinnerSeat:       play.WinnerSeat,
	})
}

func cloneHands(source [3][]carddeck.Card) [3][]carddeck.Card {
	var result [3][]carddeck.Card
	for index := range source {
		result[index] = append([]carddeck.Card(nil), source[index]...)
	}
	return result
}
'''
text = text[:start] + new_apply + text[end:]
replace_once(
    '\t\tWinnerSeat:    g.winnerSeat,\n\t}, nil\n}',
    '\t\tWinnerSeat:    g.winnerSeat,\n'
    '\t\tSettlement:    cloneSettlement(g.settlement),\n'
    '\t}, nil\n}',
)
view_anchor = '''func decodeBidCommand(payload []byte) (BidCommand, error) {
'''
clone_settlement = '''func cloneSettlement(value *settlement.Result) *settlement.Result {
	if value == nil {
		return nil
	}
	result := *value
	return &result
}

'''
if text.count(view_anchor) != 1:
    raise SystemExit("decodeBidCommand anchor mismatch")
text = text.replace(view_anchor, clone_settlement + view_anchor, 1)
game_path.write_text(text, encoding="utf-8")
