package livehand

import (
	"sync"
	"testing"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/bidding"
	"github.com/oublie6/awesome-zero-platform/server/business/gamecore"
)

func TestLiveDirectorySerializesConcurrentBidsForOneHand(t *testing.T) {
	game := newDirectBiddingGame(t, 1)
	directory, err := gamecore.NewLiveDirectory(discardFinalArchive{})
	if err != nil {
		t.Fatal(err)
	}
	if err := directory.Add(game.Descriptor(), game); err != nil {
		t.Fatal(err)
	}

	const workers = 2
	results := make([]gamecore.CommandOutcome, workers)
	errorsSeen := make([]error, workers)
	start := make(chan struct{})
	var wait sync.WaitGroup
	wait.Add(workers)
	for index := 0; index < workers; index++ {
		go func(index int) {
			defer wait.Done()
			<-start
			results[index], errorsSeen[index] = directory.Apply(game.InstanceID(), bidCommand(t, 1, 1, bidding.ScoreOne))
		}(index)
	}
	close(start)
	wait.Wait()

	accepted := 0
	rejected := 0
	for index := range results {
		if errorsSeen[index] == nil {
			accepted++
			if results[index].Version != 2 {
				t.Fatalf("worker %d version=%d want=2", index, results[index].Version)
			}
		} else {
			rejected++
		}
	}
	if accepted != 1 || rejected != 1 {
		t.Fatalf("accepted=%d rejected=%d errors=%v", accepted, rejected, errorsSeen)
	}
	public := readPublicView(t, game)
	if public.StateVersion != 2 || len(public.Bidding.Actions) != 1 || public.Bidding.Actions[0].Position != 1 {
		t.Fatalf("public=%#v", public)
	}
}

func TestLiveDirectoryKeepsSeparateHandsIndependent(t *testing.T) {
	first := newDirectBiddingGame(t, 1)
	first.id = "direct-bidding-hand-1"
	second := newDirectBiddingGame(t, 2)
	second.id = "direct-bidding-hand-2"
	directory, err := gamecore.NewLiveDirectory(discardFinalArchive{})
	if err != nil {
		t.Fatal(err)
	}
	if err := directory.Add(first.Descriptor(), first); err != nil {
		t.Fatal(err)
	}
	if err := directory.Add(second.Descriptor(), second); err != nil {
		t.Fatal(err)
	}

	var wait sync.WaitGroup
	wait.Add(2)
	errorsSeen := make(chan error, 2)
	go func() {
		defer wait.Done()
		_, err := directory.Apply(first.InstanceID(), bidCommand(t, 1, 1, bidding.ScorePass))
		errorsSeen <- err
	}()
	go func() {
		defer wait.Done()
		_, err := directory.Apply(second.InstanceID(), bidCommand(t, 2, 1, bidding.ScorePass))
		errorsSeen <- err
	}()
	wait.Wait()
	close(errorsSeen)
	for err := range errorsSeen {
		if err != nil {
			t.Fatal(err)
		}
	}
	if first.version != 2 || second.version != 2 {
		t.Fatalf("versions=%d/%d want=2/2", first.version, second.version)
	}
}

type discardFinalArchive struct{}

func (discardFinalArchive) Archive(gamecore.FinalRecord) error { return nil }
