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

type phaseTestTerminal struct {
	*testTerminal
	hands    *testHands
	commands []application.Command
}

func (p *phaseTestTerminal) advance(_ context.Context, actor domain.AccountID, cmd application.Command,
	from, to domain.HandPhase) (application.CommandResult, error) {
	hand := p.hands.snapshots[domain.HandID(cmd.AggregateID)]
	if actor != SystemActor || hand.Phase != from || cmd.ExpectedVersion != hand.Version ||
		cmd.ClientSeq != hand.Version {
		return application.CommandResult{Accepted: false}, nil
	}
	hand.Phase = to
	hand.Version++
	p.hands.snapshots[hand.ID] = hand
	p.commands = append(p.commands, cmd)
	return application.CommandResult{Accepted: true, AggregateVersion: hand.Version}, nil
}
func (p *phaseTestTerminal) StartHandPlaying(ctx context.Context, actor domain.AccountID, cmd application.Command) (application.CommandResult, error) {
	return p.advance(ctx, actor, cmd, domain.HandBidding, domain.HandPlaying)
}
func (p *phaseTestTerminal) StartHandSettlement(ctx context.Context, actor domain.AccountID, cmd application.Command) (application.CommandResult, error) {
	return p.advance(ctx, actor, cmd, domain.HandPlaying, domain.HandSettling)
}
func (p *phaseTestTerminal) CompleteHand(ctx context.Context, actor domain.AccountID, cmd application.Command) (application.CommandResult, error) {
	return p.advance(ctx, actor, cmd, domain.HandSettling, domain.HandCompleted)
}
func TestWinningBidDurablyStartsPlayingAndCompletedPlayDurablySettles(t *testing.T) {
	hands := &testHands{snapshots: map[domain.HandID]domain.HandSnapshot{
		"hand-real": testHand("hand-real", "seat-1", domain.HandBidding, 8),
	}}
	terminal := &phaseTestTerminal{testTerminal: &testTerminal{}, hands: hands}
	clock := &testClock{now: time.Unix(1800000000, 0)}
	supervisor := newTestSupervisor(t, hands, terminal, clock, &testIDs{prefix: "advance"})
	if _, err := supervisor.AfterBid(context.Background(), "seat-1", "hand-real",
		bidResult(t, "hand-real", 4, livehand.PhasePlaying, false)); err != nil {
		t.Fatal(err)
	}
	snapshot := hands.snapshots["hand-real"]
	if snapshot.Phase != domain.HandPlaying || snapshot.Version != 9 || len(terminal.commands) != 1 ||
		terminal.commands[0].Name != application.CommandHandPlayStart || terminal.commands[0].ClientSeq != 8 {
		t.Fatalf("bid failed to durably advance: snapshot=%+v commands=%+v", snapshot, terminal.commands)
	}
	payload, _ := json.Marshal(livehand.PlayResult{Version: livehand.PlayResultVersion, HandID: "hand-real",
		StateVersion: 5, Phase: livehand.PhaseCompleted})
	if err := supervisor.AfterPlay(context.Background(), "seat-1", "hand-real",
		application.LiveHandCommandResult{Version: 5, Payload: payload}); err != nil {
		t.Fatal(err)
	}
	snapshot = hands.snapshots["hand-real"]
	if snapshot.Phase != domain.HandCompleted || snapshot.Version != 11 || len(terminal.commands) != 3 {
		t.Fatalf("completed game not persisted: snapshot=%+v commands=%+v", snapshot, terminal.commands)
	}
	if terminal.commands[1].Name != application.CommandHandSettlement ||
		terminal.commands[2].Name != application.CommandHandComplete ||
		terminal.commands[1].ClientSeq != 9 || terminal.commands[2].ClientSeq != 10 {
		t.Fatalf("unexpected durable stage commands: %+v", terminal.commands)
	}
}
