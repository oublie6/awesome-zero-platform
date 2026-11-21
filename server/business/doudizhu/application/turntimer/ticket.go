package turntimer

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// Ticket is only a scheduling hint, never authority to perform a game action.
// The consumer must compare every field with the currently committed MySQL turn.
type Ticket struct {
	HandID         string `json:"handId"`
	LiveVersion    uint64 `json:"liveVersion"`
	Seat           uint8  `json:"seat"`
	DeadlineMillis int64  `json:"deadlineMillis"`
	DeliveryID     string `json:"deliveryId"`
}

const MaxTicketBytes = 1024

func (t Ticket) Validate() error {
	if len(t.HandID) == 0 || len(t.HandID) > 128 || t.LiveVersion == 0 ||
		t.Seat < 1 || t.Seat > 3 || t.DeadlineMillis <= 0 ||
		len(t.DeliveryID) == 0 || len(t.DeliveryID) > 128 {
		return errors.New("invalid turn timeout ticket")
	}
	for _, c := range t.HandID + t.DeliveryID {
		if c < 33 || c > 126 {
			return errors.New("invalid turn timeout ticket characters")
		}
	}
	return nil
}
func (t Ticket) Deadline() time.Time { return time.UnixMilli(t.DeadlineMillis).UTC() }
func Encode(t Ticket) ([]byte, error) {
	if err := t.Validate(); err != nil {
		return nil, err
	}
	data, err := json.Marshal(t)
	if err != nil {
		return nil, err
	}
	if len(data) > MaxTicketBytes {
		return nil, fmt.Errorf("timeout ticket exceeds %d bytes", MaxTicketBytes)
	}
	return data, nil
}
func Decode(data []byte) (Ticket, error) {
	var t Ticket
	if len(data) == 0 || len(data) > MaxTicketBytes {
		return t, errors.New("invalid timeout ticket length")
	}
	if err := json.Unmarshal(data, &t); err != nil {
		return t, err
	}
	return t, t.Validate()
}
