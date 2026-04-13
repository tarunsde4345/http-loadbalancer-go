package balancer

import "time"

type config struct {
	healthCheckInterval time.Duration
	healthCheckEndpoint  string
}

func defaultConfig() config {
	return config{
		healthCheckInterval: 10 * time.Second,
		healthCheckEndpoint: "/health",
	}
}

type Option func(*config)

func WithHealthCheckInterval(d time.Duration) Option {
	return func(c *config) {
		c.healthCheckInterval = d
	}
}

func WithHealthCheckEndpoint(endpoint string) Option {
	return func(c *config) {
		c.healthCheckEndpoint = endpoint
	}
}