package proxy

import (
	"sync"
	"time"
)

type CircuitState int

const (
	StateClosed CircuitState = iota
	StateOpen
	StateHalfOpen
)

type CircuitBreaker struct {
	mu                sync.RWMutex
	state             CircuitState
	failureThreshold  int           // eg 5 consecutive errors
	consecutiveErrors int           
	cooldownDuration  time.Duration // eg 30 seconds
	lastStateChange   time.Time
}

func NewCircuitBreaker(threshold int, cooldown time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		state:            StateClosed,
		failureThreshold: threshold,
		cooldownDuration: cooldown,
		lastStateChange:  time.Now(),
	}
}

// Allow checks if a request is permitted to go to the upstream service
func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	// check if the cooldown period has expired
	if cb.state == StateOpen {
		if time.Since(cb.lastStateChange) > cb.cooldownDuration {
			cb.state = StateHalfOpen
			cb.lastStateChange = time.Now()
			return true // Let this single request act as a probe!
		}
		return false // Still in cooldown, fast-fail!
	}

	return true // Closed or HalfOpen allows traffic
}

// updates the failure/success tracking metrics based on network health
func (cb *CircuitBreaker) RecordResult(err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		// Network failed
		cb.consecutiveErrors++
		
		if cb.state == StateHalfOpen || cb.consecutiveErrors >= cb.failureThreshold {
			cb.state = StateOpen
			cb.lastStateChange = time.Now()
		}
	} else {
		// Network succeeded
		if cb.state == StateHalfOpen {
			cb.state = StateClosed
		} 
		cb.consecutiveErrors = 0
	}
}