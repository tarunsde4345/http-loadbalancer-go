package balancer

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/tarunsde4345/http-loadbalancer-go/backend"
	"github.com/tarunsde4345/http-loadbalancer-go/balancer/strategy"
)

type LoadBalancer struct {
	backends  []*backend.Backend
	alive     []*backend.Backend
	mu        sync.RWMutex
	strategy  strategy.Strategy
}

func New(beConfigs []*backend.BackendConfig, factory strategy.Factory, opts ...Option) (*LoadBalancer, error) {
	lbConfig := defaultConfig()
	for _, opt := range opts {
		opt(&lbConfig)
	}

	backends := make([]*backend.Backend, 0, len(beConfigs))
	for _, cfg := range beConfigs {
		backend, err := backend.New(cfg.URL)
		if err != nil {
			return nil, err
		}
		backends = append(backends, backend)
	}

	lb := &LoadBalancer{
		backends:  backends,
		alive:    backends,
		strategy: factory(backends, beConfigs),
	}

	hc := newHealthChecker(backends, lbConfig.healthCheckInterval, lbConfig.healthCheckEndpoint, lb.setAliveBackends)
	hc.start()
	return lb, nil
}

func (lb *LoadBalancer) setAliveBackends(alive []*backend.Backend) {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	lb.alive = alive
}

func (lb *LoadBalancer) nextBackend() *backend.Backend {
	lb.mu.RLock()
	alive := lb.alive
	defer lb.mu.RUnlock()

	for range alive {
		b := lb.strategy.SelectBackend(alive)
		if b.CB.AllowRequest() {
			return b
		}
	}
	return nil
}

type responseRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (rr *responseRecorder) WriteHeader(code int) {
	rr.statusCode = code
	rr.ResponseWriter.WriteHeader(code)
}

func (lb *LoadBalancer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("received request: %s %s", r.Method, r.URL.Path)
	backend := lb.nextBackend()
	if backend == nil {
		http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
		return
	}
	log.Printf("forwarding to %s", backend.URL)
	lb.strategy.OnRequest(backend)
	defer lb.strategy.OnResponse(backend)

	startTime := time.Now()
	rec := &responseRecorder{ResponseWriter: w, statusCode: http.StatusOK}
	backend.Proxy.ServeHTTP(rec, r)

	latency := float64(time.Since(startTime).Milliseconds()) 
	isError := rec.statusCode >= 500
	backend.Metrics.RecordRequest(latency, isError)

	if isError {
		log.Printf("backend %s failed with status %d", backend.URL, rec.statusCode)
		backend.CB.RecordFailure()
	} else {
		backend.CB.RecordSuccess()
	}
}

