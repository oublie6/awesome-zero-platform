package runtime

import (
	"context"
	"errors"
	"testing"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/carddeck"
)

type fakeSeedPersistence struct {
	values map[domain.HandID]carddeck.Seed
	saveErr error
}
func (f *fakeSeedPersistence) Save(_ context.Context,id domain.HandID,seed carddeck.Seed) error {
	if f.saveErr != nil {return f.saveErr}
	if _,exists:=f.values[id];exists{return ErrSeedExists}
	f.values[id]=seed
	return nil
}
func (f *fakeSeedPersistence) Load(_ context.Context,id domain.HandID)(carddeck.Seed,error){
	v,ok:=f.values[id]
	if !ok{return carddeck.Seed{},ErrSeedNotFound}
	return v,nil
}
func (f *fakeSeedPersistence) Delete(_ context.Context,id domain.HandID)error{
	delete(f.values,id)
	return nil
}
func TestProtectedSeedVaultRecoversSameSeedAcrossProcessRestart(t *testing.T){
	store:=&fakeSeedPersistence{values:make(map[domain.HandID]carddeck.Seed)}
	first,err:=NewPersistentSeedVault(nil,store)
	if err!=nil{t.Fatal(err)}
	if _,err:=first.Prepare("hand-restart");err!=nil{t.Fatal(err)}
	before,err:=first.Read("hand-restart")
	if err!=nil{t.Fatal(err)}
	restarted,err:=NewPersistentSeedVault(nil,store)
	if err!=nil{t.Fatal(err)}
	after,err:=restarted.Read("hand-restart")
	if err!=nil || after!=before{t.Fatalf("server seed changed after restart: %v",err)}
	if !restarted.Contains("hand-restart"){t.Fatal("durable seed not found")}
	if err:=restarted.Release("hand-restart");err!=nil{t.Fatal(err)}
	if _,err:=restarted.Read("hand-restart");!errors.Is(err,ErrSeedNotFound){t.Fatalf("protected seed not removed: %v",err)}
}
func TestProtectedSeedVaultDoesNotExposeUnpersistedSeed(t *testing.T){
	store:=&fakeSeedPersistence{values:make(map[domain.HandID]carddeck.Seed),saveErr:errors.New("database down")}
	v,err:=NewPersistentSeedVault(nil,store)
	if err!=nil{t.Fatal(err)}
	if _,err:=v.Prepare("hand-fail");err==nil{t.Fatal("acknowledged unpersisted server seed")}
	if v.Contains("hand-fail"){t.Fatal("unpersisted seed left in memory")}
}
