package readiness

import (
	"context"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

type Probe interface {
	Name() string
	Ping(context.Context) error
}

type Status struct {
	Ready       bool
	Unavailable []string
}

type Checker struct {
	timeout time.Duration
	probes  []Probe
}

func New(timeout time.Duration, probes ...Probe) *Checker {
	return &Checker{
		timeout: timeout,
		probes:  probes,
	}
}

func (c *Checker) Check(ctx context.Context) Status {
	if c == nil {
		return Status{Ready: false}
	}

	status := Status{Ready: true}
	for _, probe := range c.probes {
		checkCtx, cancel := context.WithTimeout(ctx, c.timeout)
		err := probe.Ping(checkCtx)
		cancel()
		if err != nil {
			status.Ready = false
			status.Unavailable = append(status.Unavailable, probe.Name())
			logx.WithContext(ctx).Errorf("readiness check failed: dependency=%s", probe.Name())
		}
	}

	return status
}
