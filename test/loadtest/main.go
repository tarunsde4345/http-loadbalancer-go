package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

const (
	target   = "http://localhost:8080"
	duration = 5 * time.Minute

	baseRPS  = 30
	burstRPS = 200
)

func main() {
	var (
		success atomic.Int64
		failure atomic.Int64
		wg      sync.WaitGroup

		mu        sync.Mutex
		latencies []time.Duration
	)

	end := time.Now().Add(duration)

	// controls current RPS (can change dynamically)
	currentRPS := atomic.Int64{}
	currentRPS.Store(baseRPS)

	// 🔹 Burst controller
	go func() {
		for time.Now().Before(end) {
			// wait random 2–5 seconds
			sleep := time.Duration(2+rand.Intn(4)) * time.Second
			time.Sleep(sleep)

			// trigger burst
			currentRPS.Store(burstRPS)
			fmt.Println("🔥 BURST START")

			// burst lasts 0.5–1 sec
			time.Sleep(time.Duration(500+rand.Intn(500)) * time.Millisecond)

			currentRPS.Store(baseRPS)
			fmt.Println("🟢 BURST END")
		}
	}()

	start := time.Now()

	// 🔹 Request generator loop
	for time.Now().Before(end) {
		rps := currentRPS.Load()

		// interval between requests
		interval := time.Second / time.Duration(rps)

		// jitter (±20%)
		jitter := time.Duration(rand.Int63n(int64(interval/5))) - interval/10

		time.Sleep(interval + jitter)

		wg.Add(1)
		go func() {
			defer wg.Done()

			reqStart := time.Now()
			resp, err := http.Get(target)
			latency := time.Since(reqStart)

			mu.Lock()
			latencies = append(latencies, latency)
			mu.Unlock()

			if err != nil {
				failure.Add(1)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				success.Add(1)
			} else {
				failure.Add(1)
			}
		}()
	}

	wg.Wait()
	totalTime := time.Since(start)

	// 🔹 stats
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
	fmt.Printf("total time     : %v\n", totalTime)
	fmt.Printf("avg latency    : %v\n", avg)
	fmt.Printf("min latency    : %v\n", min)
	fmt.Printf("max latency    : %v\n", max)
}