package main

import (
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

const (
	target      = "http://localhost:8080"
	total       = 50
	concurrency = 10 // requests in flight at once
)

func main() {
	var (
		success atomic.Int64
		failure atomic.Int64
		wg      sync.WaitGroup
		mu      sync.Mutex
		latencies []time.Duration
	)

	// semaphore to control concurrency
	sem := make(chan struct{}, concurrency)

	start := time.Now()

	for i := range total {
		wg.Add(1)
		sem <- struct{}{} // acquire slot

		go func(reqNum int) {
			defer wg.Done()
			defer func() { <-sem }() // release slot

			reqStart := time.Now()
			resp, err := http.Get(target)
			latency := time.Since(reqStart)

			mu.Lock()
			latencies = append(latencies, latency)
			mu.Unlock()

			if err != nil {
				failure.Add(1)
				fmt.Printf("[req %02d] error: %v\n", reqNum, err)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				success.Add(1)
				fmt.Printf("[req %02d] status=%d latency=%v\n", reqNum, resp.StatusCode, latency)
			} else {
				failure.Add(1)
				fmt.Printf("[req %02d] status=%d latency=%v\n", reqNum, resp.StatusCode, latency)
			}
		}(i)
	}

	wg.Wait()
	total := time.Since(start)

	// compute stats
	var sum time.Duration
	min := latencies[0]
	max := latencies[0]
	for _, l := range latencies {
		sum += l
		if l < min {
			min = l
		}
		if l > max {
			max = l
		}
	}
	avg := sum / time.Duration(len(latencies))

	fmt.Println("\n--- results ---")
	fmt.Printf("total requests : %d\n", success.Load()+failure.Load())
	fmt.Printf("success        : %d\n", success.Load())
	fmt.Printf("failure        : %d\n", failure.Load())
	fmt.Printf("total time     : %v\n", total)
	fmt.Printf("avg latency    : %v\n", avg)
	fmt.Printf("min latency    : %v\n", min)
	fmt.Printf("max latency    : %v\n", max)
}