package domain

import "fmt"

type RoomStatus string

const (
	RoomWaitingPlayers RoomStatus = "WAITING_PLAYERS"
	RoomReady          RoomStatus = "READY"
	RoomHandActive     RoomStatus = "HAND_ACTIVE"
	RoomClosed         RoomStatus = "CLOSED"
)

const (
	EventRoomCreated      = "doudizhu.room.created.v1"
	EventRoomPlayerJoined = "doudizhu.room.player-joined.v1"
	EventRoomPlayerLeft   = "doudizhu.room.player-left.v1"
	EventRoomReadyChanged = "doudizhu.room.ready-changed.v1"
	EventRoomHandStarted  = "doudizhu.room.hand-started.v1"
	EventRoomHandFinished = "doudizhu.room.hand-finished.v1"
	EventRoomClosed       = "doudizhu.room.closed.v1"
)

type roomSeat struct {
	AccountID AccountID
	Ready     bool
}

type SeatSnapshot struct {
	Seat      Seat
	AccountID AccountID
	Ready     bool
}

type RoomSnapshot struct {
	ID           RoomID
	OwnerID      AccountID
	Status       RoomStatus
	Seats        [3]SeatSnapshot
	ActiveHandID HandID
	Version      uint64
}

type Room struct {
	id           RoomID
	ownerID      AccountID
	status       RoomStatus
	seats        [3]roomSeat
	activeHandID HandID
	version      uint64
}

type RoomCreatedPayload struct {
	OwnerID AccountID
	Seat    Seat
}

type RoomPlayerPayload struct {
	AccountID AccountID
	Seat      Seat
}

type RoomReadyChangedPayload struct {
	AccountID AccountID
	Seat      Seat
	Ready     bool
}

type RoomHandPayload struct {
	HandID HandID
}

func NewRoom(id RoomID, ownerID AccountID) (*Room, []Event, error) {
	if err := validateIdentifier("roomId", string(id)); err != nil {
		return nil, nil, err
	}
	if err := validateIdentifier("ownerId", string(ownerID)); err != nil {
		return nil, nil, err
	}

	room := &Room{
		id:      id,
		ownerID: ownerID,
		status:  RoomWaitingPlayers,
	}
	room.seats[0] = roomSeat{AccountID: ownerID}
	event := room.record(EventRoomCreated, RoomCreatedPayload{OwnerID: ownerID, Seat: SeatOne})
	return room, []Event{event}, nil
}

func (r *Room) Snapshot() RoomSnapshot {
	var seats [3]SeatSnapshot
	for index, seat := range fixedSeats {
		seats[index] = SeatSnapshot{
			Seat:      seat,
			AccountID: r.seats[index].AccountID,
			Ready:     r.seats[index].Ready,
		}
	}
	return RoomSnapshot{
		ID:           r.id,
		OwnerID:      r.ownerID,
		Status:       r.status,
		Seats:        seats,
		ActiveHandID: r.activeHandID,
		Version:      r.version,
	}
}

func (r *Room) Join(actor AccountID, expectedVersion uint64) ([]Event, error) {
	if err := checkExpectedVersion(r.version, expectedVersion); err != nil {
		return nil, err
	}
	if err := validateIdentifier("actor", string(actor)); err != nil {
		return nil, err
	}
	if r.status == RoomClosed {
		return nil, ErrRoomClosed
	}
	if r.status == RoomHandActive {
		return nil, ErrHandActive
	}
	if _, found := r.seatOf(actor); found {
		return nil, ErrAlreadySeated
	}
	for index, seat := range fixedSeats {
		if r.seats[index].AccountID == "" {
			r.seats[index] = roomSeat{AccountID: actor}
			r.recalculateStatus()
			return []Event{r.record(EventRoomPlayerJoined, RoomPlayerPayload{AccountID: actor, Seat: seat})}, nil
		}
	}
	return nil, ErrRoomFull
}

func (r *Room) Leave(actor AccountID, expectedVersion uint64) ([]Event, error) {
	if err := checkExpectedVersion(r.version, expectedVersion); err != nil {
		return nil, err
	}
	if r.status == RoomClosed {
		return nil, ErrRoomClosed
	}
	if r.status == RoomHandActive {
		return nil, ErrHandActive
	}
	seat, found := r.seatOf(actor)
	if !found {
		return nil, ErrNotSeated
	}
	if actor == r.ownerID {
		if r.occupiedCount() > 1 {
			return nil, ErrOwnerMustRemain
		}
		r.seats[seat-1] = roomSeat{}
		r.status = RoomClosed
		return []Event{r.record(EventRoomClosed, RoomPlayerPayload{AccountID: actor, Seat: seat})}, nil
	}

	r.seats[seat-1] = roomSeat{}
	r.recalculateStatus()
	return []Event{r.record(EventRoomPlayerLeft, RoomPlayerPayload{AccountID: actor, Seat: seat})}, nil
}

func (r *Room) SetReady(actor AccountID, ready bool, expectedVersion uint64) ([]Event, error) {
	if err := checkExpectedVersion(r.version, expectedVersion); err != nil {
		return nil, err
	}
	if r.status == RoomClosed {
		return nil, ErrRoomClosed
	}
	if r.status == RoomHandActive {
		return nil, ErrHandActive
	}
	seat, found := r.seatOf(actor)
	if !found {
		return nil, ErrNotSeated
	}
	current := &r.seats[seat-1]
	if current.Ready == ready {
		return nil, ErrNoStateChange
	}
	current.Ready = ready
	r.recalculateStatus()
	return []Event{r.record(EventRoomReadyChanged, RoomReadyChangedPayload{AccountID: actor, Seat: seat, Ready: ready})}, nil
}

func (r *Room) StartHand(actor AccountID, handID HandID, expectedVersion uint64) ([]Event, error) {
	if err := checkExpectedVersion(r.version, expectedVersion); err != nil {
		return nil, err
	}
	if actor != r.ownerID {
		return nil, ErrForbidden
	}
	if err := validateIdentifier("handId", string(handID)); err != nil {
		return nil, err
	}
	if r.status == RoomClosed {
		return nil, ErrRoomClosed
	}
	if r.activeHandID != "" || r.status == RoomHandActive {
		return nil, ErrHandActive
	}
	if r.status != RoomReady || r.occupiedCount() != 3 || !r.allReady() {
		return nil, ErrRoomNotReady
	}

	r.activeHandID = handID
	r.status = RoomHandActive
	for index := range r.seats {
		r.seats[index].Ready = false
	}
	return []Event{r.record(EventRoomHandStarted, RoomHandPayload{HandID: handID})}, nil
}

func (r *Room) FinishHand(handID HandID, expectedVersion uint64) ([]Event, error) {
	if err := checkExpectedVersion(r.version, expectedVersion); err != nil {
		return nil, err
	}
	if r.status != RoomHandActive || r.activeHandID == "" || r.activeHandID != handID {
		return nil, ErrHandNotActive
	}
	r.activeHandID = ""
	r.recalculateStatus()
	return []Event{r.record(EventRoomHandFinished, RoomHandPayload{HandID: handID})}, nil
}

func (r *Room) HandSeats() ([3]HandSeat, error) {
	var result [3]HandSeat
	if r.occupiedCount() != 3 {
		return result, fmt.Errorf("%w: three occupied seats required", ErrRoomNotReady)
	}
	for index, seat := range fixedSeats {
		result[index] = HandSeat{Seat: seat, AccountID: r.seats[index].AccountID}
	}
	return result, nil
}

func (r *Room) seatOf(accountID AccountID) (Seat, bool) {
	for index, current := range r.seats {
		if current.AccountID == accountID {
			return fixedSeats[index], true
		}
	}
	return 0, false
}

func (r *Room) occupiedCount() int {
	count := 0
	for _, current := range r.seats {
		if current.AccountID != "" {
			count++
		}
	}
	return count
}

func (r *Room) allReady() bool {
	for _, current := range r.seats {
		if current.AccountID == "" || !current.Ready {
			return false
		}
	}
	return true
}

func (r *Room) recalculateStatus() {
	if r.status == RoomClosed {
		return
	}
	if r.activeHandID != "" {
		r.status = RoomHandActive
		return
	}
	if r.occupiedCount() == 3 && r.allReady() {
		r.status = RoomReady
		return
	}
	r.status = RoomWaitingPlayers
}

func (r *Room) record(name string, payload any) Event {
	r.version++
	return Event{
		AggregateType: AggregateRoom,
		AggregateID:   string(r.id),
		Name:          name,
		Version:       r.version,
		Payload:       payload,
	}
}
