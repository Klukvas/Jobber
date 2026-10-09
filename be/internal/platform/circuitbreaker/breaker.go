// Package circuitbreaker provides a lightweight circuit breaker for external service calls.
//
// States:
//
//	CLOSED  → requests flow normally; consecutive failures are counted
//	OPEN    → requests fail immediately with ErrCircuitOpen
//	HALFOPEN → exactly one probe request is allowed through; success → CLOSED, failure → OPEN
package circuitbreaker

import (
	"errors"
	"sync"
	"time"
)

// ErrCircuitOpen is returned when the circuit breaker is open.
var ErrCircuitOpen = errors.New("circuit breaker is open")

// State represents the circuit breaker state.
type State int

const (
	StateClosed   State = iota
	StateOpen
	StateHalfOpen
)

// Breaker is a simple circuit breaker.
type Breaker struct {
	mu        sync.Mutex
	name      string // immutable after New
	state     State
	probing   bool          // true while a half-open probe is in flight
	failCount int
	threshold int           // consecutive failures to trip
	timeout   time.Duration // how long the circuit stays open
	openedAt  time.Time
}

// New creates a circuit breaker.
// threshold: number of consecutive failures before opening.
// timeout: duration the circuit stays open before allowing a probe.
func New(name string, threshold int, timeout time.Duration) *Breaker {
	return &Breaker{
		name:      name,
		state:     StateClosed,
		threshold: threshold,
		timeout:   timeout,
	}
}

// Execute runs fn if the circuit allows it.
// Returns ErrCircuitOpen immediately if the circuit is open and the timeout hasn't elapsed.
func (b *Breaker) Execute(fn func() error) error {
	if !b.allow() {
		return ErrCircuitOpen
	}

	err := fn()

	b.record(err)
	return err
}

// ExecuteIgnoring is Execute for calls that can end in an outcome that says
// nothing about the service's health, such as the caller giving up.
//
// When ignore reports true for the returned error, the error still goes back to
// the caller but is recorded as neither a success nor a failure: the state and
// the consecutive-failure count are left alone, and a half-open probe slot is
// released so the next request can probe instead. Recording it as a success
// would let one cancelled request re-close a breaker while the service is down.
func (b *Breaker) ExecuteIgnoring(fn func() error, ignore func(error) bool) error {
	if !b.allow() {
		return ErrCircuitOpen
	}

	err := fn()
	if err != nil && ignore != nil && ignore(err) {
		b.releaseProbe()
		return err
	}

	b.record(err)
	return err
}

// State returns the current state (for metrics/logging).
func (b *Breaker) State() State {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.currentState()
}

// Name returns the breaker name (immutable, no lock needed).
func (b *Breaker) Name() string {
	return b.name
}

func (b *Breaker) allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	switch b.currentState() {
	case StateClosed:
		return true
	case StateOpen:
		return false
	case StateHalfOpen:
		// Allow exactly one probe request
		if b.probing {
			return false
		}
		b.probing = true
		return true
	}
	return false
}

// releaseProbe frees the half-open probe slot without changing the state.
func (b *Breaker) releaseProbe() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.probing = false
}

func (b *Breaker) record(err error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.probing = false

	if err == nil {
		// Success → reset to closed
		b.failCount = 0
		b.state = StateClosed
		return
	}

	b.failCount++

	if b.currentState() == StateHalfOpen {
		// Probe failed → back to open
		b.state = StateOpen
		b.openedAt = time.Now()
		return
	}

	if b.failCount >= b.threshold {
		b.state = StateOpen
		b.openedAt = time.Now()
	}
}

// currentState returns the effective state, promoting open→halfOpen when timeout elapses.
// Must be called with mu held.
func (b *Breaker) currentState() State {
	if b.state == StateOpen && time.Since(b.openedAt) >= b.timeout {
		b.state = StateHalfOpen
	}
	return b.state
}
