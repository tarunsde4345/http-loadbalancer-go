package strategy

import (
	"sync/atomic"
	"github.com/tarunsde4345/http-loadbalancer-go/backend"
)

type roundRobin struct {
	counter atomic.Uint64
}

func NewRoundRobin() Factory {
	return func(backends []*backend.Backend, _ []*backend.BackendConfig) Strategy {
		return &roundRobin{}
	}
}

func (rr *roundRobin) SelectBackend(backends []*backend.Backend) *backend.Backend {
	if len(backends) == 0 {
		return nil
	}
	idx := rr.counter.Add(1) - 1
	return backends[idx%uint64(len(backends))]
}

func (rr *roundRobin) Name() string {
	return "Round Robin"
}

func (rr *roundRobin) OnRequest(b *backend.Backend) {}

func (rr *roundRobin) OnResponse(b *backend.Backend) {}
