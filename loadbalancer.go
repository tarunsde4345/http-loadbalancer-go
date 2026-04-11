package main 

import (
	"net/http"
	"sync/atomic"
	"log"
	"sync"
)

type LoadBalancer struct {
	backends []*Backend
	alive    []*Backend
	mu       sync.RWMutex
	counter  atomic.Uint64
}

func NewLoadBalancer(urls []string) (*LoadBalancer, error) {
	backends := make([]*Backend, 0, len(urls))

	for _, raw := range urls {
		backend, err := newBackend(raw)
		if err != nil {
			return nil, err
		}
		backends = append(backends, backend)
	}

	lb := &LoadBalancer{
		backends: backends,
		alive:    backends, // initially all are alive
	}
	return lb, nil
}

func (lb *LoadBalancer) setAliveBackends(alive []*Backend) {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	lb.alive = alive
}

func (lb *LoadBalancer) nextBackend() *Backend {
	lb.mu.RLock()
	defer lb.mu.RUnlock()
	if len(lb.alive) == 0 {
		return nil
	}
	idx := lb.counter.Add(1) - 1
	return lb.alive[idx%uint64(len(lb.alive))]
}

func (lb *LoadBalancer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("received request: %s %s", r.Method, r.URL.Path)
	backend := lb.nextBackend()
	if backend == nil {
		http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
		return
	}
	log.Printf("forwarding to %s", backend.URL)
	backend.Proxy.ServeHTTP(w, r)
}

