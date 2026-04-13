package balancer 

import (
	"log"
	"net/http"
	"time"
	"sync"

	"github.com/tarunsde4345/http-loadbalancer-go/backend"
)

type healthChecker struct {
	backends []*backend.Backend
	interval time.Duration
	endpoint string
	onUpdate func(alive []*backend.Backend)
}

func newHealthChecker(
	backends []*backend.Backend, 
	interval time.Duration, 
	endpoint string, 
	onUpdate func(alive []*backend.Backend),
) *healthChecker {
	return &healthChecker{
		backends: backends,
		interval: interval,
		endpoint: endpoint,
		onUpdate: onUpdate,
	}
}

func (hc *healthChecker) start() {
	go func() {
		ticker := time.NewTicker(hc.interval)
		defer ticker.Stop()
		for range ticker.C {
			hc.healthCheck()
		}
	}()
}

func (hc *healthChecker) healthCheck() {
	var wg sync.WaitGroup
	results := make([]bool, len(hc.backends))


	for i, b := range hc.backends {
		wg.Add(1)
		go func(i int, backend *backend.Backend) {
			defer wg.Done()
			url := backend.URL.String() + hc.endpoint
			resp, err := http.Get(url)
			alive := err == nil && resp.StatusCode == http.StatusOK
			results[i] = alive
			if alive {
				log.Printf("backend %s is up", backend.URL)
			} else {
				log.Printf("backend %s is down", backend.URL)
			}
		}(i, b)
	}

	wg.Wait()

	aliveBackends := make([]*backend.Backend, 0, len(hc.backends))
	for i, alive := range results {
		if alive {
			aliveBackends = append(aliveBackends, hc.backends[i])
		}
	}
	hc.onUpdate(aliveBackends)
}


