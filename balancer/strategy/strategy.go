package strategy

import "github.com/tarunsde4345/http-loadbalancer-go/backend"

type Strategy interface {
	SelectBackend(backends []*backend.Backend) *backend.Backend
	Name() string
	OnRequest(b *backend.Backend)
	OnResponse(b *backend.Backend)
}

type Factory func(backends []*backend.Backend, beConfigs []*backend.BackendConfig) Strategy