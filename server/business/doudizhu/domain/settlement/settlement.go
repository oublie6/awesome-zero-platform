package settlement

import (
	"errors"
	"fmt"
	"math"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/bidding"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/playing"
)

const RulesVersion = "doudizhu-settlement-v1"

const (
	maxBombs              = 13
	maxRockets            = 1
	maxMultiplierExponent = maxBombs + maxRockets + 1
)

var ErrInvalidInput = errors.New("doudizhu settlement: invalid input")

type Input struct {
	LandlordSeat uint8
	WinningScore bidding.Score
	Playing      playing.Snapshot
}

type Result struct {
	Version       string     `json:"v"`
	LandlordSeat  uint8      `json:"landlordSeat"`
	WinnerSeat    uint8      `json:"winnerSeat"`
	LandlordWon   bool       `json:"landlordWon"`
	BaseScore     uint8      `json:"baseScore"`
	BombCount     uint8      `json:"bombCount"`
	RocketCount   uint8      `json:"rocketCount"`
	Spring        bool       `json:"spring"`
	AntiSpring    bool       `json:"antiSpring"`
	Multiplier    uint64     `json:"multiplier"`
	FinalStake    uint64     `json:"finalStake"`
	SeatPoints    [3]int64   `json:"seatPoints"`
}

func Calculate(input Input) (Result, error) {
	if err := validateInput(input); err != nil {
		return Result{}, err
	}

	plays := [3]int{}
	bombs := 0
	rockets := 0
	for index, action := range input.Playing.History {
		if action.Number != uint64(index+1) || action.Seat < 1 || action.Seat > 3 {
			return Result{}, fmt.Errorf("%w: action %d ordering or seat", ErrInvalidInput, index)
		}
		switch action.Type {
		case playing.ActionPass:
			if len(action.Cards) != 0 || action.Pattern != nil {
				return Result{}, fmt.Errorf("%w: pass action %d carries cards", ErrInvalidInput, index)
			}
		case playing.ActionPlay:
			if len(action.Cards) == 0 || action.Pattern == nil {
				return Result{}, fmt.Errorf("%w: play action %d missing pattern", ErrInvalidInput, index)
			}
			analyzed, err := playing.Analyze(action.Cards)
			if err != nil || analyzed != *action.Pattern {
				return Result{}, fmt.Errorf("%w: play action %d pattern", ErrInvalidInput, index)
			}
			plays[action.Seat-1]++
			switch action.Pattern.Kind {
			case playing.KindBomb:
				bombs++
			case playing.KindRocket:
				rockets++
			}
		default:
			return Result{}, fmt.Errorf("%w: action %d type %q", ErrInvalidInput, index, action.Type)
		}
	}
	if bombs > maxBombs || rockets > maxRockets {
		return Result{}, fmt.Errorf("%w: impossible bomb or rocket count", ErrInvalidInput)
	}

	landlordWon := input.Playing.WinnerSeat == input.LandlordSeat
	farmersPlayed := 0
	for seat := uint8(1); seat <= 3; seat++ {
		if seat != input.LandlordSeat {
			farmersPlayed += plays[seat-1]
		}
	}
	spring := landlordWon && farmersPlayed == 0
	antiSpring := !landlordWon && plays[input.LandlordSeat-1] == 1

	exponent := bombs + rockets
	if spring || antiSpring {
		exponent++
	}
	if exponent > maxMultiplierExponent {
		return Result{}, fmt.Errorf("%w: multiplier exponent %d", ErrInvalidInput, exponent)
	}
	multiplier := uint64(1) << exponent
	base := uint64(input.WinningScore)
	if base > math.MaxUint64/multiplier {
		return Result{}, fmt.Errorf("%w: stake overflow", ErrInvalidInput)
	}
	stake := base * multiplier
	if stake > uint64(math.MaxInt64/2) {
		return Result{}, fmt.Errorf("%w: point overflow", ErrInvalidInput)
	}

	var points [3]int64
	stakePoints := int64(stake)
	if landlordWon {
		points[input.LandlordSeat-1] = 2 * stakePoints
		for seat := uint8(1); seat <= 3; seat++ {
			if seat != input.LandlordSeat {
				points[seat-1] = -stakePoints
			}
		}
	} else {
		points[input.LandlordSeat-1] = -2 * stakePoints
		for seat := uint8(1); seat <= 3; seat++ {
			if seat != input.LandlordSeat {
				points[seat-1] = stakePoints
			}
		}
	}

	return Result{
		Version:      RulesVersion,
		LandlordSeat: input.LandlordSeat,
		WinnerSeat:   input.Playing.WinnerSeat,
		LandlordWon:  landlordWon,
		BaseScore:    uint8(input.WinningScore),
		BombCount:    uint8(bombs),
		RocketCount:  uint8(rockets),
		Spring:       spring,
		AntiSpring:   antiSpring,
		Multiplier:   multiplier,
		FinalStake:   stake,
		SeatPoints:   points,
	}, nil
}

func validateInput(input Input) error {
	if input.LandlordSeat < 1 || input.LandlordSeat > 3 {
		return fmt.Errorf("%w: landlord seat %d", ErrInvalidInput, input.LandlordSeat)
	}
	if input.WinningScore < bidding.ScoreOne || input.WinningScore > bidding.ScoreThree {
		return fmt.Errorf("%w: winning score %d", ErrInvalidInput, input.WinningScore)
	}
	if input.Playing.Version != playing.StateVersion || !input.Playing.Complete || input.Playing.WinnerSeat < 1 || input.Playing.WinnerSeat > 3 || input.Playing.CurrentSeat != 0 {
		return fmt.Errorf("%w: incomplete or invalid playing snapshot", ErrInvalidInput)
	}
	if len(input.Playing.History) == 0 || input.Playing.Revision != uint64(len(input.Playing.History)) {
		return fmt.Errorf("%w: playing history", ErrInvalidInput)
	}
	last := input.Playing.History[len(input.Playing.History)-1]
	if last.Type != playing.ActionPlay || last.Seat != input.Playing.WinnerSeat {
		return fmt.Errorf("%w: final action does not belong to winner", ErrInvalidInput)
	}
	return nil
}
