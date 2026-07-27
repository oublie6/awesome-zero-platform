package domain

import "fmt"

func validateHandSeats(seats [3]HandSeat) error {
	seenAccounts := make(map[AccountID]struct{}, 3)
	seenSeats := make(map[Seat]struct{}, 3)
	for _, current := range seats {
		if !current.Seat.Valid() {
			return fmt.Errorf("%w: invalid seat %d", ErrInvalidArgument, current.Seat)
		}
		if err := validateIdentifier("accountId", string(current.AccountID)); err != nil {
			return err
		}
		if _, exists := seenSeats[current.Seat]; exists {
			return fmt.Errorf("%w: duplicate seat %d", ErrInvalidArgument, current.Seat)
		}
		if _, exists := seenAccounts[current.AccountID]; exists {
			return fmt.Errorf("%w: duplicate account", ErrInvalidArgument)
		}
		seenSeats[current.Seat] = struct{}{}
		seenAccounts[current.AccountID] = struct{}{}
	}
	for _, seat := range fixedSeats {
		if _, exists := seenSeats[seat]; !exists {
			return fmt.Errorf("%w: missing seat %d", ErrInvalidArgument, seat)
		}
	}
	return nil
}

func validateBeaconPlan(plan BeaconPlan) error {
	if err := validateIdentifier("beaconProvider", plan.Provider); err != nil {
		return err
	}
	if err := validateIdentifier("beaconRound", plan.Round); err != nil {
		return err
	}
	return nil
}
