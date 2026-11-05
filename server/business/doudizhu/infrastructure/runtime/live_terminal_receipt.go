package runtime

import (
 "context"
 "crypto/sha256"
 "encoding/json"
 "fmt"

 "github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
 "github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/livehand"
 "github.com/oublie6/awesome-zero-platform/server/business/gamecore"
)

// Historical results are recomputed from authenticated MySQL rows and
// protected genesis material, without creating an active runtime executor.
func (c *LiveHandCoordinator) loadTerminalReceipt(ctx context.Context, id domain.HandID, commandID string) (liveReceipt, bool, error) {
 if c == nil || c.journal == nil { return liveReceipt{}, false, nil }
 exists, err := c.journal.HasLiveCommand(ctx, id, commandID)
 if err != nil || !exists { return liveReceipt{}, false, err }
 state, genesis, actions, err := c.journal.ReadTerminalHistory(ctx, id)
 if err != nil { return liveReceipt{}, false, err }
 if !sameImmutableRecoveryHand(state, genesis) { return liveReceipt{}, false, gamecore.ErrVerificationFailed }
 material, err := c.material(genesis)
 if err != nil { return liveReceipt{}, false, err }
 artifact, err := c.module.GenerateSetup(material)
 if err != nil { return liveReceipt{}, false, err }
 if err := c.module.VerifySetup(material, artifact); err != nil { return liveReceipt{}, false, err }
 game, err := livehand.New(genesis, material, artifact)
 if err != nil { return liveReceipt{}, false, err }
 var target liveReceipt
 var found, terminal bool
 for _, action := range actions {
  position, err := game.PositionForAccount(action.Actor)
  if err != nil || position != action.ActorPosition { return liveReceipt{}, false, gamecore.ErrVerificationFailed }
  command := gamecore.Command{ActorPosition: action.ActorPosition, ExpectedVersion: action.ExpectedVersion, Payload: append([]byte(nil), action.Payload...)}
  outcome, err := game.Apply(command)
  if err != nil { return liveReceipt{}, false, fmt.Errorf("replay live action: %w", err) }
  encoded, err := json.Marshal(outcome)
  if err != nil { return liveReceipt{}, false, err }
  if outcome.Version != action.Version || sha256.Sum256(encoded) != action.ResultDigest {
   return liveReceipt{}, false, gamecore.ErrVerificationFailed
  }
  if outcome.Terminal { terminal = true }
  if action.CommandID == commandID {
   target = liveReceipt{actor: action.Actor, clientSeq: action.ClientSeq, command: command, outcome: cloneLiveOutcome(outcome)}
   found = true
  }
 }
 if err := validateReplayedPhase(state.Phase, game.Phase(), terminal); err != nil {
  return liveReceipt{}, false, fmt.Errorf("historical phase mismatch: %w", err)
 }
 return target, found, nil
}
