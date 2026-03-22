package circuitbreaker

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errTest = errors.New("test error")

func TestNew(t *testing.T) {
	t.Run("creates breaker in closed state", func(t *testing.T) {
		cb := New("test-breaker", 3, 5*time.Second)

		assert.Equal(t, StateClosed, cb.State())
		assert.Equal(t, "test-breaker", cb.Name())
	})
}

func TestBreaker_Name(t *testing.T) {
	tests := []struct {
		name     string
		cbName   string
		expected string
	}{
		{name: "simple name", cbName: "redis", expected: "redis"},
		{name: "hyphenated name", cbName: "external-api", expected: "external-api"},
		{name: "empty name", cbName: "", expected: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cb := New(tt.cbName, 3, 5*time.Second)

			assert.Equal(t, tt.expected, cb.Name())
		})
	}
}

func TestBreaker_State(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*Breaker)
		expected State
	}{
		{
			name:     "initial state is closed",
			setup:    func(_ *Breaker) {},
			expected: StateClosed,
		},
		{
			name: "open after threshold failures",
			setup: func(cb *Breaker) {
				for range 3 {
					_ = cb.Execute(func() error { return errTest })
				}
			},
			expected: StateOpen,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cb := New("test", 3, 5*time.Second)
			tt.setup(cb)

			assert.Equal(t, tt.expected, cb.State())
		})
	}
}

func TestBreaker_Execute_ClosedState(t *testing.T) {
	t.Run("runs fn and returns nil on success", func(t *testing.T) {
		cb := New("test", 3, 5*time.Second)
		called := false

		err := cb.Execute(func() error {
			called = true
			return nil
		})

		require.NoError(t, err)
		assert.True(t, called)
		assert.Equal(t, StateClosed, cb.State())
	})

	t.Run("runs fn and returns error on failure", func(t *testing.T) {
		cb := New("test", 3, 5*time.Second)

		err := cb.Execute(func() error { return errTest })

		assert.ErrorIs(t, err, errTest)
		assert.Equal(t, StateClosed, cb.State())
	})
}

func TestBreaker_Execute_TripsToOpen(t *testing.T) {
	tests := []struct {
		name       string
		threshold  int
		failCount  int
		wantState  State
	}{
		{
			name:      "stays closed below threshold",
			threshold: 3,
			failCount: 2,
			wantState: StateClosed,
		},
		{
			name:      "trips to open at threshold",
			threshold: 3,
			failCount: 3,
			wantState: StateOpen,
		},
		{
			name:      "trips to open above threshold",
			threshold: 3,
			failCount: 5,
			wantState: StateOpen,
		},
		{
			name:      "threshold of 1 trips immediately",
			threshold: 1,
			failCount: 1,
			wantState: StateOpen,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cb := New("test", tt.threshold, 5*time.Second)

			for range tt.failCount {
				// After tripping open, Execute returns ErrCircuitOpen without calling fn
				_ = cb.Execute(func() error { return errTest })
			}

			assert.Equal(t, tt.wantState, cb.State())
		})
	}
}

func TestBreaker_Execute_OpenState(t *testing.T) {
	t.Run("returns ErrCircuitOpen immediately without calling fn", func(t *testing.T) {
		cb := New("test", 1, 5*time.Second)

		// Trip the breaker
		_ = cb.Execute(func() error { return errTest })
		require.Equal(t, StateOpen, cb.State())

		called := false
		err := cb.Execute(func() error {
			called = true
			return nil
		})

		assert.ErrorIs(t, err, ErrCircuitOpen)
		assert.False(t, called)
	})
}

func TestBreaker_Execute_HalfOpen(t *testing.T) {
	t.Run("moves to half-open after timeout elapses", func(t *testing.T) {
		cb := New("test", 1, 10*time.Millisecond)

		_ = cb.Execute(func() error { return errTest })
		require.Equal(t, StateOpen, cb.State())

		time.Sleep(15 * time.Millisecond)

		assert.Equal(t, StateHalfOpen, cb.State())
	})

	t.Run("successful probe resets to closed", func(t *testing.T) {
		cb := New("test", 1, 10*time.Millisecond)

		_ = cb.Execute(func() error { return errTest })
		require.Equal(t, StateOpen, cb.State())

		time.Sleep(15 * time.Millisecond)

		err := cb.Execute(func() error { return nil })

		require.NoError(t, err)
		assert.Equal(t, StateClosed, cb.State())
	})

	t.Run("failed probe goes back to open", func(t *testing.T) {
		cb := New("test", 1, 10*time.Millisecond)

		_ = cb.Execute(func() error { return errTest })
		require.Equal(t, StateOpen, cb.State())

		time.Sleep(15 * time.Millisecond)
		require.Equal(t, StateHalfOpen, cb.State())

		err := cb.Execute(func() error { return errTest })

		assert.ErrorIs(t, err, errTest)
		assert.Equal(t, StateOpen, cb.State())
	})
}

func TestBreaker_Execute_HalfOpen_ConcurrentProbeGuard(t *testing.T) {
	t.Run("only one probe allowed in half-open state", func(t *testing.T) {
		cb := New("test", 1, 10*time.Millisecond)

		_ = cb.Execute(func() error { return errTest })
		require.Equal(t, StateOpen, cb.State())

		time.Sleep(15 * time.Millisecond)
		require.Equal(t, StateHalfOpen, cb.State())

		// Use a channel to hold the first probe inside Execute
		probeStarted := make(chan struct{})
		probeFinish := make(chan struct{})

		var wg sync.WaitGroup
		var probeErr, secondErr error

		// First goroutine: the probe request (will be allowed)
		wg.Add(1)
		go func() {
			defer wg.Done()
			probeErr = cb.Execute(func() error {
				close(probeStarted) // Signal that the probe is running
				<-probeFinish       // Wait until we release it
				return nil
			})
		}()

		// Wait for the probe to be in flight
		<-probeStarted

		// Second goroutine: should be rejected while probe is in flight
		wg.Add(1)
		go func() {
			defer wg.Done()
			secondErr = cb.Execute(func() error { return nil })
		}()

		// Give the second goroutine time to hit the guard
		time.Sleep(5 * time.Millisecond)

		// Release the probe
		close(probeFinish)
		wg.Wait()

		assert.NoError(t, probeErr)
		assert.ErrorIs(t, secondErr, ErrCircuitOpen)
		assert.Equal(t, StateClosed, cb.State())
	})
}

func TestBreaker_SuccessResetsFailCount(t *testing.T) {
	t.Run("success resets consecutive failure count", func(t *testing.T) {
		cb := New("test", 3, 5*time.Second)

		// Two failures (below threshold)
		_ = cb.Execute(func() error { return errTest })
		_ = cb.Execute(func() error { return errTest })
		require.Equal(t, StateClosed, cb.State())

		// One success resets
		_ = cb.Execute(func() error { return nil })

		// Two more failures should not trip (counter was reset)
		_ = cb.Execute(func() error { return errTest })
		_ = cb.Execute(func() error { return errTest })

		assert.Equal(t, StateClosed, cb.State())
	})
}
