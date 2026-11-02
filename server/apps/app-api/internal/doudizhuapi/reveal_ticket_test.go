package doudizhuapi

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"testing"
	"time"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/application"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
)

type revealBackendStub struct{ backendStub }

func (b *revealBackendStub) GetRevealBinding(_ context.Context, actor domain.AccountID, id domain.HandID) (application.RevealBinding, error) {
	if actor != "account-1" {
		return application.RevealBinding{}, domain.ErrNotSeated
	}
	return application.RevealBinding{HandID: id, Seat: domain.SeatOne, Version: 7,
		Commitment: domain.Commitment{1, 2, 3}, KeyID: "signed-key-v1",
		PublicKeySHA256: domain.RevealPublicKeyHash{8}, BoundAt: time.Unix(1, 0)}, nil
}
func TestSignedRevealContextMatchesGoAADAndRejectsTamper(t *testing.T) {
	backend := &revealBackendStub{}
	d := newDispatcherForTest(t, backend, &lifecycleStub{})
	if err := d.ConfigureRevealTickets(bytes.Repeat([]byte{7}, 32)); err != nil {
		t.Fatal(err)
	}
	req := RevealContextRequest{RequestID: "reveal-test-01", ClientSeq: 5, ExpectedVersion: 7}
	got, err := d.IssueRevealContext(context.Background(), "account-1", "hand-a", req)
	if err != nil {
		t.Fatal(err)
	}
	if got.Version != RevealTicketV1 || len(got.Ticket.MAC) != 64 {
		t.Fatalf("missing signed context: %+v", got)
	}
	aad, err := base64.RawURLEncoding.DecodeString(got.AAD)
	if err != nil {
		t.Fatal(err)
	}
	command := application.Command{
		Version: application.CommandProtocolV1, Name: application.CommandHandRevealSubmit,
		CommandID: req.RequestID, AggregateType: domain.AggregateHand, AggregateID: "hand-a",
		ClientSeq: 5, ExpectedVersion: 7,
		IssuedAt: time.UnixMilli(got.Ticket.Claims.IssuedAt), ExpiresAt: time.UnixMilli(got.Ticket.Claims.ExpiresAt),
	}
	expected, err := application.BuildRevealAAD(command, "account-1", domain.SeatOne, domain.Commitment{1, 2, 3}, "signed-key-v1")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(aad, expected) {
		t.Fatal("issued AAD is different from application.Open AAD")
	}
	wire := CommandRequest{Version: CommandRequestV1, Type: TypeHandReveal, RequestID: req.RequestID,
		AggregateID: "hand-a", ClientSeq: 5, ExpectedVersion: 7, RevealContext: &got.Ticket}
	issued, expires, err := d.verifyRevealContext("account-1", wire)
	if err != nil || !issued.Equal(command.IssuedAt) || !expires.Equal(command.ExpiresAt) {
		t.Fatalf("verified wrong signed context: issued=%v expires=%v err=%v", issued, expires, err)
	}
	if _, _, err := d.verifyRevealContext("account-2", wire); err == nil {
		t.Fatal("other account reused signed ticket")
	}
	wire.ExpectedVersion = 8
	if _, _, err := d.verifyRevealContext("account-1", wire); err == nil {
		t.Fatal("ticket reused for other hand version")
	}
	wire.ExpectedVersion = 7
	wire.RevealContext = &SignedRevealContext{Claims: got.Ticket.Claims, MAC: got.Ticket.MAC}
	wire.RevealContext.Claims.Seat = 2
	if _, _, err := d.verifyRevealContext("account-1", wire); err == nil {
		t.Fatal("tampered seat accepted")
	}
	wire.RevealContext.Claims.Seat = 1
	invalidMac, _ := hex.DecodeString(wire.RevealContext.MAC)
	invalidMac[0] ^= 0xff
	wire.RevealContext.MAC = hex.EncodeToString(invalidMac)
	if _, _, err := d.verifyRevealContext("account-1", wire); err == nil {
		t.Fatal("tampered signature accepted")
	}
	wire.RevealContext = &got.Ticket
	d.clock = fixedClock{now: time.UnixMilli(got.Ticket.Claims.ExpiresAt)}
	// The signed context remains authentic after expiry for exact receipt
	// replay; the application's ClaimCommand/validateFreshness path still
	// rejects first-time mutations with this expired command timestamp.
	if _, _, err := d.verifyRevealContext("account-1", wire); err != nil {
		t.Fatalf("expired exact-replay ticket lost authentication: %v", err)
	}
}
func TestRevealContextMustBeEnabledAndBoundToAuthorizedActor(t *testing.T) {
	d := newDispatcherForTest(t, &revealBackendStub{}, &lifecycleStub{})
	req := RevealContextRequest{RequestID: "reveal-test", ClientSeq: 1, ExpectedVersion: 7}
	if _, err := d.IssueRevealContext(context.Background(), "account-1", "hand", req); err == nil {
		t.Fatal("unsigned context issued")
	}
	if err := d.ConfigureRevealTickets([]byte("short")); err == nil {
		t.Fatal("weak ticket key accepted")
	}
	if err := d.ConfigureRevealTickets(bytes.Repeat([]byte{1}, 32)); err != nil {
		t.Fatal(err)
	}
	if _, err := d.IssueRevealContext(context.Background(), "account-2", "hand", req); err == nil {
		t.Fatal("nonmember received reveal context")
	}
	if _, err := d.IssueRevealContext(context.Background(), "account-1", "hand", RevealContextRequest{RequestID: "reveal-test", ClientSeq: 1, ExpectedVersion: 8}); err == nil {
		t.Fatal("stale client accepted")
	}
}
