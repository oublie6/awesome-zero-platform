package livehand

import (
	"sync"
	"testing"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/carddeck"
	"github.com/oublie6/awesome-zero-platform/server/business/gamecore"
)

func TestLiveDirectorySerializesConcurrentPlaysForOneHand(t *testing.T) {
	game := newDirectPlayingGame(t)
	directory, err := gamecore.NewLiveDirectory(discardFinalArchive{})
	if err != nil {
		t.Fatal(err)
	}
	if err := directory.Add(game.Descriptor(), game); err != nil {
		t.Fatal(err)
	}

	cards := []carddeck.Card{game.current[0][0], game.current[0][1]}
	results := make([]gamecore.CommandOutcome, len(cards))
	errorsSeen := make([]error, len(cards))
	start := make(chan struct{})
	var wait sync.WaitGroup
	wait.Add(len(cards))
	for index, card := range cards {
		index := index
		command := livePlayCommand(t, 1, 2, card)
		go func() {
			defer wait.Done()
			<-start
			results[index], errorsSeen[index] = directory.Apply(game.InstanceID(), command)
		}()
	}
	close(start)
	wait.Wait()

	accepted := 0
	rejected := 0
	for index := range results {
		if errorsSeen[index] == nil {
			accepted++
			if results[index].Version != 3 {
				t.Fatalf("worker %d version=%d want=3", index, results[index].Version)
			}
		} else {
			rejected++
		}
	}
	if accepted != 1 || rejected != 1 {
		t.Fatalf("accepted=%d rejected=%d errors=%v", accepted, rejected, errorsSeen)
	}
	if len(game.current[0]) != 19 || game.version != 3 {
		t.Fatalf("remaining=%d version=%d", len(game.current[0]), game.version)
	}
	public := readPublicView(t, game)
	if public.Playing == nil || len(public.Playing.History) != 1 || public.Playing.CurrentSeat != 2 {
		t.Fatalf("public=%#v", public)
	}
}

func TestLiveDirectoryKeepsPlayingHandsIndependent(t *testing.T) {
	first := newDirectPlayingGame(t)
	first.id = "playing-hand-1"
	second := newDirectPlayingGame(t)
	second.id = "playing-hand-2"
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

	firstCommand := livePlayCommand(t, 1, 2, first.current[0][0])
	secondCommand := livePlayCommand(t, 1, 2, second.current[0][0])
	errorsSeen := make(chan error, 2)
	var wait sync.WaitGroup
	wait.Add(2)
	go func() {
		defer wait.Done()
		_, err := directory.Apply(first.InstanceID(), firstCommand)
		errorsSeen <- err
	}()
	go func() {
		defer wait.Done()
		_, err := directory.Apply(second.InstanceID(), secondCommand)
		errorsSeen <- err
	}()
	wait.Wait()
	close(errorsSeen)
	for err := range errorsSeen {
		if err != nil {
			t.Fatal(err)
		}
	}
	if first.version != 3 || second.version != 3 || len(first.current[0]) != 19 || len(second.current[0]) != 19 {
		t.Fatalf("first=%d/%d second=%d/%d", first.version, len(first.current[0]), second.version, len(second.current[0]))
	}
}
