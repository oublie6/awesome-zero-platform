package runtime

import (
 "context"
 "crypto/sha256"
 "crypto/subtle"
 "encoding/binary"
 "encoding/hex"
 "encoding/json"
 "errors"
 "fmt"
 "io"
 "net/http"
 "strconv"
 "strings"
 "time"

 "github.com/cloudflare/circl/sign/bls"
 "github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
)

const (
 DrandQuicknetProvider = "drand-quicknet"
 DrandQuicknetChainHash = "52db9ba70e0cc0f6eaf7803dd07447a1f5477735fd3f661792ba94600c84e971"
 DrandQuicknetGenesisUnix = int64(1692803367)
 DrandQuicknetPeriod = 3*time.Second
 drandQuicknetProofPrefix = "drand-quicknet:"
 // Only a known HTTPS relay is permitted: no configurable host or redirect
 // target is constructed from an untrusted room/hand/command.
 drandQuicknetRelay = "https://api.drand.sh/v2/beacons/quicknet/rounds/"
)
var ErrUnverifiedBeacon=errors.New("independent drand beacon verification failed")

// DrandQuicknet uses the RFC9380 unchained G1 signature scheme. The public
// key is provisioned and pinned independently of the HTTP response.
// No trust is placed in relay-supplied chain metadata or random bytes.
type DrandQuicknet struct {
 key *bls.PublicKey[bls.G2]
 client *http.Client
}
func NewDrandQuicknet(pinnedPublicKeyHex string)(*DrandQuicknet,error){
 raw,err:=hex.DecodeString(pinnedPublicKeyHex)
 if err!=nil || len(raw)!=96 {return nil,fmt.Errorf("%w: require pinned 96-byte G2 public key",ErrUnverifiedBeacon)}
 key:=new(bls.PublicKey[bls.G2])
 if err:=key.UnmarshalBinary(raw);err!=nil || !key.Validate(){
  return nil,fmt.Errorf("%w: invalid pinned public key",ErrUnverifiedBeacon)
 }
 return &DrandQuicknet{key:key,client:&http.Client{
  Timeout:5*time.Second,
  CheckRedirect:func(*http.Request,[]*http.Request)error{return http.ErrUseLastResponse},
 }},nil
}
func quicknetRound(text string)(uint64,error){
 if len(text)==0 || len(text)>20 || strings.Trim(text,"0123456789")!="" || text[0]=='0'{
  return 0,ErrUnverifiedBeacon
 }
 round,err:=strconv.ParseUint(text,10,64)
 if err!=nil || round==0 {return 0,ErrUnverifiedBeacon}
 return round,nil
}
func (d *DrandQuicknet) Verify(ctx context.Context,plan domain.BeaconPlan,candidate domain.BeaconValue)error{
 if d==nil || d.key==nil || plan.Provider!=DrandQuicknetProvider ||
  candidate.Provider!=plan.Provider || candidate.Round!=plan.Round{return ErrUnverifiedBeacon}
 round,err:=quicknetRound(plan.Round)
 if err!=nil{return err}
 if !strings.HasPrefix(candidate.ProofRef,drandQuicknetProofPrefix){return ErrUnverifiedBeacon}
 sig,err:=hex.DecodeString(strings.TrimPrefix(candidate.ProofRef,drandQuicknetProofPrefix))
 if err!=nil || len(sig)!=48 {return ErrUnverifiedBeacon}
 digest:=sha256.Sum256(sig)
 if subtle.ConstantTimeCompare(digest[:],candidate.Digest[:])!=1{return ErrUnverifiedBeacon}
 var number [8]byte
 binary.BigEndian.PutUint64(number[:],round)
 message:=sha256.Sum256(number[:])
 if err:=ctx.Err();err!=nil{return err}
 if !bls.Verify[bls.G2](d.key,message[:],sig){return ErrUnverifiedBeacon}
 return nil
}
func (d *DrandQuicknet) Fetch(ctx context.Context,plan domain.BeaconPlan)(domain.BeaconValue,error){
 if d==nil || d.client==nil || plan.Provider!=DrandQuicknetProvider {
  return domain.BeaconValue{},ErrUnverifiedBeacon
 }
 round,err:=quicknetRound(plan.Round)
 if err!=nil{return domain.BeaconValue{},err}
 req,err:=http.NewRequestWithContext(ctx,http.MethodGet,drandQuicknetRelay+strconv.FormatUint(round,10),nil)
 if err!=nil{return domain.BeaconValue{},err}
 resp,err:=d.client.Do(req)
 if err!=nil{return domain.BeaconValue{},fmt.Errorf("retrieve public drand round: %w",err)}
 defer resp.Body.Close()
 if resp.StatusCode!=http.StatusOK{return domain.BeaconValue{},fmt.Errorf("%w: relay status %d",ErrUnverifiedBeacon,resp.StatusCode)}
 raw,err:=io.ReadAll(io.LimitReader(resp.Body,2049))
 if err!=nil || len(raw)>2048{return domain.BeaconValue{},ErrUnverifiedBeacon}
 var beacon struct{
  Round uint64 `json:"round"`
  Signature string `json:"signature"`
 }
 if err:=json.Unmarshal(raw,&beacon);err!=nil || beacon.Round!=round {return domain.BeaconValue{},ErrUnverifiedBeacon}
 signature,err:=hex.DecodeString(beacon.Signature)
 if err!=nil || len(signature)!=48{return domain.BeaconValue{},ErrUnverifiedBeacon}
 digest:=sha256.Sum256(signature)
 value:=domain.BeaconValue{
  Provider:DrandQuicknetProvider,Round:plan.Round,
  Digest:domain.BeaconDigest(digest),
  ProofRef:drandQuicknetProofPrefix+hex.EncodeToString(signature),
 }
 if err:=d.Verify(ctx,plan,value);err!=nil{return domain.BeaconValue{},err}
 return value,nil
}
// FutureRound pins an unobservable beacon round AT HAND CREATION, at least
// 60 seconds in the future. Even a relay that refuses to answer cannot choose
// or substitute another round after player commitments are revealed.
func DrandFutureRound(now time.Time)(string,error){
 next:=now.UTC().Add(60*time.Second).Unix()
 if next<DrandQuicknetGenesisUnix{return "",ErrUnverifiedBeacon}
 round:=uint64((next-DrandQuicknetGenesisUnix)/int64(DrandQuicknetPeriod/time.Second))+2
 return strconv.FormatUint(round,10),nil
}
