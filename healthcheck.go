package main 

import (
	"log"
	"net/http"
	"time"
	"sync"
)

func (lb *LoadBalancer) healthCheck() {
	var wg sync.WaitGroup
	results := make([] bool, len(lb.backends))

	for i, b := range lb.backends {
		wg.Add(1)
		go func(i int, backend *Backend) {
			defer wg.Done()
			url := backend.URL.String() + "/health"
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
	
	aliveBackends := make([]*Backend, 0)
	for i, alive := range results {
		if alive {
			aliveBackends = append(aliveBackends, lb.backends[i])
		}
	}
	lb.setAliveBackends(aliveBackends)
}

func (lb *LoadBalancer) StartHealthCheck(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			lb.healthCheck()
		}
	}()
}
