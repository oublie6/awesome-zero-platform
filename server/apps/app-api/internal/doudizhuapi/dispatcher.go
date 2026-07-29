package doudizhuapi

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/application"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/application/lifecycle"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/bidding"
)

const (
	CommandRequestV1  = "doudizhu-api-command-v1"
	CommandResponseV1 = "doudizhu-api-command-result-v1"

	TypeRoomCreate = "room.create"
	TypeRoomJoin   = "room.join"
	TypeRoomLeave  = "room.leave"
	TypeRoomReady  = "room.ready"
	TypeRoomStart  = "room.hand.start"
	TypeHandCommit = "hand.commit"
	TypeHandReveal = "hand.reveal"
	TypeHandBeacon = "hand.beacon"
	TypeHandDealt  = "hand.dealt"
	TypeHandBid    = "hand.bid"
	TypeHandPlay   = "hand.play"
	TypeHandPass   = "hand.pass"
	TypeHandCancel = "hand.cancel"
)

var (
	ErrInvalidRequest = errors.New("invalid Doudizhu API request")
	ErrReplayConflict = errors.New("Doudizhu API request ID is bound to another payload")
)

type Backend interface {
	CreateRoom(context.Context, domain.AccountID, application.Command) (application.CommandResult, error)
	JoinRoom(context.Context, domain.AccountID, application.Command) (application.CommandResult, error)
	LeaveRoom(context.Context, domain.AccountID, application.Command) (application.CommandResult, error)
	SetRoomReady(context.Context, domain.AccountID, application.Command, application.SetReadyInput) (application.CommandResult, error)
	StartRoomHand(context.Context, domain.AccountID, application.Command, application.StartHandInput) (application.CommandResult, error)
	SubmitHandCommit(context.Context, domain.AccountID, application.Command, application.SubmitCommitInput) (application.CommandResult, error)
	SubmitHandReveal(context.Context, domain.AccountID, application.Command, application.SubmitRevealInput) (application.CommandResult, error)
	LockHandBeacon(context.Context, domain.AccountID, application.Command, application.LockBeaconInput) (application.CommandResult, error)
	MarkHandDealt(context.Context, domain.AccountID, application.Command) (application.CommandResult, error)
	SubmitLiveHandBid(context.Context, domain.AccountID, domain.HandID, uint64, bidding.Score) (application.LiveHandCommandResult, error)
	SubmitLiveHandPlay(context.Context, domain.AccountID, domain.HandID, uint64, []string) (application.LiveHandCommandResult, error)
	SubmitLiveHandPass(context.Context, domain.AccountID, domain.HandID, uint64) (application.LiveHandCommandResult, error)
	GetHandPublicView(context.Context, domain.AccountID, domain.HandID) (application.LiveHandView, error)
	GetHandPrivateView(context.Context, domain.AccountID, domain.HandID) (application.LiveHandView, error)
}

type Evidence interface {
	Get(context.Context, domain.AccountID, domain.HandID) (application.FinalEvidenceResult, error)
}

type Lifecycle interface {
	TrackBidding(context.Context, domain.AccountID, domain.HandID) error
	AfterBid(context.Context, domain.AccountID, domain.HandID, application.LiveHandCommandResult) (lifecycle.Outcome, error)
	AfterPlay(context.Context, domain.AccountID, domain.HandID, application.LiveHandCommandResult) error
	Cancel(context.Context, domain.AccountID, domain.HandID) (lifecycle.Outcome, error)
}

type Clock interface{ Now() time.Time }

type Config struct {
	CommandTTL    time.Duration
	ReplayTTL     time.Duration
	ReplayEntries int
}

func DefaultConfig() Config {
	return Config{CommandTTL: time.Minute, ReplayTTL: 2 * time.Minute, ReplayEntries: 4096}
}

type Dispatcher struct {
	backend   Backend
	evidence  Evidence
	lifecycle Lifecycle
	clock     Clock
	config    Config
	replay    *ReplayCache
}

type CommandRequest struct {
	Version         string          `json:"v"`
	RequestID       string          `json:"requestId"`
	Type            string          `json:"type"`
	AggregateID     string          `json:"aggregateId"`
	ClientSeq       uint64          `json:"clientSeq"`
	ExpectedVersion uint64          `json:"expectedVersion"`
	Payload         json.RawMessage `json:"payload,omitempty"`
}

type CommandResponse struct {
	Version     string                     `json:"v"`
	RequestID   string                     `json:"requestId"`
	Type        string                     `json:"type"`
	Durable     *application.CommandResult `json:"durable,omitempty"`
	Live        *LiveResult                `json:"live,omitempty"`
	Termination *TerminationResult         `json:"termination,omitempty"`
}

type LiveResult struct {
	Version             uint64          `json:"version"`
	Payload             json.RawMessage `json:"payload"`
	RequiresTermination bool            `json:"requiresTermination"`
}

type TerminationResult struct {
	Triggered bool                      `json:"triggered"`
	Durable   application.CommandResult `json:"durable"`
}

type ViewResult struct {
	Version uint64          `json:"version"`
	Payload json.RawMessage `json:"payload"`
}

func NewDispatcher(backend Backend, evidence Evidence, lifecycleService Lifecycle, clock Clock, config Config) (*Dispatcher, error) {
	if backend == nil || evidence == nil || lifecycleService == nil || clock == nil || config.CommandTTL <= 0 || config.ReplayTTL <= 0 || config.ReplayEntries <= 0 {
		return nil, fmt.Errorf("%w: dispatcher configuration", ErrInvalidRequest)
	}
	return &Dispatcher{backend: backend, evidence: evidence, lifecycle: lifecycleService, clock: clock, config: config, replay: NewReplayCache(config.ReplayTTL, config.ReplayEntries, clock)}, nil
}

func (d *Dispatcher) Execute(ctx context.Context, actor domain.AccountID, request CommandRequest) (CommandResponse, error) {
	if err := validateRequest(actor, request); err != nil {
		return CommandResponse{}, err
	}
	digest, err := digestRequest(request)
	if err != nil {
		return CommandResponse{}, err
	}
	key := string(actor) + "\x00" + request.AggregateID + "\x00" + request.RequestID
	if isLiveType(request.Type) {
		return d.replay.Do(ctx, key, digest, func() (CommandResponse, error) { return d.execute(ctx, actor, request) })
	}
	return d.execute(ctx, actor, request)
}

func (d *Dispatcher) PublicView(ctx context.Context, actor domain.AccountID, handID domain.HandID) (ViewResult, error) {
	view, err := d.backend.GetHandPublicView(ctx, actor, handID)
	if err != nil {
		return ViewResult{}, err
	}
	return ViewResult{Version: view.Version, Payload: append(json.RawMessage(nil), view.Payload...)}, nil
}
func (d *Dispatcher) PrivateView(ctx context.Context, actor domain.AccountID, handID domain.HandID) (ViewResult, error) {
	view, err := d.backend.GetHandPrivateView(ctx, actor, handID)
	if err != nil {
		return ViewResult{}, err
	}
	return ViewResult{Version: view.Version, Payload: append(json.RawMessage(nil), view.Payload...)}, nil
}
func (d *Dispatcher) FinalEvidence(ctx context.Context, actor domain.AccountID, handID domain.HandID) (application.FinalEvidenceResult, error) {
	return d.evidence.Get(ctx, actor, handID)
}

func (d *Dispatcher) execute(ctx context.Context, actor domain.AccountID, request CommandRequest) (CommandResponse, error) {
	response := CommandResponse{Version: CommandResponseV1, RequestID: request.RequestID, Type: request.Type}
	command := application.Command{Version: application.CommandProtocolV1, CommandID: request.RequestID, AggregateID: request.AggregateID, ClientSeq: request.ClientSeq, ExpectedVersion: request.ExpectedVersion, IssuedAt: d.clock.Now().UTC().Truncate(time.Millisecond)}
	command.ExpiresAt = command.IssuedAt.Add(d.config.CommandTTL)
	var durable application.CommandResult
	var live application.LiveHandCommandResult
	var terminal lifecycle.Outcome
	var err error
	switch request.Type {
	case TypeRoomCreate:
		if err = requireEmptyPayload(request.Payload); err == nil {
			command.Name, command.AggregateType = application.CommandRoomCreate, domain.AggregateRoom
			durable, err = d.backend.CreateRoom(ctx, actor, command)
		}
	case TypeRoomJoin:
		if err = requireEmptyPayload(request.Payload); err == nil {
			command.Name, command.AggregateType = application.CommandRoomJoin, domain.AggregateRoom
			durable, err = d.backend.JoinRoom(ctx, actor, command)
		}
	case TypeRoomLeave:
		if err = requireEmptyPayload(request.Payload); err == nil {
			command.Name, command.AggregateType = application.CommandRoomLeave, domain.AggregateRoom
			durable, err = d.backend.LeaveRoom(ctx, actor, command)
		}
	case TypeRoomReady:
		var payload struct {
			Ready bool `json:"ready"`
		}
		if err = decodePayload(request.Payload, &payload); err == nil {
			command.Name, command.AggregateType = application.CommandRoomReadySet, domain.AggregateRoom
			durable, err = d.backend.SetRoomReady(ctx, actor, command, application.SetReadyInput{Ready: payload.Ready})
		}
	case TypeRoomStart:
		var payload struct {
			HandID string `json:"handId"`
		}
		if err = decodePayload(request.Payload, &payload); err == nil {
			command.Name, command.AggregateType = application.CommandRoomHandStart, domain.AggregateRoom
			durable, err = d.backend.StartRoomHand(ctx, actor, command, application.StartHandInput{HandID: domain.HandID(payload.HandID)})
		}
	case TypeHandCommit:
		var payload struct {
			Commitment string `json:"commitment"`
		}
		if err = decodePayload(request.Payload, &payload); err == nil {
			var value domain.Commitment
			err = decode32(payload.Commitment, value[:])
			if err == nil {
				command.Name, command.AggregateType = application.CommandHandCommitSubmit, domain.AggregateHand
				durable, err = d.backend.SubmitHandCommit(ctx, actor, command, application.SubmitCommitInput{Commitment: value})
			}
		}
	case TypeHandReveal:
		var payload struct {
			Envelope struct {
				Version         string `json:"v"`
				KeyID           string `json:"keyId"`
				Suite           string `json:"suite"`
				EncapsulatedKey string `json:"encapsulatedKey"`
				Ciphertext      string `json:"ciphertext"`
			} `json:"envelope"`
		}
		if err = decodePayload(request.Payload, &payload); err == nil {
			command.Name, command.AggregateType = application.CommandHandRevealSubmit, domain.AggregateHand
			durable, err = d.backend.SubmitHandReveal(ctx, actor, command, application.SubmitRevealInput{Envelope: application.SecureEnvelope{
				Version: payload.Envelope.Version, KeyID: payload.Envelope.KeyID, Suite: payload.Envelope.Suite,
				EncapsulatedKey: payload.Envelope.EncapsulatedKey, Ciphertext: payload.Envelope.Ciphertext,
			}})
		}
	case TypeHandBeacon:
		var payload struct {
			Provider string `json:"provider"`
			Round    string `json:"round"`
			Digest   string `json:"digest"`
			ProofRef string `json:"proofRef"`
		}
		if err = decodePayload(request.Payload, &payload); err == nil {
			var digest domain.BeaconDigest
			err = decode32(payload.Digest, digest[:])
			if err == nil {
				command.Name, command.AggregateType = application.CommandHandBeaconLock, domain.AggregateHand
				durable, err = d.backend.LockHandBeacon(ctx, actor, command, application.LockBeaconInput{Value: domain.BeaconValue{Provider: payload.Provider, Round: payload.Round, Digest: digest, ProofRef: payload.ProofRef}})
			}
		}
	case TypeHandDealt:
		if err = requireEmptyPayload(request.Payload); err == nil {
			command.Name, command.AggregateType = application.CommandHandDealt, domain.AggregateHand
			durable, err = d.backend.MarkHandDealt(ctx, actor, command)
			if err == nil && durable.Accepted {
				err = d.lifecycle.TrackBidding(ctx, actor, domain.HandID(request.AggregateID))
			}
		}
	case TypeHandBid:
		var payload struct {
			Score bidding.Score `json:"score"`
		}
		if err = decodePayload(request.Payload, &payload); err == nil {
			live, err = d.backend.SubmitLiveHandBid(ctx, actor, domain.HandID(request.AggregateID), request.ExpectedVersion, payload.Score)
			if err == nil {
				terminal, err = d.lifecycle.AfterBid(ctx, actor, domain.HandID(request.AggregateID), live)
			}
		}
	case TypeHandPlay:
		var payload struct {
			Cards []string `json:"cards"`
		}
		if err = decodePayload(request.Payload, &payload); err == nil {
			live, err = d.backend.SubmitLiveHandPlay(ctx, actor, domain.HandID(request.AggregateID), request.ExpectedVersion, payload.Cards)
			if err == nil {
				err = d.lifecycle.AfterPlay(ctx, actor, domain.HandID(request.AggregateID), live)
			}
		}
	case TypeHandPass:
		if err = requireEmptyPayload(request.Payload); err == nil {
			live, err = d.backend.SubmitLiveHandPass(ctx, actor, domain.HandID(request.AggregateID), request.ExpectedVersion)
			if err == nil {
				err = d.lifecycle.AfterPlay(ctx, actor, domain.HandID(request.AggregateID), live)
			}
		}
	case TypeHandCancel:
		if err = requireEmptyPayload(request.Payload); err == nil {
			terminal, err = d.lifecycle.Cancel(ctx, actor, domain.HandID(request.AggregateID))
		}
	default:
		return CommandResponse{}, fmt.Errorf("%w: unsupported type %q", ErrInvalidRequest, request.Type)
	}
	if err != nil {
		return CommandResponse{}, err
	}
	if durable.Version != "" {
		response.Durable = &durable
	}
	if live.Version != 0 {
		response.Live = &LiveResult{
			Version:             live.Version,
			Payload:             append(json.RawMessage(nil), live.Payload...),
			RequiresTermination: live.RequiresTermination,
		}
	}
	if terminal.Triggered {
		response.Termination = &TerminationResult{Triggered: true, Durable: cloneCommandResult(terminal.Result)}
	}
	return response, nil
}

func validateRequest(actor domain.AccountID, request CommandRequest) error {
	if strings.TrimSpace(string(actor)) == "" || request.Version != CommandRequestV1 ||
		request.RequestID != strings.TrimSpace(request.RequestID) || request.RequestID == "" || len(request.RequestID) > 128 ||
		request.Type != strings.TrimSpace(request.Type) || request.Type == "" ||
		request.AggregateID != strings.TrimSpace(request.AggregateID) || request.AggregateID == "" || len(request.AggregateID) > 128 ||
		request.ClientSeq == 0 {
		return ErrInvalidRequest
	}
	if request.Type == TypeRoomCreate && request.ExpectedVersion != 0 {
		return ErrInvalidRequest
	}
	if request.Type != TypeRoomCreate && request.ExpectedVersion == 0 {
		return ErrInvalidRequest
	}
	return nil
}
func isLiveType(value string) bool {
	return value == TypeHandBid || value == TypeHandPlay || value == TypeHandPass || value == TypeHandCancel
}
func digestRequest(request CommandRequest) ([32]byte, error) {
	encoded, err := json.Marshal(request)
	if err != nil {
		return [32]byte{}, err
	}
	return sha256.Sum256(encoded), nil
}
func decodePayload(raw json.RawMessage, destination any) error {
	if len(raw) == 0 {
		return ErrInvalidRequest
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidRequest, err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return ErrInvalidRequest
	}
	return nil
}
func requireEmptyPayload(raw json.RawMessage) error {
	if len(raw) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("{}")) || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil
	}
	return ErrInvalidRequest
}
func decode32(value string, target []byte) error {
	if len(value) != 64 || value != strings.ToLower(value) {
		return ErrInvalidRequest
	}
	decoded, err := hex.DecodeString(value)
	if err != nil || len(decoded) != 32 {
		return ErrInvalidRequest
	}
	copy(target, decoded)
	return nil
}

type ReplayCache struct {
	mu      sync.Mutex
	ttl     time.Duration
	limit   int
	clock   Clock
	entries map[string]*replayEntry
}
type replayEntry struct {
	digest   [32]byte
	expires  time.Time
	ready    chan struct{}
	response CommandResponse
	err      error
}

func NewReplayCache(ttl time.Duration, limit int, clock Clock) *ReplayCache {
	return &ReplayCache{ttl: ttl, limit: limit, clock: clock, entries: make(map[string]*replayEntry)}
}
func (c *ReplayCache) Do(ctx context.Context, key string, digest [32]byte, fn func() (CommandResponse, error)) (CommandResponse, error) {
	c.mu.Lock()
	c.pruneLocked()
	if existing := c.entries[key]; existing != nil {
		if existing.digest != digest {
			c.mu.Unlock()
			return CommandResponse{}, ErrReplayConflict
		}
		ready := existing.ready
		c.mu.Unlock()
		select {
		case <-ctx.Done():
			return CommandResponse{}, ctx.Err()
		case <-ready:
			return cloneResponse(existing.response), existing.err
		}
	}
	if len(c.entries) >= c.limit {
		c.evictLocked()
	}
	entry := &replayEntry{digest: digest, expires: c.clock.Now().UTC().Add(c.ttl), ready: make(chan struct{})}
	c.entries[key] = entry
	c.mu.Unlock()
	response, err := fn()
	c.mu.Lock()
	if err != nil {
		delete(c.entries, key)
	} else {
		entry.response = cloneResponse(response)
	}
	entry.err = err
	close(entry.ready)
	c.mu.Unlock()
	return response, err
}
func (c *ReplayCache) pruneLocked() {
	now := c.clock.Now().UTC()
	for key, entry := range c.entries {
		select {
		case <-entry.ready:
			if !now.Before(entry.expires) {
				delete(c.entries, key)
			}
		default:
		}
	}
}
func (c *ReplayCache) evictLocked() {
	var selected string
	var earliest time.Time
	for key, entry := range c.entries {
		select {
		case <-entry.ready:
			if selected == "" || entry.expires.Before(earliest) {
				selected, earliest = key, entry.expires
			}
		default:
		}
	}
	if selected != "" {
		delete(c.entries, selected)
	}
}
func cloneResponse(source CommandResponse) CommandResponse {
	result := source
	if source.Durable != nil {
		copyValue := cloneCommandResult(*source.Durable)
		result.Durable = &copyValue
	}
	if source.Live != nil {
		live := *source.Live
		live.Payload = append(json.RawMessage(nil), source.Live.Payload...)
		result.Live = &live
	}
	if source.Termination != nil {
		terminal := *source.Termination
		terminal.Durable = cloneCommandResult(source.Termination.Durable)
		result.Termination = &terminal
	}
	return result
}

func cloneCommandResult(source application.CommandResult) application.CommandResult {
	result := source
	result.Events = append([]application.EventRef(nil), source.Events...)
	if source.Failure != nil {
		failure := *source.Failure
		result.Failure = &failure
	}
	return result
}
