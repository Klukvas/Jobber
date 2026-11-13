package mcp

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

const (
	AuditQueueCapacity   = 1024
	AuditWorkerCount     = 2
	AuditShutdownTimeout = 5 * time.Second
	auditWarningInterval = time.Minute
	auditWriteTimeout    = 250 * time.Millisecond
)

// AuditWriter limits audit traffic to a fixed number of database operations.
// Start it before serving HTTP, and Close it after HTTP shutdown, before the pool.
type AuditWriter struct {
	recorder AuditRecorder
	logger   *zap.Logger
	queue    chan AuditEntry
	done     chan struct{}
	workers  sync.WaitGroup
	dropped  atomic.Uint64
	overflow atomic.Uint64

	mu                  sync.Mutex
	started             bool
	closing             bool
	aborted             bool
	pending             uint64
	cancel              context.CancelFunc
	closeErr            error
	lastOverflowWarning time.Time
}

func NewAuditWriter(recorder AuditRecorder, logger *zap.Logger) *AuditWriter {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &AuditWriter{recorder: recorder, logger: logger, queue: make(chan AuditEntry, AuditQueueCapacity), done: make(chan struct{})}
}

// Start is idempotent. Request cancellation cannot stop pending audit work;
// Close owns cancellation and each DB operation has its own short deadline.
func (w *AuditWriter) Start(ctx context.Context) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.started || w.closing {
		return
	}
	w.started = true
	workerCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	w.cancel = cancel
	for range AuditWorkerCount {
		w.workers.Go(func() { w.run(workerCtx) })
	}
	go func() {
		w.workers.Wait()
		close(w.done)
	}()
}

// Enqueue never waits for a database connection or queue capacity. The mutex
// protects only lifecycle/accounting, including concurrent Enqueue and Close.
func (w *AuditWriter) Enqueue(entry AuditEntry) bool {
	w.mu.Lock()
	if !w.started || w.closing {
		w.dropped.Add(1)
		w.mu.Unlock()
		return false
	}
	select {
	case w.queue <- entry:
		w.pending++
		w.mu.Unlock()
		return true
	default:
		w.dropped.Add(1)
		count := w.overflow.Add(1)
		now := time.Now()
		warn := w.lastOverflowWarning.IsZero() || now.Sub(w.lastOverflowWarning) >= auditWarningInterval
		if warn {
			w.lastOverflowWarning = now
		}
		w.mu.Unlock()
		if warn {
			w.logger.Warn("MCP audit queue full", zap.Uint64("dropped", count))
		}
		return false
	}
}

func (w *AuditWriter) Dropped() uint64  { return w.dropped.Load() }
func (w *AuditWriter) Overflow() uint64 { return w.overflow.Load() }

func (w *AuditWriter) run(ctx context.Context) {
	for entry := range w.queue {
		if ctx.Err() != nil {
			return
		}
		writeCtx, cancel := context.WithTimeout(ctx, auditWriteTimeout)
		err := w.recorder.Record(writeCtx, entry)
		cancel()
		w.mu.Lock()
		if w.aborted {
			w.mu.Unlock()
			return
		}
		w.pending--
		if err != nil {
			w.dropped.Add(1)
		}
		w.mu.Unlock()
		if err != nil {
			// DB errors may contain SQL parameters, so log only fixed metadata.
			w.logger.Warn("MCP audit write failed", zap.String("tool", entry.Tool))
		}
	}
}

// Close stops accepting entries and drains accepted work for at most five
// seconds. On timeout, outstanding writes are cancelled and remaining rows are
// counted as dropped; no queued request data is retained. Safe to call again.
func (w *AuditWriter) Close() error {
	w.mu.Lock()
	if !w.closing {
		w.closing = true
		close(w.queue)
		if !w.started {
			close(w.done)
		}
	}
	if w.aborted {
		err := w.closeErr
		w.mu.Unlock()
		return err
	}
	w.mu.Unlock()
	timer := time.NewTimer(AuditShutdownTimeout)
	defer timer.Stop()
	select {
	case <-w.done:
		w.mu.Lock()
		if w.cancel != nil {
			w.cancel()
		}
		err := w.closeErr
		w.mu.Unlock()
		return err
	case <-timer.C:
		return w.abort()
	}
}

func (w *AuditWriter) abort() error {
	w.mu.Lock()
	if w.aborted {
		w.mu.Unlock()
		return context.DeadlineExceeded
	}
	w.aborted = true
	w.closeErr = context.DeadlineExceeded
	dropped := w.pending
	w.pending = 0
	w.dropped.Add(dropped)
	if w.cancel != nil {
		w.cancel()
	}
	for range w.queue {
	}
	w.mu.Unlock()
	w.logger.Warn("MCP audit shutdown deadline reached", zap.Uint64("dropped", dropped))
	return context.DeadlineExceeded
}
