package delayqueue

import (
	"bufio"
	"context"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestRequiresTwoDistinctServers(t *testing.T) {
	c := Config{Endpoints: []string{"beanstalk-a:11300", "beanstalk-b:11300"}, Tube: "doudizhu-turn", RedisHost: "redis:6379"}
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	c.Endpoints = []string{"beanstalk-a:11300"}
	if c.Validate() == nil {
		t.Fatal("single node incorrectly accepted")
	}
	c.Endpoints = []string{"beanstalk-a:11300", "beanstalk-a:11300"}
	if c.Validate() == nil {
		t.Fatal("duplicate node incorrectly accepted")
	}
	c.Endpoints = []string{"beanstalk-a:11300", "beanstalk-b:11300"}
	c.RedisHost = ""
	if c.Validate() == nil {
		t.Fatal("missing dedup Redis accepted")
	}
}

func TestDeadlineAwarePublisherInterruptsSilentEstablishedConnections(t *testing.T) {
	c := Config{Endpoints: []string{"node-a:11300", "node-b:11300"},
		Tube: "turns", RedisHost: "redis:6379"}
	p, err := NewPublisher(c)
	if err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	var peers []net.Conn
	p.dial = func(ctx context.Context, _, _ string) (net.Conn, error) {
		client, server := net.Pipe()
		mu.Lock()
		peers = append(peers, server)
		mu.Unlock()
		return client, nil
	}
	defer func() {
		mu.Lock()
		defer mu.Unlock()
		for _, peer := range peers {
			_ = peer.Close()
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	start := time.Now()
	if _, err := p.At(ctx, []byte("real-ticket"), time.Now().Add(time.Second)); err == nil {
		t.Fatal("silent connections accepted publication")
	}
	if took := time.Since(start); took > 700*time.Millisecond {
		t.Fatalf("cancel blocked on silent TCP connections for %v", took)
	}
}

func TestPublisherRoundsBeanstalkDelayUpAndPreservesDQEnvelope(t *testing.T) {
	c := Config{Endpoints: []string{"node-a:11300", "node-b:11300"},
		Tube: "turns", RedisHost: "redis:6379"}
	p, err := NewPublisher(c)
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	observed := make(chan string, 2)
	p.dial = func(ctx context.Context, _, _ string) (net.Conn, error) {
		client, server := net.Pipe()
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer server.Close()
			reader := bufio.NewReader(server)
			line, err := reader.ReadString('\n')
			if err != nil || !strings.HasPrefix(line, "use turns") {
				observed <- "missing tube selection"
				return
			}
			line, err = reader.ReadString('\n')
			parts := strings.Fields(line)
			if err != nil || len(parts) != 5 || parts[0] != "put" {
				observed <- "invalid beanstalk command"
				return
			}
			size, err := strconv.Atoi(parts[4])
			if err != nil || size <= 0 {
				observed <- "invalid put length"
				return
			}
			body := make([]byte, size+2)
			if _, err := io.ReadFull(reader, body); err != nil {
				observed <- "missing put payload"
				return
			}
			if parts[2] == "0" || !strings.HasSuffix(string(body), "/real-ticket\r\n") {
				observed <- "early delivery or invalid dq envelope"
				return
			}
			_, _ = io.WriteString(server, "USING turns\r\nINSERTED 1\r\n")
			observed <- "ok"
		}()
		return client, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if _, err := p.At(ctx, []byte("real-ticket"), time.Now().Add(300*time.Millisecond)); err != nil {
		t.Fatal(err)
	}
	wg.Wait()
	for i := 0; i < 2; i++ {
		if value := <-observed; value != "ok" {
			t.Fatal(value)
		}
	}
}

func TestPublisherThirdSilentReplicaDoesNotDelayTwoNodeQuorum(t *testing.T) {
	c := Config{Endpoints: []string{"node-a:11300", "node-b:11300", "node-c:11300"},
		Tube: "turns", RedisHost: "redis:6379"}
	p, err := NewPublisher(c)
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	p.dial = func(ctx context.Context, _, endpoint string) (net.Conn, error) {
		client, server := net.Pipe()
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer server.Close()
			if endpoint == "node-c:11300" {
				// Consume nothing. An unbounded publisher waits for this socket.
				_, _ = io.Copy(io.Discard, server)
				return
			}
			reader := bufio.NewReader(server)
			_, err := reader.ReadString('\n')
			if err != nil {
				return
			}
			line, err := reader.ReadString('\n')
			if err != nil {
				return
			}
			parts := strings.Fields(line)
			if len(parts) != 5 {
				return
			}
			n, err := strconv.Atoi(parts[4])
			if err != nil {
				return
			}
			_, err = io.CopyN(io.Discard, reader, int64(n+2))
			if err == nil {
				_, _ = io.WriteString(server, "USING turns\r\nINSERTED 42\r\n")
			}
		}()
		return client, nil
	}
	start := time.Now()
	id, err := p.At(context.Background(), []byte("ticket"), time.Now().Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(id, ",") != 1 {
		t.Fatalf("two replica quorum ID expected, got %s", id)
	}
	if elapsed := time.Since(start); elapsed > 800*time.Millisecond {
		t.Fatalf("optional stalled replica delayed quorum for %v", elapsed)
	}
	wg.Wait()
}
