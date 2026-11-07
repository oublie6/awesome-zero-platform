package application

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
)

// CustodyAADV1 binds the HPKE ciphertext to the exact authenticated hand
// operation. The actor/seat association is authoritative in MySQL and the
// protected record AAD; the client is never allowed to nominate a seat.
const CustodyAADV1 = "fair-doudizhu/client-custody-aad/v1"

type SubmitCustodiedInput struct {
	Envelope SecureEnvelope `json:"envelope"`
}

func BuildCustodyAAD(command Command, seat domain.Seat) ([]byte, error) {
	if command.Name != CommandHandCustodySubmit || command.Version != CommandProtocolV1 ||
		command.AggregateType != domain.AggregateHand || !seat.Valid() ||
		command.CommandID == "" || command.AggregateID == "" ||
		command.ClientSeq == 0 || command.ExpectedVersion == 0 {
		return nil, fmt.Errorf("%w: invalid custody envelope AAD", ErrRevealInvalid)
	}
	out := bytes.NewBuffer(make([]byte, 0, 320))
	for _, value := range []string{
		CustodyAADV1, command.Version, command.Name, command.CommandID, command.AggregateID,
	} {
		writeCanonicalString(out, value)
	}
	out.WriteByte(byte(seat))
	writeCanonicalUint64(out, command.ClientSeq)
	writeCanonicalUint64(out, command.ExpectedVersion)
	return out.Bytes(), nil
}

// SubmitCustodiedContribution accepts a single server-encrypted participant
// random value and chosen verification phrase. The client does not persist
// anything. It may discard its random value after a durable accepted receipt;
// a later reconnect uses only the server's committed hand snapshot and log.
//
// All protected data and the commit/reveal aggregate events are atomically
// inserted in the SAME command transaction. Unknown COMMIT outcomes must
// be reconciled using the original command identity, never assumed failed.
func (s *Service) SubmitCustodiedContribution(
	ctx context.Context, actor domain.AccountID, command Command,
	input SubmitCustodiedInput,
) (CommandResult, error) {
	return s.execute(ctx, actor, command, CommandHandCustodySubmit, domain.AggregateHand, false, input,
		func(txCtx context.Context, tx Transaction, now time.Time) (mutationOutcome, error) {
			snapshot, err := tx.LoadHandForUpdate(txCtx, domain.HandID(command.AggregateID))
			if err != nil {
				return mutationOutcome{}, err
			}
			hand, err := domain.RestoreHand(snapshot)
			if err != nil {
				return mutationOutcome{}, wrapInfrastructure("restore hand snapshot", err)
			}
			if snapshot.FairnessMode != domain.FairnessModeClientCustodied {
				return mutationOutcome{}, &aggregateMutationError{cause: domain.ErrWrongPhase, currentVersion: snapshot.Version}
			}
			if err := checkContributionDeadline(snapshot.BeaconPlan, now); err != nil {
				return mutationOutcome{}, &aggregateMutationError{cause: err, currentVersion: snapshot.Version}
			}
			seat, _, err := resolveHandContribution(snapshot, actor)
			if err != nil {
				return mutationOutcome{}, &aggregateMutationError{cause: err, currentVersion: snapshot.Version}
			}
			if input.Envelope.KeyID != snapshot.RevealKeyID {
				return mutationOutcome{}, &aggregateMutationError{cause: ErrRevealInvalid, currentVersion: snapshot.Version}
			}
			aad, err := BuildCustodyAAD(command, seat)
			if err != nil {
				return mutationOutcome{}, &aggregateMutationError{cause: err, currentVersion: snapshot.Version}
			}
			plaintext, err := s.opener.Open(txCtx, input.Envelope, aad, RevealKeyContext{
				KeyID: snapshot.RevealKeyID,
				PublicKeySHA256: [32]byte(snapshot.RevealPublicKeySHA256),
				BoundAt: snapshot.RevealKeyBoundAt,
				UseAt: now,
			})
			if err != nil {
				return mutationOutcome{}, &aggregateMutationError{cause: ErrRevealInvalid, currentVersion: snapshot.Version}
			}
			defer clearBytes(plaintext)

			decoded, err := decodeRevealPlaintext(plaintext, snapshot.ID, seat, s.normalizer, s.config.MaxRevealPhraseBytes)
			if err != nil {
				return mutationOutcome{}, &aggregateMutationError{cause: err, currentVersion: snapshot.Version}
			}
			defer clearBytes(decoded.SecureRandom[:])
			recordID, err := s.ids.NewID()
			if err != nil {
				return mutationOutcome{}, wrapInfrastructure("generate protected contribution record ID", err)
			}
			events, err := hand.SubmitCustodiedContribution(actor, decoded.Digest, recordID, command.ExpectedVersion)
			if err != nil {
				return mutationOutcome{}, &aggregateMutationError{cause: err, currentVersion: snapshot.Version}
			}
			recordAAD, err := BuildContributionRecordAAD(recordID, snapshot.ID, seat, actor, command.CommandID, decoded.Digest)
			if err != nil {
				return mutationOutcome{}, err
			}
			protected, err := s.protector.Seal(txCtx, plaintext, recordAAD)
			if err != nil {
				return mutationOutcome{}, fmt.Errorf("%w: seal contribution: %v", ErrProtectionFailed, err)
			}
			if protected.KeyID == "" || len(protected.Nonce) == 0 || len(protected.Ciphertext) == 0 ||
				protected.AADDigest != sha256.Sum256(recordAAD) {
				return mutationOutcome{}, fmt.Errorf("%w: invalid protected contribution", ErrProtectionFailed)
			}
			record := ProtectedContributionRecord{
				RecordID: recordID, HandID: snapshot.ID, Seat: seat, ActorAccountID: actor,
				CommandID: command.CommandID, ContributionDigest: decoded.Digest,
				ProtectionKeyID: protected.KeyID, Nonce: append([]byte(nil), protected.Nonce...),
				Ciphertext: append([]byte(nil), protected.Ciphertext...),
				AADDigest: protected.AADDigest, CreatedAt: now,
			}
			if err := tx.InsertContributionRecord(txCtx, record); err != nil {
				return mutationOutcome{}, err
			}
			updated := hand.Snapshot()
			if err := tx.UpdateHand(txCtx, updated, snapshot.Version, now); err != nil {
				return mutationOutcome{}, err
			}
			return mutationOutcome{aggregateVersion: updated.Version, events: events}, nil
		})
}
