package application

import (
 "context"
 "crypto/sha256"
 "encoding/base64"
 "encoding/json"
 "errors"
 "strings"
 "testing"
 "time"

 "github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
)

type custodiedUnitOpener struct{ phrase string; fail bool }
func (o custodiedUnitOpener) Open(_ context.Context,_ SecureEnvelope,aad []byte,_ RevealKeyContext)([]byte,error){
 if o.fail || len(aad)==0{return nil,errors.New("injected bad envelope")}
 bytes,_:=json.Marshal(revealPlaintext{
  Version:RevealPlaintextV1,HandID:"hand-owned",Seat:1,
  SecureRandom:base64.RawURLEncoding.EncodeToString([]byte(strings.Repeat("r",32))),
  Phrase:o.phrase,Normalization:revealNormalizationNFKCV1,
 })
 return bytes,nil
}
type custodiedUnitProtector struct{ fail bool }
func (p custodiedUnitProtector) Seal(_ context.Context,plain,aad []byte)(ProtectedPayload,error){
 if p.fail{return ProtectedPayload{},errors.New("injected encryption failure")}
 // Test fake only; production Protector uses AES-256-GCM.
 return ProtectedPayload{KeyID:"test-only",Nonce:[]byte{1},Ciphertext:append([]byte(nil),plain...),
  AADDigest:sha256.Sum256(aad)},nil
}
func (p custodiedUnitProtector) Open(_ context.Context,protected ProtectedPayload,aad []byte)([]byte,error){
 if protected.AADDigest!=sha256.Sum256(aad){return nil,errors.New("wrong aad")}
 return append([]byte(nil),protected.Ciphertext...),nil
}

type custodiedReadStore struct{*testStore}
func (s *custodiedReadStore) LoadProtectedContribution(_ context.Context,id string,
 hand domain.HandID,seat domain.Seat,actor domain.AccountID,
)(ProtectedContributionRecord,error){
 record,ok:=s.state.records[id]
 if !ok || record.HandID!=hand || record.Seat!=seat || record.ActorAccountID!=actor{
  return ProtectedContributionRecord{},ErrNotFound
 }
 return record,nil
}

func newCustodiedTestService(t *testing.T,opener custodiedUnitOpener,
 protector custodiedUnitProtector)(*Service,*custodiedReadStore,*testClock){
 t.Helper()
 store:=&custodiedReadStore{newTestStore()}
 now:=time.Unix(1770000000,0).UTC()
 clock:=&testClock{now:now}
 service,err:=NewService(store,clock,&testIDs{},testSetup{},testBeaconVerifier{},
  testLiveRuntime{},opener,protector,testNormalizer{},DefaultConfig())
 if err!=nil{t.Fatal(err)}
 seats:=[3]domain.HandSeat{{Seat:1,AccountID:"alice"},{Seat:2,AccountID:"bob"},{Seat:3,AccountID:"carol"}}
 hand,_,err:=domain.NewClientCustodiedHand("hand-owned","room-owned",seats,
  domain.ServerCommitment{1},"reveal-test",
  domain.BeaconPlan{Provider:"drand-quicknet",Round:"999999999"},
  domain.RevealKeyBinding{PublicKeySHA256:domain.RevealPublicKeyHash{1},BoundAt:now})
 if err!=nil{t.Fatal(err)}
 store.state.hands["hand-owned"]=hand.Snapshot()
 return service,store,clock
}

func TestCustodiedContributionAtomicIdempotentAndRecoverable(t *testing.T){
 service,store,clock:=newCustodiedTestService(t,custodiedUnitOpener{phrase:"remember me"},custodiedUnitProtector{})
 cmd:=testCommand(clock.Now(),CommandHandContributionSubmit,domain.AggregateHand,"hand-owned","cmd-1",1,1)
 input:=SubmitContributionInput{Envelope:SecureEnvelope{KeyID:"reveal-test",Version:"secure-envelope-v1"}}
 result,err:=service.SubmitHandContribution(context.Background(),"alice",cmd,input)
 if err!=nil || !result.Accepted || result.Duplicate {t.Fatalf("first result=%+v err=%v",result,err)}
 snapshot:=store.state.hands["hand-owned"]
 if snapshot.FairnessMode!=domain.FairnessModeClientCustodied ||
    !snapshot.Contributions[0].Revealed || !snapshot.Contributions[0].Committed ||
    snapshot.Contributions[1].Committed || snapshot.Phase!=domain.HandFairnessCommitting{
  t.Fatalf("partial durable contributions: %+v",snapshot)
 }
 if len(store.state.records)!=1{t.Fatalf("protected record count=%d",len(store.state.records))}
 if _,err=domain.RestoreHand(snapshot);err!=nil{t.Fatalf("recover partial hand: %v",err)}
 replay,err:=service.SubmitHandContribution(context.Background(),"alice",cmd,input)
 if err!=nil || !replay.Duplicate || !replay.Accepted || len(store.state.records)!=1 {
  t.Fatalf("replayed result=%+v err=%v",replay,err)
 }
 cmd.CommandID="cmd-2"
 cmd.ClientSeq=2
 cmd.ExpectedVersion=snapshot.Version
 rejected,err:=service.SubmitHandContribution(context.Background(),"alice",cmd,input)
 if err!=nil || rejected.Accepted || rejected.Failure==nil{
  t.Fatalf("repeated contribution not durably refused: %+v %v",rejected,err)
 }
 if len(store.state.records)!=1{t.Fatal("rejected request created another protected record")}
}

func TestCustodiedContributionEncryptionFailureRollsBack(t *testing.T){
 service,store,clock:=newCustodiedTestService(t,custodiedUnitOpener{phrase:"remember me"},
  custodiedUnitProtector{fail:true})
 cmd:=testCommand(clock.Now(),CommandHandContributionSubmit,domain.AggregateHand,"hand-owned","cmd-fail",1,1)
 _,err:=service.SubmitHandContribution(context.Background(),"alice",cmd,
   SubmitContributionInput{Envelope:SecureEnvelope{KeyID:"reveal-test"}})
 if err==nil || len(store.state.records)!=0 || store.state.hands["hand-owned"].Version!=1 ||
    len(store.state.outbox)!=0 || len(store.state.commands)!=0{
  t.Fatalf("partial atomic write on encryption fail: err=%v records=%d",err,len(store.state.records))
 }
}

func TestCustodiedContributionCannotReadBeforeCompleted(t *testing.T){
 service,_,_:=newCustodiedTestService(t,custodiedUnitOpener{phrase:"remember me"},custodiedUnitProtector{})
 if _,err:=service.GetOwnContributionVerification(context.Background(),"alice","hand-owned");err==nil{
  t.Fatal("undisclosed pre-completion material leaked")
 }
 if _,err:=service.GetOwnContributionVerification(context.Background(),"outsider","hand-owned");err==nil{
  t.Fatal("nonparticipant query accepted")
 }
}
