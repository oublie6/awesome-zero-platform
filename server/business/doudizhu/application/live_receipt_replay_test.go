package application

import (
	"context"
	"errors"
	"testing"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/bidding"
	"github.com/oublie6/awesome-zero-platform/server/business/gamecore"
)

type historicalReceiptStub struct {
	testLiveRuntime
	found bool
	err error
	calls int
	result LiveHandCommandResult
}

func (s *historicalReceiptStub) lookup(actor domain.AccountID, id domain.HandID, commandID string, version uint64) (LiveHandCommandResult, bool, error) {
	s.calls++
	if actor != "player-1" || id != "hand-1" || commandID != "accepted-1" || version != 1 {
		return LiveHandCommandResult{}, true, gamecore.ErrVerificationFailed
	}
	return s.result, s.found, s.err
}
func (s *historicalReceiptStub) FindBidReceipt(_ context.Context, id domain.HandID, actor domain.AccountID,
	commandID string, version uint64, score bidding.Score,
) (LiveHandCommandResult, bool, error) {
	if score != bidding.ScoreThree { return LiveHandCommandResult{},true,gamecore.ErrVerificationFailed }
	return s.lookup(actor,id,commandID,version)
}
func (s *historicalReceiptStub) FindPlayReceipt(_ context.Context, id domain.HandID, actor domain.AccountID,
	commandID string, version uint64, cards []string,
) (LiveHandCommandResult, bool, error) {
	if len(cards)!=1 || cards[0]!="C3" {return LiveHandCommandResult{},true,gamecore.ErrVerificationFailed}
	return s.lookup(actor,id,commandID,version)
}
func (s *historicalReceiptStub) FindPassReceipt(_ context.Context, id domain.HandID, actor domain.AccountID,
	commandID string, version uint64,
) (LiveHandCommandResult, bool, error) {
	return s.lookup(actor,id,commandID,version)
}

func TestHistoricalBidReceiptBeforePhaseGuard(t *testing.T) {
	for _, phase:=range []domain.HandPhase{domain.HandBidding,domain.HandPlaying,domain.HandSettling,domain.HandCompleted} {
		t.Run(string(phase),func(t *testing.T){
			store:=&bidStore{snapshot:biddingHandSnapshot(phase)}
			runtime:=&historicalReceiptStub{found:true,result:LiveHandCommandResult{Version:2,Payload:[]byte("original-bid")}}
			svc:=&Service{store:store,liveHands:runtime}
			result,err:=svc.SubmitLiveHandBidWithID(context.Background(),"player-1","hand-1","accepted-1",1,bidding.ScoreThree)
			if err!=nil || result.Version!=2 || string(result.Payload)!="original-bid" || runtime.calls!=1 ||
				result.Historical!=(phase!=domain.HandBidding){
				t.Fatalf("phase=%s result=%+v calls=%d err=%v",phase,result,runtime.calls,err)
			}
		})
	}
}

func TestHistoricalPlayAndPassReceiptsBeforePhaseGuard(t *testing.T) {
	for _,phase:=range []domain.HandPhase{domain.HandPlaying,domain.HandSettling,domain.HandCompleted}{
		for _,kind:=range []string{"play","pass"}{
			t.Run(string(phase)+"/"+kind,func(t *testing.T){
				runtime:=&historicalReceiptStub{found:true,result:LiveHandCommandResult{Version:8,Payload:[]byte("original-result")}}
				svc:=&Service{store:&bidStore{snapshot:biddingHandSnapshot(phase)},liveHands:runtime}
				var got LiveHandCommandResult
				var err error
				if kind=="play"{got,err=svc.SubmitLiveHandPlayWithID(context.Background(),"player-1","hand-1","accepted-1",1,[]string{"C3"})}else{
					got,err=svc.SubmitLiveHandPassWithID(context.Background(),"player-1","hand-1","accepted-1",1)
				}
				if err!=nil || got.Version!=8 || runtime.calls!=1 || got.Historical!=(phase==domain.HandSettling||phase==domain.HandCompleted){
					t.Fatalf("phase=%s kind=%s got=%+v calls=%d err=%v",phase,kind,got,runtime.calls,err)
				}
			})
		}
	}
}

func TestUnknownHistoricalIDCannotBypassPhaseOrAuthorization(t *testing.T){
	for _,phase:=range []domain.HandPhase{domain.HandPlaying,domain.HandSettling,domain.HandCompleted}{
		for _,kind:=range []string{"bid","play","pass"}{
			t.Run(string(phase)+"/"+kind,func(t *testing.T){
				runtime:=&historicalReceiptStub{found:false}
				svc:=&Service{store:&bidStore{snapshot:biddingHandSnapshot(phase)},liveHands:runtime}
				invoke:=func(actor domain.AccountID)(LiveHandCommandResult,error){
					switch kind {
					case "bid":return svc.SubmitLiveHandBidWithID(context.Background(),actor,"hand-1","accepted-1",1,bidding.ScoreThree)
					case "play":return svc.SubmitLiveHandPlayWithID(context.Background(),actor,"hand-1","accepted-1",1,[]string{"C3"})
					default:return svc.SubmitLiveHandPassWithID(context.Background(),actor,"hand-1","accepted-1",1)
					}
				}
				if _,err:=invoke("outsider");!errors.Is(err,domain.ErrNotSeated)||runtime.calls!=0{
					t.Fatalf("outsider accessed receipt: %v calls=%d",err,runtime.calls)
				}
				if kind!="bid" && phase==domain.HandPlaying {return}
				if _,err:=invoke("player-1");!errors.Is(err,domain.ErrWrongPhase)||runtime.calls!=1{
					t.Fatalf("unknown ID executed across phase: %v calls=%d",err,runtime.calls)
				}
			})
		}
	}
}

func TestHistoricalReceiptRejectsChangedRequestAndInvalidVersion(t *testing.T){
	runtime:=&historicalReceiptStub{found:true,err:gamecore.ErrVerificationFailed}
	svc:=&Service{store:&bidStore{snapshot:biddingHandSnapshot(domain.HandPlaying)},liveHands:runtime}
	if _,err:=svc.SubmitLiveHandBidWithID(context.Background(),"player-1","hand-1","accepted-1",1,bidding.ScoreThree);
		!errors.Is(err,gamecore.ErrVerificationFailed){t.Fatalf("mismatch accepted: %v",err)}
	if _,err:=svc.SubmitLiveHandBidWithID(context.Background(),"player-1","hand-1","accepted-1",0,bidding.ScoreThree);
		!errors.Is(err,ErrInvalidCommand)||runtime.calls!=1{t.Fatalf("invalid version hit index: %v calls=%d",err,runtime.calls)}
}
