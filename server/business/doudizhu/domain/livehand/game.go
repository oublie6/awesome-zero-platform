package livehand

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/carddeck"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/randomizedsetup"
	"github.com/oublie6/awesome-zero-platform/server/business/gamecore"
)

const (
	PublicViewVersion   = "doudizhu-live-public-view-v1"
	PrivateViewVersion  = "doudizhu-live-private-view-v1"
	TerminalPayloadV1   = "doudizhu-live-terminal-v1"
	PhaseBidding        = "BIDDING"
	PhaseAborted        = "ABORTED"
)

var (
	ErrUnsupportedCommand = errors.New("doudizhu live hand: gameplay command is not implemented")
	ErrViewerNotSeated     = errors.New("doudizhu live hand: viewer is not seated")
)

type Game struct {
	id          gamecore.InstanceID
	seats       [3]domain.HandSeat
	phase       string
	version     uint64
	artifact    gamecore.SetupArtifact
	setup       randomizedsetup.Setup
	material    gamecore.FairnessMaterial
	transcript  carddeck.Transcript
	current     [3][carddeck.CardsPerSeat]carddeck.Card
	terminal    bool
}

type SeatView struct {
	Position       uint8            `json:"position"`
	AccountID      domain.AccountID `json:"accountId"`
	RemainingCards int              `json:"remainingCards"`
}

type PublicView struct {
	Version       string       `json:"v"`
	HandID        string       `json:"handId"`
	Phase         string       `json:"phase"`
	StateVersion  uint64       `json:"stateVersion"`
	Seats         [3]SeatView  `json:"seats"`
	SetupDigest   string       `json:"setupDigest"`
	DeckDigest    string       `json:"deckDigest"`
	DealDigest    string       `json:"dealDigest"`
	LandlordCount int          `json:"landlordCardCount"`
}

type PrivateView struct {
	Version  string     `json:"v"`
	Public   PublicView `json:"public"`
	Position uint8      `json:"position"`
	Cards    []string   `json:"cards"`
}

type TerminalPayload struct {
	Version          string       `json:"v"`
	HandID           string       `json:"handId"`
	Status           string       `json:"status"`
	Reason           string       `json:"reason"`
	StateVersion     uint64       `json:"stateVersion"`
	SetupArtifact    string       `json:"setupArtifact"`
	SetupDigest      string       `json:"setupDigest"`
	Transcript       string       `json:"transcript"`
	TranscriptDigest string       `json:"transcriptDigest"`
	CurrentHands     [3][]string  `json:"currentHands"`
	LandlordCards    []string     `json:"landlordCards"`
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
	game := &Game{
		id:         material.InstanceID,
		seats:      snapshot.Seats,
		phase:      PhaseBidding,
		version:    1,
		artifact:   artifact,
		setup:      setup,
		material:   material.Clone(),
		transcript: transcript,
		current:    setup.Hands,
	}
	return game, nil
}

func (g *Game) Descriptor() gamecore.Descriptor { return randomizedsetup.Descriptor() }
func (g *Game) InstanceID() gamecore.InstanceID { return g.id }

func (g *Game) Apply(gamecore.Command) (gamecore.CommandOutcome, error) {
	return gamecore.CommandOutcome{}, ErrUnsupportedCommand
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
	cards, err := cardCodes(g.current[request.ViewerPosition-1][:])
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
		current[index], err = cardCodes(g.current[index][:])
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
	}, nil
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
