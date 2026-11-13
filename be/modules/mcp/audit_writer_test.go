package mcp

import (
	"context"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestAuditWriterBoundedBurstDoesNotDelayTools(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var active, peak, persisted atomic.Int64
		release := make(chan struct{})
		recorder := auditRecorderFunc(func(ctx context.Context, _ AuditEntry) error {
			current := active.Add(1)
			defer active.Add(-1)
			for previous := peak.Load(); current > previous; previous = peak.Load() {
				if peak.CompareAndSwap(previous, current) {
					break
				}
			}
			select {
			case <-release:
				persisted.Add(1)
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		})
		core, logs := observer.New(zap.WarnLevel)
		writer := auditTestWriter(t, recorder, zap.New(core))
		writer.Start(t.Context()) // A second Start must not create more workers.
		for range AuditWorkerCount {
			require.True(t, writer.Enqueue(AuditEntry{Tool: "list_jobs"}))
		}
		synctest.Wait()
		require.Equal(t, int64(AuditWorkerCount), active.Load())
		server := sdk.NewServer(&sdk.Implementation{Name: "audit-burst", Version: "1"}, nil)
		handlers := &tools{audit: writer, logger: zap.NewNop()}
		register(server, handlers, "audit_probe", "test", true, true,
			func(context.Context, string, auditProbeInput) (auditProbeInput, error) {
				return auditProbeInput{Text: "ok"}, nil
			})
		server.AddReceivingMiddleware(handlers.auditMiddleware)
		router := auditHTTPRouter(server, nil, testHTTPConfig())
		started := time.Now()
		const overflow = 100
		for range AuditQueueCapacity + overflow {
			response := rpcHTTP(router, http.MethodPost, "Bearer "+validPAT, auditProbeCall)
			require.Equal(t, 200, response.Code)
			require.Contains(t, response.Body.String(), `"text":"ok"`)
		}
		require.Less(t, time.Since(started), auditWriteTimeout, "tools must finish while DB workers remain blocked")
		require.Equal(t, int64(AuditWorkerCount), peak.Load())
		require.Equal(t, uint64(overflow), writer.Overflow())
		require.Equal(t, uint64(overflow), writer.Dropped())
		require.Len(t, logs.All(), 1, "overflow warnings must be rate limited")
		require.Equal(t, "MCP audit queue full", logs.All()[0].Message)
		require.Equal(t, map[string]any{"dropped": uint64(1)}, logs.All()[0].ContextMap())
		close(release)
		require.NoError(t, writer.Close())
		require.Equal(t, int64(AuditQueueCapacity+AuditWorkerCount), persisted.Load())
		require.Zero(t, active.Load())
	})
}

func TestAuditWriterCloseDrainsPendingExactlyOnce(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var calls, persisted atomic.Int64
		writer := NewAuditWriter(auditRecorderFunc(func(ctx context.Context, _ AuditEntry) error {
			calls.Add(1)
			select {
			case <-time.After(20 * time.Millisecond):
				persisted.Add(1)
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		}), nil)
		writer.Start(t.Context())
		require.True(t, writer.Enqueue(AuditEntry{ID: "one", Tool: "list_jobs"}))
		synctest.Wait()
		closed := make(chan error, 1)
		go func() { closed <- writer.Close() }()
		synctest.Wait()
		require.Zero(t, persisted.Load(), "Close must wait for pending persistence")
		require.False(t, writer.Enqueue(AuditEntry{ID: "after-close"}))
		time.Sleep(20 * time.Millisecond)
		require.NoError(t, <-closed)
		require.Equal(t, int64(1), calls.Load())
		require.Equal(t, int64(1), persisted.Load())
		require.Equal(t, uint64(1), writer.Dropped(), "only the rejected late enqueue is dropped")
		require.NoError(t, writer.Close())
	})
}

func TestAuditWriterCloseTimeoutCountsAndCancelsPending(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		core, logs := observer.New(zap.WarnLevel)
		var active atomic.Int64
		writer := NewAuditWriter(auditRecorderFunc(func(ctx context.Context, _ AuditEntry) error {
			active.Add(1)
			defer active.Add(-1)
			<-ctx.Done()
			return ctx.Err()
		}), zap.New(core))
		writer.Start(t.Context())
		for range AuditWorkerCount {
			require.True(t, writer.Enqueue(AuditEntry{}))
		}
		synctest.Wait()
		for range AuditQueueCapacity {
			require.True(t, writer.Enqueue(AuditEntry{}))
		}
		started := time.Now()
		require.ErrorIs(t, writer.Close(), context.DeadlineExceeded)
		require.Equal(t, AuditShutdownTimeout, time.Since(started))
		synctest.Wait()
		require.Zero(t, active.Load())
		require.Empty(t, writer.queue)
		require.Equal(t, uint64(AuditQueueCapacity+AuditWorkerCount), writer.Dropped())
		require.Zero(t, writer.Overflow())
		require.ErrorIs(t, writer.Close(), context.DeadlineExceeded)
		deadlineLogs := logs.FilterMessage("MCP audit shutdown deadline reached").All()
		require.Len(t, deadlineLogs, 1)
		require.Positive(t, deadlineLogs[0].ContextMap()["dropped"])
		select {
		case <-writer.done:
		default:
			t.Error("workers must stop on shutdown cancellation")
		}
	})
}

func TestAuditWriterCloseBeforeStart(t *testing.T) {
	writer := NewAuditWriter(nil, nil)
	require.False(t, writer.Enqueue(AuditEntry{}))
	require.NoError(t, writer.Close())
	writer.Start(t.Context())
	require.False(t, writer.Enqueue(AuditEntry{}))
	require.NoError(t, writer.Close())
	require.Equal(t, uint64(2), writer.Dropped())
}

func TestAuditWriterConcurrentEnqueueAndClose(t *testing.T) {
	var accepted, persisted atomic.Uint64
	writer := NewAuditWriter(auditRecorderFunc(func(context.Context, AuditEntry) error {
		persisted.Add(1)
		return nil
	}), nil)
	writer.Start(t.Context())
	var producers sync.WaitGroup
	const producersCount, rowsPerProducer = 8, 50
	start := make(chan struct{})
	for range producersCount {
		producers.Go(func() {
			<-start
			for range rowsPerProducer {
				if writer.Enqueue(AuditEntry{Tool: "list_jobs"}) {
					accepted.Add(1)
				}
			}
		})
	}
	closed := make(chan error, 1)
	go func() { <-start; closed <- writer.Close() }()
	close(start)
	producers.Wait()
	require.NoError(t, <-closed)
	require.NoError(t, writer.Close())
	require.Equal(t, accepted.Load(), persisted.Load())
	require.Equal(t, uint64(producersCount*rowsPerProducer), persisted.Load()+writer.Dropped())
}
