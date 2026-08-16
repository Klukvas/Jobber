package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/andreypavlenko/jobber/internal/platform/telegram"
	userRepo "github.com/andreypavlenko/jobber/modules/users/repository"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// userColumns mirrors the SELECT column list of UserRepository.GetByID, with
// the Postgres type OIDs pgx uses to scan each destination field.
func userColumns() []pgCol {
	return []pgCol{
		{"id", 25}, {"email", 25}, {"name", 25}, {"password_hash", 25},
		{"locale", 25}, {"email_verified", 16}, {"created_at", 1184}, {"updated_at", 1184},
	}
}

// userRow builds a single canned user row for the given name/email.
func userRow(id, name, email string) [][]byte {
	ts := "2023-01-02 03:04:05+00"
	return [][]byte{
		[]byte(id), []byte(email), []byte(name), []byte("hash"),
		[]byte("en"), []byte("t"), []byte(ts), []byte(ts),
	}
}

// newPool builds a real *pgxpool.Pool wired to the given DSN. The pool connects
// lazily, so construction never blocks on an unreachable server.
func newPool(t *testing.T, dsn string) *pgxpool.Pool {
	t.Helper()
	cfg, err := pgxpool.ParseConfig(dsn)
	require.NoError(t, err)
	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	require.NoError(t, err)
	return pool
}

// captureTransport is an http.RoundTripper that records the outgoing Telegram
// request and returns a scripted response.
type captureTransport struct {
	statusCode int
	respBody   string
	err        error

	gotURL  string
	gotBody string
	calls   int
}

func (c *captureTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	c.calls++
	c.gotURL = r.URL.String()
	if r.Body != nil {
		b, _ := io.ReadAll(r.Body)
		c.gotBody = string(b)
	}
	if c.err != nil {
		return nil, c.err
	}
	status := c.statusCode
	if status == 0 {
		status = http.StatusOK
	}
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(bytes.NewBufferString(c.respBody)),
		Header:     make(http.Header),
	}, nil
}

// withTelegramTransport swaps http.DefaultTransport (which telegram.Client uses
// via its zero-value Transport) for the duration of the test. These tests must
// not run in parallel because DefaultTransport is process-global.
func withTelegramTransport(t *testing.T, tr http.RoundTripper) {
	t.Helper()
	orig := http.DefaultTransport
	http.DefaultTransport = tr
	t.Cleanup(func() { http.DefaultTransport = orig })
}

func TestSupportService_Submit_Success(t *testing.T) {
	t.Run("includes the page line when page is provided", func(t *testing.T) {
		fpg := newFakePG(t, userColumns(), userRow("user-1", "Jane Doe", "jane@example.com"), "")
		defer fpg.Close()
		pool := newPool(t, fpg.dsn())
		defer pool.Close()

		tr := &captureTransport{statusCode: http.StatusOK, respBody: `{"ok":true}`}
		withTelegramTransport(t, tr)

		svc := NewSupportService(telegram.NewClient("tok", "chat"), userRepo.NewUserRepository(pool))
		err := svc.Submit(context.Background(), "user-1", "Login broken", "I cannot log in at all", "/dashboard")

		require.NoError(t, err)
		require.Equal(t, 1, tr.calls)
		assert.Contains(t, tr.gotURL, "/bottok/sendMessage")
		// The Telegram payload is JSON with an escaped "text" field.
		assert.Contains(t, tr.gotBody, "Jane Doe")
		assert.Contains(t, tr.gotBody, "jane@example.com")
		assert.Contains(t, tr.gotBody, "Login broken")
		assert.Contains(t, tr.gotBody, "I cannot log in at all")
		assert.Contains(t, tr.gotBody, "Page:")
		assert.Contains(t, tr.gotBody, "/dashboard")
	})

	t.Run("omits the page line when page is empty", func(t *testing.T) {
		fpg := newFakePG(t, userColumns(), userRow("user-2", "Bob", "bob@example.com"), "")
		defer fpg.Close()
		pool := newPool(t, fpg.dsn())
		defer pool.Close()

		tr := &captureTransport{statusCode: http.StatusOK, respBody: `{"ok":true}`}
		withTelegramTransport(t, tr)

		svc := NewSupportService(telegram.NewClient("tok", "chat"), userRepo.NewUserRepository(pool))
		err := svc.Submit(context.Background(), "user-2", "Question", "This is my question", "")

		require.NoError(t, err)
		require.Equal(t, 1, tr.calls)
		assert.NotContains(t, tr.gotBody, "Page:")
		assert.Contains(t, tr.gotBody, "Question")
	})

	t.Run("HTML-escapes user-controlled fields", func(t *testing.T) {
		// A user whose name contains HTML must not be able to inject markup into
		// the Telegram message — the service html.EscapeString-es every field.
		fpg := newFakePG(t, userColumns(), userRow("user-3", "<b>Mallory</b>", "m@x.com"), "")
		defer fpg.Close()
		pool := newPool(t, fpg.dsn())
		defer pool.Close()

		tr := &captureTransport{statusCode: http.StatusOK, respBody: `{"ok":true}`}
		withTelegramTransport(t, tr)

		svc := NewSupportService(telegram.NewClient("tok", "chat"), userRepo.NewUserRepository(pool))
		err := svc.Submit(context.Background(), "user-3", "<script>", "hello there world", "<img>")

		require.NoError(t, err)
		// The service html.EscapeString-es every user field, so the raw markup
		// never reaches Telegram. In the JSON payload, '<'/'>'/'&' are further
		// unicode-escaped by encoding/json, so the escaped entities appear as
		// &lt; etc. Decode the JSON "text" field back to compare cleanly.
		var payload struct {
			Text string `json:"text"`
		}
		require.NoError(t, json.Unmarshal([]byte(tr.gotBody), &payload))
		assert.NotContains(t, payload.Text, "<b>Mallory</b>")
		assert.Contains(t, payload.Text, "&lt;b&gt;Mallory&lt;/b&gt;")
		assert.Contains(t, payload.Text, "&lt;script&gt;")
		assert.Contains(t, payload.Text, "&lt;img&gt;")
	})
}

func TestSupportService_Submit_UserLookupError(t *testing.T) {
	// An unreachable pool makes GetByID fail fast (connection refused), which is
	// the "find user" error branch. Telegram must never be called.
	pool := newPool(t, "postgres://u:p@127.0.0.1:1/db?sslmode=disable")
	defer pool.Close()

	tr := &captureTransport{statusCode: http.StatusOK, respBody: `{"ok":true}`}
	withTelegramTransport(t, tr)

	svc := NewSupportService(telegram.NewClient("tok", "chat"), userRepo.NewUserRepository(pool))
	err := svc.Submit(context.Background(), "user-x", "subject here", "message body long enough", "")

	require.Error(t, err)
	assert.True(t, strings.HasPrefix(err.Error(), "support: find user:"), "got %q", err.Error())
	assert.Equal(t, 0, tr.calls, "telegram must not be called when user lookup fails")
}

func TestSupportService_Submit_TelegramError(t *testing.T) {
	t.Run("transport error is wrapped", func(t *testing.T) {
		fpg := newFakePG(t, userColumns(), userRow("user-1", "Jane", "jane@example.com"), "")
		defer fpg.Close()
		pool := newPool(t, fpg.dsn())
		defer pool.Close()

		tr := &captureTransport{err: errors.New("network down")}
		withTelegramTransport(t, tr)

		svc := NewSupportService(telegram.NewClient("tok", "chat"), userRepo.NewUserRepository(pool))
		err := svc.Submit(context.Background(), "user-1", "subject here", "message body long enough", "")

		require.Error(t, err)
		assert.True(t, strings.HasPrefix(err.Error(), "support: send telegram message:"), "got %q", err.Error())
	})

	t.Run("telegram API ok:false is wrapped", func(t *testing.T) {
		fpg := newFakePG(t, userColumns(), userRow("user-1", "Jane", "jane@example.com"), "")
		defer fpg.Close()
		pool := newPool(t, fpg.dsn())
		defer pool.Close()

		tr := &captureTransport{statusCode: http.StatusOK, respBody: `{"ok":false,"description":"chat not found"}`}
		withTelegramTransport(t, tr)

		svc := NewSupportService(telegram.NewClient("tok", "chat"), userRepo.NewUserRepository(pool))
		err := svc.Submit(context.Background(), "user-1", "subject here", "message body long enough", "")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "support: send telegram message:")
	})
}

func TestNewSupportService(t *testing.T) {
	svc := NewSupportService(telegram.NewClient("t", "c"), userRepo.NewUserRepository(nil))
	require.NotNil(t, svc)
}
