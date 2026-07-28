package gamecore

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"sync"
)

type FinalStatus string

const (
	FinalStatusCompleted FinalStatus = "completed"
	FinalStatusAborted   FinalStatus = "aborted"
)

const finalRecordDigestDomain = "gamecore/final-record/v1"

type Command struct {
	ActorPosition   uint8
	ExpectedVersion uint64
	Payload         []byte
}

type CommandOutcome struct {
	Version      uint64
	Payload      []byte
	Terminal     bool
	FinalPayload []byte
}

type ViewRequest struct {
	ViewerPosition uint8
	PublicOnly     bool
}

type GameView struct {
	Version uint64
	Payload []byte
}

type AbortOutcome struct {
	Version      uint64
	FinalPayload []byte
}

type LiveGame interface {
	Descriptor() Descriptor
	InstanceID() InstanceID
	Apply(Command) (CommandOutcome, error)
	View(ViewRequest) (GameView, error)
	Abort(reason string) (AbortOutcome, error)
}

type FinalRecord struct {
	instanceID InstanceID
	descriptor Descriptor
	status     FinalStatus
	version    uint64
	payload    []byte
	digest     Digest
}

func NewFinalRecord(id InstanceID, descriptor Descriptor, status FinalStatus, version uint64, payload []byte) (FinalRecord, error) {
	if err := validateInstanceID(id); err != nil {
		return FinalRecord{}, err
	}
	if err := descriptor.Validate(); err != nil {
		return FinalRecord{}, err
	}
	if status != FinalStatusCompleted && status != FinalStatusAborted {
		return FinalRecord{}, fmt.Errorf("%w: final status %q", ErrInvalidArgument, status)
	}
	if version == 0 {
		return FinalRecord{}, fmt.Errorf("%w: zero final version", ErrInvalidArgument)
	}
	if err := validatePayload("final payload", payload, false); err != nil {
		return FinalRecord{}, err
	}
	record := FinalRecord{instanceID: id, descriptor: descriptor, status: status, version: version, payload: cloneBytes(payload)}
	record.digest = record.computeDigest()
	return record, nil
}

func (r FinalRecord) Validate() error {
	restored, err := NewFinalRecord(r.instanceID, r.descriptor, r.status, r.version, r.payload)
	if err != nil {
		return err
	}
	if allZero(r.digest) || restored.digest != r.digest {
		return fmt.Errorf("%w: final record digest mismatch", ErrVerificationFailed)
	}
	return nil
}

func (r FinalRecord) InstanceID() InstanceID { return r.instanceID }
func (r FinalRecord) Descriptor() Descriptor { return r.descriptor }
func (r FinalRecord) Status() FinalStatus    { return r.status }
func (r FinalRecord) Version() uint64        { return r.version }
func (r FinalRecord) Payload() []byte        { return cloneBytes(r.payload) }
func (r FinalRecord) Digest() Digest         { return r.digest }

func (r FinalRecord) computeDigest() Digest {
	h := sha256.New()
	writeDomain(h, finalRecordDigestDomain)
	writeString(h, string(r.instanceID))
	writeString(h, string(r.descriptor.GameID()))
	writeString(h, string(r.descriptor.RulesetVersion()))
	writeString(h, string(r.descriptor.ModuleVersion()))
	writeString(h, string(r.descriptor.FairnessSuiteID()))
	_, _ = h.Write([]byte{r.descriptor.ParticipantCount()})
	writeString(h, string(r.status))
	writeU64(h, r.version)
	writeBytes(h, r.payload)
	var digest Digest
	copy(digest[:], h.Sum(nil))
	return digest
}

type FinalRecordArchive interface {
	Archive(FinalRecord) error
}

type liveEntry struct {
	mu         sync.Mutex
	game       LiveGame
	descriptor Descriptor
	pending    *FinalRecord
	closed     bool
}

type LiveDirectory struct {
	mu      sync.RWMutex
	entries map[InstanceID]*liveEntry
	archive FinalRecordArchive
}

func NewLiveDirectory(archive FinalRecordArchive) (*LiveDirectory, error) {
	if archive == nil {
		return nil, fmt.Errorf("%w: nil final record archive", ErrInvalidArgument)
	}
	return &LiveDirectory{entries: make(map[InstanceID]*liveEntry), archive: archive}, nil
}

func (d *LiveDirectory) Add(expected Descriptor, game LiveGame) error {
	if d == nil {
		return fmt.Errorf("%w: nil live directory", ErrInvalidArgument)
	}
	if game == nil {
		return fmt.Errorf("%w: nil live game", ErrInvalidArgument)
	}
	if err := expected.Validate(); err != nil {
		return err
	}
	descriptor := game.Descriptor()
	if err := descriptor.Validate(); err != nil {
		return err
	}
	if !descriptor.Equal(expected) {
		return fmt.Errorf("%w: live game descriptor mismatch", ErrInvalidArgument)
	}
	id := game.InstanceID()
	if err := validateInstanceID(id); err != nil {
		return err
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, exists := d.entries[id]; exists {
		return fmt.Errorf("%w: %s", ErrInstanceExists, id)
	}
	d.entries[id] = &liveEntry{game: game, descriptor: expected}
	return nil
}

func (d *LiveDirectory) Apply(id InstanceID, command Command) (CommandOutcome, error) {
	entry, err := d.entry(id)
	if err != nil {
		return CommandOutcome{}, err
	}
	entry.mu.Lock()
	defer entry.mu.Unlock()
	if entry.closed {
		return CommandOutcome{}, fmt.Errorf("%w: %s", ErrInstanceNotFound, id)
	}
	if entry.pending != nil {
		return CommandOutcome{}, fmt.Errorf("%w: %s", ErrFinalizationPending, id)
	}
	descriptor := entry.descriptor
	if command.ActorPosition < 1 || command.ActorPosition > descriptor.ParticipantCount() {
		return CommandOutcome{}, fmt.Errorf("%w: actor position %d", ErrInvalidArgument, command.ActorPosition)
	}
	command.Payload = cloneBytes(command.Payload)
	outcome, err := entry.game.Apply(command)
	if err != nil {
		return CommandOutcome{}, err
	}
	outcome.Payload = cloneBytes(outcome.Payload)
	outcome.FinalPayload = cloneBytes(outcome.FinalPayload)
	if !outcome.Terminal {
		if len(outcome.FinalPayload) != 0 {
			return CommandOutcome{}, fmt.Errorf("%w: non-terminal outcome includes final payload", ErrVerificationFailed)
		}
		return outcome, nil
	}
	record, err := NewFinalRecord(id, descriptor, FinalStatusCompleted, outcome.Version, outcome.FinalPayload)
	if err != nil {
		return CommandOutcome{}, err
	}
	entry.pending = &record
	if err := d.archivePendingLocked(id, entry); err != nil {
		return outcome, err
	}
	return outcome, nil
}

func (d *LiveDirectory) View(id InstanceID, request ViewRequest) (GameView, error) {
	entry, err := d.entry(id)
	if err != nil {
		return GameView{}, err
	}
	entry.mu.Lock()
	defer entry.mu.Unlock()
	if entry.closed {
		return GameView{}, fmt.Errorf("%w: %s", ErrInstanceNotFound, id)
	}
	if !request.PublicOnly {
		descriptor := entry.descriptor
		if request.ViewerPosition < 1 || request.ViewerPosition > descriptor.ParticipantCount() {
			return GameView{}, fmt.Errorf("%w: viewer position %d", ErrInvalidArgument, request.ViewerPosition)
		}
	}
	view, err := entry.game.View(request)
	if err != nil {
		return GameView{}, err
	}
	view.Payload = cloneBytes(view.Payload)
	return view, nil
}

func (d *LiveDirectory) Abort(id InstanceID, reason string) (FinalRecord, error) {
	entry, err := d.entry(id)
	if err != nil {
		return FinalRecord{}, err
	}
	entry.mu.Lock()
	defer entry.mu.Unlock()
	if entry.closed {
		return FinalRecord{}, fmt.Errorf("%w: %s", ErrInstanceNotFound, id)
	}
	if entry.pending != nil {
		return FinalRecord{}, fmt.Errorf("%w: %s", ErrFinalizationPending, id)
	}
	if err := validateIdentifier("abortReason", reason); err != nil {
		return FinalRecord{}, err
	}
	outcome, err := entry.game.Abort(reason)
	if err != nil {
		return FinalRecord{}, err
	}
	record, err := NewFinalRecord(id, entry.descriptor, FinalStatusAborted, outcome.Version, outcome.FinalPayload)
	if err != nil {
		return FinalRecord{}, err
	}
	entry.pending = &record
	if err := d.archivePendingLocked(id, entry); err != nil {
		return record, err
	}
	return record, nil
}

func (d *LiveDirectory) RetryArchive(id InstanceID) (FinalRecord, error) {
	entry, err := d.entry(id)
	if err != nil {
		return FinalRecord{}, err
	}
	entry.mu.Lock()
	defer entry.mu.Unlock()
	if entry.closed {
		return FinalRecord{}, fmt.Errorf("%w: %s", ErrInstanceNotFound, id)
	}
	if entry.pending == nil {
		return FinalRecord{}, fmt.Errorf("%w: %s", ErrNotFinalizing, id)
	}
	record := *entry.pending
	if err := d.archivePendingLocked(id, entry); err != nil {
		return record, err
	}
	return record, nil
}

func (d *LiveDirectory) PendingFinalRecord(id InstanceID) (FinalRecord, bool, error) {
	entry, err := d.entry(id)
	if err != nil {
		return FinalRecord{}, false, err
	}
	entry.mu.Lock()
	defer entry.mu.Unlock()
	if entry.closed {
		return FinalRecord{}, false, fmt.Errorf("%w: %s", ErrInstanceNotFound, id)
	}
	if entry.pending == nil {
		return FinalRecord{}, false, nil
	}
	return *entry.pending, true, nil
}

func (d *LiveDirectory) Contains(id InstanceID) bool {
	if d == nil {
		return false
	}
	d.mu.RLock()
	_, exists := d.entries[id]
	d.mu.RUnlock()
	return exists
}

func (d *LiveDirectory) Count() int {
	if d == nil {
		return 0
	}
	d.mu.RLock()
	defer d.mu.RUnlock()
	return len(d.entries)
}

func (d *LiveDirectory) entry(id InstanceID) (*liveEntry, error) {
	if d == nil {
		return nil, fmt.Errorf("%w: nil live directory", ErrInvalidArgument)
	}
	if err := validateInstanceID(id); err != nil {
		return nil, err
	}
	d.mu.RLock()
	entry := d.entries[id]
	d.mu.RUnlock()
	if entry == nil {
		return nil, fmt.Errorf("%w: %s", ErrInstanceNotFound, id)
	}
	return entry, nil
}

func (d *LiveDirectory) archivePendingLocked(id InstanceID, entry *liveEntry) error {
	if entry.pending == nil {
		return fmt.Errorf("%w: %s", ErrNotFinalizing, id)
	}
	if err := d.archive.Archive(*entry.pending); err != nil {
		return errors.Join(ErrArchiveFailed, err)
	}
	d.mu.Lock()
	if current := d.entries[id]; current == entry {
		delete(d.entries, id)
	}
	d.mu.Unlock()
	entry.pending = nil
	entry.closed = true
	return nil
}
