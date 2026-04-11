package main

import (
	"log"
	"net/http"
	"time"
)


func main() {
	urls := []string{
		"http://localhost:9090",
		"http://localhost:9091",
		"http://localhost:9092",
	}

	lb, err := NewLoadBalancer(urls)
	if err != nil {
		log.Fatal(err)
	}

	lb.StartHealthCheck(10 * time.Second)

	server := &http.Server{
		Addr:    ":8080",
		Handler: lb,
	}

	log.Println("load balancer listening on :8080")
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}