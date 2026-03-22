package email

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// ---------------------------------------------------------------------------
// Mock Sender
// ---------------------------------------------------------------------------

type mockSender struct {
	mu    sync.Mutex
	calls []mockCall
	// Optional delay to simulate slow sends.
	delay time.Duration
	// Optional error to return from each send.
	err error
}

type mockCall struct {
	method string
	to     string
	code   string
	locale string
}

func (m *mockSender) SendVerificationEmail(_ context.Context, to, code, locale string) error {
	if m.delay > 0 {
		time.Sleep(m.delay)
	}
	m.mu.Lock()
	m.calls = append(m.calls, mockCall{method: "verification", to: to, code: code, locale: locale})
	m.mu.Unlock()
	return m.err
}

func (m *mockSender) SendPasswordResetEmail(_ context.Context, to, code, locale string) error {
	if m.delay > 0 {
		time.Sleep(m.delay)
	}
	m.mu.Lock()
	m.calls = append(m.calls, mockCall{method: "password_reset", to: to, code: code, locale: locale})
	m.mu.Unlock()
	return m.err
}

func (m *mockSender) getCalls() []mockCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]mockCall, len(m.calls))
	copy(out, m.calls)
	return out
}

// ---------------------------------------------------------------------------
// NewAsyncSender tests
// ---------------------------------------------------------------------------

func TestNewAsyncSender(t *testing.T) {
	tests := []struct {
		name            string
		maxConcurrent   int
		wantSemCapacity int
	}{
		{
			name:            "uses default concurrency when zero",
			maxConcurrent:   0,
			wantSemCapacity: defaultMaxConcurrent,
		},
		{
			name:            "uses default concurrency when negative",
			maxConcurrent:   -1,
			wantSemCapacity: defaultMaxConcurrent,
		},
		{
			name:            "uses custom concurrency",
			maxConcurrent:   3,
			wantSemCapacity: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inner := &mockSender{}
			sender := NewAsyncSender(inner, zap.NewNop(), tt.maxConcurrent)

			require.NotNil(t, sender)
			assert.Equal(t, tt.wantSemCapacity, cap(sender.sem))
			assert.Equal(t, inner, sender.inner)
		})
	}
}

func TestNewAsyncSender_NilLogger(t *testing.T) {
	sender := NewAsyncSender(&mockSender{}, nil, 1)
	require.NotNil(t, sender)
	assert.NotNil(t, sender.logger, "nil logger should be replaced with nop logger")
}

// ---------------------------------------------------------------------------
// SendVerificationEmail tests
// ---------------------------------------------------------------------------

func TestAsyncSender_SendVerificationEmail(t *testing.T) {
	inner := &mockSender{}
	sender := NewAsyncSender(inner, zap.NewNop(), 5)

	err := sender.SendVerificationEmail(context.Background(), "alice@example.com", "code123", "en")
	require.NoError(t, err, "SendVerificationEmail always returns nil")

	// Wait for the async goroutine to complete.
	require.Eventually(t, func() bool {
		return len(inner.getCalls()) == 1
	}, 2*time.Second, 10*time.Millisecond)

	calls := inner.getCalls()
	assert.Equal(t, "verification", calls[0].method)
	assert.Equal(t, "alice@example.com", calls[0].to)
	assert.Equal(t, "code123", calls[0].code)
	assert.Equal(t, "en", calls[0].locale)
}

// ---------------------------------------------------------------------------
// SendPasswordResetEmail tests
// ---------------------------------------------------------------------------

func TestAsyncSender_SendPasswordResetEmail(t *testing.T) {
	inner := &mockSender{}
	sender := NewAsyncSender(inner, zap.NewNop(), 5)

	err := sender.SendPasswordResetEmail(context.Background(), "bob@example.com", "reset-tok", "ru")
	require.NoError(t, err, "SendPasswordResetEmail always returns nil")

	require.Eventually(t, func() bool {
		return len(inner.getCalls()) == 1
	}, 2*time.Second, 10*time.Millisecond)

	calls := inner.getCalls()
	assert.Equal(t, "password_reset", calls[0].method)
	assert.Equal(t, "bob@example.com", calls[0].to)
	assert.Equal(t, "reset-tok", calls[0].code)
	assert.Equal(t, "ru", calls[0].locale)
}

// ---------------------------------------------------------------------------
// maskEmail tests
// ---------------------------------------------------------------------------

func TestMaskEmail(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"standard email", "john@example.com", "j***@example.com"},
		{"single char local", "a@example.com", "a***@example.com"},
		{"no at sign", "noemail", "***"},
		{"empty string", "", "***"},
		{"at sign at start", "@domain.com", "***"},
		{"long local part", "longuser@domain.com", "l***@domain.com"},
		{"multiple at signs", "user@sub@domain.com", "u***@domain.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, maskEmail(tt.input))
		})
	}
}

// ---------------------------------------------------------------------------
// Non-blocking when semaphore is full
// ---------------------------------------------------------------------------

func TestAsyncSender_NonBlockingWhenSemaphoreFull(t *testing.T) {
	// Use a semaphore of size 1 and a slow inner sender so the slot stays occupied.
	inner := &mockSender{delay: 500 * time.Millisecond}
	sender := NewAsyncSender(inner, zap.NewNop(), 1)

	// First call occupies the single semaphore slot.
	err := sender.SendVerificationEmail(context.Background(), "a@b.com", "c1", "en")
	require.NoError(t, err)

	// Give the goroutine a moment to acquire the semaphore slot.
	time.Sleep(50 * time.Millisecond)

	// Second call should NOT block; it returns immediately (email is dropped).
	done := make(chan struct{})
	go func() {
		_ = sender.SendVerificationEmail(context.Background(), "x@y.com", "c2", "en")
		close(done)
	}()

	select {
	case <-done:
		// returned promptly -- expected
	case <-time.After(200 * time.Millisecond):
		t.Fatal("SendVerificationEmail blocked when semaphore was full")
	}

	// Wait for the first goroutine to finish.
	time.Sleep(600 * time.Millisecond)

	// Only the first email should have been dispatched to the inner sender.
	calls := inner.getCalls()
	assert.Equal(t, 1, len(calls), "only the first email should be delivered; second should be dropped")
	assert.Equal(t, "a@b.com", calls[0].to)
}
