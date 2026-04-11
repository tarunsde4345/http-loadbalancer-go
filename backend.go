package main

import (
	"net/http/httputil"
	"net/url"
	"sync"
)

type Backend struct {
	URL   *url.URL
	Proxy *httputil.ReverseProxy
	alive bool
	mu    sync.RWMutex
}

func newBackend(raw string) (*Backend, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	return &Backend{
		URL:   u,
		Proxy: httputil.NewSingleHostReverseProxy(u),
		alive: true,
	}, nil
}

func (b *Backend) IsAlive() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.alive
}

func (b *Backend) SetAlive(alive bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.alive = alive
}
