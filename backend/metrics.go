package backend

import (
	"sort"
	"sync"
	"sync/atomic"
)

type Metrics struct {
	mu            sync.Mutex
	TotalRequests atomic.Int64
	TotalErrors   atomic.Int64

	window []float64
	wIndex atomic.Int64
	wFull  bool
}

func NewMetrics(windowSize int) *Metrics {
	return &Metrics{
		window: make([]float64, windowSize),
	}
}

func (m *Metrics) RecordRequest(latency float64, success bool) {
	m.TotalRequests.Add(1)
	if !success {
		m.TotalErrors.Add(1)
	}

	idx := m.wIndex.Add(1) - 1
	m.window[idx%int64(len(m.window))] = latency
	if idx >= int64(len(m.window)) {
		m.wFull = true
	}
}

type MetricsSnapshot struct {
	TotalRequests int64
	TotalErrors   int64
	AvgLatency    float64
	P99Latency    float64
}

func (m *Metrics) Snapshot() MetricsSnapshot {
	m.mu.Lock()

	var samples []float64

	if m.wFull {
		samples = make([]float64, len(m.window))
		copy(samples, m.window)
	} else {
		samples = make([]float64, m.wIndex.Load())
		copy(samples, m.window[:m.wIndex.Load()])
	}

	m.mu.Unlock()

	snap := MetricsSnapshot{
		TotalRequests: m.TotalRequests.Load(),
		TotalErrors:   m.TotalErrors.Load(),
	}

	if len(samples) == 0 {
		return snap
	}

	sum := 0.0
	
	for _, v := range samples {
		sum += v
	}
	snap.AvgLatency = sum / float64(len(samples))

	sort.Float64s(samples)
	p99Index := int(float64(len(samples)) * 0.99)
	if p99Index > 0 {
		p99Index -= 1
	}
	snap.P99Latency = samples[p99Index]

	return snap
}