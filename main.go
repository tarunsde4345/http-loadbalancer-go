package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"
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

	mux := http.NewServeMux()
	mux.Handle("/metrics", balancer.NewMetricsHandler(lb))
	mux.Handle("/dashboard/", http.StripPrefix("/dashboard/", http.FileServer(http.Dir("web/dashboard"))))
	mux.Handle("/", lb)

	server := &http.Server{
		Addr:    ":80",
		Handler: mux,
	}

	log.Println("load balancer listening on :80")
	log.Println("metrics available at http://localhost:80/metrics")
	log.Println("dashboard available at http://localhost:80/dashboard/")

	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	<-ctx.Done()

	log.Println("shutdown signal received")

	// give in-flight requests 30s to finish
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown error: %v", err)
	}

	log.Println("shutdown complete")
}


// metrics & dashboard hit should not be considered in lb metrics.