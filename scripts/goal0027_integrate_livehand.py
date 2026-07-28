from pathlib import Path

path = Path("server/business/doudizhu/domain/livehand/game.go")
text = path.read_text(encoding="utf-8")


def replace_once(old: str, new: str) -> None:
    global text
    if text.count(old) != 1:
        raise SystemExit(f"expected exactly one match, found {text.count(old)} for:\n{old[:200]}")
    text = text.replace(old, new, 1)


replace_once(
    '\t"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/carddeck"\n',
    '\t"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/carddeck"\n'
    '\t"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/playing"\n',
)

replace_once(
    '\tBidResultVersion   = "doudizhu-live-bid-result-v1"\n',
    '\tBidResultVersion   = "doudizhu-live-bid-result-v1"\n'
    '\tPlayCommandVersion = "doudizhu-live-play-command-v1"\n'
    '\tPassCommandVersion = "doudizhu-live-pass-command-v1"\n'
    '\tPlayResultVersion  = "doudizhu-live-play-result-v1"\n',
)

replace_once(
    '\tPhasePlaying       = "PLAYING"\n',
    '\tPhasePlaying          = "PLAYING"\n'
    '\tPhaseGameplayComplete = "GAMEPLAY_COMPLETE"\n',
)

replace_once(
    '\tErrMalformedCommand   = errors.New("doudizhu live hand: malformed command")\n',
    '\tErrMalformedCommand   = errors.New("doudizhu live hand: malformed command")\n'
    '\tErrCardNotHeld        = errors.New("doudizhu live hand: card not held")\n',
)

replace_once(
    '\tauction      *bidding.State\n\tcurrent      [3][]carddeck.Card\n',
    '\tauction      *bidding.State\n\tplay         *playing.State\n\tcurrent      [3][]carddeck.Card\n',
)

replace_once(
    '\tplayingSeat  uint8\n\tterminal     bool\n',
    '\tplayingSeat  uint8\n\twinnerSeat   uint8\n\tterminal     bool\n',
)

replace_once(
    '''type BidResult struct {
\tVersion             string           `json:"v"`
\tHandID              string           `json:"handId"`
\tStateVersion        uint64           `json:"stateVersion"`
\tPhase               string           `json:"phase"`
\tBidding             bidding.Snapshot `json:"bidding"`
\tLandlordSeat        uint8            `json:"landlordSeat,omitempty"`
\tWinningScore        bidding.Score    `json:"winningScore,omitempty"`
\tPlayingSeat         uint8            `json:"playingSeat,omitempty"`
\tRequiresTermination bool             `json:"requiresTermination"`
}
''',
    '''type BidResult struct {
\tVersion             string           `json:"v"`
\tHandID              string           `json:"handId"`
\tStateVersion        uint64           `json:"stateVersion"`
\tPhase               string           `json:"phase"`
\tBidding             bidding.Snapshot `json:"bidding"`
\tLandlordSeat        uint8            `json:"landlordSeat,omitempty"`
\tWinningScore        bidding.Score    `json:"winningScore,omitempty"`
\tPlayingSeat         uint8            `json:"playingSeat,omitempty"`
\tRequiresTermination bool             `json:"requiresTermination"`
}

type PlayCommand struct {
\tVersion string   `json:"v"`
\tCards   []string `json:"cards"`
}

type PassCommand struct {
\tVersion string `json:"v"`
}

type PlayResult struct {
\tVersion      string           `json:"v"`
\tHandID       string           `json:"handId"`
\tStateVersion uint64           `json:"stateVersion"`
\tPhase        string           `json:"phase"`
\tPlaying      playing.Snapshot `json:"playing"`
\tWinnerSeat   uint8            `json:"winnerSeat,omitempty"`
}
''',
)

replace_once(
    '\tLandlordCards []string         `json:"landlordCards,omitempty"`\n',
    '\tLandlordCards []string          `json:"landlordCards,omitempty"`\n'
    '\tPlaying       *playing.Snapshot `json:"playing,omitempty"`\n'
    '\tWinnerSeat    uint8             `json:"winnerSeat,omitempty"`\n',
)

old_apply = '''func (g *Game) Apply(command gamecore.Command) (gamecore.CommandOutcome, error) {
\tif g == nil || g.terminal {
\t\treturn gamecore.CommandOutcome{}, fmt.Errorf("%w: terminal live hand", gamecore.ErrInstanceNotFound)
\t}
\tif g.phase != PhaseBidding {
\t\treturn gamecore.CommandOutcome{}, fmt.Errorf("%w: phase %s", ErrUnsupportedCommand, g.phase)
\t}
\tif command.ExpectedVersion != g.version {
\t\treturn gamecore.CommandOutcome{}, fmt.Errorf("%w: got %d want %d", ErrVersionConflict, command.ExpectedVersion, g.version)
\t}
\tbid, err := decodeBidCommand(command.Payload)
\tif err != nil {
\t\treturn gamecore.CommandOutcome{}, err
\t}
\tsnapshot, err := g.auction.Submit(command.ActorPosition, bid.Score)
\tif err != nil {
\t\treturn gamecore.CommandOutcome{}, err
\t}

\tg.version++
\trequiresTermination := false
\tif snapshot.Complete {
\t\tswitch {
\t\tcase snapshot.NoLandlord:
\t\t\tg.phase = PhaseNoLandlord
\t\t\trequiresTermination = true
\t\tcase snapshot.Landlord != 0:
\t\t\tindex := snapshot.Landlord - 1
\t\t\tg.current[index] = append(g.current[index], g.setup.LandlordCards[:]...)
\t\t\tg.landlordSeat = snapshot.Landlord
\t\t\tg.winningScore = snapshot.HighestScore
\t\t\tg.playingSeat = snapshot.Landlord
\t\t\tg.phase = PhasePlaying
\t\t}
\t}

\tpayload, err := json.Marshal(BidResult{
\t\tVersion:             BidResultVersion,
\t\tHandID:              string(g.id),
\t\tStateVersion:        g.version,
\t\tPhase:               g.phase,
\t\tBidding:             snapshot,
\t\tLandlordSeat:        g.landlordSeat,
\t\tWinningScore:        g.winningScore,
\t\tPlayingSeat:         g.playingSeat,
\t\tRequiresTermination: requiresTermination,
\t})
\tif err != nil {
\t\treturn gamecore.CommandOutcome{}, err
\t}
\treturn gamecore.CommandOutcome{Version: g.version, Payload: payload}, nil
}
'''

new_apply = '''func (g *Game) Apply(command gamecore.Command) (gamecore.CommandOutcome, error) {
\tif g == nil || g.terminal {
\t\treturn gamecore.CommandOutcome{}, fmt.Errorf("%w: terminal live hand", gamecore.ErrInstanceNotFound)
\t}
\tif g.phase != PhaseBidding && g.phase != PhasePlaying {
\t\treturn gamecore.CommandOutcome{}, fmt.Errorf("%w: phase %s", ErrUnsupportedCommand, g.phase)
\t}
\tif command.ExpectedVersion != g.version {
\t\treturn gamecore.CommandOutcome{}, fmt.Errorf("%w: got %d want %d", ErrVersionConflict, command.ExpectedVersion, g.version)
\t}
\tif g.phase == PhasePlaying {
\t\treturn g.applyPlaying(command)
\t}
\treturn g.applyBid(command)
}

func (g *Game) applyBid(command gamecore.Command) (gamecore.CommandOutcome, error) {
\tbid, err := decodeBidCommand(command.Payload)
\tif err != nil {
\t\treturn gamecore.CommandOutcome{}, err
\t}
\tsnapshot, err := g.auction.Submit(command.ActorPosition, bid.Score)
\tif err != nil {
\t\treturn gamecore.CommandOutcome{}, err
\t}

\tg.version++
\trequiresTermination := false
\tif snapshot.Complete {
\t\tswitch {
\t\tcase snapshot.NoLandlord:
\t\t\tg.phase = PhaseNoLandlord
\t\t\trequiresTermination = true
\t\tcase snapshot.Landlord != 0:
\t\t\tturns, err := playing.NewState(snapshot.Landlord)
\t\t\tif err != nil {
\t\t\t\treturn gamecore.CommandOutcome{}, err
\t\t\t}
\t\t\tindex := snapshot.Landlord - 1
\t\t\tg.current[index] = append(g.current[index], g.setup.LandlordCards[:]...)
\t\t\tg.landlordSeat = snapshot.Landlord
\t\t\tg.winningScore = snapshot.HighestScore
\t\t\tg.playingSeat = snapshot.Landlord
\t\t\tg.play = turns
\t\t\tg.phase = PhasePlaying
\t\t}
\t}

\tpayload, err := json.Marshal(BidResult{
\t\tVersion:             BidResultVersion,
\t\tHandID:              string(g.id),
\t\tStateVersion:        g.version,
\t\tPhase:               g.phase,
\t\tBidding:             snapshot,
\t\tLandlordSeat:        g.landlordSeat,
\t\tWinningScore:        g.winningScore,
\t\tPlayingSeat:         g.playingSeat,
\t\tRequiresTermination: requiresTermination,
\t})
\tif err != nil {
\t\treturn gamecore.CommandOutcome{}, err
\t}
\treturn gamecore.CommandOutcome{Version: g.version, Payload: payload}, nil
}

func (g *Game) applyPlaying(command gamecore.Command) (gamecore.CommandOutcome, error) {
\tif g.play == nil {
\t\treturn gamecore.CommandOutcome{}, fmt.Errorf("%w: missing playing state", ErrUnsupportedCommand)
\t}
\tversion, err := commandPayloadVersion(command.Payload)
\tif err != nil {
\t\treturn gamecore.CommandOutcome{}, err
\t}
\tcurrent := g.play.Snapshot()
\tif command.ActorPosition != current.CurrentSeat {
\t\treturn gamecore.CommandOutcome{}, fmt.Errorf("%w: got %d want %d", playing.ErrWrongTurn, command.ActorPosition, current.CurrentSeat)
\t}

\tvar snapshot playing.Snapshot
\tswitch version {
\tcase PlayCommandVersion:
\t\t_, cards, err := decodePlayCommand(command.Payload)
\t\tif err != nil {
\t\t\treturn gamecore.CommandOutcome{}, err
\t\t}
\t\tremaining, err := removeHeldCards(g.current[command.ActorPosition-1], cards)
\t\tif err != nil {
\t\t\treturn gamecore.CommandOutcome{}, err
\t\t}
\t\tsnapshot, err = g.play.Play(command.ActorPosition, cards, len(remaining) == 0)
\t\tif err != nil {
\t\t\treturn gamecore.CommandOutcome{}, err
\t\t}
\t\tg.current[command.ActorPosition-1] = remaining
\tcase PassCommandVersion:
\t\tif _, err := decodePassCommand(command.Payload); err != nil {
\t\t\treturn gamecore.CommandOutcome{}, err
\t\t}
\t\tsnapshot, err = g.play.Pass(command.ActorPosition)
\t\tif err != nil {
\t\t\treturn gamecore.CommandOutcome{}, err
\t\t}
\tcase BidCommandVersion:
\t\treturn gamecore.CommandOutcome{}, fmt.Errorf("%w: bidding already completed", ErrUnsupportedCommand)
\tdefault:
\t\treturn gamecore.CommandOutcome{}, fmt.Errorf("%w: playing command %q", gamecore.ErrUnsupportedVersion, version)
\t}

\tg.version++
\tg.playingSeat = snapshot.CurrentSeat
\tif snapshot.Complete {
\t\tg.phase = PhaseGameplayComplete
\t\tg.winnerSeat = snapshot.WinnerSeat
\t}
\tpayload, err := json.Marshal(PlayResult{
\t\tVersion:      PlayResultVersion,
\t\tHandID:       string(g.id),
\t\tStateVersion: g.version,
\t\tPhase:        g.phase,
\t\tPlaying:      snapshot,
\t\tWinnerSeat:   g.winnerSeat,
\t})
\tif err != nil {
\t\treturn gamecore.CommandOutcome{}, err
\t}
\treturn gamecore.CommandOutcome{Version: g.version, Payload: payload}, nil
}
'''
replace_once(old_apply, new_apply)

replace_once(
    '''\tvar landlordCards []string
\tvar err error
\tif g.landlordSeat != 0 {
''',
    '''\tvar landlordCards []string
\tvar playSnapshot *playing.Snapshot
\tvar err error
\tif g.landlordSeat != 0 {
''',
)

replace_once(
    '''\t\tif err != nil {
\t\t\treturn PublicView{}, err
\t\t}
\t}
\treturn PublicView{
''',
    '''\t\tif err != nil {
\t\t\treturn PublicView{}, err
\t\t}
\t}
\tif g.play != nil {
\t\tsnapshot := g.play.Snapshot()
\t\tplaySnapshot = &snapshot
\t}
\treturn PublicView{
''',
)

replace_once(
    '\t\tLandlordCards: landlordCards,\n\t}, nil\n}\n\nfunc decodeBidCommand',
    '\t\tLandlordCards: landlordCards,\n'
    '\t\tPlaying:       playSnapshot,\n'
    '\t\tWinnerSeat:    g.winnerSeat,\n'
    '\t}, nil\n}\n\n'
    '''func commandPayloadVersion(payload []byte) (string, error) {
\tif len(payload) == 0 {
\t\treturn "", fmt.Errorf("%w: empty payload", ErrMalformedCommand)
\t}
\tvar header struct {
\t\tVersion string `json:"v"`
\t}
\tif err := decodeStrictJSON(payload, &header, false); err != nil {
\t\treturn "", err
\t}
\tif strings.TrimSpace(header.Version) == "" {
\t\treturn "", fmt.Errorf("%w: missing command version", ErrMalformedCommand)
\t}
\treturn header.Version, nil
}

func decodePlayCommand(payload []byte) (PlayCommand, []carddeck.Card, error) {
\tvar command PlayCommand
\tif err := decodeStrictJSON(payload, &command, true); err != nil {
\t\treturn PlayCommand{}, nil, err
\t}
\tif command.Version != PlayCommandVersion {
\t\treturn PlayCommand{}, nil, fmt.Errorf("%w: play command %q", gamecore.ErrUnsupportedVersion, command.Version)
\t}
\tif len(command.Cards) == 0 || len(command.Cards) > 20 {
\t\treturn PlayCommand{}, nil, fmt.Errorf("%w: play card count %d", ErrMalformedCommand, len(command.Cards))
\t}
\tcards := make([]carddeck.Card, len(command.Cards))
\tfor index, code := range command.Cards {
\t\tcard, err := carddeck.ParseCard(code)
\t\tif err != nil {
\t\t\treturn PlayCommand{}, nil, fmt.Errorf("%w: card[%d]: %v", ErrMalformedCommand, index, err)
\t\t}
\t\tcards[index] = card
\t}
\treturn command, cards, nil
}

func decodePassCommand(payload []byte) (PassCommand, error) {
\tvar command PassCommand
\tif err := decodeStrictJSON(payload, &command, true); err != nil {
\t\treturn PassCommand{}, err
\t}
\tif command.Version != PassCommandVersion {
\t\treturn PassCommand{}, fmt.Errorf("%w: pass command %q", gamecore.ErrUnsupportedVersion, command.Version)
\t}
\treturn command, nil
}

func decodeStrictJSON(payload []byte, target any, disallowUnknown bool) error {
\tif len(payload) == 0 {
\t\treturn fmt.Errorf("%w: empty payload", ErrMalformedCommand)
\t}
\tdecoder := json.NewDecoder(bytes.NewReader(payload))
\tif disallowUnknown {
\t\tdecoder.DisallowUnknownFields()
\t}
\tif err := decoder.Decode(target); err != nil {
\t\treturn fmt.Errorf("%w: %v", ErrMalformedCommand, err)
\t}
\tif err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
\t\treturn fmt.Errorf("%w: trailing JSON", ErrMalformedCommand)
\t}
\treturn nil
}

func removeHeldCards(hand, played []carddeck.Card) ([]carddeck.Card, error) {
\tremaining := append([]carddeck.Card(nil), hand...)
\tfor _, card := range played {
\t\tfound := -1
\t\tfor index, held := range remaining {
\t\t\tif held == card {
\t\t\t\tfound = index
\t\t\t\tbreak
\t\t\t}
\t\t}
\t\tif found == -1 {
\t\t\tcode, _ := card.Code()
\t\t\treturn nil, fmt.Errorf("%w: %s", ErrCardNotHeld, code)
\t\t}
\t\tremaining = append(remaining[:found], remaining[found+1:]...)
\t}
\treturn remaining, nil
}

func decodeBidCommand''',
)

path.write_text(text, encoding="utf-8")
