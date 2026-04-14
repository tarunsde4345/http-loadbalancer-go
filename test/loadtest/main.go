package main

import (
	"flag"
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

type sample struct {
	latency time.Duration
	success bool
}

func main() {
	target := flag.String("target", "http://localhost", "load balancer target URL")
	duration := flag.Duration("duration", 2*time.Minute, "total duration of the test")
	baseRPS := flag.Int("base-rps", 30, "steady-state requests per second")
	spikeRPS := flag.Int("spike-rps", 60, "requests per second during spikes")
	jitter := flag.Int("jitter", 2, "random per-second RPS variance added in the range [-jitter, +jitter]")
	spikeEvery := flag.Duration("spike-every", 30*time.Second, "time between spike starts")
	spikeDuration := flag.Duration("spike-duration", 5*time.Second, "duration of each traffic spike")
	timeout := flag.Duration("timeout", 5*time.Second, "per-request timeout")
	flag.Parse()

	if *baseRPS <= 0 || *spikeRPS <= 0 {
		panic("base-rps and spike-rps must be greater than 0")
	}
	if *spikeDuration <= 0 || *spikeEvery <= 0 {
		panic("spike-duration and spike-every must be greater than 0")
	}
	if *jitter < 0 {
		panic("jitter must be greater than or equal to 0")
	}

	client := &http.Client{Timeout: *timeout}
	start := time.Now()
	end := start.Add(*duration)
	random := rand.New(rand.NewSource(time.Now().UnixNano()))

	fmt.Printf("target         : %s\n", *target)
	fmt.Printf("duration       : %v\n", *duration)
	fmt.Printf("base rps       : %d\n", *baseRPS)
	fmt.Printf("spike rps      : %d\n", *spikeRPS)
	fmt.Printf("jitter         : +/- %d\n", *jitter)
	fmt.Printf("spike every    : %v\n", *spikeEvery)
	fmt.Printf("spike duration : %v\n\n", *spikeDuration)

	var (
		successes atomic.Int64
		failures  atomic.Int64
		samplesMu sync.Mutex
		samples   []sample
		wg        sync.WaitGroup
	)

	secondTick := time.NewTicker(time.Second)
	defer secondTick.Stop()

	secondIndex := 0
	for now := range secondTick.C {
		if !now.Before(end) {
			break
		}

		rps := *baseRPS
		if inSpikeWindow(secondIndex, *spikeEvery, *spikeDuration) {
			rps = *spikeRPS
		}
		if *jitter > 0 {
			rps += random.Intn(*jitter*2+1) - *jitter
		}
		if rps < 1 {
			rps = 1
		}

		fmt.Printf("[%s] generating %d requests\n", now.Format("15:04:05"), rps)

		for i := 0; i < rps; i++ {
			scheduledAt := now.Add(time.Duration(i) * time.Second / time.Duration(rps))
			sleepUntil(scheduledAt)

			wg.Add(1)
			go func() {
				defer wg.Done()

				reqStart := time.Now()
				resp, err := client.Get(*target)
				latency := time.Since(reqStart)

				ok := err == nil && resp != nil && resp.StatusCode == http.StatusOK
				if err != nil {
					failures.Add(1)
				} else {
					if resp.Body != nil {
						resp.Body.Close()
					}
					if ok {
						successes.Add(1)
					} else {
						failures.Add(1)
					}
				}

				samplesMu.Lock()
				samples = append(samples, sample{
					latency: latency,
					success: ok,
				})
				samplesMu.Unlock()
			}()
		}

		secondIndex++
	}

	wg.Wait()
	printSummary(time.Since(start), successes.Load(), failures.Load(), samples)
}

func inSpikeWindow(secondIndex int, spikeEvery time.Duration, spikeDuration time.Duration) bool {
	spikeEverySeconds := int(spikeEvery / time.Second)
	spikeDurationSeconds := int(spikeDuration / time.Second)

	if spikeEverySeconds <= 0 {
		return false
	}

	return secondIndex%spikeEverySeconds < spikeDurationSeconds
}

func sleepUntil(t time.Time) {
	if d := time.Until(t); d > 0 {
		time.Sleep(d)
	}
}

func printSummary(totalTime time.Duration, successCount int64, failureCount int64, samples []sample) {
	fmt.Println("\n--- results ---")
	fmt.Printf("total requests : %d\n", successCount+failureCount)
	fmt.Printf("success        : %d\n", successCount)
	fmt.Printf("failure        : %d\n", failureCount)
	fmt.Printf("total time     : %v\n", totalTime)

	if len(samples) == 0 {
		fmt.Println("avg latency    : n/a")
		fmt.Println("min latency    : n/a")
		fmt.Println("max latency    : n/a")
		return
	}

	var (
		sum time.Duration
		min = samples[0].latency
		max = samples[0].latency
	)

	for _, s := range samples {
		sum += s.latency
		if s.latency < min {
			min = s.latency
		}
		if s.latency > max {
			max = s.latency
		}
	}

	avg := sum / time.Duration(len(samples))
	fmt.Printf("avg latency    : %v\n", avg)
	fmt.Printf("min latency    : %v\n", min)
	fmt.Printf("max latency    : %v\n", max)
}
