package main

import (
	"log"
	"net/http"
	"time"

	"github.com/tarunsde4345/http-loadbalancer-go/backend"
	"github.com/tarunsde4345/http-loadbalancer-go/balancer"
	"github.com/tarunsde4345/http-loadbalancer-go/balancer/strategy"
)


func main() {
	// urls := []string{
	// 	"http://localhost:9001",
	// 	"http://localhost:9002",
	// 	"http://localhost:9003",
	// }

	backendConfig := []*backend.BackendConfig{
		{URL: "http://localhost:9001", Weight: 5},
		{URL: "http://localhost:9002", Weight: 2},
		{URL: "http://localhost:9003", Weight: 1},
	}

	// roundRobin := strategy.NewRoundRobin()
	leastConnection := strategy.NewLeastConnection()
	// weightedRoundRobin := strategy.NewWeightedRoundRobin()

	lb, err := balancer.New(backendConfig, leastConnection,
		balancer.WithHealthCheckInterval(10 * time.Second),
		balancer.WithHealthCheckEndpoint("/health"),
	)
	if err != nil {
		log.Fatal(err)
	}

	server := &http.Server{
		Addr:    ":8080",
		Handler: lb,
	}

	log.Println("load balancer listening on :8080")
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}