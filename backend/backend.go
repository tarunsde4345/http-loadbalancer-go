package backend

import (
	"net/http/httputil"
	"net/url"
	"sync/atomic"
)

type Backend struct {
	URL   				*url.URL
	Proxy 				*httputil.ReverseProxy
    CB    				*CircuitBreaker
	ActiveConnections 	atomic.Uint64
}

type BackendConfig struct {
	URL    string
	Weight int64
}

func New(raw string) (*Backend, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	return &Backend{
		URL:   u,
		Proxy: httputil.NewSingleHostReverseProxy(u),
		CB:    NewCircuitBreaker(),
	}, nil
}
