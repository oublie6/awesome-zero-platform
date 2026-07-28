package livehand

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/bidding"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/carddeck"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/playing"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/randomizedsetup"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/settlement"
	"github.com/oublie6/awesome-zero-platform/server/business/gamecore"
)

const (
	PublicViewVersion     = "doudizhu-live-public-view-v1"
	PrivateViewVersion    = "doudizhu-live-private-view-v1"
	TerminalPayloadV1     = "doudizhu-live-terminal-v1"
	CompletedPayloadV1    = "doudizhu-live-completed-v1"
	BidCommandVersion     = "doudizhu-live-bid-command-v1"
	BidResultVersion      = "doudizhu-live-bid-result-v1"
	PlayCommandVersion    = "doudizhu-live-play-command-v1"
	PassCommandVersion    = "doudizhu-live-pass-command-v1"
	PlayResultVersion     = "doudizhu-live-play-result-v1"
	PhaseBidding          = "BIDDING"
	PhaseNoLandlord       = "NO_LANDLORD"
	PhasePlaying          = "PLAYING"
	PhaseGameplayComplete = "GAMEPLAY_COMPLETE"
	PhaseCompleted        = "COMPLETED"
	PhaseAborted          = "ABORTED"
)

var (
	ErrUnsupportedCommand = errors.New("doudizhu live hand: unsupported gameplay command")
	ErrViewerNotSeated    = errors.New("doudizhu live hand: viewer is not seated")
	ErrVersionConflict    = errors.New("doudizhu live hand: version conflict")
	ErrMalformedCommand   = errors.New("doudizhu live hand: malformed command")
	ErrCardNotHeld        = errors.New("doudizhu live hand: card not held")
)

type Game struct {
	id           gamecore.InstanceID
	seats        [3]domain.HandSeat
	phase        string
	version      uint64
	artifact     gamecore.SetupArtifact
	setup        randomizedsetup.Setup
	material     gamecore.FairnessMaterial
	transcript   carddeck.Transcript
	auction      *bidding.State
	play         *playing.State
	current      [3][]carddeck.Card
	landlordSeat uint8
	winningScore bidding.Score
	playingSeat  uint8
	winnerSeat   uint8
	settlement   *settlement.Result
	terminal     bool
}

type BidCommand struct {
	Version string        `json:"v"`
	Score   bidding.Score `json:"score"`
}

type BidResult struct {
	Version             string           `json:"v"`
	HandID              string           `json:"handId"`
	StateVersion        uint64           `json:"stateVersion"`
	Phase               string           `json:"phase"`
	Bidding             bidding.Snapshot `json:"bidding"`
	LandlordSeat        uint8            `json:"landlordSeat,omitempty"`
	WinningScore        bidding.Score    `json:"winningScore,omitempty"`
	PlayingSeat         uint8            `json:"playingSeat,omitempty"`
	RequiresTermination bool             `json:"requiresTermination"`
}

type PlayCommand struct {
	Version string   `json:"v"`
	Cards   []string `json:"cards"`
}

type PassCommand struct {
	Version string `json:"v"`
}

type PlayResult struct {
	Version      string             `json:"v"`
	HandID       string             `json:"handId"`
	StateVersion uint64             `json:"stateVersion"`
	Phase        string             `json:"phase"`
	Playing      playing.Snapshot   `json:"playing"`
	WinnerSeat   uint8              `json:"winnerSeat,omitempty"`
	Settlement   *settlement.Result `json:"settlement,omitempty"`
}

type SeatView struct {
	Position       uint8            `json:"position"`
	AccountID      domain.AccountID `json:"accountId"`
	RemainingCards int              `json:"remainingCards"`
}

type PublicView struct {
	Version       string             `json:"v"`
	HandID        string             `json:"handId"`
	Phase         string             `json:"phase"`
	StateVersion  uint64             `json:"stateVersion"`
	Seats         [3]SeatView        `json:"seats"`
	SetupDigest   string             `json:"setupDigest"`
	DeckDigest    string             `json:"deckDigest"`
	DealDigest    string             `json:"dealDigest"`
	LandlordCount int                `json:"landlordCardCount"`
	Bidding       bidding.Snapshot   `json:"bidding"`
	LandlordSeat  uint8              `json:"landlordSeat,omitempty"`
	WinningScore  bidding.Score      `json:"winningScore,omitempty"`
	PlayingSeat   uint8              `json:"playingSeat,omitempty"`
	LandlordCards []string           `json:"landlordCards,omitempty"`
	Playing       *playing.Snapshot  `json:"playing,omitempty"`
	WinnerSeat    uint8              `json:"winnerSeat,omitempty"`
	Settlement    *settlement.Result `json:"settlement,omitempty"`
}

type PrivateView struct {
	Version  string     `json:"v"`
	Public   PublicView `json:"public"`
	Position uint8      `json:"position"`
	Cards    []string   `json:"cards"`
}

type TerminalPayload struct {
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
	Version          string            `json:"v"`
	HandID           string            `json:"handId"`
	Status           string            `json:"status"`
	StateVersion     uint64            `json:"stateVersion"`
	SetupArtifact    string            `json:"setupArtifact"`
	SetupDigest      string            `json:"setupDigest"`
	Transcript       string            `json:"transcript"`
	TranscriptDigest string            `json:"transcriptDigest"`
	Bidding          bidding.Snapshot  `json:"bidding"`
	Playing          playing.Snapshot  `json:"playing"`
	Settlement       settlement.Result `json:"settlement"`
	FinalHands       [3][]string       `json:"finalHands"`
	LandlordCards    []string          `json:"landlordCards"`
	LandlordSeat     uint8             `json:"landlordSeat"`
	WinningScore     bidding.Score     `json:"winningScore"`
	WinnerSeat       uint8             `json:"winnerSeat"`
}

func New(snapshot domain.HandSnapshot, material gamecore.FairnessMaterial, artifact gamecore.SetupArtifact) (*Game, error) {
	if snapshot.Phase != domain.HandDealing {
		return nil, fmt.Errorf("%w: hand phase %s", domain.ErrWrongPhase, snapshot.Phase)
	}
	if _, err := domain.RestoreHand(snapshot); err != nil {
		return nil, err
	}
	if err := material.Validate(); err != nil {
		return nil, err
	}
	if material.InstanceID != gamecore.InstanceID(snapshot.ID) || !material.Descriptor.Equal(randomizedsetup.Descriptor()) {
		return nil, fmt.Errorf("%w: live hand fairness identity", gamecore.ErrInvalidArgument)
	}
	if gamecore.Digest(snapshot.ServerCommitment) != material.ServerCommitment {
		return nil, fmt.Errorf("%w: server commitment mismatch", gamecore.ErrVerificationFailed)
	}
	if snapshot.Beacon == nil || snapshot.Beacon.Provider != material.Beacon.Provider || snapshot.Beacon.Round != material.Beacon.Round || gamecore.Digest(snapshot.Beacon.Digest) != material.Beacon.Digest || snapshot.Beacon.ProofRef != material.Beacon.ProofRef {
		return nil, fmt.Errorf("%w: beacon evidence mismatch", gamecore.ErrVerificationFailed)
	}
	if snapshot.RevealKeyID != material.RevealKey.KeyID || gamecore.Digest(snapshot.RevealPublicKeySHA256) != material.RevealKey.PublicKeySHA256 {
		return nil, fmt.Errorf("%w: reveal-key audit mismatch", gamecore.ErrVerificationFailed)
	}
	for index, contribution := range snapshot.Contributions {
		participant := material.Participants[index]
		if !contribution.Committed || !contribution.Revealed || participant.Position != uint8(contribution.Seat) || participant.Contribution != gamecore.Digest(contribution.Digest) || participant.Commitment != gamecore.Digest(contribution.Commitment) {
			return nil, fmt.Errorf("%w: participant %d fairness mismatch", gamecore.ErrVerificationFailed, index+1)
		}
	}
	module := randomizedsetup.NewModule()
	if err := module.VerifySetup(material, artifact); err != nil {
		return nil, err
	}
	setup, err := randomizedsetup.DecodeArtifact(artifact)
	if err != nil {
		return nil, err
	}
	transcript, err := buildTranscript(material)
	if err != nil {
		return nil, err
	}
	if transcript.Deck != setup.Deck || transcript.DeckDigest != setup.DeckDigest || transcript.Deal.Hands() != setup.Hands || transcript.Deal.LandlordCards() != setup.LandlordCards || transcript.DealDigest != setup.DealDigest {
		return nil, fmt.Errorf("%w: transcript does not match setup", gamecore.ErrVerificationFailed)
	}
	auction, err := bidding.New(setup.DealDigest)
	if err != nil {
		return nil, err
	}
	var current [3][]carddeck.Card
	for index := range setup.Hands {
		current[index] = append([]carddeck.Card(nil), setup.Hands[index][:]...)
	}
	game := &Game{
		id:         material.InstanceID,
		seats:      snapshot.Seats,
		phase:      PhaseBidding,
		version:    1,
		artifact:   artifact,
		setup:      setup,
		material:   material.Clone(),
		transcript: transcript,
		auction:    auction,
		current:    current,
	}
	return game, nil
}

func (g *Game) Descriptor() gamecore.Descriptor { return randomizedsetup.Descriptor() }
func (g *Game) InstanceID() gamecore.InstanceID { return g.id }

func (g *Game) Apply(command gamecore.Command) (gamecore.CommandOutcome, error) {
	if g == nil || g.terminal {
		return gamecore.CommandOutcome{}, fmt.Errorf("%w: terminal live hand", gamecore.ErrInstanceNotFound)
	}
	if g.phase != PhaseBidding && g.phase != PhasePlaying {
		return gamecore.CommandOutcome{}, fmt.Errorf("%w: phase %s", ErrUnsupportedCommand, g.phase)
	}
	if command.ExpectedVersion != g.version {
		return gamecore.CommandOutcome{}, fmt.Errorf("%w: got %d want %d", ErrVersionConflict, command.ExpectedVersion, g.version)
	}
	if g.phase == PhasePlaying {
		return g.applyPlaying(command)
	}
	return g.applyBid(command)
}

func (g *Game) applyBid(command gamecore.Command) (gamecore.CommandOutcome, error) {
	bid, err := decodeBidCommand(command.Payload)
	if err != nil {
		return gamecore.CommandOutcome{}, err
	}
	snapshot, err := g.auction.Submit(command.ActorPosition, bid.Score)
	if err != nil {
		return gamecore.CommandOutcome{}, err
	}

	g.version++
	requiresTermination := false
	if snapshot.Complete {
		switch {
		case snapshot.NoLandlord:
			g.phase = PhaseNoLandlord
			requiresTermination = true
		case snapshot.Landlord != 0:
			turns, err := playing.NewState(snapshot.Landlord)
			if err != nil {
				return gamecore.CommandOutcome{}, err
			}
			index := snapshot.Landlord - 1
			g.current[index] = append(g.current[index], g.setup.LandlordCards[:]...)
			g.landlordSeat = snapshot.Landlord
			g.winningScore = snapshot.HighestScore
			g.playingSeat = snapshot.Landlord
			g.play = turns
			g.phase = PhasePlaying
		}
	}

	payload, err := json.Marshal(BidResult{
		Version:             BidResultVersion,
		HandID:              string(g.id),
		StateVersion:        g.version,
		Phase:               g.phase,
		Bidding:             snapshot,
		LandlordSeat:        g.landlordSeat,
		WinningScore:        g.winningScore,
		PlayingSeat:         g.playingSeat,
		RequiresTermination: requiresTermination,
	})
	if err != nil {
		return gamecore.CommandOutcome{}, err
	}
	return gamecore.CommandOutcome{Version: g.version, Payload: payload}, nil
}

func (g *Game) applyPlaying(command gamecore.Command) (gamecore.CommandOutcome, error) {
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

func (g *Game) View(request gamecore.ViewRequest) (gamecore.GameView, error) {
	if g == nil || g.terminal {
		return gamecore.GameView{}, fmt.Errorf("%w: terminal live hand", gamecore.ErrInstanceNotFound)
	}
	public, err := g.publicView()
	if err != nil {
		return gamecore.GameView{}, err
	}
	if request.PublicOnly {
		payload, err := json.Marshal(public)
		return gamecore.GameView{Version: g.version, Payload: payload}, err
	}
	if request.ViewerPosition < 1 || request.ViewerPosition > 3 {
		return gamecore.GameView{}, fmt.Errorf("%w: viewer position %d", gamecore.ErrInvalidArgument, request.ViewerPosition)
	}
	cards, err := cardCodes(g.current[request.ViewerPosition-1])
	if err != nil {
		return gamecore.GameView{}, err
	}
	payload, err := json.Marshal(PrivateView{Version: PrivateViewVersion, Public: public, Position: request.ViewerPosition, Cards: cards})
	return gamecore.GameView{Version: g.version, Payload: payload}, err
}

func (g *Game) Abort(reason string) (gamecore.AbortOutcome, error) {
	if g == nil || g.terminal {
		return gamecore.AbortOutcome{}, fmt.Errorf("%w: terminal live hand", gamecore.ErrInstanceNotFound)
	}
	reason = strings.TrimSpace(reason)
	if reason == "" || len(reason) > 128 {
		return gamecore.AbortOutcome{}, fmt.Errorf("%w: abort reason", gamecore.ErrInvalidArgument)
	}
	transcriptBytes, err := g.transcript.CanonicalBytes()
	if err != nil {
		return gamecore.AbortOutcome{}, err
	}
	var current [3][]string
	for index := range g.current {
		current[index], err = cardCodes(g.current[index])
		if err != nil {
			return gamecore.AbortOutcome{}, err
		}
	}
	landlord, err := cardCodes(g.setup.LandlordCards[:])
	if err != nil {
		return gamecore.AbortOutcome{}, err
	}
	g.version++
	payload, err := json.Marshal(TerminalPayload{
		Version:          TerminalPayloadV1,
		HandID:           string(g.id),
		Status:           string(gamecore.FinalStatusAborted),
		Reason:           reason,
		StateVersion:     g.version,
		SetupArtifact:    base64.RawURLEncoding.EncodeToString(g.artifact.Payload()),
		SetupDigest:      hexDigest(g.artifact.Digest()),
		Transcript:       base64.RawURLEncoding.EncodeToString(transcriptBytes),
		TranscriptDigest: hex.EncodeToString(g.transcript.TranscriptDigest[:]),
		CurrentHands:     current,
		LandlordCards:    landlord,
	})
	if err != nil {
		g.version--
		return gamecore.AbortOutcome{}, err
	}
	g.phase = PhaseAborted
	g.terminal = true
	clearMaterial(&g.material)
	return gamecore.AbortOutcome{Version: g.version, FinalPayload: payload}, nil
}

func (g *Game) PositionForAccount(accountID domain.AccountID) (uint8, error) {
	if g == nil {
		return 0, ErrViewerNotSeated
	}
	for _, seat := range g.seats {
		if seat.AccountID == accountID {
			return uint8(seat.Seat), nil
		}
	}
	return 0, ErrViewerNotSeated
}

func (g *Game) publicView() (PublicView, error) {
	var seats [3]SeatView
	for index, seat := range g.seats {
		seats[index] = SeatView{Position: uint8(seat.Seat), AccountID: seat.AccountID, RemainingCards: len(g.current[index])}
	}
	var landlordCards []string
	var playSnapshot *playing.Snapshot
	var err error
	if g.landlordSeat != 0 {
		landlordCards, err = cardCodes(g.setup.LandlordCards[:])
		if err != nil {
			return PublicView{}, err
		}
	}
	if g.play != nil {
		snapshot := g.play.Snapshot()
		playSnapshot = &snapshot
	}
	return PublicView{
		Version:       PublicViewVersion,
		HandID:        string(g.id),
		Phase:         g.phase,
		StateVersion:  g.version,
		Seats:         seats,
		SetupDigest:   hexDigest(g.artifact.Digest()),
		DeckDigest:    hex.EncodeToString(g.setup.DeckDigest[:]),
		DealDigest:    hex.EncodeToString(g.setup.DealDigest[:]),
		LandlordCount: carddeck.LandlordCardCount,
		Bidding:       g.auction.Snapshot(),
		LandlordSeat:  g.landlordSeat,
		WinningScore:  g.winningScore,
		PlayingSeat:   g.playingSeat,
		LandlordCards: landlordCards,
		Playing:       playSnapshot,
		WinnerSeat:    g.winnerSeat,
		Settlement:    cloneSettlement(g.settlement),
	}, nil
}

func commandPayloadVersion(payload []byte) (string, error) {
	if len(payload) == 0 {
		return "", fmt.Errorf("%w: empty payload", ErrMalformedCommand)
	}
	var header struct {
		Version string `json:"v"`
	}
	if err := decodeStrictJSON(payload, &header, false); err != nil {
		return "", err
	}
	if strings.TrimSpace(header.Version) == "" {
		return "", fmt.Errorf("%w: missing command version", ErrMalformedCommand)
	}
	return header.Version, nil
}

func decodePlayCommand(payload []byte) (PlayCommand, []carddeck.Card, error) {
	var command PlayCommand
	if err := decodeStrictJSON(payload, &command, true); err != nil {
		return PlayCommand{}, nil, err
	}
	if command.Version != PlayCommandVersion {
		return PlayCommand{}, nil, fmt.Errorf("%w: play command %q", gamecore.ErrUnsupportedVersion, command.Version)
	}
	if len(command.Cards) == 0 || len(command.Cards) > 20 {
		return PlayCommand{}, nil, fmt.Errorf("%w: play card count %d", ErrMalformedCommand, len(command.Cards))
	}
	cards := make([]carddeck.Card, len(command.Cards))
	for index, code := range command.Cards {
		card, err := carddeck.ParseCard(code)
		if err != nil {
			return PlayCommand{}, nil, fmt.Errorf("%w: card[%d]: %v", ErrMalformedCommand, index, err)
		}
		cards[index] = card
	}
	return command, cards, nil
}

func decodePassCommand(payload []byte) (PassCommand, error) {
	var command PassCommand
	if err := decodeStrictJSON(payload, &command, true); err != nil {
		return PassCommand{}, err
	}
	if command.Version != PassCommandVersion {
		return PassCommand{}, fmt.Errorf("%w: pass command %q", gamecore.ErrUnsupportedVersion, command.Version)
	}
	return command, nil
}

func decodeStrictJSON(payload []byte, target any, disallowUnknown bool) error {
	if len(payload) == 0 {
		return fmt.Errorf("%w: empty payload", ErrMalformedCommand)
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	if disallowUnknown {
		decoder.DisallowUnknownFields()
	}
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("%w: %v", ErrMalformedCommand, err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return fmt.Errorf("%w: trailing JSON", ErrMalformedCommand)
	}
	return nil
}

func removeHeldCards(hand, played []carddeck.Card) ([]carddeck.Card, error) {
	remaining := append([]carddeck.Card(nil), hand...)
	for _, card := range played {
		found := -1
		for index, held := range remaining {
			if held == card {
				found = index
				break
			}
		}
		if found == -1 {
			code, _ := card.Code()
			return nil, fmt.Errorf("%w: %s", ErrCardNotHeld, code)
		}
		remaining = append(remaining[:found], remaining[found+1:]...)
	}
	return remaining, nil
}

func cloneSettlement(value *settlement.Result) *settlement.Result {
	if value == nil {
		return nil
	}
	result := *value
	return &result
}

func decodeBidCommand(payload []byte) (BidCommand, error) {
	if len(payload) == 0 {
		return BidCommand{}, fmt.Errorf("%w: empty payload", ErrMalformedCommand)
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	var command BidCommand
	if err := decoder.Decode(&command); err != nil {
		return BidCommand{}, fmt.Errorf("%w: %v", ErrMalformedCommand, err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return BidCommand{}, fmt.Errorf("%w: trailing JSON", ErrMalformedCommand)
	}
	if command.Version != BidCommandVersion {
		return BidCommand{}, fmt.Errorf("%w: bid command %q", gamecore.ErrUnsupportedVersion, command.Version)
	}
	return command, nil
}

func buildTranscript(material gamecore.FairnessMaterial) (carddeck.Transcript, error) {
	input := carddeck.TranscriptInput{
		HandID:           string(material.InstanceID),
		ServerSeed:       carddeck.Seed(material.ServerSeed),
		ServerCommitment: carddeck.Commitment(material.ServerCommitment),
		Beacon: carddeck.BeaconEvidence{
			Provider: material.Beacon.Provider,
			Round:    material.Beacon.Round,
			Digest:   carddeck.BeaconDigest(material.Beacon.Digest),
			ProofRef: material.Beacon.ProofRef,
		},
		RevealKey: carddeck.RevealKeyAudit{
			KeyID:           material.RevealKey.KeyID,
			PublicKeySHA256: carddeck.Digest(material.RevealKey.PublicKeySHA256),
		},
	}
	for index, participant := range material.Participants {
		input.Contributions[index] = carddeck.ContributionEvidence{
			Seat:       participant.Position,
			Digest:     carddeck.ContributionDigest(participant.Contribution),
			Commitment: carddeck.Commitment(participant.Commitment),
		}
	}
	transcript, err := carddeck.BuildTranscript(input)
	if err != nil {
		return carddeck.Transcript{}, err
	}
	if err := carddeck.VerifyTranscript(transcript); err != nil {
		return carddeck.Transcript{}, err
	}
	return transcript, nil
}

func cardCodes(cards []carddeck.Card) ([]string, error) {
	codes := make([]string, len(cards))
	for index, card := range cards {
		code, err := card.Code()
		if err != nil {
			return nil, err
		}
		codes[index] = code
	}
	return codes, nil
}

func hexDigest(value gamecore.Digest) string { return hex.EncodeToString(value[:]) }

func clearMaterial(material *gamecore.FairnessMaterial) {
	if material == nil {
		return
	}
	clear(material.ServerSeed[:])
	for index := range material.Participants {
		clear(material.Participants[index].Contribution[:])
		clear(material.Participants[index].Commitment[:])
	}
	material.Participants = nil
}
