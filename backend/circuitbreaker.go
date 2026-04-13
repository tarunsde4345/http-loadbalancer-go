package backend 

import (
	"sync"
	"time"
)

type CBState int

const (
	Closed CBState = iota
	Open
	HalfOpen
)

const (
	failureThreshold = 5
	successThreshold = 1
	resetTimeout     = 30 * time.Second
)

type CircuitBreaker struct {
	mu          sync.Mutex
	state       CBState
	failure     int
	lastFailure time.Time
}

func NewCircuitBreaker() *CircuitBreaker {
	return &CircuitBreaker{
		state: Closed,
	}
}

func (cb *CircuitBreaker) AllowRequest() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case Closed:
		return true
	case Open:
		if time.Since(cb.lastFailure) > resetTimeout {
			cb.state = HalfOpen
			return true
		}
		return false
	case HalfOpen:
		return false
	}
	return false
}

func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failure++
	cb.lastFailure = time.Now()

	if cb.failure >= failureThreshold {
		cb.state = Open
	}
}

func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == HalfOpen {
		cb.failure = 0
		cb.state = Closed
	}
}

func (cb *CircuitBreaker) CBState() CBState {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.state
}

