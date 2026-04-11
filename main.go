package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync/atomic"
)

type Backend struct {
	URL   *url.URL
	Proxy *httputil.ReverseProxy
}

type LoadBalancer struct {
	backends []*Backend
	counter  atomic.Uint64
}

func NewLoadBalancer(urls []string) (*LoadBalancer, error) {
	backends := make([]*Backend, 0, len(urls))

	for _, raw := range urls {
		u, err := url.Parse(raw)
		if err != nil {
			return nil, err
		}
		backends = append(backends, &Backend{
			URL:   u,
			Proxy: httputil.NewSingleHostReverseProxy(u),
		})
	}

	return &LoadBalancer{backends: backends}, nil
}

func (lb *LoadBalancer) nextBackend() *Backend {
	idx := lb.counter.Add(1) - 1
	return lb.backends[idx%uint64(len(lb.backends))]
}

func (lb *LoadBalancer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	backend := lb.nextBackend()
	log.Printf("forwarding to %s", backend.URL)
	backend.Proxy.ServeHTTP(w, r)
}

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

	server := &http.Server{
		Addr:    ":8080",
		Handler: lb,
	}

	log.Println("load balancer listening on :8080")
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}