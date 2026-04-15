# HTTP Load Balancer in Go

A modular HTTP reverse proxy / load balancer written in Go with:

- multiple balancing strategies
- active health checks
- circuit breaker protection
- per-backend metrics
- a live dashboard
- mock backends and traffic generators for local experimentation

The project is designed as a small systems-design style playground: you can simulate fast/slow backends, inject server-side failures, switch strategies in code, and observe how routing and resiliency behave.

## What It Does

The load balancer:

- accepts incoming HTTP requests
- forwards them to one of several backend servers
- keeps unhealthy backends out of rotation using health checks
- tracks per-backend requests, errors, average latency, and p99 latency
- applies a circuit breaker to avoid repeatedly sending traffic to failing backends
- exposes a JSON metrics endpoint and a browser dashboard

## Implemented Features

### Load-balancing strategies

- `Round Robin`
- `Least Connection`
- `Weighted Round Robin`

Strategies are created through a factory-based interface so they can initialize internal state using the concrete backend list and backend config.

### Health checking

- background health checks run on a configurable interval
- each backend is probed on a configurable endpoint such as `/health`
- the balancer maintains an `alive` pool and only routes to healthy backends

### Circuit breaker

Each backend has its own circuit breaker with these states:

- `Closed`
- `Open`
- `HalfOpen`

Current behavior:

- repeated `5xx` responses increment backend failure count
- after the threshold is reached, the backend circuit opens
- after a timeout, one trial request is allowed in `HalfOpen`
- a successful trial closes the circuit again

Note:
- the current breaker uses a raw failure threshold, not a time-window-based threshold yet
- a time-aware failure window is the next planned improvement

### Metrics

The load balancer exposes `GET /metrics` and returns:

- uptime
- total requests
- total errors
- alive/dead backend counts
- per-backend:
  - URL
  - request count
  - error count
  - average latency
  - p99 latency
  - circuit breaker state

Latency metrics use a rolling in-memory sample window.

### Dashboard

A built-in HTML/CSS/JS dashboard is served at:

- `GET /dashboard/`

It polls `/metrics` every 2 seconds and shows:

- top-level summary cards
- requests vs errors over time
- aggregate latency chart
- backend status table

Dashboard behavior:

- cards show cumulative totals
- the traffic chart shows per-poll deltas rather than cumulative totals
- the chart keeps a rolling 2-minute window
- x-axis labels are intentionally sparse to reduce clutter

### Mock backend servers

Local mock backends support:

- configurable port
- configurable response delay
- configurable error probability

This makes it easy to simulate:

- slow backends
- intermittent backend failures
- uneven backend quality

Example mock backend config:

- `PORT=9001`
- `DELAY=50ms`
- `ERROR_PROBABILITY=0.15`

### Load testing / traffic simulation

The traffic generator supports:

- configurable duration
- configurable base RPS
- configurable spike RPS
- configurable spike interval
- configurable spike duration
- configurable RPS jitter

This lets you model traffic that is not perfectly flat, for example:

- base traffic: `30 RPS`
- bursts to `60 RPS`
- 5-second spikes every 30 seconds
- per-second jitter such as `+/- 2`

## Project Structure

```text
.
├── main.go
├── backend/
│   ├── backend.go
│   ├── circuitbreaker.go
│   └── metrics.go
├── balancer/
│   ├── loadbalancer.go
│   ├── healthcheck.go
│   ├── metrics.go
│   ├── options.go
│   └── strategy/
│       ├── strategy.go
│       ├── roundRobin.go
│       ├── leastConnection.go
│       └── weightedRoundRobin.go
├── web/dashboard/
│   ├── index.html
│   ├── styles.css
│   └── app.js
└── test/
    ├── mockserver/main.go
    ├── loadtest/main.go
    └── start.sh
```

## Architecture Overview

### `main.go`

Wires everything together:

- backend config
- strategy selection
- load balancer creation
- metrics route
- dashboard route
- root proxy route

### `backend` package

Represents runtime backend instances.

Each backend contains:

- parsed URL
- reverse proxy
- circuit breaker
- metrics collector

### `balancer` package

Owns request routing and backend orchestration.

Key responsibilities:

- create backend runtime objects
- keep track of healthy backends
- select the next backend using the chosen strategy
- forward requests using Go's `httputil.ReverseProxy`
- update metrics and circuit breaker state after responses

### `strategy` package

Defines the strategy contract and implementations.

The strategy interface includes:

- `SelectBackend(...)`
- `Name()`
- `OnRequest(...)`
- `OnResponse(...)`

The `OnRequest/OnResponse` hooks are used by strategies such as least-connections that need to maintain live request counts.

## Current Default Configuration

In the current `main.go`, the project is wired with:

- 3 backends:
  - `http://localhost:9001`
  - `http://localhost:9002`
  - `http://localhost:9003`
- backend weights:
  - `9001 -> 5`
  - `9002 -> 2`
  - `9003 -> 1`
- current selected strategy:
  - `Least Connection`
- health check interval:
  - `10s`
- health check endpoint:
  - `/health`
- server listen address:
  - `:80`

Important:
- binding to port `80` may require elevated privileges depending on your OS/environment

## How To Run

### 1. Start the load balancer

```bash
go run main.go
```

Then open:

- Metrics: [http://localhost/metrics](http://localhost/metrics)
- Dashboard: [http://localhost/dashboard/](http://localhost/dashboard/)

### 2. Start mock backends and traffic simulation

```bash
bash test/start.sh
```

This script:

- starts 3 mock backend servers
- assigns each backend a delay and error probability
- waits briefly for startup
- runs the load test generator against the load balancer

## Mock Server Configuration

The mock server reads environment variables:

- `PORT`
- `DELAY`
- `ERROR_PROBABILITY`

Example:

```bash
PORT=9001 DELAY=50ms ERROR_PROBABILITY=0.10 go run test/mockserver/main.go
```

Behavior:

- waits for `DELAY`
- returns `502` with probability `ERROR_PROBABILITY`
- otherwise returns `200`

## Load Test Configuration

The load test supports these flags:

```bash
go run test/loadtest/main.go \
  -target http://localhost \
  -duration 2m \
  -base-rps 30 \
  -spike-rps 60 \
  -jitter 2 \
  -spike-every 30s \
  -spike-duration 5s
```

Meaning:

- steady-state traffic at `30 RPS`
- bursts to `60 RPS`
- each second gets random variation in `[-2, +2]`
- burst windows last `5s`
- bursts repeat every `30s`

## Example Metrics Payload

```json
{
  "uptime_seconds": "53s",
  "total_requests": 2,
  "total_errors": 2,
  "alive_backends": 3,
  "dead_backends": 0,
  "backends": [
    {
      "url": "http://localhost:9001",
      "requests": 2,
      "errors": 2,
      "average_latency_ms": 0,
      "p99_latency_ms": 0,
      "circuit_breaker_state": "closed"
    }
  ]
}
```

## Design Choices

### Why strategy factories?

Strategies are created using:

```go
type Factory func(backends []*backend.Backend, beConfigs []*backend.BackendConfig) Strategy
```

This allows strategies to initialize themselves using:

- the concrete backend instances
- backend configuration such as weights

This is especially useful for:

- weighted round robin
- least connection
- future strategies that need startup-time state

### Why keep metrics inside backend objects?

Per-backend metrics are attached to each runtime backend so the balancer can:

- update request/error counts immediately after proxying
- compute snapshots independently per backend
- expose aggregated metrics without external storage

## Known Limitations / Next Steps

- Circuit breaker currently uses a raw failure threshold, not a rolling time window.
- Health checker does not perform an immediate first probe on startup; it begins on the ticker interval.
- `main.go` currently hardcodes backend config and selected strategy.
- Dynamic runtime control from the dashboard is a planned next phase:
  - add/remove backends
  - update strategy at runtime
  - richer admin controls

## Why This Project Is Interview-Relevant

This project demonstrates:

- Go concurrency with goroutines, mutexes, atomics, and wait groups
- use of interfaces and factories for pluggable behavior
- reverse proxying with the Go standard library
- resiliency patterns:
  - health checks
  - circuit breakers
  - fallback behavior
- observability:
  - metrics endpoint
  - live dashboard
  - local simulation tooling
- systems thinking around traffic shaping, failure injection, and strategy comparison

## Talking Points For Interview

If asked what you would improve next, strong answers are:

- add a time-window-based circuit breaker threshold
- allow runtime strategy switching through an admin API
- allow dynamic backend registration/removal
- move from custom JSON metrics to Prometheus metrics for production observability
- make graceful shutdown flow more robust and explicit

