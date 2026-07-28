package playing

import (
	"errors"
	"fmt"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/carddeck"
)

const StateVersion = "doudizhu-playing-state-v1"

type ActionType string

const (
	ActionPlay ActionType = "PLAY"
	ActionPass ActionType = "PASS"
)

var (
	ErrInvalidSeat     = errors.New("doudizhu playing: invalid seat")
	ErrWrongTurn       = errors.New("doudizhu playing: wrong turn")
	ErrCannotPass      = errors.New("doudizhu playing: cannot pass")
	ErrDoesNotBeat     = errors.New("doudizhu playing: play does not beat leader")
	ErrPlayingComplete = errors.New("doudizhu playing: playing is complete")
)

type Action struct {
	Number  uint64          `json:"number"`
	Type    ActionType      `json:"type"`
	Seat    uint8           `json:"seat"`
	Cards   []carddeck.Card `json:"cards,omitempty"`
	Pattern *Pattern        `json:"pattern,omitempty"`
}

type Snapshot struct {
	Version        string          `json:"v"`
	Revision       uint64          `json:"revision"`
	CurrentSeat    uint8           `json:"currentSeat,omitempty"`
	LeadingSeat    uint8           `json:"leadingSeat,omitempty"`
	LeadingCards   []carddeck.Card `json:"leadingCards,omitempty"`
	LeadingPattern *Pattern        `json:"leadingPattern,omitempty"`
	PassCount      uint8           `json:"passCount"`
	Complete       bool            `json:"complete"`
	WinnerSeat     uint8           `json:"winnerSeat,omitempty"`
	History        []Action        `json:"history"`
}

type State struct {
	revision       uint64
	currentSeat    uint8
	leadingSeat    uint8
	leadingCards   []carddeck.Card
	leadingPattern *Pattern
	passCount      uint8
	complete       bool
	winnerSeat     uint8
	history        []Action
}

func NewState(firstSeat uint8) (*State, error) {
	if !validSeat(firstSeat) {
		return nil, fmt.Errorf("%w: %d", ErrInvalidSeat, firstSeat)
	}
	return &State{currentSeat: firstSeat}, nil
}

func (s *State) Play(seat uint8, cards []carddeck.Card, handEmpty bool) (Snapshot, error) {
	if err := s.validateActor(seat); err != nil {
		return Snapshot{}, err
	}
	candidate, err := Analyze(cards)
	if err != nil {
		return Snapshot{}, err
	}
	if s.leadingPattern != nil {
		beats, err := CanBeat(candidate, *s.leadingPattern)
		if err != nil {
			return Snapshot{}, err
		}
		if !beats {
			return Snapshot{}, ErrDoesNotBeat
		}
	}

	s.revision++
	storedCards := append([]carddeck.Card(nil), cards...)
	storedPattern := candidate
	s.history = append(s.history, Action{
		Number:  s.revision,
		Type:    ActionPlay,
		Seat:    seat,
		Cards:   storedCards,
		Pattern: &storedPattern,
	})
	s.leadingSeat = seat
	s.leadingCards = append(s.leadingCards[:0], cards...)
	s.leadingPattern = &storedPattern
	s.passCount = 0
	if handEmpty {
		s.complete = true
		s.winnerSeat = seat
		s.currentSeat = 0
	} else {
		s.currentSeat = nextSeat(seat)
	}
	return s.Snapshot(), nil
}

func (s *State) Pass(seat uint8) (Snapshot, error) {
	if err := s.validateActor(seat); err != nil {
		return Snapshot{}, err
	}
	if s.leadingPattern == nil || s.leadingSeat == 0 {
		return Snapshot{}, ErrCannotPass
	}

	s.revision++
	s.history = append(s.history, Action{Number: s.revision, Type: ActionPass, Seat: seat})
	s.passCount++
	if s.passCount == 2 {
		s.currentSeat = s.leadingSeat
		s.leadingSeat = 0
		s.leadingCards = nil
		s.leadingPattern = nil
		s.passCount = 0
	} else {
		s.currentSeat = nextSeat(seat)
	}
	return s.Snapshot(), nil
}

func (s *State) Clone() *State {
	if s == nil {
		return nil
	}
	clone := &State{
		revision:     s.revision,
		currentSeat:  s.currentSeat,
		leadingSeat:  s.leadingSeat,
		leadingCards: append([]carddeck.Card(nil), s.leadingCards...),
		passCount:    s.passCount,
		complete:     s.complete,
		winnerSeat:   s.winnerSeat,
		history:      make([]Action, len(s.history)),
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

func (s *State) Snapshot() Snapshot {
	if s == nil {
		return Snapshot{Version: StateVersion}
	}
	result := Snapshot{
		Version:      StateVersion,
		Revision:     s.revision,
		CurrentSeat:  s.currentSeat,
		LeadingSeat:  s.leadingSeat,
		LeadingCards: append([]carddeck.Card(nil), s.leadingCards...),
		PassCount:    s.passCount,
		Complete:     s.complete,
		WinnerSeat:   s.winnerSeat,
		History:      make([]Action, len(s.history)),
	}
	if s.leadingPattern != nil {
		pattern := *s.leadingPattern
		result.LeadingPattern = &pattern
	}
	for index, action := range s.history {
		result.History[index] = cloneAction(action)
	}
	return result
}

func (s *State) validateActor(seat uint8) error {
	if s == nil {
		return fmt.Errorf("%w: nil state", ErrPlayingComplete)
	}
	if !validSeat(seat) {
		return fmt.Errorf("%w: %d", ErrInvalidSeat, seat)
	}
	if s.complete {
		return ErrPlayingComplete
	}
	if seat != s.currentSeat {
		return fmt.Errorf("%w: got %d want %d", ErrWrongTurn, seat, s.currentSeat)
	}
	return nil
}

func cloneAction(action Action) Action {
	result := action
	result.Cards = append([]carddeck.Card(nil), action.Cards...)
	if action.Pattern != nil {
		pattern := *action.Pattern
		result.Pattern = &pattern
	}
	return result
}

func validSeat(seat uint8) bool { return seat >= 1 && seat <= 3 }

func nextSeat(seat uint8) uint8 { return seat%3 + 1 }
