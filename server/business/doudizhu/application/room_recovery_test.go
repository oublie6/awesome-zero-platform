package application

import (
 "context"
 "testing"

 "github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
)

type discoverStore struct {
 *testStore
 ids []domain.RoomID
}
func (s *discoverStore) FindRecoverableRoomIDs(_ context.Context,_ domain.AccountID)([]domain.RoomID,error){
 return append([]domain.RoomID(nil),s.ids...),nil
}
func TestRecoverableRoomsRequireCurrentMembershipAndExcludeClosed(t *testing.T){
 ctx:=context.Background()
 active,_,err:=domain.NewRoom("active-one","actor-A")
 if err!=nil{t.Fatal(err)}
 for _,actor:=range []domain.AccountID{"actor-B","actor-C"}{
   if _,err=active.Join(actor,active.Snapshot().Version);err!=nil{t.Fatal(err)}
 }
 for _,actor:=range []domain.AccountID{"actor-A","actor-B","actor-C"}{
   if _,err=active.SetReady(actor,true,active.Snapshot().Version);err!=nil{t.Fatal(err)}
 }
 if _,err=active.StartHand("actor-A","active-hand",active.Snapshot().Version);err!=nil{t.Fatal(err)}
 waiting,_,err:=domain.NewRoom("waiting-one","actor-A")
 if err!=nil{t.Fatal(err)}
 outsider,_,err:=domain.NewRoom("other-private","someone-else")
 if err!=nil{t.Fatal(err)}
 closed,_,err:=domain.NewRoom("closed-one","actor-A")
 if err!=nil{t.Fatal(err)}
 if _,err=closed.Leave("actor-A",closed.Snapshot().Version);err!=nil{t.Fatal(err)}
 fake:=newTestStore()
 for id,room:=range map[string]*domain.Room{
   "active-one":active,"waiting-one":waiting,"other-private":outsider,"closed-one":closed,
 }{fake.state.rooms[id]=room.Snapshot()}
 fake.state.sequences[testSequenceKey(domain.AggregateRoom,"active-one","actor-B")]=17
 store:=&discoverStore{testStore:fake,ids:[]domain.RoomID{
   "active-one","other-private","closed-one","waiting-one",
 }}
 service:=&Service{store:store}
 views,err:=service.GetRecoverableRooms(ctx,"actor-B")
 if err!=nil{t.Fatal(err)}
 if len(views)!=1 || views[0].RoomID!="active-one" ||
  views[0].Status!=domain.RoomHandActive || views[0].ActiveHandID!="active-hand" ||
  views[0].LastClientSequence!=17 || !views[0].Member {
  t.Fatalf("discovery lost active member or leaked others: %+v",views)
 }
 outsiderViews,err:=service.GetRecoverableRooms(ctx,"someone-else")
 if err!=nil{t.Fatal(err)}
 if len(outsiderViews)!=1 || outsiderViews[0].RoomID!="other-private" ||
   outsiderViews[0].ActiveHandID!=""{
  t.Fatalf("outsider received another room's private locator: %+v",outsiderViews)
 }
 store.ids=[]domain.RoomID{"active-one","active-one"}
 if _,err=service.GetRecoverableRooms(ctx,"actor-B");err==nil{
  t.Fatal("repeated discovery identity silently accepted")
 }
 if _,err=service.GetRecoverableRooms(ctx,"");err==nil{
  t.Fatal("unauthenticated discovery accepted")
 }
}
