package delayqueue

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/beanstalkd/go-beanstalk"
	"github.com/zeromicro/go-queue/dq"
	zeroredis "github.com/zeromicro/go-zero/core/stores/redis"
)

// Config isolates infrastructure; two independent Beanstalkd replicas and
// Redis DB 0 are required by the existing go-queue/dq consumer protocol.
type Config struct {
	Endpoints []string
	Tube      string
	RedisHost string
	RedisUser string
	RedisPass string
}

func (c Config) Validate() error {
	if len(c.Endpoints) < 2 || len(c.Endpoints) > 3 || strings.TrimSpace(c.Tube) == "" ||
		strings.TrimSpace(c.RedisHost) == "" {
		return errors.New("dq requires two distinct beanstalkd nodes, tube and redis")
	}
	distinct := make(map[string]struct{}, len(c.Endpoints))
	for _, ep := range c.Endpoints {
		if strings.TrimSpace(ep) != ep || ep == "" {
			return errors.New("invalid beanstalkd endpoint")
		}
		if _, ok := distinct[ep]; ok {
			return errors.New("duplicate beanstalkd endpoint")
		}
		distinct[ep] = struct{}{}
	}
	return nil
}
func (c Config) beanstalks() []dq.Beanstalk {
	list := make([]dq.Beanstalk, 0, len(c.Endpoints))
	for _, ep := range c.Endpoints {
		list = append(list, dq.Beanstalk{Endpoint: ep, Tube: c.Tube})
	}
	return list
}

const publishTimeout = 2 * time.Second

// go-queue/dq Producer.At offers no context/deadline hook and can block for
// ever on a TCP connection that remains established but stops replying.
// Retain go-queue's exact wire envelope and 2-replica quorum, but use its
// own go-beanstalk driver behind a bounded, context-aware publisher adapter.
// A fresh connection per publish intentionally trades setup cost for deadline
// isolation; bounded per-turn writes have no unbounded stuck worker pool.
type Publisher struct {
	nodes []dq.Beanstalk
	dial  func(context.Context, string, string) (net.Conn, error)
}

func NewPublisher(c Config) (*Publisher, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	dialer := &net.Dialer{Timeout: publishTimeout, KeepAlive: 10 * time.Second}
	return &Publisher{nodes: c.beanstalks(), dial: dialer.DialContext}, nil
}

// At may return an error after one replica accepted a job. Durable MySQL
// reconciliation and identity fencing handle unknown/duplicate results.
// In particular, do not attempt unbounded network revocation on partial writes.
func (p *Publisher) At(ctx context.Context, body []byte, deadline time.Time) (string, error) {
	if p == nil || len(p.nodes) < 2 || p.dial == nil {
		return "", errors.New("dq producer unavailable")
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if len(body) == 0 || len(body) > 1024 || deadline.IsZero() {
		return "", errors.New("invalid delay queue message or deadline")
	}
	// Unlike go-queue Producer.At, overdue work is scheduled immediately.
	// Beanstalkd accepts only whole seconds: CEIL instead of truncating to
	// avoid consuming and discarding an early callback.
	at := deadline
	if now := time.Now(); at.Before(now) {
		at = now
	}
	wrapped := append([]byte(strconv.FormatInt(at.UnixNano(), 10)+"/"), body...)
	publishCtx, cancel := context.WithTimeout(ctx, publishTimeout)
	defer cancel()
	type result struct {
		id  string
		err error
	}
	results := make(chan result, len(p.nodes))
	for _, node := range p.nodes {
		go func(node dq.Beanstalk) {
			id, err := p.put(publishCtx, node, wrapped, at)
			results <- result{id: id, err: err}
		}(node)
	}
	var ids []string
	var failures []error
	for range p.nodes {
		r := <-results
		if r.err != nil {
			failures = append(failures, r.err)
		} else {
			ids = append(ids, r.id)
			if len(ids) >= 2 {
				// Do not wait for an optional stalled third replica after the
				// two durable writes required by go-queue have succeeded.
				// Cancel closes its socket; the buffered result cannot leak.
				return strings.Join(ids, ","), nil
			}
		}
	}
	if len(ids) < 2 {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		return "", fmt.Errorf("dq publication accepted by %d/%d replicas: %w", len(ids), len(p.nodes), errors.Join(failures...))
	}
	return strings.Join(ids, ","), nil
}

func (p *Publisher) put(ctx context.Context, node dq.Beanstalk, wrapped []byte, at time.Time) (string, error) {
	raw, err := p.dial(ctx, "tcp", node.Endpoint)
	if err != nil {
		return "", err
	}
	defer raw.Close()
	// Cancel also interrupts already established, silently stalled connections.
	stop := context.AfterFunc(ctx, func() { _ = raw.Close() })
	defer stop()
	if deadline, ok := ctx.Deadline(); ok {
		if err := raw.SetDeadline(deadline); err != nil {
			return "", err
		}
	}
	conn := beanstalk.NewConn(raw)
	conn.Tube.Name = node.Tube
	remaining := at.Sub(time.Now())
	var delay time.Duration
	if remaining > 0 {
		delay = ((remaining + time.Second - 1) / time.Second) * time.Second
	}
	id, err := conn.Put(wrapped, dq.PriNormal, delay, 5*time.Second)
	if err != nil {
		return "", err
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/%s/%d", node.Endpoint, node.Tube, id), nil
}

func (p *Publisher) Close() error { return nil }

type Consumer struct{ inner dq.Consumer }

func NewConsumer(c Config) (*Consumer, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	red := zeroredis.RedisConf{Host: c.RedisHost, Type: zeroredis.NodeType, User: c.RedisUser, Pass: c.RedisPass}
	return &Consumer{inner: dq.NewConsumer(dq.DqConf{Beanstalks: c.beanstalks(), Redis: red})}, nil
}

// Consume blocks. The library deletes beanstalkd jobs BEFORE calling handler
// and Redis deduplicates bodies for one hour. The caller MUST reconcile failed
// and missing jobs from the durable MySQL schedule; a callback alone is unsafe.
func (c *Consumer) Consume(handler func([]byte) error, onError func(error)) {
	if c == nil || c.inner == nil {
		if onError != nil {
			onError(errors.New("dq consumer unavailable"))
		}
		return
	}
	c.inner.Consume(func(body []byte) {
		err := handler(body)
		if err != nil && onError != nil {
			onError(err)
		}
	})
}
