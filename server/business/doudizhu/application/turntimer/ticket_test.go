package turntimer

import (
	"reflect"
	"testing"
	"time"
)

func TestTicketRoundTrip(t *testing.T) {
	input := Ticket{HandID: "hand-01", LiveVersion: 42, Seat: 3, DeadlineMillis: time.Date(2026, 11, 17, 12, 0, 0, 0, time.UTC).UnixMilli(), DeliveryID: "d-1"}
	b, err := Encode(input)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Decode(b)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(input, got) {
		t.Fatalf("roundtrip mismatch: %#v", got)
	}
	if got.Deadline().Location() != time.UTC {
		t.Fatal("deadline must be UTC")
	}
}
func TestRejectMalformedTickets(t *testing.T) {
	base := Ticket{HandID: "h", LiveVersion: 1, Seat: 1, DeadlineMillis: 1000, DeliveryID: "d"}
	for _, m := range []func(*Ticket){
		func(x *Ticket) { x.HandID = "" }, func(x *Ticket) { x.HandID = "h\nnot-valid" },
		func(x *Ticket) { x.LiveVersion = 0 }, func(x *Ticket) { x.Seat = 0 },
		func(x *Ticket) { x.Seat = 4 }, func(x *Ticket) { x.DeadlineMillis = 0 },
		func(x *Ticket) { x.DeliveryID = "" }, func(x *Ticket) { x.DeliveryID = "bad\nlog" },
	} {
		x := base
		m(&x)
		if _, err := Encode(x); err == nil {
			t.Fatalf("accepted %#v", x)
		}
	}
	if _, err := Decode([]byte("{")); err == nil {
		t.Fatal("accepted invalid JSON")
	}
}
