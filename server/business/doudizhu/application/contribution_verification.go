package application

import (
 "context"
 "crypto/subtle"
 "encoding/base64"
 "encoding/hex"
 "fmt"

 "github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
)

// OwnContributionVerificationV1 is supplied only to the authenticated original
// seat after hand completion. Phrase is never echoed; the client re-enters the
// user's own phrase and locally validates the original commitment.
const OwnContributionVerificationV1 = "doudizhu-own-contribution-v1"

type OwnContributionVerification struct {
 Version string `json:"v"`
 FairnessMode domain.FairnessMode `json:"fairnessMode"`
 HandID domain.HandID `json:"handId"`
 Seat domain.Seat `json:"seat"`
 SecureRandom string `json:"secureRandom"`
 Commitment string `json:"commitment"`
 ContributionDigest string `json:"contributionDigest"`
}

// GetOwnContributionVerification never returns any other participant's
// plaintext, does not expose the user's phrase, and rejects incomplete or
// unarchived hands. Caller must first confirm an independently verified final
// archive through FinalEvidenceService.Get.
func (s *Service) GetOwnContributionVerification(ctx context.Context,actor domain.AccountID,
 handID domain.HandID,
) (OwnContributionVerification,error) {
 if s==nil || s.store==nil || s.protector==nil || ctx==nil {
  return OwnContributionVerification{},fmt.Errorf("%w: verification dependencies",ErrInvalidCommand)
 }
 hand,err:=s.store.LoadHand(ctx,handID)
 if err!=nil{return OwnContributionVerification{},err}
 if hand.ID!=handID || hand.FairnessMode!=domain.FairnessModeClientCustodied ||
    hand.Phase!=domain.HandCompleted {
  return OwnContributionVerification{},fmt.Errorf("%w: contribution not eligible",domain.ErrWrongPhase)
 }
 seat,own,err:=resolveHandContribution(hand,actor)
 if err!=nil{return OwnContributionVerification{},err}
 if !own.Committed || !own.Revealed || own.RecordRef=="" {
  return OwnContributionVerification{},fmt.Errorf("%w: no complete contribution",ErrRevealInvalid)
 }
 reader,ok:=s.store.(interface {
  LoadProtectedContribution(context.Context,string,domain.HandID,domain.Seat,domain.AccountID)(ProtectedContributionRecord,error)
 })
 if !ok {return OwnContributionVerification{},fmt.Errorf("%w: protected read unavailable",ErrProtectionFailed)}
 opener,ok:=s.protector.(interface {
  Open(context.Context,ProtectedPayload,[]byte)([]byte,error)
 })
 if !ok{return OwnContributionVerification{},fmt.Errorf("%w: decryptor unavailable",ErrProtectionFailed)}
 record,err:=reader.LoadProtectedContribution(ctx,own.RecordRef,handID,seat,actor)
 if err!=nil{return OwnContributionVerification{},err}
 if record.RecordID!=own.RecordRef || record.Seat!=seat ||
    record.HandID!=handID || record.ActorAccountID!=actor ||
    subtle.ConstantTimeCompare(record.ContributionDigest[:],own.Digest[:])!=1{
  return OwnContributionVerification{},fmt.Errorf("%w: contribution identity mismatch",ErrProtectionFailed)
 }
 aad,err:=BuildContributionRecordAAD(record.RecordID,handID,seat,actor,record.CommandID,own.Digest)
 if err!=nil{return OwnContributionVerification{},err}
 defer clearBytes(aad)
 plaintext,err:=opener.Open(ctx,ProtectedPayload{
  KeyID:record.ProtectionKeyID,Nonce:record.Nonce,
  Ciphertext:record.Ciphertext,AADDigest:record.AADDigest,
 },aad)
 if err!=nil{return OwnContributionVerification{},fmt.Errorf("%w: protected contribution decrypt failed: %v",ErrProtectionFailed,err)}
 defer clearBytes(plaintext)
 decoded,err:=decodeRevealPlaintext(plaintext,handID,seat,s.normalizer,s.config.MaxRevealPhraseBytes)
 if err!=nil{return OwnContributionVerification{},err}
 defer clearBytes(decoded.SecureRandom[:])
 expectedCommitment:=domain.ComputeClientCommitment(handID,seat,decoded.Digest)
 if subtle.ConstantTimeCompare(decoded.Digest[:],own.Digest[:])!=1 ||
    subtle.ConstantTimeCompare(expectedCommitment[:],own.Commitment[:])!=1 {
  return OwnContributionVerification{},fmt.Errorf("%w: protected contribution digest mismatch",ErrProtectionFailed)
 }
 return OwnContributionVerification{
  Version:OwnContributionVerificationV1,FairnessMode:hand.FairnessMode,
  HandID:handID,Seat:seat,SecureRandom:base64.RawURLEncoding.EncodeToString(decoded.SecureRandom[:]),
  Commitment:hex.EncodeToString(own.Commitment[:]),
  ContributionDigest:hex.EncodeToString(own.Digest[:]),
 },nil
}
