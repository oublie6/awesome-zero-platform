package doudizhuapi

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/application"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/bidding"
)

type historicalBackendStub struct{ backendStub }
func (b *historicalBackendStub) SubmitLiveHandBidWithID(context.Context,domain.AccountID,domain.HandID,string,uint64,bidding.Score)(application.LiveHandCommandResult,error){
	return application.LiveHandCommandResult{Version:2,Payload:[]byte(`{"kind":"bid"}`),Historical:true},nil
}
func (b *historicalBackendStub) SubmitLiveHandPlayWithID(context.Context,domain.AccountID,domain.HandID,string,uint64,[]string)(application.LiveHandCommandResult,error){
	return application.LiveHandCommandResult{Version:3,Payload:[]byte(`{"kind":"play"}`),Historical:true},nil
}
func (b *historicalBackendStub) SubmitLiveHandPassWithID(context.Context,domain.AccountID,domain.HandID,string,uint64)(application.LiveHandCommandResult,error){
	return application.LiveHandCommandResult{Version:4,Payload:[]byte(`{"kind":"pass"}`),Historical:true},nil
}

func TestRecoveredHistoricalLiveResultsDoNotRepeatLifecycleSideEffects(t *testing.T){
	backend:=&historicalBackendStub{}
	life:=&lifecycleStub{}
	dispatcher:=newDispatcherForTest(t,backend,life)
	for _,tt:=range []struct{name string;payload json.RawMessage;want uint64}{
		{TypeHandBid,json.RawMessage(`{"score":3}`),2},
		{TypeHandPlay,json.RawMessage(`{"cards":["C3"]}`),3},
		{TypeHandPass,nil,4},
	}{
		req:=CommandRequest{Version:CommandRequestV1,RequestID:"old-"+tt.name,Type:tt.name,
			AggregateID:"hand-1",ClientSeq:91,ExpectedVersion:1,Payload:tt.payload}
		got,err:=dispatcher.Execute(context.Background(),"account-1",req)
		if err!=nil||got.Live==nil||got.Live.Version!=tt.want||got.Termination!=nil {
			t.Fatalf("historical %s result=%+v err=%v",tt.name,got,err)
		}
	}
	if life.bidCalls!=0||life.playCalls!=0{
		t.Fatalf("historical results restarted lifecycle: bid=%d play=%d",life.bidCalls,life.playCalls)
	}
}
