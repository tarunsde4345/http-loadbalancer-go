package strategy

import (
	"math"
	"sync/atomic"

	"github.com/tarunsde4345/http-loadbalancer-go/backend"
)

type lcBackend struct {
	connections atomic.Uint64
}

type LeastConnection struct {
	entries map[*backend.Backend]*lcBackend
}

func NewLeastConnection() Factory {
	return func(backends []*backend.Backend, _ []*backend.BackendConfig) Strategy {
		entries := make(map[*backend.Backend]*lcBackend, len(backends))
		for _, b := range backends {
			entries[b] = &lcBackend{}
		}

		return &LeastConnection{
			entries: entries,
		}
	}
}

func (lc *LeastConnection) SelectBackend(backends []*backend.Backend) *backend.Backend {
	if len(backends) == 0 {
		return nil
	}

	var selected *backend.Backend
	minConnections := uint64(math.MaxUint64)
	for _, b := range backends {
		entry, ok := lc.entries[b]
		if !ok {
			continue
		}
		if entry.connections.Load() < minConnections {
			minConnections = entry.connections.Load()
			selected = b
		}
	}

	return selected
}

func (lc *LeastConnection) OnRequest(b *backend.Backend) {
	if entry, ok := lc.entries[b]; ok {
		entry.connections.Add(1)
	}
}

func (lc *LeastConnection) OnResponse(b *backend.Backend) {
	if entry, ok := lc.entries[b]; ok {
		entry.connections.Add(^uint64(0))
	}
}

func (lc *LeastConnection) Name() string {
	return "Least Connection"
}
