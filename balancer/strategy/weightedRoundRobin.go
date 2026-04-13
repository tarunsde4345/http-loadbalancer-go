package strategy

import (
	"sync"

	"github.com/tarunsde4345/http-loadbalancer-go/backend"
)

type wrrBackend struct {
	weight        int64
	currentWeight int64
}

type weightedRoundRobin struct {
	entries     map[*backend.Backend]*wrrBackend
	mu 		    sync.Mutex
	totalWeight int64
}

func NewWeightedRoundRobin() Factory {
	return func(backends []*backend.Backend, beConfig []*backend.BackendConfig) Strategy {
		entries := make(map[*backend.Backend]*wrrBackend, len(backends))
		totalWeight := int64(0)
		for i, b := range backends {
			weight := beConfig[i].Weight
			entries[b] = &wrrBackend{weight: weight}
			totalWeight += weight
		}

		return &weightedRoundRobin{
			entries:    entries,
			totalWeight: totalWeight,
		}
	}
}

func (rr *weightedRoundRobin) SelectBackend(backends []*backend.Backend) *backend.Backend {
	rr.mu.Lock()
	defer rr.mu.Unlock()
	
	if len(backends) == 0 {
		return nil
	}

	maxWeight := int64(0)
	var selected *backend.Backend
	for _, b := range backends {
		entry, ok := rr.entries[b]
		if !ok {
			continue
		}
		entry.currentWeight += entry.weight
		if entry.currentWeight > maxWeight {
			maxWeight = entry.currentWeight
			selected = b
		}
	}

	if selected == nil {
		return nil
	}

	rr.entries[selected].currentWeight -= rr.totalWeight

	return selected
}

func (rr *weightedRoundRobin) Name() string {
	return "Weighted Round Robin"
}

func (rr *weightedRoundRobin) OnRequest(b *backend.Backend) {}

func (rr *weightedRoundRobin) OnResponse(b *backend.Backend) {}
