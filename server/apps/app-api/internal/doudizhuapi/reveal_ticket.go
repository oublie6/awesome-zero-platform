package doudizhuapi

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/application"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
)

const RevealTicketV1 = "doudizhu-reveal-context-v1"
const CustodiedTicketV1 = "doudizhu-custodied-context-v1"
const revealTicketDomain = "doudizhu/reveal-ticket/hmac-sha256/v1:"

type RevealTicketClaims struct {
	Version         string `json:"v"`
	Actor           string `json:"actor"`
	HandID          string `json:"handId"`
	RequestID       string `json:"requestId"`
	ClientSeq       uint64 `json:"clientSeq"`
	ExpectedVersion uint64 `json:"expectedVersion"`
	Seat            uint8  `json:"seat"`
	Commitment      string `json:"commitment"`
	KeyID           string `json:"keyId"`
	PublicKeySHA256 string `json:"publicKeySha256"`
	IssuedAt        int64  `json:"issuedAt"`
	ExpiresAt       int64  `json:"expiresAt"`
}
type SignedRevealContext struct {
	Claims RevealTicketClaims `json:"claims"`
	MAC    string             `json:"mac"`
}
type RevealContextResponse struct {
	Version string              `json:"v"`
	Ticket  SignedRevealContext `json:"ticket"`
	AAD     string              `json:"aad"`
}
type RevealContextRequest struct {
	RequestID       string `json:"requestId"`
	ClientSeq       uint64 `json:"clientSeq"`
	ExpectedVersion uint64 `json:"expectedVersion"`
}

type revealTicketIssuer struct {
	key [32]byte
	ttl time.Duration
}

// ConfigureRevealTickets derives a domain-specific MAC key from the existing
// secure application root key, shared across authorized server instances.
// Never expose this secret in responses or client code.
func (d *Dispatcher) ConfigureRevealTickets(rootKey []byte) error {
	if d == nil || len(rootKey) != 32 {
		return fmt.Errorf("%w: protected reveal ticket key", ErrInvalidRequest)
	}
	digest := sha256.New()
	digest.Write([]byte(revealTicketDomain))
	digest.Write(rootKey)
	key := digest.Sum(nil)
	var fixed [32]byte
	copy(fixed[:], key)
	clear(key)
	d.revealTickets = &revealTicketIssuer{key: fixed, ttl: d.config.CommandTTL}
	return nil
}

func (d *Dispatcher) IssueRevealContext(ctx context.Context, actor domain.AccountID, handID domain.HandID, request RevealContextRequest) (RevealContextResponse, error) {
	if d == nil || d.revealTickets == nil || d.backend == nil {
		return RevealContextResponse{}, ErrInvalidRequest
	}
	if strings.TrimSpace(string(actor)) == "" || request.RequestID == "" ||
		request.RequestID != strings.TrimSpace(request.RequestID) || len(request.RequestID) > 128 ||
		request.ClientSeq == 0 || request.ExpectedVersion == 0 || strings.TrimSpace(string(handID)) == "" ||
		len(handID) > 128 {
		return RevealContextResponse{}, ErrInvalidRequest
	}
	loader, ok := d.backend.(interface {
		GetRevealBinding(context.Context, domain.AccountID, domain.HandID) (application.RevealBinding, error)
	})
	if !ok {
		return RevealContextResponse{}, ErrInvalidRequest
	}
	binding, err := loader.GetRevealBinding(ctx, actor, handID)
	if err != nil {
		return RevealContextResponse{}, err
	}
	if binding.HandID != handID || binding.Version != request.ExpectedVersion || !binding.Seat.Valid() {
		return RevealContextResponse{}, fmt.Errorf("%w: stale reveal binding", domain.ErrVersionConflict)
	}
	now := d.clock.Now().UTC().Truncate(time.Millisecond)
    version:=RevealTicketV1
    commitment:=hex.EncodeToString(binding.Commitment[:])
    if binding.Mode==domain.FairnessModeClientCustodied {
      version=CustodiedTicketV1
      commitment=""
    }
	ticket := SignedRevealContext{Claims: RevealTicketClaims{
		Version: version, Actor: string(actor), HandID: string(handID), RequestID: request.RequestID,
		ClientSeq: request.ClientSeq, ExpectedVersion: request.ExpectedVersion, Seat: uint8(binding.Seat),
		Commitment: commitment, KeyID: binding.KeyID,
		PublicKeySHA256: hex.EncodeToString(binding.PublicKeySHA256[:]),
		IssuedAt:        now.UnixMilli(), ExpiresAt: now.Add(d.revealTickets.ttl).UnixMilli(),
	}}
	ticket.MAC = d.revealTickets.sign(ticket.Claims)
	aad, err := revealTicketAAD(ticket.Claims)
	if err != nil {
		return RevealContextResponse{}, err
	}
	return RevealContextResponse{Version: version, Ticket: ticket, AAD: base64.RawURLEncoding.EncodeToString(aad)}, nil
}
func (s *revealTicketIssuer) sign(claims RevealTicketClaims) string {
	encoded, _ := json.Marshal(claims)
	mac := hmac.New(sha256.New, s.key[:])
	mac.Write(encoded)
	return hex.EncodeToString(mac.Sum(nil))
}
func revealTicketAAD(claims RevealTicketClaims) ([]byte,error) {
  command:=application.Command{
    Version:application.CommandProtocolV1,
    CommandID:claims.RequestID,AggregateType:domain.AggregateHand,
    AggregateID:claims.HandID,ClientSeq:claims.ClientSeq,ExpectedVersion:claims.ExpectedVersion,
    IssuedAt:time.UnixMilli(claims.IssuedAt).UTC(),ExpiresAt:time.UnixMilli(claims.ExpiresAt).UTC(),
  }
  if claims.Version==CustodiedTicketV1 {
    if claims.Commitment!="" { return nil,ErrInvalidRequest }
    command.Name=application.CommandHandContributionSubmit
    return application.BuildCustodiedAAD(command,domain.AccountID(claims.Actor),domain.Seat(claims.Seat),claims.KeyID)
  }
  if claims.Version!=RevealTicketV1 {return nil,ErrInvalidRequest}
  var commitment domain.Commitment
  decoded,err:=hex.DecodeString(claims.Commitment)
  if err!=nil || len(decoded)!=32{return nil,ErrInvalidRequest}
  copy(commitment[:],decoded)
  command.Name=application.CommandHandRevealSubmit
  return application.BuildRevealAAD(command,domain.AccountID(claims.Actor),domain.Seat(claims.Seat),commitment,claims.KeyID)
}
func (d *Dispatcher) verifyRevealContext(actor domain.AccountID, request CommandRequest) (time.Time, time.Time, error) {
	if d == nil || d.revealTickets == nil || request.RevealContext == nil {
		return time.Time{}, time.Time{}, ErrInvalidRequest
	}
	t := request.RevealContext
	c := t.Claims
	received, err := hex.DecodeString(t.MAC)
	if err != nil || len(received) != sha256.Size || t.MAC != strings.ToLower(t.MAC) {
		return time.Time{}, time.Time{}, ErrInvalidRequest
	}
	expected, err := hex.DecodeString(d.revealTickets.sign(c))
	if err != nil || !hmac.Equal(received, expected) {
		return time.Time{}, time.Time{}, ErrInvalidRequest
	}
	expectedVersion:=RevealTicketV1
    if request.Type==TypeHandContribution { expectedVersion=CustodiedTicketV1 }
    if c.Version != expectedVersion || c.Actor != string(actor) || c.HandID != request.AggregateID ||
		c.RequestID != request.RequestID || c.ClientSeq != request.ClientSeq ||
		c.ExpectedVersion != request.ExpectedVersion || !domain.Seat(c.Seat).Valid() ||
		len(c.KeyID) == 0 || len(c.KeyID) > 128 || len(c.PublicKeySHA256) != 64 {
		return time.Time{}, time.Time{}, ErrInvalidRequest
	}
	if _, err := revealTicketAAD(c); err != nil {
		return time.Time{}, time.Time{}, ErrInvalidRequest
	}
	hash, err := hex.DecodeString(c.PublicKeySHA256)
	if err != nil || len(hash) != 32 {
		return time.Time{}, time.Time{}, ErrInvalidRequest
	}
	issued, expires := time.UnixMilli(c.IssuedAt).UTC(), time.UnixMilli(c.ExpiresAt).UTC()
	now := d.clock.Now().UTC()
	// Preserve signed context verification for exact already-committed retries
	// after the ticket expires. The application transaction checks stored
	// command identity before freshness; uncommitted expired requests still
	// fail inside MySQL, without changing an acknowledged contribution.
	if expires.Sub(issued) != d.revealTickets.ttl || now.Before(issued.Add(-5*time.Second)) ||
		expires.Sub(now) > d.revealTickets.ttl+5*time.Second {
		return time.Time{}, time.Time{}, errors.Join(ErrInvalidRequest, errors.New("invalid signed reveal context time window"))
	}
	return issued, expires, nil
}
