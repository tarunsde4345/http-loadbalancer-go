package balancer

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/tarunsde4345/http-loadbalancer-go/backend"
)

type backendMetricsJSON struct {
	URL   				string  `json:"url"`
	Requests 			int64   `json:"requests"`
	Errors 				int64   `json:"errors"`
	AverageLatencyMs 	float64 `json:"average_latency_ms"`
	P99LatencyMs 		float64 `json:"p99_latency_ms"`
	CircuitBreakerState 	string  `json:"circuit_breaker_state"`
}

type loadBalancerMetricsJSON struct {
	UptimeSeconds string                    `json:"uptime_seconds"`
	TotalRequests int64                     `json:"total_requests"`
	TotalErrors   int64                     `json:"total_errors"`
	AliveBackends int                       `json:"alive_backends"`
	DeadBackends  int                       `json:"dead_backends"`
	Backends      []backendMetricsJSON      `json:"backends"`
}

type metricsHandler struct {
	lb       	*LoadBalancer
	startTime 	time.Time
}

func NewMetricsHandler(lb *LoadBalancer) *metricsHandler {
	return &metricsHandler{
		lb:        lb,
		startTime: time.Now(),
	}
}

func (mh *metricsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	mh.lb.mu.RLock()
	alive := mh.lb.alive
	mh.lb.mu.RUnlock()

	aliveSet := make(map[*backend.Backend]struct{}, len(alive))
	for _, b := range alive {
		aliveSet[b] = struct{}{}
	}

	totalRequests := int64(0)
	totalErrors := int64(0)
	backendStats := make([]backendMetricsJSON, 0, len(mh.lb.backends))

	for _, b := range mh.lb.backends {
		snap := b.Metrics.Snapshot()
		totalRequests += snap.TotalRequests
		totalErrors += snap.TotalErrors

		// circuit breaker state as string
		cbState := "closed"
		switch b.CB.CBState() {
		case backend.Open:
			cbState = "open"
		case backend.HalfOpen:
			cbState = "half-open"
		}

		backendStats = append(backendStats, backendMetricsJSON{
			URL:            b.URL.String(),
			Requests:       snap.TotalRequests,
			Errors:         snap.TotalErrors,
			AverageLatencyMs:   snap.AvgLatency,
			P99LatencyMs:   snap.P99Latency,
			CircuitBreakerState: cbState,
		})
	}

	resp := loadBalancerMetricsJSON{
		UptimeSeconds: time.Since(mh.startTime).Round(time.Second).String(),
		TotalRequests: totalRequests,
		TotalErrors:   totalErrors,
		AliveBackends: len(alive),
		DeadBackends:  len(mh.lb.backends) - len(alive),
		Backends:      backendStats,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}