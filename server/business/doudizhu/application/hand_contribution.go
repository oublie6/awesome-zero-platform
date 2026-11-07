package application

import (
 "context"
 "crypto/sha256"
 "fmt"
 "time"

 "github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
)

// SubmitContributionInput contains only an HPKE envelope. The user phrase and
// client-generated random bytes are absent from durable command payloads,
// ordinary logs and broadcasts; they are encrypted at rest before SQL insert.
type SubmitContributionInput struct{ Envelope SecureEnvelope }

// SubmitHandContribution commits and reveals one authenticated seat in the
// same SQL transaction as the encrypted record and command receipt. Restart
// recovery needs only the server-side protected record, not a client backup.
func (s *Service) SubmitHandContribution(
 ctx context.Context, actor domain.AccountID, command Command, input SubmitContributionInput,
) (CommandResult,error) {
 return s.execute(ctx,actor,command,CommandHandContributionSubmit,domain.AggregateHand,false,input,
  func(txCtx context.Context,tx Transaction,now time.Time)(mutationOutcome,error){
   snapshot,err:=tx.LoadHandForUpdate(txCtx,domain.HandID(command.AggregateID))
   if err!=nil{return mutationOutcome{},err}
   if snapshot.FairnessMode!=domain.FairnessModeClientCustodied {
    return mutationOutcome{},&aggregateMutationError{cause:domain.ErrWrongPhase,currentVersion:snapshot.Version}
   }
   hand,err:=domain.RestoreHand(snapshot)
   if err!=nil{return mutationOutcome{},wrapInfrastructure("restore custodied hand",err)}
   if err=checkContributionDeadline(snapshot.BeaconPlan,now);err!=nil{
    return mutationOutcome{},&aggregateMutationError{cause:err,currentVersion:snapshot.Version}
   }
   seat,own,err:=resolveHandContribution(snapshot,actor)
   if err!=nil{return mutationOutcome{},&aggregateMutationError{cause:err,currentVersion:snapshot.Version}}
   if snapshot.Phase!=domain.HandFairnessCommitting || own.Committed || own.Revealed{
    return mutationOutcome{},&aggregateMutationError{cause:domain.ErrWrongPhase,currentVersion:snapshot.Version}
   }
   if input.Envelope.KeyID != snapshot.RevealKeyID {
    return mutationOutcome{},&aggregateMutationError{cause:ErrRevealInvalid,currentVersion:snapshot.Version}
   }
   aad,err:=BuildCustodiedAAD(command,actor,seat,snapshot.RevealKeyID)
   if err!=nil{return mutationOutcome{},&aggregateMutationError{cause:err,currentVersion:snapshot.Version}}
   defer clearBytes(aad)
   plaintext,err:=s.opener.Open(txCtx,input.Envelope,aad,RevealKeyContext{
    KeyID:snapshot.RevealKeyID,
    PublicKeySHA256:[32]byte(snapshot.RevealPublicKeySHA256),
    BoundAt:snapshot.RevealKeyBoundAt,
    UseAt:now,
   })
   if err!=nil{return mutationOutcome{},&aggregateMutationError{cause:ErrRevealInvalid,currentVersion:snapshot.Version}}
   defer clearBytes(plaintext)
   decoded,err:=decodeRevealPlaintext(plaintext,snapshot.ID,seat,s.normalizer,s.config.MaxRevealPhraseBytes)
   if err!=nil{return mutationOutcome{},&aggregateMutationError{cause:err,currentVersion:snapshot.Version}}
   defer clearBytes(decoded.SecureRandom[:])
   recordID,err:=s.ids.NewID()
   if err!=nil{return mutationOutcome{},wrapInfrastructure("new protected record ID",err)}
   events,err:=hand.SubmitCustodiedContribution(actor,decoded.Digest,recordID,command.ExpectedVersion)
   if err!=nil{return mutationOutcome{},&aggregateMutationError{cause:err,currentVersion:snapshot.Version}}
   recordAAD,err:=BuildContributionRecordAAD(recordID,snapshot.ID,seat,actor,command.CommandID,decoded.Digest)
   if err!=nil{return mutationOutcome{},err}
   defer clearBytes(recordAAD)
   protected,err:=s.protector.Seal(txCtx,plaintext,recordAAD)
   if err!=nil{return mutationOutcome{},fmt.Errorf("%w: encrypt custodied player material: %v",ErrProtectionFailed,err)}
   if protected.KeyID=="" || len(protected.Nonce)==0 || len(protected.Ciphertext)==0 ||
      protected.AADDigest!=sha256.Sum256(recordAAD) {
    return mutationOutcome{},fmt.Errorf("%w: malformed encrypted record",ErrProtectionFailed)
   }
   record:=ProtectedContributionRecord{
    RecordID:recordID,HandID:snapshot.ID,Seat:seat,ActorAccountID:actor,
    CommandID:command.CommandID,ContributionDigest:decoded.Digest,
    ProtectionKeyID:protected.KeyID,Nonce:append([]byte(nil),protected.Nonce...),
    Ciphertext:append([]byte(nil),protected.Ciphertext...),AADDigest:protected.AADDigest,CreatedAt:now,
   }
   if err=tx.InsertContributionRecord(txCtx,record);err!=nil{return mutationOutcome{},err}
   updated:=hand.Snapshot()
   if err=tx.UpdateHand(txCtx,updated,snapshot.Version,now);err!=nil{return mutationOutcome{},err}
   return mutationOutcome{aggregateVersion:updated.Version,events:events},nil
  })
}
