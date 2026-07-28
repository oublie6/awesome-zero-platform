package bidding

import (
	"crypto/sha256"
	"errors"
	"fmt"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/carddeck"
	"github.com/oublie6/awesome-zero-platform/server/business/gamecore"
)

const (
	RulesVersion       = "doudizhu-score-bidding-v1"
	FirstBidderVersion = "doudizhu-bidding-first-seat-v1"
)

const firstBidderDomain = "doudizhu/bidding-first-seat/v1"

var (
	ErrInvalidDealDigest = errors.New("doudizhu bidding: invalid deal digest")
	ErrInvalidPosition   = errors.New("doudizhu bidding: invalid position")
	ErrInvalidScore      = errors.New("doudizhu bidding: invalid score")
	ErrWrongTurn         = errors.New("doudizhu bidding: wrong turn")
	ErrBidNotHigher      = errors.New("doudizhu bidding: positive bid must exceed current highest score")
	ErrBiddingComplete   = errors.New("doudizhu bidding: bidding is complete")
)

type Score uint8

const (
	ScorePass Score = iota
	ScoreOne
	ScoreTwo
	ScoreThree
)

func (s Score) Valid() bool { return s <= ScoreThree }

type Action struct {
	Position uint8 `json:"position"`
	Score    Score `json:"score"`
}

type Snapshot struct {
	Version       string   `json:"v"`
	FirstBidder   uint8    `json:"firstBidder"`
	CurrentBidder uint8    `json:"currentBidder"`
	HighestScore  Score    `json:"highestScore"`
	HighestBidder uint8    `json:"highestBidder"`
	Actions       []Action `json:"actions"`
	Complete      bool     `json:"complete"`
	NoLandlord    bool     `json:"noLandlord"`
	Landlord      uint8    `json:"landlord"`
}

type State struct {
	firstBidder   uint8
	currentBidder uint8
	highestScore  Score
	highestBidder uint8
	actions       [3]Action
	actionCount   int
	complete      bool
	noLandlord    bool
	landlord      uint8
}

func New(dealDigest carddeck.DealDigest) (*State, error) {
	first, err := DeriveFirstBidder(dealDigest)
	if err != nil {
		return nil, err
	}
	return newState(first), nil
}

func newState(first uint8) *State {
	return &State{firstBidder: first, currentBidder: first}
}

func DeriveFirstBidder(dealDigest carddeck.DealDigest) (uint8, error) {
	if digestIsZero(dealDigest) {
		return 0, ErrInvalidDealDigest
	}
	h := sha256.New()
	_, _ = h.Write([]byte(firstBidderDomain))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write(dealDigest[:])
	var seed gamecore.Seed
	copy(seed[:], h.Sum(nil))
	stream, err := gamecore.NewStream(seed)
	if err != nil {
		return 0, fmt.Errorf("derive first bidder stream: %w", err)
	}
	value, err := gamecore.Uniform(stream, 3)
	if err != nil {
		return 0, fmt.Errorf("derive first bidder position: %w", err)
	}
	return uint8(value + 1), nil
}

func (s *State) Submit(position uint8, score Score) (Snapshot, error) {
	if s == nil {
		return Snapshot{}, fmt.Errorf("%w: nil state", gamecore.ErrInvalidArgument)
	}
	if s.complete {
		return Snapshot{}, ErrBiddingComplete
	}
	if position < 1 || position > 3 {
		return Snapshot{}, fmt.Errorf("%w: %d", ErrInvalidPosition, position)
	}
	if !score.Valid() {
		return Snapshot{}, fmt.Errorf("%w: %d", ErrInvalidScore, score)
	}
	if position != s.currentBidder {
		return Snapshot{}, fmt.Errorf("%w: got %d want %d", ErrWrongTurn, position, s.currentBidder)
	}
	if score > ScorePass && score <= s.highestScore {
		return Snapshot{}, fmt.Errorf("%w: got %d current %d", ErrBidNotHigher, score, s.highestScore)
	}

	s.actions[s.actionCount] = Action{Position: position, Score: score}
	s.actionCount++
	if score > s.highestScore {
		s.highestScore = score
		s.highestBidder = position
	}

	if score == ScoreThree || s.actionCount == len(s.actions) {
		s.complete = true
		s.currentBidder = 0
		if s.highestScore == ScorePass {
			s.noLandlord = true
		} else {
			s.landlord = s.highestBidder
		}
		return s.Snapshot(), nil
	}

	s.currentBidder = nextPosition(position)
	return s.Snapshot(), nil
}

func (s *State) Snapshot() Snapshot {
	if s == nil {
		return Snapshot{}
	}
	actions := make([]Action, s.actionCount)
	copy(actions, s.actions[:s.actionCount])
	return Snapshot{
		Version:       RulesVersion,
		FirstBidder:   s.firstBidder,
		CurrentBidder: s.currentBidder,
		HighestScore:  s.highestScore,
		HighestBidder: s.highestBidder,
		Actions:       actions,
		Complete:      s.complete,
		NoLandlord:    s.noLandlord,
		Landlord:      s.landlord,
	}
}

func nextPosition(position uint8) uint8 { return position%3 + 1 }

func digestIsZero(value carddeck.DealDigest) bool {
	var combined byte
	for _, item := range value {
		combined |= item
	}
	return combined == 0
}
