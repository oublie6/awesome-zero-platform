package lifecycle

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/application"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/livehand"
)

func TestOldBidNeverExtendsSamePhaseBiddingDeadline(t *testing.T) {
	clock := &testClock{now: time.Unix(1800000000,0)}
	hands := &testHands{snapshots:map[domain.HandID]domain.HandSnapshot{
		"hand-1":testHand("hand-1","seat-1",domain.HandBidding,8),
	}}
	terminal:=&testTerminal{}
	s:=newTestSupervisor(t,hands,terminal,clock,&testIDs{prefix:"historical"})
	if err:=s.TrackBidding(context.Background(),"seat-1","hand-1");err!=nil{t.Fatal(err)}
	clock.Advance(4*time.Second)
	old:=bidResult(t,"hand-1",2,livehand.PhaseBidding,false)
	old.Historical=true
	if err:=s.AfterHistoricalBid(context.Background(),"seat-1","hand-1",old);err!=nil{t.Fatal(err)}
	clock.Advance(2*time.Second)
	_,err:=s.Sweep(context.Background())
	if err!=nil||len(terminal.expireCalls)!=1||terminal.expireCalls[0].reason!=ReasonBiddingTimeout{
		t.Fatalf("old bid prolonged original 5s deadline: %+v err=%v",terminal.expireCalls,err)
	}
}

func TestOldBidCannotDowngradePlayingCountdown(t *testing.T) {
	clock:=&testClock{now:time.Unix(1800000000,0)}
	hands:=&testHands{snapshots:map[domain.HandID]domain.HandSnapshot{
		"hand-1":testHand("hand-1","seat-1",domain.HandBidding,8),
	}}
	terminal:=&testTerminal{}
	s:=newTestSupervisor(t,hands,terminal,clock,&testIDs{prefix:"historical"})
	_,err:=s.AfterBid(context.Background(),"seat-1","hand-1",bidResult(t,"hand-1",3,livehand.PhasePlaying,false))
	if err!=nil{t.Fatal(err)}
	snapshot:=hands.snapshots["hand-1"]
	snapshot.Phase=domain.HandPlaying
	hands.snapshots["hand-1"]=snapshot
	clock.Advance(4*time.Second)
	old:=bidResult(t,"hand-1",2,livehand.PhaseBidding,false)
	old.Historical=true
	if err:=s.AfterHistoricalBid(context.Background(),"seat-1","hand-1",old);err!=nil{t.Fatal(err)}
	clock.Advance(2*time.Second)
	_,err=s.Sweep(context.Background())
	if err!=nil||len(terminal.expireCalls)!=0{
		t.Fatalf("old bidding receipt overwrote playing timer: %+v err=%v",terminal.expireCalls,err)
	}
	clock.Advance(4*time.Second)
	_,err=s.Sweep(context.Background())
	if err!=nil||len(terminal.expireCalls)!=1||terminal.expireCalls[0].reason!=ReasonPlayingTimeout{
		t.Fatalf("playing timer never fired at original 10s: %+v err=%v",terminal.expireCalls,err)
	}
}

func TestOldPlayAndPassNeverExtendPlayingCountdown(t *testing.T) {
	for _,name:=range []string{"play","pass"}{
		t.Run(name,func(t *testing.T){
			clock:=&testClock{now:time.Unix(1800000000,0)}
			hands:=&testHands{snapshots:map[domain.HandID]domain.HandSnapshot{
				"hand-1":testHand("hand-1","seat-1",domain.HandPlaying,9),
			}}
			terminal:=&testTerminal{}
			s:=newTestSupervisor(t,hands,terminal,clock,&testIDs{prefix:"historical"})
			result:=application.LiveHandCommandResult{Version:4}
			payload,_:=json.Marshal(livehand.PlayResult{Version:livehand.PlayResultVersion,
				HandID:"hand-1",StateVersion:4,Phase:livehand.PhasePlaying})
			result.Payload=payload
			if err:=s.AfterPlay(context.Background(),"seat-1","hand-1",result);err!=nil{t.Fatal(err)}
			clock.Advance(9*time.Second)
			result.Historical=true
			if err:=s.AfterHistoricalPlay(context.Background(),"seat-1","hand-1",result);err!=nil{t.Fatal(err)}
			clock.Advance(2*time.Second)
			_,err:=s.Sweep(context.Background())
			if err!=nil||len(terminal.expireCalls)!=1||terminal.expireCalls[0].reason!=ReasonPlayingTimeout{
				t.Fatalf("old %s reset playing timer: %+v err=%v",name,terminal.expireCalls,err)
			}
		})
	}
}

func TestOldAcceptedTerminalResultReconcilesOnlyMissingDurableStages(t *testing.T) {
	hands:=&testHands{snapshots:map[domain.HandID]domain.HandSnapshot{
		"hand-1":testHand("hand-1","seat-1",domain.HandPlaying,9),
	}}
	terminal:=&phaseTestTerminal{testTerminal:&testTerminal{},hands:hands}
	s:=newTestSupervisor(t,hands,terminal,&testClock{now:time.Unix(1800000000,0)},&testIDs{prefix:"historical"})
	payload,_:=json.Marshal(livehand.PlayResult{Version:livehand.PlayResultVersion,
		HandID:"hand-1",StateVersion:60,Phase:livehand.PhaseCompleted})
	result:=application.LiveHandCommandResult{Version:60,Payload:payload,Historical:true}
	if err:=s.AfterHistoricalPlay(context.Background(),"seat-1","hand-1",result);err!=nil{t.Fatal(err)}
	if hands.snapshots["hand-1"].Phase!=domain.HandCompleted||len(terminal.commands)!=2{
		t.Fatalf("terminal did not reconcile exactly once: %+v cmds=%d",hands.snapshots["hand-1"],len(terminal.commands))
	}
	if err:=s.AfterHistoricalPlay(context.Background(),"seat-1","hand-1",result);err!=nil{t.Fatal(err)}
	if len(terminal.commands)!=2{t.Fatal("historical completion ran twice")}
}

func TestOldAcceptedWinningBidReconcilesOnlyMissingDurablePhase(t *testing.T) {
	hands:=&testHands{snapshots:map[domain.HandID]domain.HandSnapshot{
		"hand-1":testHand("hand-1","seat-1",domain.HandBidding,8),
	}}
	terminal:=&phaseTestTerminal{testTerminal:&testTerminal{},hands:hands}
	clock:=&testClock{now:time.Unix(1800000000,0)}
	s:=newTestSupervisor(t,hands,terminal,clock,&testIDs{prefix:"historical"})
	bid:=bidResult(t,"hand-1",2,livehand.PhasePlaying,false)
	bid.Historical=true
	if err:=s.AfterHistoricalBid(context.Background(),"seat-1","hand-1",bid);err!=nil{t.Fatal(err)}
	if hands.snapshots["hand-1"].Phase!=domain.HandPlaying||len(terminal.commands)!=1{
		t.Fatalf("historical winning bid did not complete its missing phase: %+v cmds=%d",hands.snapshots["hand-1"],len(terminal.commands))
	}
	clock.Advance(4*time.Second)
	if err:=s.AfterHistoricalBid(context.Background(),"seat-1","hand-1",bid);err!=nil{t.Fatal(err)}
	clock.Advance(7*time.Second)
	_,err:=s.Sweep(context.Background())
	if err!=nil||len(terminal.expireCalls)!=1||terminal.expireCalls[0].reason!=ReasonPlayingTimeout{
		t.Fatalf("historical winning bid postponed playing timer: %+v err=%v",terminal.expireCalls,err)
	}
}
