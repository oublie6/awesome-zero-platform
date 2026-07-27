package domain

import (
	"fmt"
	"strings"
)

type AccountID string
type RoomID string
type HandID string

type Seat uint8

const (
	SeatOne   Seat = 1
	SeatTwo   Seat = 2
	SeatThree Seat = 3
)

var fixedSeats = [3]Seat{SeatOne, SeatTwo, SeatThree}

func AllSeats() [3]Seat {
	return fixedSeats
}

func (s Seat) Valid() bool {
	return s >= SeatOne && s <= SeatThree
}

func validateIdentifier(name, value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("%w: %s is empty", ErrInvalidArgument, name)
	}
	if len(value) > 128 {
		return fmt.Errorf("%w: %s exceeds 128 bytes", ErrInvalidArgument, name)
	}
	return nil
}
