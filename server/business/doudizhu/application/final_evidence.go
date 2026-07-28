package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/livehand"
	"github.com/oublie6/awesome-zero-platform/server/business/gamecore"
)

const FinalEvidenceResultV1 = "doudizhu-final-evidence-result-v1"

var (
	ErrFinalEvidenceInvalid   = errors.New("doudizhu application: invalid final evidence query")
	ErrFinalEvidenceForbidden = errors.New("doudizhu application: final evidence is restricted to seated participants")
)

type FinalEvidenceHandReader interface {
	LoadHand(context.Context, domain.HandID) (domain.HandSnapshot, error)
}

type FinalRecordLoader interface {
	LoadFinalRecord(context.Context, gamecore.InstanceID) (gamecore.FinalRecord, time.Time, error)
}

type FinalRecordVerifier interface {
	Verify(gamecore.FinalRecord) (livehand.FinalVerificationReport, error)
}

type DoudizhuFinalRecordVerifier struct{}

func (DoudizhuFinalRecordVerifier) Verify(record gamecore.FinalRecord) (livehand.FinalVerificationReport, error) {
	return livehand.VerifyFinalRecord(record)
}

type FinalEvidenceService struct {
	hands    FinalEvidenceHandReader
	records  FinalRecordLoader
	verifier FinalRecordVerifier
}

type FinalEvidenceResult struct {
	Version      string                           `json:"v"`
	HandID       domain.HandID                    `json:"handId"`
	Status       gamecore.FinalStatus             `json:"status"`
	FinalVersion uint64                           `json:"finalVersion"`
	ArchivedAt   time.Time                        `json:"archivedAt"`
	Payload      json.RawMessage                  `json:"payload"`
	Verification livehand.FinalVerificationReport `json:"verification"`
}

func NewFinalEvidenceService(hands FinalEvidenceHandReader, records FinalRecordLoader, verifier FinalRecordVerifier) (*FinalEvidenceService, error) {
	if hands == nil || records == nil || verifier == nil {
		return nil, fmt.Errorf("%w: dependencies", ErrFinalEvidenceInvalid)
	}
	return &FinalEvidenceService{hands: hands, records: records, verifier: verifier}, nil
}

func (s *FinalEvidenceService) Get(ctx context.Context, actor domain.AccountID, handID domain.HandID) (FinalEvidenceResult, error) {
	if s == nil || s.hands == nil || s.records == nil || s.verifier == nil || ctx == nil {
		return FinalEvidenceResult{}, fmt.Errorf("%w: service", ErrFinalEvidenceInvalid)
	}
	if strings.TrimSpace(string(actor)) == "" || actor != domain.AccountID(strings.TrimSpace(string(actor))) || len(actor) > 128 ||
		strings.TrimSpace(string(handID)) == "" || handID != domain.HandID(strings.TrimSpace(string(handID))) || len(handID) > 128 {
		return FinalEvidenceResult{}, fmt.Errorf("%w: identity", ErrFinalEvidenceInvalid)
	}
	hand, err := s.hands.LoadHand(ctx, handID)
	if err != nil {
		return FinalEvidenceResult{}, fmt.Errorf("load hand for final evidence: %w", err)
	}
	if hand.ID != handID || !isSeatedParticipant(hand, actor) {
		return FinalEvidenceResult{}, ErrFinalEvidenceForbidden
	}
	record, archivedAt, err := s.records.LoadFinalRecord(ctx, gamecore.InstanceID(handID))
	if err != nil {
		return FinalEvidenceResult{}, fmt.Errorf("load final evidence: %w", err)
	}
	if record.InstanceID() != gamecore.InstanceID(handID) || archivedAt.IsZero() {
		return FinalEvidenceResult{}, fmt.Errorf("%w: archive identity", ErrFinalEvidenceInvalid)
	}
	report, err := s.verifier.Verify(record)
	if err != nil {
		return FinalEvidenceResult{}, fmt.Errorf("verify final evidence: %w", err)
	}
	payload := record.Payload()
	return FinalEvidenceResult{
		Version:      FinalEvidenceResultV1,
		HandID:       handID,
		Status:       record.Status(),
		FinalVersion: record.Version(),
		ArchivedAt:   archivedAt.UTC(),
		Payload:      append(json.RawMessage(nil), payload...),
		Verification: report,
	}, nil
}

func isSeatedParticipant(hand domain.HandSnapshot, actor domain.AccountID) bool {
	for _, seat := range hand.Seats {
		if seat.AccountID == actor {
			return true
		}
	}
	return false
}
