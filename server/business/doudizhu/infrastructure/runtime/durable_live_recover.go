package runtime

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/application"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/livehand"
	"github.com/oublie6/awesome-zero-platform/server/business/gamecore"
)

// RecoverActive fences previous writers in MySQL before replay and accepting
// requests. It refuses any missing/corrupt action rather than regenerating a
// different deal or silently skipping acknowledged live operations.
// Call before registering the realtime/HTTP handlers or launching sweeps.
func (c *LiveHandCoordinator) RecoverActive(ctx context.Context,
	loadHand func(context.Context,domain.HandID)(domain.HandSnapshot,error),
)error{
	if c==nil || c.journal==nil || loadHand==nil {return fmt.Errorf("durable recovery dependencies unavailable")}
	ids,err:=c.journal.List(ctx)
	if err!=nil{return fmt.Errorf("list live recovery: %w",err)}
	for _,id:=range ids {
		snapshot,err:=loadHand(ctx,id)
		if errors.Is(err,application.ErrNotFound){continue} // orphan from rolled-back genesis transaction
		if err!=nil{return fmt.Errorf("load authoritative hand %s: %w",id,err)}
		if snapshot.Phase!=domain.HandBidding {
			// Completed/cancelled hands are driven by durable domain state and
			// final evidence. Never resurrect them as active executors.
			continue
		}
		epoch,watermark,genesis,err:=c.journal.Acquire(ctx,id)
		if err!=nil{return fmt.Errorf("take fenced live ownership %s: %w",id,err)}
		material,err:=c.material(genesis)
		if err!=nil{return fmt.Errorf("recover protected fairness material %s: %w",id,err)}
		artifact,err:=c.module.GenerateSetup(material)
		if err!=nil{return fmt.Errorf("regenerate committed deal %s: %w",id,err)}
		if err:=c.module.VerifySetup(material,artifact);err!=nil{return fmt.Errorf("verify recovered setup %s: %w",id,err)}
		game,err:=livehand.New(genesis,material,artifact)
		if err!=nil{return fmt.Errorf("restore live genesis %s: %w",id,err)}
		actions,err:=c.journal.Read(ctx,id)
		if err!=nil{return fmt.Errorf("load protected live log %s: %w",id,err)}
		if watermark!=uint64(len(actions))+1 {
			return fmt.Errorf("%w: live watermark disagrees with replay log for %s",gamecore.ErrVerificationFailed,id)
		}
		var finalOutcome gamecore.CommandOutcome
		for _,item:=range actions {
			position,err:=game.PositionForAccount(item.Actor)
			if err!=nil || position!=item.ActorPosition {
				return fmt.Errorf("%w: forged or changed live actor for %s",gamecore.ErrVerificationFailed,id)
			}
			outcome,err:=game.Apply(gamecore.Command{ActorPosition:item.ActorPosition,
				ExpectedVersion:item.ExpectedVersion,Payload:append([]byte(nil),item.Payload...)})
			if err!=nil{return fmt.Errorf("replay live action %s/%d: %w",id,item.Version,err)}
			raw,err:=json.Marshal(outcome)
			if err!=nil{return err}
			digest:=sha256.Sum256(raw)
			if outcome.Version!=item.Version || digest!=item.ResultDigest {
				return fmt.Errorf("%w: divergent live result for %s/%d",gamecore.ErrVerificationFailed,id,item.Version)
			}
			if outcome.Terminal {finalOutcome=outcome}
		}
		if finalOutcome.Terminal {
			record,err:=c.directory.ArchiveRecoveredFinal(game,finalOutcome)
			if err!=nil{return fmt.Errorf("archive recovered game completion %s: %w",id,err)}
			c.finish(id,record)
			continue
		}
		if err:=c.directory.Add(game.Descriptor(),game);err!=nil{return err}
		c.mu.Lock()
		c.games[id]=game
		c.epochs[id]=epoch
		c.mu.Unlock()
	}
	return nil
}
